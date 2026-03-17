// Package extensions discovers installed Sindri extensions from the status ledger.
package extensions

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// ledgerEntry represents a single line in the Sindri status ledger.
type ledgerEntry struct {
	ExtensionName string `json:"extension_name"`
	StateAfter    string `json:"state_after"`
}

// Discover reads ~/.sindri/status_ledger.jsonl and returns the names of
// extensions whose latest state is "Installed". If the ledger cannot be
// read (e.g. Sindri CLI not present), it returns an empty slice.
func Discover(logger *slog.Logger) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("extensions: cannot determine home directory", "error", err)
		return nil
	}

	ledgerPath := filepath.Join(home, ".sindri", "status_ledger.jsonl")
	f, err := os.Open(ledgerPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("extensions: status ledger not found, no extensions to report", "path", ledgerPath)
		} else {
			logger.Warn("extensions: cannot open status ledger", "path", ledgerPath, "error", err)
		}
		return nil
	}
	defer func() { _ = f.Close() }()

	// Keep the latest state per extension name
	latest := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry ledgerEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			logger.Debug("extensions: skipping malformed ledger line", "error", err)
			continue
		}
		if entry.ExtensionName != "" && entry.StateAfter != "" {
			latest[entry.ExtensionName] = entry.StateAfter
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Warn("extensions: error reading status ledger", "error", err)
	}

	var installed []string
	for name, state := range latest {
		if state == "Installed" {
			installed = append(installed, name)
		}
	}

	if len(installed) > 0 {
		logger.Info("extensions: discovered installed extensions", "count", len(installed))
	}
	return installed
}
