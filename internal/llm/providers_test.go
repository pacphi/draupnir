package llm

import (
	"testing"
)

func TestParseAnthropicUsage(t *testing.T) {
	body := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"usage": {
			"input_tokens": 1500,
			"output_tokens": 500,
			"cache_read_input_tokens": 200,
			"cache_creation_input_tokens": 100
		}
	}`)

	input, output, cacheRead, cacheWrite, model := parseAnthropicUsage(body)
	if input != 1500 {
		t.Errorf("input_tokens: expected 1500, got %d", input)
	}
	if output != 500 {
		t.Errorf("output_tokens: expected 500, got %d", output)
	}
	if cacheRead != 200 {
		t.Errorf("cache_read: expected 200, got %d", cacheRead)
	}
	if cacheWrite != 100 {
		t.Errorf("cache_write: expected 100, got %d", cacheWrite)
	}
	if model != "claude-sonnet-4-20250514" {
		t.Errorf("model: expected claude-sonnet-4-20250514, got %s", model)
	}
}

func TestParseOpenAIUsage(t *testing.T) {
	body := []byte(`{
		"model": "gpt-4o-2024-08-06",
		"usage": {
			"prompt_tokens": 2000,
			"completion_tokens": 800
		}
	}`)

	input, output, _, _, model := parseOpenAIUsage(body)
	if input != 2000 {
		t.Errorf("prompt_tokens: expected 2000, got %d", input)
	}
	if output != 800 {
		t.Errorf("completion_tokens: expected 800, got %d", output)
	}
	if model != "gpt-4o-2024-08-06" {
		t.Errorf("model: expected gpt-4o-2024-08-06, got %s", model)
	}
}

func TestParseGoogleUsage(t *testing.T) {
	body := []byte(`{
		"modelVersion": "gemini-2.5-flash",
		"usageMetadata": {
			"promptTokenCount": 3000,
			"candidatesTokenCount": 1200
		}
	}`)

	input, output, _, _, model := parseGoogleUsage(body)
	if input != 3000 {
		t.Errorf("promptTokenCount: expected 3000, got %d", input)
	}
	if output != 1200 {
		t.Errorf("candidatesTokenCount: expected 1200, got %d", output)
	}
	if model != "gemini-2.5-flash" {
		t.Errorf("model: expected gemini-2.5-flash, got %s", model)
	}
}

func TestParseOllamaUsage(t *testing.T) {
	body := []byte(`{
		"model": "llama3:8b",
		"prompt_eval_count": 500,
		"eval_count": 200
	}`)

	input, output, _, _, model := parseOllamaUsage(body)
	if input != 500 {
		t.Errorf("prompt_eval_count: expected 500, got %d", input)
	}
	if output != 200 {
		t.Errorf("eval_count: expected 200, got %d", output)
	}
	if model != "llama3:8b" {
		t.Errorf("model: expected llama3:8b, got %s", model)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	input, output, _, _, model := parseAnthropicUsage([]byte("not json"))
	if input != 0 || output != 0 || model != "" {
		t.Error("expected zeros for invalid JSON")
	}
}

func TestIsLLMAPIHost(t *testing.T) {
	tests := []struct {
		host     string
		expected string
		found    bool
	}{
		{"api.anthropic.com", "anthropic", true},
		{"api.openai.com", "openai", true},
		{"api.groq.com", "groq", true},
		{"api.x.ai", "xai", true},
		{"generativelanguage.googleapis.com", "google", true},
		{"example.com", "", false},
		{"api.anthropic.com:443", "anthropic", true},
		{"bedrock-runtime.us-east-1.amazonaws.com", "bedrock", true},
	}

	for _, tt := range tests {
		name, found := isLLMAPIHost(tt.host)
		if found != tt.found {
			t.Errorf("isLLMAPIHost(%q): found=%v, expected=%v", tt.host, found, tt.found)
		}
		if name != tt.expected {
			t.Errorf("isLLMAPIHost(%q): name=%q, expected=%q", tt.host, name, tt.expected)
		}
	}
}

func TestKnownProviders_Count(t *testing.T) {
	providers := KnownProviders()
	if len(providers) < 8 {
		t.Errorf("expected at least 8 providers, got %d", len(providers))
	}
}
