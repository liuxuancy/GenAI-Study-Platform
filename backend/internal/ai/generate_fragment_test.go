package ai_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"caveclass/internal/ai"
	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
	"caveclass/internal/ai/prompt"
	"caveclass/internal/platform/logs"
)

func TestGenerateChapterAndActivityFromMock(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("AI_MODEL_PRIMARY", "")
	t.Setenv("AI_MODEL_FALLBACK", "")
	service, err := ai.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	ctx := logs.WithRequestID(context.Background(), "req-fragment")

	chapter, chapterMeta, err := service.GenerateChapter(ctx, ai.GenerateChapterInput{
		Topic:            "Git 入门",
		Audience:         course.AudienceNonCS,
		TargetChapterKey: "ch-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chapter.Validate(); err != nil {
		t.Fatal(err)
	}
	if chapter.Key != "ch-1" || chapter.Order != 1 || len(chapter.Activities) != 2 {
		t.Fatalf("chapter = %+v", chapter)
	}
	if chapterMeta.Scene != ai.SceneChapter || chapterMeta.TemplateID != prompt.ChapterV1 || chapterMeta.TemplateVersion != prompt.Version1 {
		t.Fatalf("chapter meta = %+v", chapterMeta)
	}
	if chapterMeta.ModelUsed == "" || chapterMeta.RequestID != "req-fragment" || !chapterMeta.Validation.Passed {
		t.Fatalf("chapter meta = %+v", chapterMeta)
	}
	if !containsWarning(chapterMeta.Warnings, "contextCourse was not provided") {
		t.Fatalf("warnings = %#v", chapterMeta.Warnings)
	}

	activity, activityMeta, err := service.GenerateActivity(ctx, ai.GenerateActivityInput{
		Topic:             "Git 入门",
		Audience:          course.AudienceNonCS,
		TargetChapterKey:  "ch-1",
		TargetActivityKey: "act-1-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := activity.Validate(); err != nil {
		t.Fatal(err)
	}
	if activity.Key != "act-1-1" || activity.Type != course.ActivityTypeRichText || activity.Order != 1 {
		t.Fatalf("activity = %+v", activity)
	}
	if activityMeta.Scene != ai.SceneActivity || activityMeta.TemplateID != prompt.ActivityV1 || activityMeta.ModelUsed == "" {
		t.Fatalf("activity meta = %+v", activityMeta)
	}
	if !activityMeta.Validation.Passed || !containsWarning(activityMeta.Warnings, "contextCourse was not provided") {
		t.Fatalf("activity meta = %+v", activityMeta)
	}
}

func TestGenerateChapterUsesContextAndRewritesKey(t *testing.T) {
	stub := &stubClient{text: llm.MockChapterText(), model: "stub-model"}
	service, err := ai.New(llm.Config{ModelPrimary: "gpt-4o-mini"}, stub)
	if err != nil {
		t.Fatal(err)
	}
	contextCourse := &course.StandardCourse{Title: "Existing Git Course", Audience: course.AudienceCS, Language: course.LanguageZhCN}
	chapter, meta, err := service.GenerateChapter(context.Background(), ai.GenerateChapterInput{
		Topic:            "分支",
		Audience:         course.AudienceCS,
		TargetChapterKey: "ch-9",
		ContextCourse:    contextCourse,
	})
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Key != "ch-9" {
		t.Fatalf("key = %s", chapter.Key)
	}
	if !strings.Contains(stub.prompt, "Existing Git Course") || !strings.Contains(stub.prompt, "templateId: chapter_v1") {
		t.Fatalf("prompt = %s", stub.prompt)
	}
	if containsWarning(meta.Warnings, "contextCourse was not provided") {
		t.Fatalf("context was provided, warnings = %#v", meta.Warnings)
	}
	if !containsField(meta.Validation.RepairedFields, "key") || !meta.Validation.Passed {
		t.Fatalf("validation = %+v", meta.Validation)
	}
}

func TestGenerateActivityRejectsMissingKeyBeforeModel(t *testing.T) {
	stub := &stubClient{text: llm.MockActivityText()}
	service, err := ai.New(llm.Config{}, stub)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.GenerateActivity(context.Background(), ai.GenerateActivityInput{
		Topic:            "Git",
		Audience:         course.AudienceGeneral,
		TargetChapterKey: "ch-1",
	})
	if !errors.Is(err, aierr.ErrInvalidInput) {
		t.Fatalf("err = %v", err)
	}
	if stub.calls != 0 {
		t.Fatal("invalid input must not call the model")
	}
}

func containsWarning(warnings []string, text string) bool {
	for _, warning := range warnings {
		if warning == text {
			return true
		}
	}
	return false
}

func containsField(fields []string, field string) bool {
	for _, item := range fields {
		if item == field {
			return true
		}
	}
	return false
}
