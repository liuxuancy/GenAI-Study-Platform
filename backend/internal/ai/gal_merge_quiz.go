package ai

import (
	"encoding/json"
	"strings"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai/gal"
)

// MergeChapterQuizzes 将章内 quiz_game 题目插入剧本后段，并跳过与已有 choice 重复的题干。
func MergeChapterQuizzes(chapter aicourse.Chapter, script []gal.ScriptItem) []gal.ScriptItem {
	quizChoices := choicesFromChapterQuizzes(chapter)
	if len(quizChoices) == 0 {
		return script
	}
	seen := choicePromptsInScript(script)
	var toInsert []gal.ScriptItem
	for _, item := range quizChoices {
		key := normalizePromptKey(item.Prompt)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		toInsert = append(toInsert, item)
	}
	if len(toInsert) == 0 {
		return script
	}
	insertAt := len(script) * 2 / 3
	if insertAt < 1 {
		insertAt = 1
	}
	out := make([]gal.ScriptItem, 0, len(script)+len(toInsert))
	out = append(out, script[:insertAt]...)
	out = append(out, toInsert...)
	out = append(out, script[insertAt:]...)
	return out
}

func choicesFromChapterQuizzes(chapter aicourse.Chapter) []gal.ScriptItem {
	var items []gal.ScriptItem
	for _, activity := range chapter.Activities {
		if activity.Type != aicourse.ActivityTypeQuizGame {
			continue
		}
		var payload aicourse.QuizGamePayload
		if json.Unmarshal(activity.Payload, &payload) != nil {
			continue
		}
		for _, question := range payload.Questions {
			if len(question.Options) < 2 {
				continue
			}
			correct := question.CorrectIndex
			if correct < 0 || correct >= len(question.Options) {
				continue
			}
			feedbackCorrect := strings.TrimSpace(question.Explanation)
			if feedbackCorrect == "" {
				feedbackCorrect = "回答正确，继续吧。"
			}
			items = append(items, gal.ScriptItem{
				Kind:             gal.KindChoice,
				Prompt:           strings.TrimSpace(question.Question),
				Options:          append([]string(nil), question.Options...),
				CorrectIndex:     correct,
				FeedbackCorrect:  feedbackCorrect,
				FeedbackWrong:    "再想想：" + feedbackCorrect,
				EmotionOnCorrect: "happy",
				EmotionOnWrong:   "shy",
			})
		}
	}
	return items
}

func choicePromptsInScript(script []gal.ScriptItem) map[string]bool {
	seen := make(map[string]bool)
	for _, item := range script {
		if item.Kind != gal.KindChoice {
			continue
		}
		key := normalizePromptKey(item.Prompt)
		if key != "" {
			seen[key] = true
		}
	}
	return seen
}

func normalizePromptKey(prompt string) string {
	return strings.ToLower(strings.TrimSpace(prompt))
}
