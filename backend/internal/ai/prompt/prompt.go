// Package prompt 加载版本化的 Prompt 模板。
package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"caveclass/internal/ai/aierr"
)

const (
	FullCourseV1     = "full_course_v1"
	ChapterV1        = "chapter_v1"
	ActivityV1       = "activity_v1"
	ContentQualityV1 = "content_quality_v1"
	Version1         = "1"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var (
	fullCourseTemplate = template.Must(template.ParseFS(templateFiles, "templates/full_course_v1.tmpl"))
	chapterTemplate    = template.Must(template.ParseFS(templateFiles, "templates/chapter_v1.tmpl"))
	activityTemplate   = template.Must(template.ParseFS(templateFiles, "templates/activity_v1.tmpl"))
	qualityTemplate    = template.Must(template.ParseFS(templateFiles, "templates/content_quality_v1.tmpl"))
)

// Info 写入 meta.templateId 和 meta.templateVersion。
type Info struct {
	ID      string
	Version string
}

// FullCourseData 是整课模板的数据。调用方应先做入参校验。
type FullCourseData struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	ChapterCountHint  int
}

// ChapterData 是单章模板的数据。ContextJSON 为空表示调用方没有提供上下文。
type ChapterData struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	TargetChapterKey  string
	ContextJSON       string
}

// ActivityData 是单 activity 模板的数据。
type ActivityData struct {
	Topic             string
	Audience          string
	Language          string
	ExtraRequirements string
	TargetChapterKey  string
	TargetActivityKey string
	ContextJSON       string
}

// QualityData 是质量评估模板的数据。Rubric 为空表示调用方没有附加评分要求。
type QualityData struct {
	Rubric     string
	CourseJSON string
}

// Render 渲染指定模板。未知 templateId 返回 AI_INVALID_INPUT。
func Render(templateID string, data any) (string, Info, error) {
	var parsed *template.Template
	info := Info{Version: Version1}
	switch templateID {
	case FullCourseV1:
		parsed = fullCourseTemplate
		info.ID = FullCourseV1
	case ChapterV1:
		parsed = chapterTemplate
		info.ID = ChapterV1
	case ActivityV1:
		parsed = activityTemplate
		info.ID = ActivityV1
	case ContentQualityV1:
		parsed = qualityTemplate
		info.ID = ContentQualityV1
	default:
		return "", Info{}, aierr.InvalidInput(fmt.Errorf("unknown template %q", templateID), "templateId")
	}
	var buffer bytes.Buffer
	if err := parsed.Execute(&buffer, data); err != nil {
		return "", Info{}, aierr.Internal(err, "templateId="+templateID)
	}
	return buffer.String(), info, nil
}
