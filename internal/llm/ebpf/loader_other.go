//go:build !linux

package ebpf

import (
	"context"
	"log/slog"
	"time"
)

// Available returns false on non-Linux platforms — eBPF is Linux-only.
func Available() bool {
	return false
}

// InterceptedRequest is the event type from eBPF capture.
type InterceptedRequest struct {
	Host         string
	Method       string
	Path         string
	ResponseBody []byte
	Timestamp    time.Time
	PID          uint32
}

// SSLInterceptor is a no-op on non-Linux platforms.
type SSLInterceptor struct{}

// NewSSLInterceptor returns a no-op interceptor on non-Linux.
func NewSSLInterceptor(_ chan<- InterceptedRequest, _ *slog.Logger) *SSLInterceptor {
	return &SSLInterceptor{}
}

// Run is a no-op on non-Linux.
func (s *SSLInterceptor) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// LLMAPIHosts returns the list of LLM API hostnames.
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
