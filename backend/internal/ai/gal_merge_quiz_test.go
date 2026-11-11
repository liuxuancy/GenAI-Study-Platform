package ai

import (
	"encoding/json"
	"testing"

	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai/gal"
)

func TestMergeChapterQuizzesInsertsFromQuizGame(t *testing.T) {
	payload, err := json.Marshal(aicourse.QuizGamePayload{
		InstructionsMarkdown: "test",
		Questions: []aicourse.QuizQuestion{
			{
				Question:     "Q1?",
				Options:      []string{"yes", "no"},
				CorrectIndex: 0,
				Explanation:  "because",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	chapter := aicourse.Chapter{
		Key: "ch1",
		Activities: []aicourse.Activity{
			{Key: "q1", Type: aicourse.ActivityTypeQuizGame, Payload: payload},
		},
	}
	script := []gal.ScriptItem{
		{Kind: gal.KindDialogue, Speaker: "小洞", Text: "a", Emotion: "normal"},
		{Kind: gal.KindDialogue, Speaker: "小洞", Text: "b", Emotion: "normal"},
		{Kind: gal.KindDialogue, Speaker: "小洞", Text: "c", Emotion: "normal"},
	}
	merged := MergeChapterQuizzes(chapter, script)
	foundChoice := false
	for _, item := range merged {
		if item.Kind == gal.KindChoice && item.Prompt == "Q1?" {
			foundChoice = true
		}
	}
	if !foundChoice {
		t.Fatalf("merged = %+v", merged)
	}
}
