package llm

import (
	"context"
	"fmt"

	"caveclass/internal/ai/aierr"
)

// Usage 与 ai.Meta 里的 usage 字段一致。这里单独定义，避免 llm 引用门面包。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Request 是一次补全请求。Model 为空时使用配置里的主模型。
// JSONResponse 为 true 时，OpenAI 兼容客户端会要求模型返回 json_object（仅用于课程等结构化生成）。
type Request struct {
	Model        string
	Prompt       string
	JSONResponse bool
}

// Response 是模型原文，尚未解析成课程。FallbackUsed 表示这次成功来自备用模型。
type Response struct {
	Text         string
	Model        string
	Usage        Usage
	FallbackUsed bool
}

// Client 调用大模型。mock 返回固定文本；真实厂商在后续阶段接入。
type Client interface {
	Complete(ctx context.Context, request Request) (Response, error)
}

// NewClient 按配置创建客户端。
// mock 返回固定 JSON。deepseek 走 OpenAI 兼容的 chat completions。teamily 走 agent responses。openai / azure 仍返回 AI_MODEL_UNAVAILABLE。
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "", ProviderMock:
		model := cfg.ModelPrimary
		if model == "" {
			model = DefaultModelPrimary
		}
		return &mockClient{model: model, text: MockText()}, nil
	case ProviderDeepSeek:
		if cfg.APIKey == "" {
			return nil, aierr.InvalidInput(fmt.Errorf("missing API key"), "AI_API_KEY")
		}
		endpoint := cfg.BaseURL
		if endpoint == "" {
			endpoint = DeepSeekEndpoint
		}
		model := cfg.ModelPrimary
		if model == "" || model == DefaultModelPrimary {
			model = DefaultModelDeepSeek
		}
		return NewCompatibleClient(endpoint, cfg.APIKey, model), nil
	case ProviderTeamily:
		if cfg.APIKey == "" {
			return nil, aierr.InvalidInput(fmt.Errorf("missing API key"), "AI_API_KEY")
		}
		endpoint := cfg.BaseURL
		if endpoint == "" {
			endpoint = TeamilyEndpoint
		}
		model := cfg.ModelPrimary
		if model == "" || model == DefaultModelPrimary {
			model = DefaultModelTeamily
		}
		return NewTeamilyClient(endpoint, cfg.APIKey, model), nil
	case ProviderOpenAI, ProviderAzure:
		return nil, aierr.ModelUnavailable(fmt.Errorf("%s client is not implemented", cfg.Provider), "provider="+cfg.Provider)
	default:
		return nil, aierr.InvalidInput(fmt.Errorf("unknown provider %q", cfg.Provider), "AI_PROVIDER")
	}
}
