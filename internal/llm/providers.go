// Package llm provides LLM API traffic interception for token usage tracking.
package llm

import (
	"encoding/json"
	"strings"
)

// ProviderConfig describes how to extract token usage from a provider's API response.
type ProviderConfig struct {
	Name       string
	Hosts      []string
	TargetURL  string // Real API base URL to forward to
	ParseUsage func(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string)
}

// KnownProviders returns the configured LLM providers for the proxy.
func KnownProviders() []ProviderConfig {
	return []ProviderConfig{
		{
			Name:       "anthropic",
			Hosts:      []string{"api.anthropic.com"},
			TargetURL:  "https://api.anthropic.com",
			ParseUsage: parseAnthropicUsage,
		},
		{
			Name:       "openai",
			Hosts:      []string{"api.openai.com"},
			TargetURL:  "https://api.openai.com",
			ParseUsage: parseOpenAIUsage,
		},
		{
			Name:       "google",
			Hosts:      []string{"generativelanguage.googleapis.com"},
			TargetURL:  "https://generativelanguage.googleapis.com",
			ParseUsage: parseGoogleUsage,
		},
		{
			Name:       "groq",
			Hosts:      []string{"api.groq.com"},
			TargetURL:  "https://api.groq.com",
			ParseUsage: parseOpenAIUsage, // Same format as OpenAI
		},
		{
			Name:       "mistral",
			Hosts:      []string{"api.mistral.ai"},
			TargetURL:  "https://api.mistral.ai",
			ParseUsage: parseOpenAIUsage, // Same format as OpenAI
		},
		{
			Name:       "xai",
			Hosts:      []string{"api.x.ai"},
			TargetURL:  "https://api.x.ai",
			ParseUsage: parseOpenAIUsage, // Same format as OpenAI
		},
		{
			Name:       "cohere",
			Hosts:      []string{"api.cohere.ai", "api.cohere.com"},
			TargetURL:  "https://api.cohere.com",
			ParseUsage: parseCohereUsage,
		},
		{
			Name:       "together",
			Hosts:      []string{"api.together.xyz"},
			TargetURL:  "https://api.together.xyz",
			ParseUsage: parseOpenAIUsage, // Same format as OpenAI
		},
	}
}

// --- Provider-specific usage parsers ---

// parseAnthropicUsage extracts tokens from Anthropic API response.
// Response format: { "usage": { "input_tokens": N, "output_tokens": N, "cache_creation_input_tokens": N, "cache_read_input_tokens": N }, "model": "..." }
func parseAnthropicUsage(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string) {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0, ""
	}
	return resp.Usage.InputTokens, resp.Usage.OutputTokens,
		resp.Usage.CacheReadInputTokens, resp.Usage.CacheCreationInputTokens,
		resp.Model
}

// parseOpenAIUsage extracts tokens from OpenAI-compatible API responses.
// Format: { "usage": { "prompt_tokens": N, "completion_tokens": N }, "model": "..." }
func parseOpenAIUsage(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string) {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0, ""
	}
	return resp.Usage.PromptTokens, resp.Usage.CompletionTokens, 0, 0, resp.Model
}

// parseGoogleUsage extracts tokens from Google Gemini API responses.
// Format: { "usageMetadata": { "promptTokenCount": N, "candidatesTokenCount": N }, "modelVersion": "..." }
func parseGoogleUsage(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string) {
	var resp struct {
		ModelVersion  string `json:"modelVersion"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0, ""
	}
	return resp.UsageMetadata.PromptTokenCount, resp.UsageMetadata.CandidatesTokenCount, 0, 0, resp.ModelVersion
}

// parseCohereUsage extracts tokens from Cohere API responses.
func parseCohereUsage(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string) {
	var resp struct {
		Meta struct {
			BilledUnits struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"billed_units"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0, ""
	}
	return resp.Meta.BilledUnits.InputTokens, resp.Meta.BilledUnits.OutputTokens, 0, 0, ""
}

// parseOllamaUsage extracts tokens from Ollama API responses.
// Format: { "model": "...", "prompt_eval_count": N, "eval_count": N }
func parseOllamaUsage(body []byte) (inputTokens, outputTokens, cacheRead, cacheWrite int, model string) { //nolint:unparam // cacheRead kept for interface consistency with other parsers
	var resp struct {
		Model           string `json:"model"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0, ""
	}
	return resp.PromptEvalCount, resp.EvalCount, 0, 0, resp.Model
}

// isLLMAPIHost checks if a hostname matches a known LLM API provider.
func isLLMAPIHost(host string) (string, bool) {
	host = strings.ToLower(host)
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	for _, p := range KnownProviders() {
		for _, h := range p.Hosts {
			if host == h {
				return p.Name, true
			}
		}
	}
	// Check for Bedrock
	if strings.Contains(host, "bedrock-runtime") && strings.Contains(host, "amazonaws.com") {
		return "bedrock", true
	}
	return "", false
}
