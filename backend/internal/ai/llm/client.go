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
type Request struct {
	Model  string
	Prompt string
}

// Response 是模型原文，尚未解析成课程。
type Response struct {
	Text  string
	Model string
	Usage Usage
}

// Client 调用大模型。mock 返回固定文本；真实厂商在后续阶段接入。
type Client interface {
	Complete(ctx context.Context, request Request) (Response, error)
}

// NewClient 按配置创建客户端。
// mock 返回固定课程 JSON。openai / azure 本阶段只占位，返回 AI_MODEL_UNAVAILABLE。
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "", ProviderMock:
		model := cfg.ModelPrimary
		if model == "" {
			model = DefaultModelPrimary
		}
		return &mockClient{model: model, text: MockText()}, nil
	case ProviderOpenAI, ProviderAzure:
		return nil, aierr.ModelUnavailable(fmt.Errorf("%s client is not implemented", cfg.Provider), "provider="+cfg.Provider)
	default:
		return nil, aierr.InvalidInput(fmt.Errorf("unknown provider %q", cfg.Provider), "AI_PROVIDER")
	}
}
