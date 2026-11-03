package ai

import (
	"context"
	"strings"
	"time"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/llm"
)

// CompletePrompt 直接发送已经按预算裁剪过的提示词。
func (service *Service) CompletePrompt(ctx context.Context, scene, requestedModel, promptText string) (string, Meta, error) {
	if err := service.ready(); err != nil {
		return "", Meta{}, err
	}
	if len(promptText) > chapterPromptBudget() {
		return "", Meta{}, aierr.InputTooLong(nil, "prompt exceeds input budget")
	}
	meta := NewMeta(ctx, scene)
	meta.TemplateID = "workspace_prompt"
	meta.TemplateVersion = "1"
	meta.ModelRequested = requestedModel
	if meta.ModelRequested == "" {
		meta.ModelRequested = service.config.ModelPrimary
	}
	if meta.ModelRequested == "" {
		meta.ModelRequested = llm.DefaultModelPrimary
	}
	started := time.Now()
	response, err := llm.Call(ctx, service.client, service.config, llm.Request{Model: requestedModel, Prompt: promptText})
	meta.LatencyMs = time.Since(started).Milliseconds()
	if err != nil {
		return "", meta, err
	}
	meta.FallbackUsed = response.FallbackUsed
	meta.ModelUsed = response.Model
	meta.Usage = Usage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}
	answer := strings.TrimSpace(response.Text)
	if answer == "" {
		return "", meta, aierr.OutputSchemaFailed(nil, "empty answer")
	}
	meta.Validation.Passed = true
	return answer, meta, nil
}
