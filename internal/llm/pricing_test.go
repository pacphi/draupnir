package llm

import (
	"math"
	"testing"
)

func TestComputeCost_ExactMatch(t *testing.T) {
	cost := ComputeCost("anthropic", "claude-sonnet-4", 1_000_000, 1_000_000)
	// 1M input * $3/M + 1M output * $15/M = $18
	if math.Abs(cost-18.0) > 0.001 {
		t.Errorf("expected $18.00, got $%.6f", cost)
	}
}

func TestComputeCost_PrefixMatch(t *testing.T) {
	// "claude-sonnet-4-20250514" should prefix-match "claude-sonnet-4"
	cost := ComputeCost("anthropic", "claude-sonnet-4-20250514", 1_000_000, 0)
	if math.Abs(cost-3.0) > 0.001 {
		t.Errorf("expected $3.00 for prefix match, got $%.6f", cost)
	}
}

func TestComputeCost_WildcardMatch(t *testing.T) {
	// Ollama uses wildcard "*" → $0
	cost := ComputeCost("ollama", "llama3:8b", 1_000_000, 1_000_000)
	if cost != 0 {
		t.Errorf("expected $0.00 for Ollama, got $%.6f", cost)
	}
}

func TestComputeCost_UnknownProvider(t *testing.T) {
	cost := ComputeCost("unknown-provider", "some-model", 1000, 1000)
	if cost != 0 {
		t.Errorf("expected $0.00 for unknown provider, got $%.6f", cost)
	}
}

func TestComputeCost_ZeroTokens(t *testing.T) {
	cost := ComputeCost("openai", "gpt-4o", 0, 0)
	if cost != 0 {
		t.Errorf("expected $0.00 for zero tokens, got $%.6f", cost)
	}
}

func TestFindPricing_AllProviders(t *testing.T) {
	providers := []string{"anthropic", "openai", "google", "groq", "mistral", "xai", "ollama"}
	for _, p := range providers {
		found := false
		for _, entry := range pricingTable {
			if entry.Provider == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected provider %q in pricing table", p)
		}
	}
}
