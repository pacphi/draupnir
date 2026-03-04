package llm

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pacphi/draupnir/pkg/protocol"
)

// Sender is the interface for sending WebSocket envelopes.
type Sender interface {
	Send(env protocol.Envelope) error
}

// Reporter batches UsageEvents and sends them to Mimir via WebSocket at a configured interval.
type Reporter struct {
	sender   Sender
	usageCh  <-chan UsageEvent
	interval time.Duration
	logger   *slog.Logger

	mu      sync.Mutex
	pending []UsageEvent
}

// NewReporter creates a new LLM usage reporter.
func NewReporter(sender Sender, usageCh <-chan UsageEvent, interval time.Duration, logger *slog.Logger) *Reporter {
	return &Reporter{
		sender:   sender,
		usageCh:  usageCh,
		interval: interval,
		logger:   logger,
	}
}

// Run starts the reporter loop. It collects events from usageCh and flushes
// them to Mimir in batches at the configured interval.
func (r *Reporter) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-r.usageCh:
			if !ok {
				r.flush()
				return
			}
			r.mu.Lock()
			r.pending = append(r.pending, event)
			r.mu.Unlock()

		case <-ticker.C:
			r.flush()

		case <-ctx.Done():
			r.flush()
			return
		}
	}
}

func (r *Reporter) flush() {
	r.mu.Lock()
	if len(r.pending) == 0 {
		r.mu.Unlock()
		return
	}
	batch := r.pending
	r.pending = nil
	r.mu.Unlock()

	records := make([]protocol.LLMUsageRecord, len(batch))
	for i, e := range batch {
		records[i] = protocol.LLMUsageRecord{
			Provider:         e.Provider,
			Model:            e.Model,
			Operation:        e.Operation,
			InputTokens:      e.InputTokens,
			OutputTokens:     e.OutputTokens,
			CacheReadTokens:  e.CacheReadTokens,
			CacheWriteTokens: e.CacheWriteTokens,
			CostUsd:          e.CostUsd,
			CaptureTier:      e.CaptureTier,
			Ts:               e.Timestamp.UnixMilli(),
		}
	}

	env := protocol.Envelope{
		ProtocolVersion: protocol.ProtocolVersion,
		Type:            protocol.MsgLLMUsageBatch,
		Payload: protocol.LLMUsageBatchPayload{
			Records: records,
		},
	}

	if err := r.sender.Send(env); err != nil {
		r.logger.Warn("failed to send LLM usage batch", "count", len(records), "error", err)
		// Re-queue failed batch
		r.mu.Lock()
		r.pending = append(batch, r.pending...)
		r.mu.Unlock()
	} else {
		r.logger.Info("LLM usage batch sent", "count", len(records))
	}
}
