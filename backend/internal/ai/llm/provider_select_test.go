package llm

import (
	"testing"
)

func TestConfigForProviderDeepSeekUsesDedicatedKey(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "sk-deepseek-test")
	t.Setenv("AI_TEAMILY_API_KEY", "tmly-test")
	base := Config{Provider: ProviderMock, ModelPrimary: DefaultModelPrimary}

	cfg, err := ConfigForProvider(base, ProviderDeepSeek)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sk-deepseek-test" {
		t.Fatalf("api key: got %q", cfg.APIKey)
	}
	if cfg.ModelPrimary != DefaultModelDeepSeek {
		t.Fatalf("model: got %q", cfg.ModelPrimary)
	}
}

func TestConfigForProviderTeamilyUsesDedicatedKey(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "sk-deepseek-test")
	t.Setenv("AI_TEAMILY_API_KEY", "tmly-test")
	base := Config{Provider: ProviderMock, ModelPrimary: DefaultModelPrimary}

	cfg, err := ConfigForProvider(base, ProviderTeamily)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "tmly-test" {
		t.Fatalf("api key: got %q", cfg.APIKey)
	}
	if cfg.ModelPrimary != DefaultModelTeamily {
		t.Fatalf("model: got %q", cfg.ModelPrimary)
	}
}

func TestProviderAvailability(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TEAMILY_API_KEY", "")
	base := Config{Provider: ProviderDeepSeek, APIKey: "legacy-deepseek"}

	status := ProviderAvailability(base)
	if !status.DeepSeek {
		t.Fatal("expected deepseek from base when AI_PROVIDER=deepseek")
	}
	if status.Teamily {
		t.Fatal("teamily should be false")
	}
}

func TestGenerateProviderOptionsExposeMockOnlyInMockMode(t *testing.T) {
	if !GenerateProviderOptionsFor(Config{Provider: ProviderMock}).Providers.Mock {
		t.Fatal("mock should be selectable in mock mode")
	}
	if GenerateProviderOptionsFor(Config{Provider: ProviderDeepSeek}).Providers.Mock {
		t.Fatal("mock should not be selectable in real provider mode")
	}
}

func TestProviderAPIKeyFromSharedEnvWhenProviderMatches(t *testing.T) {
	t.Setenv("AI_DEEPSEEK_API_KEY", "")
	t.Setenv("AI_TEAMILY_API_KEY", "")
	t.Setenv("AI_PROVIDER", "teamily")
	t.Setenv("AI_API_KEY", "shared-key")
	base := Config{Provider: ProviderMock}

	key := ProviderAPIKey("AI_TEAMILY_API_KEY", base, ProviderTeamily)
	if key != "shared-key" {
		t.Fatalf("key: got %q", key)
	}
}

func TestProviderBaseURLPreservesLoadConfigWhenProviderMatches(t *testing.T) {
	t.Setenv("AI_TEAMILY_BASE_URL", "")
	t.Setenv("AI_PROVIDER", "teamily")
	t.Setenv("AI_BASE_URL", "https://example.test/responses")
	base := Config{Provider: ProviderTeamily, BaseURL: "https://example.test/responses"}

	cfg, err := ConfigForProvider(base, ProviderTeamily)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://example.test/responses" {
		t.Fatalf("base url: got %q", cfg.BaseURL)
	}
}
