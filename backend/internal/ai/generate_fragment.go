package ai

import (
	"context"
	"encoding/json"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
	"caveclass/internal/ai/validate"
)

const contextMissingWarning = "contextCourse was not provided"

// GenerateChapterInput 是单章生成的入参。ContextCourse 可选。
type GenerateChapterInput struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	TargetChapterKey  string
	ContextCourse     *course.StandardCourse
	Model             string
	TemplateID        string
}

// GenerateActivityInput 是单 activity 生成的入参。ContextCourse 可选。
type GenerateActivityInput struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	TargetChapterKey  string
	TargetActivityKey string
	ContextCourse     *course.StandardCourse
	Model             string
	TemplateID        string
}

// GenerateChapter 生成一个章节。没有 ContextCourse 时仍然成功，并在 meta.warnings 记录。
func (service *Service) GenerateChapter(ctx context.Context, input GenerateChapterInput) (course.Chapter, Meta, error) {
	if err := service.ready(); err != nil {
		return course.Chapter{}, Meta{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	checked, err := validate.NormalizeChapter(validate.ChapterInput{
		CourseInput: validate.CourseInput{
			Topic:             input.Topic,
			Audience:          input.Audience,
			Language:          input.Language,
			ExtraRequirements: input.ExtraRequirements,
			Model:             input.Model,
			TemplateID:        input.TemplateID,
		},
		TargetChapterKey: input.TargetChapterKey,
	}, service.config.MaxTopicLen)
	if err != nil {
		return course.Chapter{}, Meta{}, err
	}
	if checked.TemplateID == "" {
		checked.TemplateID = prompt.ChapterV1
	} else if checked.TemplateID != prompt.ChapterV1 {
		return course.Chapter{}, Meta{}, aierr.InvalidInput(nil, "templateId")
	}
	contextText, err := marshalContext(input.ContextCourse)
	if err != nil {
		return course.Chapter{}, Meta{}, err
	}
	response, meta, err := service.runModel(ctx, SceneChapter, checked.TemplateID, checked.Model, prompt.ChapterData{
		Topic:             checked.Topic,
		Audience:          checked.Audience,
		Language:          checked.Language,
		ExtraRequirements: checked.ExtraRequirements,
		TargetChapterKey:  checked.TargetChapterKey,
		ContextJSON:       contextText,
	})
	if err != nil {
		return course.Chapter{}, meta, err
	}
	parsed, report, err := parse.ParseChapterJSON(response.Text, true)
	if err != nil {
		return course.Chapter{}, meta, err
	}
	if parsed.Key != checked.TargetChapterKey {
		parsed.Key = checked.TargetChapterKey
		report.RepairedFields = append(report.RepairedFields, "key")
	}
	if err := parsed.Validate(); err != nil {
		return course.Chapter{}, meta, aierr.OutputSchemaFailed(err, err.Error())
	}
	applyReport(&meta, report)
	noteMissingContext(&meta, input.ContextCourse)
	return parsed, meta, nil
}

// GenerateActivity 生成一个 activity。没有 ContextCourse 时仍然成功，并在 meta.warnings 记录。
func (service *Service) GenerateActivity(ctx context.Context, input GenerateActivityInput) (course.Activity, Meta, error) {
	if err := service.ready(); err != nil {
		return course.Activity{}, Meta{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	checked, err := validate.NormalizeActivity(validate.ActivityInput{
		ChapterInput: validate.ChapterInput{
			CourseInput: validate.CourseInput{
				Topic:             input.Topic,
				Audience:          input.Audience,
				Language:          input.Language,
				ExtraRequirements: input.ExtraRequirements,
				Model:             input.Model,
				TemplateID:        input.TemplateID,
			},
			TargetChapterKey: input.TargetChapterKey,
		},
		TargetActivityKey: input.TargetActivityKey,
	}, service.config.MaxTopicLen)
	if err != nil {
		return course.Activity{}, Meta{}, err
	}
	if checked.TemplateID == "" {
		checked.TemplateID = prompt.ActivityV1
	} else if checked.TemplateID != prompt.ActivityV1 {
		return course.Activity{}, Meta{}, aierr.InvalidInput(nil, "templateId")
	}
	contextText, err := marshalActivityContext(input.ContextCourse, checked.TargetChapterKey)
	if err != nil {
		return course.Activity{}, Meta{}, err
	}
	response, meta, err := service.runModel(ctx, SceneActivity, checked.TemplateID, checked.Model, prompt.ActivityData{
		Topic:             checked.Topic,
		Audience:          checked.Audience,
		Language:          checked.Language,
		ExtraRequirements: checked.ExtraRequirements,
		TargetChapterKey:  checked.TargetChapterKey,
		TargetActivityKey: checked.TargetActivityKey,
		ContextJSON:       contextText,
	})
	if err != nil {
		return course.Activity{}, meta, err
	}
	parsed, report, err := parse.ParseActivityJSON(response.Text, true)
	if err != nil {
		return course.Activity{}, meta, err
	}
	if parsed.Key != checked.TargetActivityKey {
		parsed.Key = checked.TargetActivityKey
		report.RepairedFields = append(report.RepairedFields, "key")
	}
	if err := parsed.Validate(); err != nil {
		return course.Activity{}, meta, aierr.OutputSchemaFailed(err, err.Error())
	}
	applyReport(&meta, report)
	noteMissingContext(&meta, input.ContextCourse)
	return parsed, meta, nil
}

func (service *Service) ready() error {
	if service == nil || service.client == nil {
		return aierr.Internal(nil, "ai service is not configured")
	}
	return nil
}

func marshalContext(contextCourse *course.StandardCourse) (string, error) {
	if contextCourse == nil {
		return "", nil
	}
	encoded, err := json.Marshal(contextCourse)
	if err != nil {
		return "", aierr.InvalidInput(err, "contextCourse")
	}
	return string(encoded), nil
}

func noteMissingContext(meta *Meta, contextCourse *course.StandardCourse) {
	if contextCourse != nil {
		return
	}
	meta.Warnings = append(meta.Warnings, contextMissingWarning)
}
