// Package ai 是 B 侧 AI 子系统的对外门面。
// 目前提供整课生成、调用元数据和错误日志。章节、activity 和质量评估在后续阶段补上。
// 不访问数据库，也不注册 HTTP 路由。
package ai

import (
	"context"
	"log/slog"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/platform/logs"
)

const (
	SceneFullCourse = "full_course"
	SceneChapter    = "chapter"
	SceneActivity   = "activity"
	SceneEvaluate   = "evaluate"
)

// Usage 是一次模型调用的 token 用量。
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Validation 记录输出校验是否通过，以及 repair 补过的字段。
type Validation struct {
	Passed         bool     `json:"passed"`
	RepairedFields []string `json:"repairedFields"`
}

// Meta 是每次 AI 调用附带的元数据，业务层放进 data.meta。
type Meta struct {
	Scene           string     `json:"scene"`
	TemplateID      string     `json:"templateId"`
	TemplateVersion string     `json:"templateVersion"`
	ModelRequested  string     `json:"modelRequested"`
	ModelUsed       string     `json:"modelUsed"`
	FallbackUsed    bool       `json:"fallbackUsed"`
	LatencyMs       int64      `json:"latencyMs"`
	Usage           Usage      `json:"usage"`
	Warnings        []string   `json:"warnings"`
	Validation      Validation `json:"validation"`
	RequestID       string     `json:"requestId"`
}

// NewMeta 创建一条元数据，并从 ctx 填入 requestId。没有请求 ID 时留空。
func NewMeta(ctx context.Context, scene string) Meta {
	meta := Meta{
		Scene:    scene,
		Warnings: []string{},
		Validation: Validation{
			RepairedFields: []string{},
		},
	}
	meta.FillRequestID(ctx)
	return meta
}

// FillRequestID 把日志 ctx 里的请求 ID 抄到 meta，便于和响应头 X-Request-ID 对照。
func (meta *Meta) FillRequestID(ctx context.Context) {
	if meta == nil {
		return
	}
	meta.RequestID = logs.GetRequestID(ctx)
}

// LogError 把 AI 失败打成结构化日志。字段为 aiCode、details、meta、uid。
// uid 由业务层传入，本包不查库。
func LogError(ctx context.Context, err error, meta Meta, uid int64) {
	if ctx == nil {
		ctx = context.Background()
	}
	code := aierr.CodeInternal
	message := aierr.ErrInternal.Msg
	details := []string{}
	if aiError, ok := aierr.As(err); ok {
		code = aiError.Code
		message = aiError.Msg
		if aiError.Details != nil {
			details = aiError.Details
		}
	} else if err != nil {
		details = []string{err.Error()}
	}
	slog.ErrorContext(ctx, message,
		slog.String("aiCode", code),
		slog.Any("details", details),
		slog.Any("meta", meta),
		slog.Int64("uid", uid),
	)
}
