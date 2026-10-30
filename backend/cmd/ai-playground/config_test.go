package main

import (
	"testing"

	"caveclass/internal/ai/llm"
)

func TestConfigForProviderDeepSeekUsesDedicatedKey(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "sk-deepseek-test")
	t.Setenv("AI_TEAMILY_API_KEY", "tmly-test")
	base := llm.Config{Provider: llm.ProviderMock, ModelPrimary: llm.DefaultModelPrimary}

	cfg, err := configForProvider(base, llm.ProviderDeepSeek)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sk-deepseek-test" {
		t.Fatalf("api key: got %q", cfg.APIKey)
	}
	if cfg.ModelPrimary != llm.DefaultModelDeepSeek {
		t.Fatalf("model: got %q", cfg.ModelPrimary)
	}
}

func TestConfigForProviderTeamilyUsesDedicatedKey(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "sk-deepseek-test")
	t.Setenv("AI_TEAMILY_API_KEY", "tmly-test")
	base := llm.Config{Provider: llm.ProviderMock, ModelPrimary: llm.DefaultModelPrimary}

	cfg, err := configForProvider(base, llm.ProviderTeamily)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "tmly-test" {
		t.Fatalf("api key: got %q", cfg.APIKey)
	}
	if cfg.ModelPrimary != llm.DefaultModelTeamily {
		t.Fatalf("model: got %q", cfg.ModelPrimary)
	}
}

func TestProviderAvailability(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TEAMILY_API_KEY", "")
	base := llm.Config{Provider: llm.ProviderDeepSeek, APIKey: "legacy-deepseek"}

	status := providerAvailability(base)
	if !status.DeepSeek {
		t.Fatal("expected deepseek from base when AI_PROVIDER=deepseek")
	}
	if status.Teamily {
		t.Fatal("teamily should be false")
	}
}

func TestProviderAPIKeyFromSharedEnvWhenProviderMatches(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TEAMILY_API_KEY", "")
	t.Setenv("AI_PROVIDER", "teamily")
	t.Setenv("AI_API_KEY", "shared-key")
	base := llm.Config{Provider: llm.ProviderMock}

	key := providerAPIKey("AI_TEAMILY_API_KEY", base, llm.ProviderTeamily)
	if key != "shared-key" {
		t.Fatalf("key: got %q", key)
	}
}

func TestProviderBaseURLPreservesLoadConfigWhenProviderMatches(t *testing.T) {
	t.Setenv("AI_TEAMILY_BASE_URL", "")
	t.Setenv("AI_PROVIDER", "teamily")
	t.Setenv("AI_BASE_URL", "https://example.test/responses")
	base := llm.Config{Provider: llm.ProviderTeamily, BaseURL: "https://example.test/responses"}

	cfg, err := configForProvider(base, llm.ProviderTeamily)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://example.test/responses" {
		t.Fatalf("base url: got %q", cfg.BaseURL)
	}
}
