package ai

import (
	"fmt"

	"caveclass/internal/ai/llm"
)

// Service 是供业务层调用的 AI 门面。
type Service struct {
	client llm.Client
	config llm.Config
}

// New 用已有配置和客户端构造服务。client 不能为空。
func New(cfg llm.Config, client llm.Client) (*Service, error) {
	if client == nil {
		return nil, fmt.Errorf("ai client is nil")
	}
	return &Service{client: client, config: cfg}, nil
}

// NewFromEnv 读取 AI_* 并创建客户端。AI_PROVIDER 默认是 mock。
func NewFromEnv() (*Service, error) {
	cfg, err := llm.LoadConfig()
	if err != nil {
		return nil, err
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return New(cfg, client)
}
