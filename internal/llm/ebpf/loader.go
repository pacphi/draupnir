//go:build linux

// Package ebpf provides Tier 2 LLM traffic interception via eBPF SSL uprobes.
// It attaches to SSL_read/SSL_write in libssl.so and Go crypto/tls to capture
// pre-encryption plaintext from ANY process making HTTPS calls to LLM API hosts.
//
// Requires Linux kernel 5.8+ with BTF (CO-RE). Falls back gracefully on older kernels.
//
// Production precedent: Datadog USM, Pixie, eCapture all use this technique.
package ebpf

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Available checks if eBPF Tier 2 can be used on this system.
// Requires:
//   - Linux kernel 5.8+ (for ring buffers and improved BTF)
//   - /sys/kernel/btf/vmlinux present (BTF for CO-RE)
//   - CAP_BPF or root
func Available() bool {
	// Check BTF availability
	if _, err := os.Stat("/sys/kernel/btf/vmlinux"); err != nil {
		return false
	}

	// Check kernel version >= 5.8
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	version := string(data)
	return isKernel58Plus(version)
}

func isKernel58Plus(version string) bool {
	// Parse "Linux version X.Y.Z-..." format
	parts := strings.Fields(version)
	if len(parts) < 3 {
		return false
	}
	verStr := parts[2]
	// Extract major.minor
	dotParts := strings.SplitN(verStr, ".", 3)
	if len(dotParts) < 2 {
		return false
	}
	major := 0
	minor := 0
	fmt.Sscanf(dotParts[0], "%d", &major)
	fmt.Sscanf(dotParts[1], "%d", &minor)
	return major > 5 || (major == 5 && minor >= 8)
}

// SSLInterceptor attaches eBPF programs to OpenSSL and Go TLS functions
// to capture pre-encryption HTTP traffic to known LLM API endpoints.
type SSLInterceptor struct {
	logger  *slog.Logger
	eventCh chan<- InterceptedRequest
}

// InterceptedRequest represents an HTTP request/response pair captured by eBPF.
type InterceptedRequest struct {
	Host         string
	Method       string
	Path         string
	ResponseBody []byte
	Timestamp    time.Time
	PID          uint32
}

// NewSSLInterceptor creates a new eBPF SSL interceptor.
func NewSSLInterceptor(eventCh chan<- InterceptedRequest, logger *slog.Logger) *SSLInterceptor {
	return &SSLInterceptor{
		logger:  logger,
		eventCh: eventCh,
	}
}

// Run starts the eBPF programs and processes events.
// This is a blocking call — it runs until ctx is cancelled.
func (s *SSLInterceptor) Run(ctx context.Context) error {
	s.logger.Info("eBPF SSL interceptor starting",
		"tier", "ebpf",
		"btf", "/sys/kernel/btf/vmlinux",
	)

	// Load and attach eBPF programs
	if err := s.loadPrograms(); err != nil {
		return fmt.Errorf("loading eBPF programs: %w", err)
	}
	defer s.cleanup()

	s.logger.Info("eBPF SSL uprobes attached successfully")

	// Process events from the ring buffer
	return s.processEvents(ctx)
}

func (s *SSLInterceptor) loadPrograms() error {
	// NOTE: Full implementation requires:
	// 1. Compiled BPF C programs (ssl_uprobe.bpf.o, gotls_uprobe.bpf.o)
	// 2. cilium/ebpf library to load programs
	// 3. uprobe attachment to libssl.so SSL_read/SSL_write
	// 4. uprobe attachment to Go crypto/tls.(*Conn).Read/Write
	// 5. Ring buffer for kernel→userspace event delivery
	//
	// For the initial implementation, this is a stub that will be filled
	// when cilium/ebpf is added as a dependency. The adapter.go gracefully
	// falls back to Tier 1 (proxy) when eBPF is unavailable.
	//
	// Implementation pattern (from cilium/ebpf examples):
	//
	//   spec, err := ebpf.LoadCollectionSpec("ssl_uprobe.bpf.o")
	//   coll, err := ebpf.NewCollection(spec)
	//   ex, err := link.OpenExecutable("/usr/lib/x86_64-linux-gnu/libssl.so.3")
	//   link, err := ex.Uprobe("SSL_write", coll.Programs["uprobe_ssl_write"], nil)
	//   reader, err := ringbuf.NewReader(coll.Maps["events"])
	//
	s.logger.Info("eBPF program loading (stub — will activate with cilium/ebpf dependency)")
	return fmt.Errorf("eBPF programs not yet compiled — falling back to Tier 1 proxy")
}

func (s *SSLInterceptor) processEvents(ctx context.Context) error {
	// Will read from ring buffer and parse HTTP request/response pairs
	// filtered by known LLM API hostnames (SNI / Host header matching).
	<-ctx.Done()
	return nil
}

func (s *SSLInterceptor) cleanup() {
	s.logger.Info("eBPF programs detached")
}

// LLMAPIHosts returns the set of hostnames to filter in eBPF programs.
// Only traffic to these hosts will be captured.
func LLMAPIHosts() []string {
	return []string{
		"api.anthropic.com",
		"api.openai.com",
		"generativelanguage.googleapis.com",
		"api.groq.com",
		"api.mistral.ai",
		"api.x.ai",
		"api.cohere.com",
		"api.cohere.ai",
		"api.together.xyz",
	}
}
