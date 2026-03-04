package llm

// LLMModelPrice holds per-million-token pricing for a model.
type LLMModelPrice struct {
	Provider         string
	Model            string
	InputPerMillion  float64
	OutputPerMillion float64
}

// pricingTable is the embedded LLM pricing table, synced with Mimir's llm-pricing.ts.
var pricingTable = []LLMModelPrice{
	// Anthropic
	{Provider: "anthropic", Model: "claude-opus-4", InputPerMillion: 15, OutputPerMillion: 75},
	{Provider: "anthropic", Model: "claude-sonnet-4", InputPerMillion: 3, OutputPerMillion: 15},
	{Provider: "anthropic", Model: "claude-haiku-4", InputPerMillion: 0.8, OutputPerMillion: 4},
	{Provider: "anthropic", Model: "claude-3-5-sonnet", InputPerMillion: 3, OutputPerMillion: 15},
	{Provider: "anthropic", Model: "claude-3-5-haiku", InputPerMillion: 0.8, OutputPerMillion: 4},
	{Provider: "anthropic", Model: "claude-3-opus", InputPerMillion: 15, OutputPerMillion: 75},
	{Provider: "anthropic", Model: "claude-3-haiku", InputPerMillion: 0.25, OutputPerMillion: 1.25},
	// OpenAI
	{Provider: "openai", Model: "gpt-4o", InputPerMillion: 2.5, OutputPerMillion: 10},
	{Provider: "openai", Model: "gpt-4o-mini", InputPerMillion: 0.15, OutputPerMillion: 0.6},
	{Provider: "openai", Model: "o3", InputPerMillion: 10, OutputPerMillion: 40},
	{Provider: "openai", Model: "o3-mini", InputPerMillion: 1.1, OutputPerMillion: 4.4},
	{Provider: "openai", Model: "o4-mini", InputPerMillion: 1.1, OutputPerMillion: 4.4},
	// Google
	{Provider: "google", Model: "gemini-2.5-pro", InputPerMillion: 1.25, OutputPerMillion: 10},
	{Provider: "google", Model: "gemini-2.5-flash", InputPerMillion: 0.15, OutputPerMillion: 0.6},
	{Provider: "google", Model: "gemini-2.0-flash", InputPerMillion: 0.1, OutputPerMillion: 0.4},
	// Groq
	{Provider: "groq", Model: "llama-3.3-70b", InputPerMillion: 0.59, OutputPerMillion: 0.79},
	{Provider: "groq", Model: "llama-3.1-8b", InputPerMillion: 0.05, OutputPerMillion: 0.08},
	// Mistral
	{Provider: "mistral", Model: "mistral-large", InputPerMillion: 2, OutputPerMillion: 6},
	{Provider: "mistral", Model: "codestral", InputPerMillion: 0.3, OutputPerMillion: 0.9},
	// xAI
	{Provider: "xai", Model: "grok-3", InputPerMillion: 3, OutputPerMillion: 15},
	{Provider: "xai", Model: "grok-3-mini", InputPerMillion: 0.3, OutputPerMillion: 0.5},
	// Ollama (local — $0)
	{Provider: "ollama", Model: "*", InputPerMillion: 0, OutputPerMillion: 0},
}

// ComputeCost calculates the USD cost for a single LLM API call.
func ComputeCost(provider, model string, inputTokens, outputTokens int) float64 {
	p := findPricing(provider, model)
	if p == nil {
		return 0
	}
	cost := float64(inputTokens)/1_000_000*p.InputPerMillion +
		float64(outputTokens)/1_000_000*p.OutputPerMillion
	return cost
}

func findPricing(provider, model string) *LLMModelPrice {
	// Exact match
	for i := range pricingTable {
		if pricingTable[i].Provider == provider && pricingTable[i].Model == model {
			return &pricingTable[i]
		}
	}
	// Prefix match (model starts with table entry)
	var best *LLMModelPrice
	bestLen := 0
	for i := range pricingTable {
		if pricingTable[i].Provider == provider && pricingTable[i].Model != "*" {
			if len(model) >= len(pricingTable[i].Model) && model[:len(pricingTable[i].Model)] == pricingTable[i].Model {
				if len(pricingTable[i].Model) > bestLen {
					best = &pricingTable[i]
					bestLen = len(pricingTable[i].Model)
				}
			}
		}
	}
	if best != nil {
		return best
	}
	// Wildcard match
	for i := range pricingTable {
		if pricingTable[i].Provider == provider && pricingTable[i].Model == "*" {
			return &pricingTable[i]
		}
	}
	return nil
}
