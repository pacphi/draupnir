package llm

import (
	"context"
	"log/slog"
	"time"

	"github.com/pacphi/draupnir/internal/llm/ebpf"
)

// AdapterMode determines which capture tiers are active.
type AdapterMode string

const (
	ModeAuto  AdapterMode = "auto"  // Try eBPF, fallback to proxy
	ModeProxy AdapterMode = "proxy" // Tier 1 only
	ModeEBPF  AdapterMode = "ebpf"  // Tier 2 only (requires Linux 5.8+)
	ModeNone  AdapterMode = "none"  // Disabled
)

// Adapter is the top-level LLM traffic interception controller.
// It starts the appropriate capture tiers based on the configured mode
// and system capabilities.
type Adapter struct {
	mode           AdapterMode
	proxyPort      int
	reportInterval time.Duration
	sender         Sender
	logger         *slog.Logger
}

// NewAdapter creates a new LLM adapter.
func NewAdapter(mode string, proxyPort int, reportInterval time.Duration, sender Sender, logger *slog.Logger) *Adapter {
	m := AdapterMode(mode)
	switch m {
	case ModeAuto, ModeProxy, ModeEBPF, ModeNone:
		// valid
	default:
		m = ModeAuto
	}
	return &Adapter{
		mode:           m,
		proxyPort:      proxyPort,
		reportInterval: reportInterval,
		sender:         sender,
		logger:         logger,
	}
}

// Run starts all configured capture tiers and the usage reporter.
// This is a blocking call — run in a goroutine.
func (a *Adapter) Run(ctx context.Context) {
	if a.mode == ModeNone {
		a.logger.Info("LLM adapter disabled (SINDRI_LLM_ADAPTER=none)")
		return
	}

	// Buffered channel for usage events from all tiers
	usageCh := make(chan UsageEvent, 1024)

	// Start the reporter (batches events and sends to Mimir)
	reporter := NewReporter(a.sender, usageCh, a.reportInterval, a.logger)
	go reporter.Run(ctx)

	// Tier 1: Local HTTP reverse proxy
	proxyStarted := false
	if a.mode == ModeAuto || a.mode == ModeProxy {
		proxy := NewProxy(a.proxyPort, usageCh, a.logger)
		go func() {
			if err := proxy.Start(ctx); err != nil {
				a.logger.Warn("LLM proxy failed to start", "error", err)
			}
		}()
		proxyStarted = true
	}

	// Tier 2: eBPF SSL interception (Linux 5.8+ only)
	ebpfStarted := false
	if a.mode == ModeAuto || a.mode == ModeEBPF {
		if ebpf.Available() {
			a.logger.Info("eBPF Tier 2 available — starting SSL uprobe interception")
			interceptCh := make(chan ebpf.InterceptedRequest, 256)
			interceptor := ebpf.NewSSLInterceptor(interceptCh, a.logger)

			// Process intercepted requests in a separate goroutine
			go a.processEBPFEvents(ctx, interceptCh, usageCh)

			go func() {
				if err := interceptor.Run(ctx); err != nil {
					a.logger.Warn("eBPF Tier 2 failed — Tier 1 proxy remains active",
						"error", err,
					)
				}
			}()
			ebpfStarted = true
		} else {
			if a.mode == ModeEBPF {
				a.logger.Warn("eBPF requested but not available (requires Linux 5.8+ with BTF)")
			} else {
				a.logger.Info("eBPF not available on this system — using Tier 1 proxy only")
			}
		}
	}

	// Ollama detector (always active when adapter is enabled)
	ollamaDetector := NewOllamaDetector(usageCh, a.logger)
	go ollamaDetector.Run(ctx)

	a.logger.Info("LLM adapter started",
		"mode", a.mode,
		"proxy", proxyStarted,
		"ebpf", ebpfStarted,
		"ollama_detector", true,
		"proxy_port", a.proxyPort,
		"report_interval", a.reportInterval,
	)

	// Block until context cancelled
	<-ctx.Done()
	a.logger.Info("LLM adapter shutting down")
}

// processEBPFEvents converts eBPF intercepted HTTP requests into UsageEvents.
func (a *Adapter) processEBPFEvents(ctx context.Context, interceptCh <-chan ebpf.InterceptedRequest, usageCh chan<- UsageEvent) {
	providers := make(map[string]ProviderConfig)
	for _, p := range KnownProviders() {
		for _, h := range p.Hosts {
			providers[h] = p
		}
	}

	for {
		select {
		case req, ok := <-interceptCh:
			if !ok {
				return
			}
			provider, found := providers[req.Host]
			if !found {
				continue
			}
			if provider.ParseUsage == nil || len(req.ResponseBody) == 0 {
				continue
			}

			inputTokens, outputTokens, cacheRead, cacheWrite, model := provider.ParseUsage(req.ResponseBody)
			if inputTokens == 0 && outputTokens == 0 {
				continue
			}

			cost := ComputeCost(provider.Name, model, inputTokens, outputTokens)

			event := UsageEvent{
				Provider:         provider.Name,
				Model:            model,
				Operation:        "chat",
				InputTokens:      inputTokens,
				OutputTokens:     outputTokens,
				CacheReadTokens:  cacheRead,
				CacheWriteTokens: cacheWrite,
				CostUsd:          cost,
				CaptureTier:      "ebpf",
				Timestamp:        req.Timestamp,
			}

			select {
			case usageCh <- event:
				a.logger.Debug("eBPF usage captured",
					"provider", provider.Name,
					"model", model,
					"input_tokens", inputTokens,
					"output_tokens", outputTokens,
					"pid", req.PID,
				)
			default:
				a.logger.Warn("Usage channel full, dropping eBPF event")
			}

		case <-ctx.Done():
			return
		}
	}
}
