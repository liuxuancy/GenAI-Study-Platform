package ai

import (
	"strings"
	"testing"

	"caveclass/internal/ai/course"
	"caveclass/internal/ai/prompt"
)

func TestLearningInsightPromptLanguage(t *testing.T) {
	for _, item := range []struct {
		language string
		want     string
	}{
		{language: "", want: course.LanguageZhCN},
		{language: "zh-CN", want: course.LanguageZhCN},
		{language: course.LanguageEnUS, want: course.LanguageEnUS},
		{language: "en", want: course.LanguageZhCN},
	} {
		if got := insightLanguage(item.language); got != item.want {
			t.Fatalf("insightLanguage(%q) = %q, want %q", item.language, got, item.want)
		}
	}

	text, info, err := prompt.Render(prompt.LearningInsightV1, prompt.LearningInsightData{
		SummaryJSON: `{"pendingQuizCount":1}`,
		Language:    course.LanguageEnUS,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != prompt.LearningInsightV1 {
		t.Fatalf("template id = %q", info.ID)
	}
	if !strings.Contains(text, "Write all user-facing text in en-US.") {
		t.Fatalf("prompt missing language line: %s", text)
	}
}
