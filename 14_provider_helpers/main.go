// Tutorial 14: Provider Presets & Resolution Helpers
//
// Demonstrates: PresetFor, ProviderPreset (BaseURL, AuthHeader, Protocol),
//               ResolveProtocol, ResolveBaseURL, ResolveAuthHeader,
//               IsNoAuthProvider

package main

import (
	"fmt"

	"gitlab2024.bds421-cloud.com/bds421/rho/llm"
	_ "gitlab2024.bds421-cloud.com/bds421/rho/llm/provider"
)

func main() {
	// --- Test 1: PresetFor all known providers ---
	fmt.Println("=== Provider Presets ===")
	providers := []string{
		"anthropic", "gemini", "openai", "xai", "groq",
		"cerebras", "mistral", "openrouter", "ollama", "vllm", "lmstudio",
		"unknown_provider",
	}
	for _, prov := range providers {
		preset, ok := llm.PresetFor(prov)
		if !ok {
			fmt.Printf("  %-14s -> not found\n", prov)
			continue
		}
		fmt.Printf("  %-14s -> BaseURL: %-45s Auth: %-10s Protocol: %s\n",
			prov, preset.BaseURL, preset.AuthHeader, preset.Protocol)
	}
	fmt.Println()

	// --- Test 2: IsNoAuthProvider ---
	fmt.Println("=== No-Auth Providers ===")
	testProviders := []string{"anthropic", "gemini", "ollama", "vllm", "lmstudio", "openai", "custom"}
	for _, prov := range testProviders {
		noAuth := llm.IsNoAuthProvider(prov)
		fmt.Printf("  %-14s needs auth: %v\n", prov, !noAuth)
	}
	fmt.Println()

	// --- Test 3: ResolveProtocol ---
	fmt.Println("=== Protocol Resolution ===")
	configs := []llm.Config{
		{Provider: "anthropic"},
		{Provider: "gemini"},
		{Provider: "openai"},
		{Provider: "ollama"},
		{Provider: "xai"},
		{Provider: "custom"},
	}
	for _, cfg := range configs {
		protocol := llm.ResolveProtocol(cfg)
		fmt.Printf("  %-14s -> protocol: %s\n", cfg.Provider, protocol)
	}
	fmt.Println()

	// --- Test 4: ResolveBaseURL with and without override ---
	fmt.Println("=== BaseURL Resolution ===")
	{
		// Default — uses preset
		cfg := llm.Config{Provider: "anthropic"}
		fmt.Printf("  anthropic (default):  %s\n", llm.ResolveBaseURL(cfg))

		// With override
		cfg = llm.Config{Provider: "anthropic", BaseURL: "https://my-proxy.example.com/v1"}
		fmt.Printf("  anthropic (override): %s\n", llm.ResolveBaseURL(cfg))

		// Ollama default
		cfg = llm.Config{Provider: "ollama"}
		fmt.Printf("  ollama (default):     %s\n", llm.ResolveBaseURL(cfg))
	}
	fmt.Println()

	// --- Test 5: ResolveAuthHeader ---
	fmt.Println("=== Auth Header Resolution ===")
	{
		cfg := llm.Config{Provider: "anthropic"}
		fmt.Printf("  anthropic (default):  %s\n", llm.ResolveAuthHeader(cfg))

		cfg = llm.Config{Provider: "openai"}
		fmt.Printf("  openai (default):     %s\n", llm.ResolveAuthHeader(cfg))

		cfg = llm.Config{Provider: "openai", AuthHeader: "X-Custom-Key"}
		fmt.Printf("  openai (override):    %s\n", llm.ResolveAuthHeader(cfg))

		cfg = llm.Config{Provider: "gemini"}
		fmt.Printf("  gemini (default):     %s\n", llm.ResolveAuthHeader(cfg))
	}
}
