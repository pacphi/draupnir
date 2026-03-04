package llm

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const ollamaDefaultAddr = "http://localhost:11434"

// OllamaDetector monitors local Ollama inference at localhost:11434.
// It periodically checks if Ollama is running and polls its API for usage.
type OllamaDetector struct {
	addr    string
	usageCh chan<- UsageEvent
	logger  *slog.Logger
}

// NewOllamaDetector creates a detector for local Ollama inference.
func NewOllamaDetector(usageCh chan<- UsageEvent, logger *slog.Logger) *OllamaDetector {
	return &OllamaDetector{
		addr:    ollamaDefaultAddr,
		usageCh: usageCh,
		logger:  logger,
	}
}

// Run starts the Ollama detector loop. It checks for Ollama every 30 seconds.
func (d *OllamaDetector) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Check immediately on start
	if d.isRunning(ctx) {
		d.logger.Info("Ollama detected at localhost:11434", "tier", "ollama")
	}

	for {
		select {
		case <-ticker.C:
			if d.isRunning(ctx) {
				d.checkRunningModels(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *OllamaDetector) isRunning(ctx context.Context) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", d.addr+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req) //nolint:gosec // URL is from trusted constant ollamaDefaultAddr
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == 200
}

func (d *OllamaDetector) checkRunningModels(ctx context.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", d.addr+"/api/ps", nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req) //nolint:gosec // URL is from trusted constant ollamaDefaultAddr
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var psResp struct {
		Models []struct {
			Name    string `json:"name"`
			Model   string `json:"model"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&psResp); err != nil {
		return
	}

	for _, m := range psResp.Models {
		d.logger.Debug("Ollama model running",
			"model", m.Name,
			"parameter_size", m.Details.ParameterSize,
		)
	}
}

// InterceptOllamaResponse parses an Ollama API response body for token usage.
// This is called when the proxy forwards requests to localhost:11434.
func InterceptOllamaResponse(body []byte, usageCh chan<- UsageEvent, logger *slog.Logger) {
	inputTokens, outputTokens, _, _, model := parseOllamaUsage(body)
	if inputTokens == 0 && outputTokens == 0 {
		return
	}

	event := UsageEvent{
		Provider:     "ollama",
		Model:        model,
		Operation:    "chat",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUsd:      0, // Local inference — $0 by default
		CaptureTier:  "ollama",
		Timestamp:    time.Now(),
	}

	select {
	case usageCh <- event:
		logger.Debug("Ollama usage captured",
			"model", model,
			"input_tokens", inputTokens,
			"output_tokens", outputTokens,
		)
	default:
		logger.Warn("LLM usage channel full, dropping Ollama event")
	}
}

// OllamaProxyAddr returns the address string for Ollama proxy integration.
func OllamaProxyAddr() string {
	return "http://localhost:11434"
}
