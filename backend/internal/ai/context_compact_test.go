package ai

import (
	"encoding/json"
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
