package ai

import (
	"context"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/prompt"
)

const SceneChapterQA = "chapter_qa"

const (
	maxChapterHistoryMessages = 8
	maxChapterHistoryRunes    = 2000
	maxChapterHistoryTotal    = 6000
	maxChapterSelectionRunes  = 2000
)

// HistoryTurn 是学生与助手的一轮原文。内容不可信，只能作为对话上下文。
type HistoryTurn struct {
	Role         string
	Content      string
	SectionTitle string
}

// ChapterQuestion 是一次章节提问。章节正文仍由服务端读取，不由这里传入。
type ChapterQuestion struct {
	Question    string
	ActivityKey string
	Selection   string
	History     []HistoryTurn
}

// NormalizeChapterHistory 拒绝非法角色，并截断过长的历史。空内容会被丢掉。
func NormalizeChapterHistory(history []HistoryTurn) ([]HistoryTurn, error) {
	if len(history) == 0 {
		return nil, nil
	}
	cleaned := make([]HistoryTurn, 0, len(history))
	for _, turn := range history {
		role := strings.TrimSpace(turn.Role)
		if role != "user" && role != "assistant" {
			return nil, aierr.InvalidInput(nil, "history role must be user or assistant")
		}
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		cleaned = append(cleaned, HistoryTurn{
			Role: role, Content: truncateRunes(content, maxChapterHistoryRunes), SectionTitle: strings.TrimSpace(turn.SectionTitle),
		})
	}
	if len(cleaned) == 0 {
		return nil, nil
	}
	if len(cleaned) > maxChapterHistoryMessages {
		cleaned = cleaned[len(cleaned)-maxChapterHistoryMessages:]
	}
	total := 0
	start := len(cleaned)
	for index := len(cleaned) - 1; index >= 0; index-- {
		count := utf8.RuneCountInString(cleaned[index].Content)
		if total+count > maxChapterHistoryTotal && start < len(cleaned) {
			break
		}
		total += count
		start = index
		if total > maxChapterHistoryTotal {
			break
		}
	}
	return cleaned[start:], nil
}

// AskChapter 把当前章和截断后的对话历史交给模型。历史不能替代章节 JSON。
func (service *Service) AskChapter(ctx context.Context, chapter course.Chapter, language string, ask ChapterQuestion) (string, Meta, error) {
	if err := service.ready(); err != nil {
		return "", Meta{}, err
	}
	data, err := fitChapterQuestion(chapter, language, ask, service.config.ModelPrimary)
	if err != nil {
		return "", Meta{}, err
	}
	response, meta, err := service.runModel(ctx, SceneChapterQA, prompt.ChapterQAV1, "", data)
	if err != nil {
		return "", meta, err
	}
	answer := strings.TrimSpace(response.Text)
	if answer == "" {
		return "", meta, aierr.OutputSchemaFailed(nil, "empty answer")
	}
	meta.Validation.Passed = true
	return answer, meta, nil
}

func chapterQuestionData(chapter course.Chapter, language string, ask ChapterQuestion) (prompt.ChapterQAData, error) {
	if err := chapter.Validate(); err != nil {
		return prompt.ChapterQAData{}, invalidCourse(err)
	}
	question := strings.TrimSpace(ask.Question)
	if question == "" || utf8.RuneCountInString(question) > 2000 {
		return prompt.ChapterQAData{}, aierr.InvalidInput(nil, "question must contain 1 to 2000 characters")
	}
	if language != course.LanguageEnUS {
		language = course.LanguageZhCN
	}
	history, err := NormalizeChapterHistory(ask.History)
	if err != nil {
		return prompt.ChapterQAData{}, err
	}
	return fitChapterQuestion(chapter, language, ChapterQuestion{
		Question: question, ActivityKey: ask.ActivityKey, Selection: ask.Selection, History: history,
	}, "")
}

func matchedActivity(chapter course.Chapter, key string) (string, string, string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", ""
	}
	for _, activity := range chapter.Activities {
		if activity.Key == key {
			return strings.TrimSpace(activity.Title), activity.Key, excerptFromActivity(activity, 4000)
		}
	}
	return "", "", ""
}
