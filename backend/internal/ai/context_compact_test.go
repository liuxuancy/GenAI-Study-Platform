package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	aicourse "caveclass/internal/ai/course"
)

func TestMarshalActivityContextStaysWithinBudget(t *testing.T) {
	longMarkdown := strings.Repeat("章节内容。", 8000)
	payload, err := json.Marshal(map[string]any{
		"markdown":   longMarkdown,
		"highlights": []string{"a", "b", "c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	course := &aicourse.StandardCourse{
		SchemaVersion: "1.0",
		Title:         "大数据",
		Description:   strings.Repeat("描述", 500),
		Audience:      aicourse.AudienceGeneral,
		Language:      "zh-CN",
		Chapters: []aicourse.Chapter{
			{
				Key:     "ch1",
				Title:   "第一章",
				Summary: strings.Repeat("摘要", 400),
				Activities: []aicourse.Activity{
					{Key: "a1", Type: aicourse.ActivityTypeRichText, Title: "阅读", Order: 1, Payload: payload},
				},
			},
			{Key: "ch2", Title: "第二章"},
		},
	}
	text, err := marshalActivityContext(course, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > activityContextBudget {
		t.Fatalf("context length %d exceeds budget %d", len(text), activityContextBudget)
	}
	if !strings.Contains(text, "ch1") {
		t.Fatal("expected target chapter in context")
	}
	if strings.Contains(text, `"ch2"`) {
		t.Fatal("other chapters should be omitted")
	}
}

func TestMarshalCourseForQualityStaysWithinBudget(t *testing.T) {
	longMarkdown := strings.Repeat("正文。", 6000)
	payload, err := json.Marshal(map[string]any{"markdown": longMarkdown, "highlights": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	quizPayload, err := json.Marshal(map[string]any{
		"instructionsMarkdown": longMarkdown,
		"questions": []map[string]any{{
			"question":     longMarkdown,
			"options":      []string{"A", "B", "C", "D"},
			"correctIndex": 0,
			"explanation":  longMarkdown,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	chapters := make([]aicourse.Chapter, 0, 12)
	for index := 0; index < 12; index++ {
		chapters = append(chapters, aicourse.Chapter{
			Key: fmt.Sprintf("ch%d", index+1),
			Title: fmt.Sprintf("第 %d 章", index+1),
			Summary: strings.Repeat("摘要", 80),
			Order: index + 1,
			Activities: []aicourse.Activity{
				{Key: fmt.Sprintf("a%d-1", index), Type: aicourse.ActivityTypeRichText, Title: "阅读", Order: 1, Payload: payload},
				{Key: fmt.Sprintf("a%d-2", index), Type: aicourse.ActivityTypeQuizGame, Title: "测验", Order: 2, Payload: quizPayload},
			},
		})
	}
	subject := aicourse.StandardCourse{
		SchemaVersion: "1.0",
		Title:         "C++ 入门",
		Description:   strings.Repeat("简介", 300),
		Audience:      aicourse.AudienceGeneral,
		Language:      "zh-CN",
		Chapters:      chapters,
	}
	text, err := marshalCourseForQuality(subject)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > qualityContextBudget {
		t.Fatalf("quality context length %d exceeds budget %d", len(text), qualityContextBudget)
	}
}
