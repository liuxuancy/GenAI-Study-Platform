package prompt_test

import (
	"errors"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/prompt"
)

func TestRenderFullCourseWrapsUserData(t *testing.T) {
	text, info, err := prompt.Render(prompt.FullCourseV1, prompt.FullCourseData{
		Topic:             "Git 入门",
		Audience:          "NON_CS",
		Language:          "zh-CN",
		ExtraRequirements: "多举例子",
		ChapterCountHint:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != prompt.FullCourseV1 || info.Version != prompt.Version1 {
		t.Fatalf("info = %+v", info)
	}
	for _, part := range []string{
		"Do not wrap the JSON in markdown fences",
		"<user_topic>",
		"Git 入门",
		"</user_topic>",
		"<extra_requirements>",
		"多举例子",
		"Aim for about 4 chapters",
		"zh-CN",
		"NON_CS",
	} {
		if !strings.Contains(text, part) {
			t.Fatalf("prompt missing %q\n%s", part, text)
		}
	}
}

func TestRenderChapterAndActivityTemplates(t *testing.T) {
	chapter, info, err := prompt.Render(prompt.ChapterV1, prompt.ChapterData{
		Topic: "分支", Audience: "CS", Language: "zh-CN", TargetChapterKey: "ch-9",
		ContextJSON: `{"title":"Existing"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != prompt.ChapterV1 || info.Version != prompt.Version1 || !strings.HasPrefix(chapter, "templateId: chapter_v1\n") {
		t.Fatalf("chapter prompt = %s info = %+v", chapter, info)
	}
	if !strings.Contains(chapter, "ch-9") || !strings.Contains(chapter, `"title":"Existing"`) {
		t.Fatalf("chapter prompt = %s", chapter)
	}

	activity, info, err := prompt.Render(prompt.ActivityV1, prompt.ActivityData{
		Topic: "命令", Audience: "GENERAL", Language: "en-US",
		TargetChapterKey: "ch-1", TargetActivityKey: "act-1-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != prompt.ActivityV1 || !strings.HasPrefix(activity, "templateId: activity_v1\n") || !strings.Contains(activity, "act-1-1") {
		t.Fatalf("activity prompt = %s info = %+v", activity, info)
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	_, _, err := prompt.Render("missing_v1", prompt.FullCourseData{})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
