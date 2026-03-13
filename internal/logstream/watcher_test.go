package logstream

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pacphi/draupnir/pkg/protocol"
)

// captureSender records envelopes it receives.
type captureSender struct {
	mu        sync.Mutex
	envelopes []protocol.Envelope
}

func (s *captureSender) Send(env protocol.Envelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.envelopes = append(s.envelopes, env)
	return nil
}

func (s *captureSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.envelopes)
}

func (s *captureSender) all() []protocol.Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]protocol.Envelope, len(s.envelopes))
	copy(out, s.envelopes)
	return out
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestWatcher_SubscribeAndReceiveLines(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "app.log")

	// Create the file so the tailer can open it.
	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &captureSender{}
	w := NewWatcher(dir, sender, testLogger())
	defer w.Close()

	w.Subscribe([]string{"app.log"})

	// Give the tailer time to start and poll once.
	time.Sleep(100 * time.Millisecond)

	// Append a line to the log file.
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(f, "[INFO] something happened")
	f.Close()

	// Wait for the tailer to pick it up and flush.
	time.Sleep(500 * time.Millisecond)

	if sender.count() == 0 {
		t.Fatal("expected at least one envelope to be sent")
	}

	envs := sender.all()
	found := false
	for _, env := range envs {
		if env.Type == protocol.MsgLogLine {
			found = true
			payload, ok := env.Payload.(protocol.LogLinePayload)
			if !ok {
				t.Fatalf("payload type = %T, want LogLinePayload", env.Payload)
			}
			if payload.Path != "app.log" {
				t.Errorf("path = %q, want app.log", payload.Path)
			}
			if payload.Level != "INFO" {
				t.Errorf("level = %q, want INFO", payload.Level)
			}
		}
	}
	if !found {
		// Might have been sent as a batch.
		for _, env := range envs {
			if env.Type == protocol.MsgLogBatch {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected a log:line or log:batch envelope")
	}
}

func TestWatcher_BatchFlush(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "bulk.log")

	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &captureSender{}
	w := NewWatcher(dir, sender, testLogger())
	defer w.Close()

	w.Subscribe([]string{"bulk.log"})
	time.Sleep(100 * time.Millisecond)

	// Write 60 lines — should trigger a batch flush at 50.
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		fmt.Fprintf(f, "line %d\n", i)
	}
	f.Close()

	time.Sleep(600 * time.Millisecond)

	envs := sender.all()
	foundBatch := false
	for _, env := range envs {
		if env.Type == protocol.MsgLogBatch {
			foundBatch = true
			break
		}
	}
	if !foundBatch {
		t.Error("expected at least one log:batch envelope for 60 lines")
	}
}

func TestWatcher_FileTruncation(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "rotate.log")

	// Write initial content.
	if err := os.WriteFile(logFile, []byte("old line 1\nold line 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &captureSender{}
	w := NewWatcher(dir, sender, testLogger())
	defer w.Close()

	w.Subscribe([]string{"rotate.log"})
	time.Sleep(400 * time.Millisecond)

	// Truncate the file (simulate log rotation).
	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(400 * time.Millisecond)

	// Write new content after truncation.
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(f, "[ERROR] after rotation")
	f.Close()

	time.Sleep(500 * time.Millisecond)

	envs := sender.all()
	found := false
	for _, env := range envs {
		switch env.Type {
		case protocol.MsgLogLine:
			payload, ok := env.Payload.(protocol.LogLinePayload)
			if ok && payload.Level == "ERROR" {
				found = true
			}
		case protocol.MsgLogBatch:
			payload, ok := env.Payload.(protocol.LogBatchPayload)
			if ok {
				for _, entry := range payload.Lines {
					if entry.Level == "ERROR" {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Error("expected to receive the [ERROR] line written after truncation")
	}
}

func TestWatcher_PathValidation(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"app.log", false},
		{"sub/dir/app.log", false},
		{"../escape.log", true},
		{"/etc/passwd", true},
		{"foo/../../bar.log", true},
		{"", true},
	}

	for _, tt := range tests {
		err := validatePath(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("validatePath(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
		}
	}
}

func TestWatcher_Unsubscribe(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "unsub.log")

	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &captureSender{}
	w := NewWatcher(dir, sender, testLogger())
	defer w.Close()

	w.Subscribe([]string{"unsub.log"})
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe.
	w.Unsubscribe([]string{"unsub.log"})
	time.Sleep(100 * time.Millisecond)

	beforeCount := sender.count()

	// Write after unsubscribe — should NOT be received.
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(f, "[WARN] should not appear")
	f.Close()

	time.Sleep(500 * time.Millisecond)

	if sender.count() != beforeCount {
		t.Errorf("expected no new envelopes after unsubscribe, got %d new", sender.count()-beforeCount)
	}
}

func TestWatcher_SubscribeIdempotent(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "idem.log")

	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &captureSender{}
	w := NewWatcher(dir, sender, testLogger())
	defer w.Close()

	w.Subscribe([]string{"idem.log"})
	w.Subscribe([]string{"idem.log"}) // duplicate — should be a no-op

	w.mu.Lock()
	count := len(w.tailers)
	w.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 tailer, got %d", count)
	}
}
