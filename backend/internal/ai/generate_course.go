package ai

import (
	"context"
	"errors"
	"time"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
	"caveclass/internal/ai/validate"
)

// GenerateCourseInput 是整课生成的入参。
type GenerateCourseInput struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	ChapterCountHint  int
	Model             string
	TemplateID        string
}

// GenerateCourse 校验入参，渲染 full_course_v1，调用模型，再解析并校验课程。
// 生成路径固定 repair=true。openai / azure 客户端尚未实现，会返回 AI_MODEL_UNAVAILABLE。
func (service *Service) GenerateCourse(ctx context.Context, input GenerateCourseInput) (course.StandardCourse, Meta, error) {
	if service == nil || service.client == nil {
		return course.StandardCourse{}, Meta{}, aierr.Internal(nil, "ai service is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	checked, err := validate.NormalizeCourse(validate.CourseInput{
		Topic:             input.Topic,
		Audience:          input.Audience,
		Language:          input.Language,
		ExtraRequirements: input.ExtraRequirements,
		ChapterCountHint:  input.ChapterCountHint,
		Model:             input.Model,
		TemplateID:        input.TemplateID,
	}, service.config.MaxTopicLen)
	if err != nil {
		return course.StandardCourse{}, Meta{}, err
	}
	if checked.TemplateID == "" {
		checked.TemplateID = prompt.FullCourseV1
	} else if checked.TemplateID != prompt.FullCourseV1 {
		return course.StandardCourse{}, Meta{}, aierr.InvalidInput(nil, "templateId")
	}
	response, meta, err := service.runModel(ctx, SceneFullCourse, checked.TemplateID, checked.Model, prompt.FullCourseData{
		Topic:             checked.Topic,
		Audience:          checked.Audience,
		Language:          checked.Language,
		ExtraRequirements: checked.ExtraRequirements,
		ChapterCountHint:  checked.ChapterCountHint,
	})
	if err != nil {
		return course.StandardCourse{}, meta, err
	}
	parsed, report, err := parse.ParseCourseJSON(response.Text, true)
	if err != nil {
		return course.StandardCourse{}, meta, err
	}
	applyReport(&meta, report)
	return parsed, meta, nil
}

func (service *Service) runModel(ctx context.Context, scene, templateID, requestedModel string, data any) (llm.Response, Meta, error) {
	rendered, info, err := prompt.Render(templateID, data)
	if err != nil {
		return llm.Response{}, Meta{}, err
	}
	meta := NewMeta(ctx, scene)
	meta.TemplateID = info.ID
	meta.TemplateVersion = info.Version
	meta.ModelRequested = requestedModel
	if meta.ModelRequested == "" {
		meta.ModelRequested = service.config.ModelPrimary
	}
	if meta.ModelRequested == "" {
		meta.ModelRequested = llm.DefaultModelPrimary
	}
	started := time.Now()
	response, err := llm.Call(ctx, service.client, service.config, llm.Request{Model: requestedModel, Prompt: rendered})
	meta.LatencyMs = time.Since(started).Milliseconds()
	if err != nil {
		return llm.Response{}, meta, completeError(err)
	}
	meta.FallbackUsed = response.FallbackUsed
	meta.ModelUsed = response.Model
	meta.Usage = Usage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}
	return response, meta, nil
}

func applyReport(meta *Meta, report parse.Report) {
	meta.Warnings = append([]string{}, report.Warnings...)
	if len(report.Normalize.DroppedActivities) > 0 {
		meta.Warnings = append(meta.Warnings, "dropped activities with illegal type")
	}
	if len(report.Normalize.DroppedChapters) > 0 {
		meta.Warnings = append(meta.Warnings, "dropped empty chapters")
	}
	if meta.Warnings == nil {
		meta.Warnings = []string{}
	}
	repaired := report.RepairedFields
	if repaired == nil {
		repaired = []string{}
	}
	meta.Validation = Validation{Passed: true, RepairedFields: repaired}
}

func completeError(err error) error {
	if _, ok := aierr.As(err); ok {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return aierr.Timeout(err)
	}
	return aierr.ModelUnavailable(err)
}
