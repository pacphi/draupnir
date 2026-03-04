package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// Proxy is the Tier 1 local HTTP reverse proxy that intercepts LLM API traffic.
// Extensions call localhost:PORT/v1/{provider}/... and the proxy forwards to the real API,
// parsing token usage from responses.
type Proxy struct {
	port      int
	providers map[string]ProviderConfig
	usageCh   chan<- UsageEvent
	logger    *slog.Logger
	server    *http.Server
}

// UsageEvent represents a captured LLM API call with token usage.
type UsageEvent struct {
	Provider         string
	Model            string
	Operation        string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	CostUsd          float64
	CaptureTier      string // "proxy" or "ebpf" or "ollama"
	Timestamp        time.Time
}

// NewProxy creates a new Tier 1 LLM proxy.
func NewProxy(port int, usageCh chan<- UsageEvent, logger *slog.Logger) *Proxy {
	provMap := make(map[string]ProviderConfig)
	for _, p := range KnownProviders() {
		provMap[p.Name] = p
	}
	return &Proxy{
		port:      port,
		providers: provMap,
		usageCh:   usageCh,
		logger:    logger,
	}
}

// Start begins listening on the configured port.
func (p *Proxy) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	// Route pattern: /v1/{provider}/...
	mux.HandleFunc("/", p.handleRequest)

	p.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	p.logger.Info("LLM proxy starting", "port", p.port, "tier", "proxy")

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.server.Shutdown(shutCtx)
	}()

	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("llm proxy listen: %w", err)
	}
	return nil
}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse provider from path: /v1/{provider}/...
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 3)
	if len(parts) < 2 {
		http.Error(w, "invalid path — expected /v1/{provider}/...", http.StatusBadRequest)
		return
	}

	providerName := parts[1]
	provider, ok := p.providers[providerName]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown provider: %s", providerName), http.StatusBadRequest)
		return
	}

	// Reconstruct the target URL: strip /v1/{provider} prefix, forward the rest
	targetPath := ""
	if len(parts) >= 3 {
		targetPath = "/" + parts[2]
	}
	targetURL := provider.TargetURL + targetPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Forward the request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy all headers (including auth headers)
	for key, vals := range r.Header {
		for _, val := range vals {
			proxyReq.Header.Add(key, val)
		}
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(proxyReq) //nolint:gosec // URL constructed from trusted provider config
	if err != nil {
		p.logger.Warn("proxy request failed", "provider", providerName, "error", err)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response body for usage extraction
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read upstream response", http.StatusBadGateway)
		return
	}

	// Copy response headers to client
	for key, vals := range resp.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, bytes.NewReader(body))

	// Extract token usage from successful responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && provider.ParseUsage != nil {
		go p.extractAndReport(providerName, provider, body)
	}
}

func (p *Proxy) extractAndReport(providerName string, provider ProviderConfig, body []byte) {
	inputTokens, outputTokens, cacheRead, cacheWrite, model := provider.ParseUsage(body)
	if inputTokens == 0 && outputTokens == 0 {
		return
	}

	cost := ComputeCost(providerName, model, inputTokens, outputTokens)

	event := UsageEvent{
		Provider:         providerName,
		Model:            model,
		Operation:        "chat",
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		CostUsd:          cost,
		CaptureTier:      "proxy",
		Timestamp:        time.Now(),
	}

	select {
	case p.usageCh <- event:
		p.logger.Debug("LLM usage captured",
			"provider", providerName,
			"model", model,
			"input_tokens", inputTokens,
			"output_tokens", outputTokens,
			"cost_usd", cost,
			"tier", "proxy",
		)
	default:
		p.logger.Warn("LLM usage channel full, dropping event")
	}
}
