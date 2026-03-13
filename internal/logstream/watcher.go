// Package logstream watches log files and streams new lines to the console.
package logstream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pacphi/draupnir/pkg/protocol"
)

const (
	pollInterval     = 250 * time.Millisecond
	batchFlushDelay  = 100 * time.Millisecond
	batchMaxLines    = 50
	retryNotFoundSec = 5
)

// levelPattern matches common log level indicators like [ERROR], [WARN], etc.
var levelPattern = regexp.MustCompile(`\[(ERROR|WARN|WARNING|INFO|DEBUG)\]`)

// Sender is any type that can send an Envelope over the transport layer.
type Sender interface {
	Send(env protocol.Envelope) error
}

// Watcher monitors log files and sends new lines to the console.
type Watcher struct {
	mu       sync.Mutex
	basePath string
	sender   Sender
	logger   *slog.Logger
	tailers  map[string]*fileTailer
}

// NewWatcher creates a Watcher rooted at basePath.
// If basePath is empty, it defaults to ~/.sindri/logs/.
func NewWatcher(basePath string, sender Sender, logger *slog.Logger) *Watcher {
	if basePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/"
		}
		basePath = filepath.Join(home, ".sindri", "logs")
	}
	return &Watcher{
		basePath: basePath,
		sender:   sender,
		logger:   logger,
		tailers:  make(map[string]*fileTailer),
	}
}

// Subscribe starts tailing the specified files. Paths are relative to basePath.
func (w *Watcher) Subscribe(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, p := range paths {
		if err := validatePath(p); err != nil {
			w.logger.Warn("log subscribe: invalid path", "path", p, "error", err)
			continue
		}
		if _, exists := w.tailers[p]; exists {
			w.logger.Debug("log subscribe: already tailing", "path", p)
			continue
		}

		absPath := filepath.Join(w.basePath, p)
		ctx, cancel := context.WithCancel(context.Background())
		ft := &fileTailer{
			relPath: p,
			absPath: absPath,
			sender:  w.sender,
			logger:  w.logger,
			cancel:  cancel,
		}
		w.tailers[p] = ft
		go ft.run(ctx)
		w.logger.Info("log subscribe: started tailing", "path", p)
	}
}

// Unsubscribe stops tailing specified files. If paths is empty, stops all.
func (w *Watcher) Unsubscribe(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(paths) == 0 {
		for p, ft := range w.tailers {
			ft.cancel()
			delete(w.tailers, p)
		}
		w.logger.Info("log unsubscribe: stopped all tailers")
		return
	}

	for _, p := range paths {
		if ft, ok := w.tailers[p]; ok {
			ft.cancel()
			delete(w.tailers, p)
			w.logger.Info("log unsubscribe: stopped tailing", "path", p)
		}
	}
}

// Close stops all active tailers and cleans up.
func (w *Watcher) Close() {
	w.Unsubscribe(nil)
}

// validatePath rejects paths that escape the base directory.
func validatePath(p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}
	if filepath.IsAbs(p) {
		return fmt.Errorf("absolute path not allowed")
	}
	cleaned := filepath.Clean(p)
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	return nil
}

// fileTailer reads new lines appended to a single log file.
type fileTailer struct {
	relPath string
	absPath string
	sender  Sender
	logger  *slog.Logger
	cancel  context.CancelFunc
}

func (ft *fileTailer) run(ctx context.Context) {
	// Seek to the end of the file on start.
	var offset int64
	if info, err := os.Stat(ft.absPath); err == nil {
		offset = info.Size()
	}

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	var batch []protocol.LogLineEntry

	flushTimer := time.NewTimer(batchFlushDelay)
	flushTimer.Stop()
	defer flushTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			ft.flush(batch)
			return

		case <-flushTimer.C:
			batch = ft.flush(batch)

		case <-pollTicker.C:
			info, err := os.Stat(ft.absPath)
			if err != nil {
				if os.IsNotExist(err) {
					ft.logger.Debug("log file not found, will retry", "path", ft.relPath)
					offset = 0
					// Wait longer before retrying.
					select {
					case <-time.After(retryNotFoundSec * time.Second):
					case <-ctx.Done():
						ft.flush(batch)
						return
					}
				}
				continue
			}

			currentSize := info.Size()

			// Handle file truncation (log rotation).
			if currentSize < offset {
				ft.logger.Info("log file truncated, re-reading from start", "path", ft.relPath)
				offset = 0
			}

			if currentSize == offset {
				continue
			}

			// Read new content.
			newEntries, newOffset := ft.readNewLines(offset, currentSize)
			offset = newOffset

			batch = append(batch, newEntries...)

			if len(batch) >= batchMaxLines {
				batch = ft.flush(batch)
			} else if len(batch) > 0 {
				flushTimer.Reset(batchFlushDelay)
			}
		}
	}
}

func (ft *fileTailer) readNewLines(offset, limit int64) ([]protocol.LogLineEntry, int64) {
	f, err := os.Open(ft.absPath)
	if err != nil {
		ft.logger.Warn("failed to open log file", "path", ft.relPath, "error", err)
		return nil, offset
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		ft.logger.Warn("failed to seek log file", "path", ft.relPath, "error", err)
		return nil, offset
	}

	reader := bufio.NewReader(io.LimitReader(f, limit-offset))
	var entries []protocol.LogLineEntry
	bytesRead := offset

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			bytesRead += int64(len(line))
			line = strings.TrimRight(line, "\n\r")
			if line != "" {
				entries = append(entries, protocol.LogLineEntry{
					Line:      line,
					Timestamp: time.Now().UnixMilli(),
					Level:     parseLevel(line),
				})
			}
		}
		if err != nil {
			break
		}
	}

	return entries, bytesRead
}

func (ft *fileTailer) flush(batch []protocol.LogLineEntry) []protocol.LogLineEntry { //nolint:unparam // nil return is intentional for slice reuse pattern
	if len(batch) == 0 {
		return nil
	}

	if len(batch) == 1 {
		env := protocol.Envelope{
			ProtocolVersion: protocol.ProtocolVersion,
			Type:            protocol.MsgLogLine,
			Payload: protocol.LogLinePayload{
				Path:      ft.relPath,
				Line:      batch[0].Line,
				Timestamp: batch[0].Timestamp,
				Level:     batch[0].Level,
			},
		}
		if err := ft.sender.Send(env); err != nil {
			ft.logger.Warn("log line send failed", "path", ft.relPath, "error", err)
		}
	} else {
		env := protocol.Envelope{
			ProtocolVersion: protocol.ProtocolVersion,
			Type:            protocol.MsgLogBatch,
			Payload: protocol.LogBatchPayload{
				Path:  ft.relPath,
				Lines: batch,
			},
		}
		if err := ft.sender.Send(env); err != nil {
			ft.logger.Warn("log batch send failed", "path", ft.relPath, "error", err)
		}
	}

	return nil
}

// parseLevel extracts a log level from the line content.
func parseLevel(line string) string {
	match := levelPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return ""
	}
	level := strings.ToUpper(match[1])
	if level == "WARNING" {
		return "WARN"
	}
	return level
}
