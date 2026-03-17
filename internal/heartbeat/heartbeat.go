// Package heartbeat manages periodic pings from the agent to the Console.
package heartbeat

import (
	"context"
	"log/slog"
	"time"

	"github.com/pacphi/draupnir/internal/extensions"
	"github.com/pacphi/draupnir/pkg/protocol"
)

// Sender is any type that can send an Envelope over the transport layer.
type Sender interface {
	Send(env protocol.Envelope) error
}

// Manager ticks at a fixed interval, sending HeartbeatPayloads via a Sender.
type Manager struct {
	instanceID      string
	distro          string
	interval        time.Duration
	sender          Sender
	startTime       time.Time
	logger          *slog.Logger
	extensions      []string
	extRefreshCount int
}

// New creates a Manager. interval must be > 0.
func New(instanceID string, distro string, interval time.Duration, sender Sender, logger *slog.Logger) *Manager {
	return &Manager{
		instanceID: instanceID,
		distro:     distro,
		interval:   interval,
		sender:     sender,
		startTime:  time.Now(),
		logger:     logger,
	}
}

// Run blocks, sending heartbeats until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Send the first heartbeat immediately so the Console sees us right away.
	m.sendOnce()

	for {
		select {
		case <-ticker.C:
			m.sendOnce()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) sendOnce() {
	// Refresh extensions every 10 heartbeats (~5 min at 30s interval)
	if m.extRefreshCount%10 == 0 {
		m.extensions = extensions.Discover(m.logger)
	}
	m.extRefreshCount++

	uptime := int64(time.Since(m.startTime).Seconds())
	env := protocol.Envelope{
		Type: protocol.MsgHeartbeat,
		Payload: protocol.HeartbeatPayload{
			InstanceID: m.instanceID,
			Timestamp:  time.Now().UTC(),
			Uptime:     uptime,
			Distro:     m.distro,
			Extensions: m.extensions,
		},
	}
	if err := m.sender.Send(env); err != nil {
		m.logger.Warn("heartbeat send failed", "error", err)
	} else {
		m.logger.Debug("heartbeat sent", "uptime_seconds", uptime)
	}
}
