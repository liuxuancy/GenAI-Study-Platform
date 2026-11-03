package llm

import (
	"fmt"
	"os"
	"strings"
)

// TeamilyMaxInputChars 是 Teamily agent input 字段的上限（与官方校验一致）。
const TeamilyMaxInputChars = 32000

// ProviderStatus 表示各厂商是否已在环境变量里配好密钥。
type ProviderStatus struct {
	Mock     bool `json:"mock"`
	DeepSeek bool `json:"deepseek"`
	Teamily  bool `json:"teamily"`
}

// GenerateProviderOptions 供前端展示可选模型与默认项。
type GenerateProviderOptions struct {
	Default   string         `json:"default"`
	Providers ProviderStatus `json:"providers"`
}

// ConfigForProvider 按请求选择厂商，不修改进程里的 AI_PROVIDER。
func ConfigForProvider(base Config, provider string) (Config, error) {
	switch provider {
	case ProviderMock, ProviderDeepSeek, ProviderTeamily:
	default:
		return Config{}, fmt.Errorf("provider must be mock, deepseek, or teamily")
	}
	cfg := base
	cfg.Provider = provider
	switch provider {
	case ProviderMock:
		cfg.APIKey = ""
		cfg.BaseURL = ""
		if os.Getenv("AI_MODEL_PRIMARY") == "" {
			cfg.ModelPrimary = DefaultModelPrimary
		}
	case ProviderDeepSeek:
		cfg.APIKey = ProviderAPIKey("AI_DEEPSEEK_API_KEY", base, ProviderDeepSeek)
		cfg.BaseURL = providerBaseURL("AI_DEEPSEEK_BASE_URL", base, ProviderDeepSeek)
		if model := os.Getenv("AI_DEEPSEEK_MODEL_PRIMARY"); model != "" {
			cfg.ModelPrimary = model
		} else if os.Getenv("AI_MODEL_PRIMARY") == "" || cfg.ModelPrimary == DefaultModelPrimary {
			cfg.ModelPrimary = DefaultModelDeepSeek
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = DeepSeekEndpoint
		}
	case ProviderTeamily:
		cfg.APIKey = ProviderAPIKey("AI_TEAMILY_API_KEY", base, ProviderTeamily)
		cfg.BaseURL = providerBaseURL("AI_TEAMILY_BASE_URL", base, ProviderTeamily)
		if model := os.Getenv("AI_TEAMILY_MODEL_PRIMARY"); model != "" {
			cfg.ModelPrimary = model
		} else if os.Getenv("AI_MODEL_PRIMARY") == "" || cfg.ModelPrimary == DefaultModelPrimary {
			cfg.ModelPrimary = DefaultModelTeamily
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = TeamilyEndpoint
		}
	}
	return cfg, nil
}

// ProviderAPIKey 读取厂商专用密钥，否则回退到与 AI_PROVIDER 一致的 AI_API_KEY。
func ProviderAPIKey(specificEnv string, base Config, provider string) string {
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

func providerBaseURL(specificEnv string, base Config, provider string) string {
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

// ProviderAvailability 判断 mock / deepseek / teamily 是否可用（mock 恒为 true）。
func ProviderAvailability(base Config) ProviderStatus {
	return ProviderStatus{
		Mock:     true,
		DeepSeek: ProviderAPIKey("AI_DEEPSEEK_API_KEY", base, ProviderDeepSeek) != "",
		Teamily:  ProviderAPIKey("AI_TEAMILY_API_KEY", base, ProviderTeamily) != "",
	}
}

// DefaultGenerateProvider 是整课生成未指定 provider 时的默认值。
func DefaultGenerateProvider(base Config) string {
	switch base.Provider {
	case ProviderDeepSeek, ProviderTeamily:
		if ProviderAPIKey(providerKeyEnv(base.Provider), base, base.Provider) != "" {
			return base.Provider
		}
	case ProviderMock:
		return ProviderMock
	}
	status := ProviderAvailability(base)
	if status.DeepSeek {
		return ProviderDeepSeek
	}
	if status.Teamily {
		return ProviderTeamily
	}
	if base.Provider == ProviderMock {
		return ProviderMock
	}
	return base.Provider
}

func providerKeyEnv(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return "AI_DEEPSEEK_API_KEY"
	case ProviderTeamily:
		return "AI_TEAMILY_API_KEY"
	default:
		return ""
	}
}

// GenerateProviderOptionsFor 返回前端可用的 provider 列表与默认项（不含 mock）。
func GenerateProviderOptionsFor(base Config) GenerateProviderOptions {
	status := ProviderAvailability(base)
	defaultProvider := DefaultGenerateProvider(base)
	if defaultProvider == ProviderMock {
		if status.DeepSeek {
			defaultProvider = ProviderDeepSeek
		} else if status.Teamily {
			defaultProvider = ProviderTeamily
		}
	}
	if defaultProvider != ProviderDeepSeek && defaultProvider != ProviderTeamily {
		if status.DeepSeek {
			defaultProvider = ProviderDeepSeek
		} else if status.Teamily {
			defaultProvider = ProviderTeamily
		} else {
			defaultProvider = ProviderDeepSeek
		}
	}
	return GenerateProviderOptions{
		Default:   defaultProvider,
		Providers: ProviderStatus{DeepSeek: status.DeepSeek, Teamily: status.Teamily},
	}
}
