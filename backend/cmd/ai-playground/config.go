package main

import (
	"fmt"
	"os"
	"strings"

	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
)

// configForProvider 在 playground 里按请求选择厂商，不修改进程里的 AI_PROVIDER。
func configForProvider(base llm.Config, provider string) (llm.Config, error) {
	switch provider {
	case llm.ProviderMock, llm.ProviderDeepSeek, llm.ProviderTeamily:
	default:
		return llm.Config{}, fmt.Errorf("provider must be mock, deepseek, or teamily")
	}
	cfg := base
	cfg.Provider = provider
	switch provider {
	case llm.ProviderMock:
		cfg.APIKey = ""
		cfg.BaseURL = ""
		if os.Getenv("AI_MODEL_PRIMARY") == "" {
			cfg.ModelPrimary = llm.DefaultModelPrimary
		}
	case llm.ProviderDeepSeek:
		cfg.APIKey = providerAPIKey("AI_DEEPSEEK_API_KEY", base, llm.ProviderDeepSeek)
		cfg.BaseURL = providerBaseURL("AI_DEEPSEEK_BASE_URL", base, llm.ProviderDeepSeek)
		if model := os.Getenv("AI_DEEPSEEK_MODEL_PRIMARY"); model != "" {
			cfg.ModelPrimary = model
		} else if os.Getenv("AI_MODEL_PRIMARY") == "" || cfg.ModelPrimary == llm.DefaultModelPrimary {
			cfg.ModelPrimary = llm.DefaultModelDeepSeek
		}
	case llm.ProviderTeamily:
		cfg.APIKey = providerAPIKey("AI_TEAMILY_API_KEY", base, llm.ProviderTeamily)
		cfg.BaseURL = providerBaseURL("AI_TEAMILY_BASE_URL", base, llm.ProviderTeamily)
		if model := os.Getenv("AI_TEAMILY_MODEL_PRIMARY"); model != "" {
			cfg.ModelPrimary = model
		} else if os.Getenv("AI_MODEL_PRIMARY") == "" || cfg.ModelPrimary == llm.DefaultModelPrimary {
			cfg.ModelPrimary = llm.DefaultModelTeamily
		}
	}
	return cfg, nil
}

func providerAPIKey(specificEnv string, base llm.Config, provider string) string {
	if key := os.Getenv(specificEnv); key != "" {
		return key
	}
	if base.Provider == provider && base.APIKey != "" {
		return base.APIKey
	}
	if envProviderMatches(provider) {
		return strings.TrimSpace(os.Getenv("AI_API_KEY"))
	}
	return ""
}

func providerBaseURL(specificEnv string, base llm.Config, provider string) string {
	if url := strings.TrimSpace(os.Getenv(specificEnv)); url != "" {
		return url
	}
	if base.Provider == provider && strings.TrimSpace(base.BaseURL) != "" {
		return base.BaseURL
	}
	if envProviderMatches(provider) {
		return strings.TrimSpace(os.Getenv("AI_BASE_URL"))
	}
	return ""
}

func envProviderMatches(provider string) bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("AI_PROVIDER")), provider)
}

func newServiceForProvider(base llm.Config, provider string) (*ai.Service, error) {
	cfg, err := configForProvider(base, provider)
	if err != nil {
		return nil, err
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return ai.New(cfg, client)
}

type providerStatus struct {
	Mock     bool `json:"mock"`
	DeepSeek bool `json:"deepseek"`
	Teamily  bool `json:"teamily"`
}

func providerAvailability(base llm.Config) providerStatus {
	return providerStatus{
		Mock:     true,
		DeepSeek: providerAPIKey("AI_DEEPSEEK_API_KEY", base, llm.ProviderDeepSeek) != "",
		Teamily:  providerAPIKey("AI_TEAMILY_API_KEY", base, llm.ProviderTeamily) != "",
	}
}
