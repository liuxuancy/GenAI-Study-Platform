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

func TestRenderUnknownTemplate(t *testing.T) {
	_, _, err := prompt.Render("chapter_v1", prompt.FullCourseData{})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
