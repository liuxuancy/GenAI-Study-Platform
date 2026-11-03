package ai

import (
	"context"
	"encoding/json"
	"strings"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/prompt"
)

const SceneLearningInsight = "learning_insight"

// GenerateLearningInsight 根据学生学习摘要生成 JSON 建议。
func (service *Service) GenerateLearningInsight(ctx context.Context, summaryJSON, language string) (string, Meta, error) {
	if err := service.ready(); err != nil {
		return "", Meta{}, err
	}
	summaryJSON = strings.TrimSpace(summaryJSON)
	if summaryJSON == "" {
		return "", Meta{}, aierr.InvalidInput(nil, "summary is required")
	}
	response, meta, err := service.runModel(ctx, SceneLearningInsight, prompt.LearningInsightV1, "", prompt.LearningInsightData{
		SummaryJSON: summaryJSON,
		Language:    insightLanguage(language),
	})
	if err != nil {
		return "", meta, err
	}
	text := strings.TrimSpace(response.Text)
	if text == "" {
		return "", meta, aierr.OutputSchemaFailed(nil, "empty insight")
	}
	meta.Validation.Passed = true
	return text, meta, nil
}

// ParseLearningInsightResponse 解析模型返回的 JSON。
func ParseLearningInsightResponse(raw string) (suggestion string, actionTitle, actionDetail string, ok bool) {
	var parsed struct {
		Suggestion string `json:"suggestion"`
		RecommendedAction struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"recommendedAction"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", "", "", false
	}
	return strings.TrimSpace(parsed.Suggestion),
		strings.TrimSpace(parsed.RecommendedAction.Title),
		strings.TrimSpace(parsed.RecommendedAction.Detail),
		strings.TrimSpace(parsed.Suggestion) != ""
}

func insightLanguage(language string) string {
	if strings.TrimSpace(language) == course.LanguageEnUS {
		return course.LanguageEnUS
	}
	return course.LanguageZhCN
}
