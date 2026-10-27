package ai

import (
	"context"
	"encoding/json"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/parse"
	"caveclass/internal/ai/prompt"
	"caveclass/internal/ai/quality"
	"caveclass/internal/ai/validate"
)

// EvaluateCourse 按 content_quality_v1 评估一门已经符合 schema 的课程。
// rubric 可空。课程不合法或 rubric 含注入时不会调用模型。
func (service *Service) EvaluateCourse(ctx context.Context, subject course.StandardCourse, rubric string) (quality.Report, Meta, error) {
	if err := service.ready(); err != nil {
		return quality.Report{}, Meta{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := subject.Validate(); err != nil {
		return quality.Report{}, Meta{}, invalidCourse(err)
	}
	checkedRubric, err := validate.NormalizeRubric(rubric)
	if err != nil {
		return quality.Report{}, Meta{}, err
	}
	encoded, err := json.Marshal(subject)
	if err != nil {
		return quality.Report{}, Meta{}, aierr.InvalidInput(err, "course")
	}
	response, meta, err := service.runModel(ctx, SceneEvaluate, prompt.ContentQualityV1, "", prompt.QualityData{
		Rubric:     checkedRubric,
		CourseJSON: string(encoded),
	})
	if err != nil {
		return quality.Report{}, meta, err
	}
	parsed, report, err := parse.ParseQualityJSON(response.Text, true)
	if err != nil {
		return quality.Report{}, meta, err
	}
	applyReport(&meta, report)
	return parsed, meta, nil
}

func invalidCourse(err error) error {
	validationError, ok := err.(*course.ValidationError)
	if !ok {
		return aierr.InvalidInput(err, "course")
	}
	details := make([]string, len(validationError.Problems))
	for index, problem := range validationError.Problems {
		details[index] = problem.Field + ": " + problem.Message
	}
	return aierr.InvalidInput(nil, details...)
}
