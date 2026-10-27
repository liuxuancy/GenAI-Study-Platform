package llm

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	ProviderMock   = "mock"
	ProviderOpenAI = "openai"
	ProviderAzure  = "azure"

	DefaultModelPrimary = "gpt-4o-mini"
	DefaultTimeout      = 60 * time.Second
	// DefaultMaxRetries 是每个模型首次失败后的额外尝试次数。0 表示不重试。
	DefaultMaxRetries  = 1
	DefaultMaxTopicLen = 500
)

// Config 是 AI 子系统的环境变量配置。尚未接入进程启动配置，调用方自行 LoadConfig。
type Config struct {
	Provider      string
	APIKey        string
	ModelPrimary  string
	ModelFallback string
	Timeout       time.Duration
	MaxRetries    int
	MaxTopicLen   int
}

// LoadConfig 读取 AI_* 环境变量。未设置 AI_PROVIDER 时默认为 mock。
func LoadConfig() (Config, error) {
	cfg := Config{
		Provider:      getenv("AI_PROVIDER", ProviderMock),
		APIKey:        os.Getenv("AI_API_KEY"),
		ModelPrimary:  getenv("AI_MODEL_PRIMARY", DefaultModelPrimary),
		ModelFallback: os.Getenv("AI_MODEL_FALLBACK"),
		Timeout:       DefaultTimeout,
		MaxRetries:    DefaultMaxRetries,
		MaxTopicLen:   DefaultMaxTopicLen,
	}
	switch cfg.Provider {
	case ProviderMock, ProviderOpenAI, ProviderAzure:
	default:
		return Config{}, fmt.Errorf("AI_PROVIDER: must be mock, openai, or azure, got %q", cfg.Provider)
	}
	if value := os.Getenv("AI_TIMEOUT"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return Config{}, fmt.Errorf("AI_TIMEOUT: invalid duration %q", value)
		}
		cfg.Timeout = duration
	}
	if value := os.Getenv("AI_MAX_RETRIES"); value != "" {
		retries, err := strconv.Atoi(value)
		if err != nil || retries < 0 {
			return Config{}, fmt.Errorf("AI_MAX_RETRIES: invalid integer %q", value)
		}
		cfg.MaxRetries = retries
	}
	if value := os.Getenv("AI_MAX_TOPIC_LEN"); value != "" {
		topicLen, err := strconv.Atoi(value)
		if err != nil || topicLen <= 0 {
			return Config{}, fmt.Errorf("AI_MAX_TOPIC_LEN: invalid integer %q", value)
		}
		cfg.MaxTopicLen = topicLen
	}
	return cfg, nil
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
