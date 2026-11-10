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
	FullCourseV1      = "full_course_v1"
	CourseOutlineV1   = "course_outline_v1"
	ChapterV1         = "chapter_v1"
	ActivityV1        = "activity_v1"
	ContentQualityV1  = "content_quality_v1"
	ChapterQAV1       = "chapter_qa_v1"
	LearningInsightV1 = "learning_insight_v1"
	PixelSpriteV1     = "pixel_sprite_v1"
	PixelDialogueV1   = "pixel_dialogue_v1"
	Version1          = "1"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var (
	fullCourseTemplate      = template.Must(template.ParseFS(templateFiles, "templates/full_course_v1.tmpl"))
	courseOutlineTemplate   = template.Must(template.ParseFS(templateFiles, "templates/course_outline_v1.tmpl"))
	chapterTemplate         = template.Must(template.ParseFS(templateFiles, "templates/chapter_v1.tmpl"))
	activityTemplate        = template.Must(template.ParseFS(templateFiles, "templates/activity_v1.tmpl"))
	qualityTemplate         = template.Must(template.ParseFS(templateFiles, "templates/content_quality_v1.tmpl"))
	chapterQATemplate       = template.Must(template.ParseFS(templateFiles, "templates/chapter_qa_v1.tmpl"))
	learningInsightTemplate = template.Must(template.ParseFS(templateFiles, "templates/learning_insight_v1.tmpl"))
	pixelSpriteTemplate     = template.Must(template.ParseFS(templateFiles, "templates/pixel_sprite_v1.tmpl"))
	pixelDialogueTemplate   = template.Must(template.ParseFS(templateFiles, "templates/pixel_dialogue_v1.tmpl"))
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

// ChapterQATurn 是一段不可信的对话历史。
type ChapterQATurn struct {
	Role         string
	Content      string
	SectionTitle string
}

// ChapterQAData 的章节、历史、选中文字和问题都属于不可信数据。
type ChapterQAData struct {
	Language      string
	ChapterJSON   string
	Question      string
	ActivityKey   string
	ActivityTitle string
	ActivityBody  string
	Selection     string
	History       []ChapterQATurn
}

// LearningInsightData 学生学习摘要 JSON。
type LearningInsightData struct {
	SummaryJSON string
	Language    string
}

// PixelSpriteData 是已经抽好的外观属性。模型只画网格，不能改这些属性。
type PixelSpriteData struct {
	CourseID           int64
	Title              string
	RoleType           string
	BodyType           string
	HairColor          string
	HairStyle          string
	FacialFeature      string
	Accessory          string
	SilhouetteKey      string
	Personality        string
	ValidationFeedback string
	PreviousSprite     string
}

// Render 渲染指定模板。未知 templateId 返回 AI_INVALID_INPUT。
func Render(templateID string, data any) (string, Info, error) {
	var parsed *template.Template
	info := Info{Version: Version1}
	switch templateID {
	case FullCourseV1:
		parsed = fullCourseTemplate
		info.ID = FullCourseV1
	case CourseOutlineV1:
		parsed = courseOutlineTemplate
		info.ID = CourseOutlineV1
	case ChapterV1:
		parsed = chapterTemplate
		info.ID = ChapterV1
	case ActivityV1:
		parsed = activityTemplate
		info.ID = ActivityV1
	case ContentQualityV1:
		parsed = qualityTemplate
		info.ID = ContentQualityV1
	case ChapterQAV1:
		parsed = chapterQATemplate
		info.ID = ChapterQAV1
	case LearningInsightV1:
		parsed = learningInsightTemplate
		info.ID = LearningInsightV1
	case PixelSpriteV1:
		parsed = pixelSpriteTemplate
		info.ID = PixelSpriteV1
	case PixelDialogueV1:
		parsed = pixelDialogueTemplate
		info.ID = PixelDialogueV1
	default:
		return "", Info{}, aierr.InvalidInput(fmt.Errorf("unknown template %q", templateID), "templateId")
	}
	var buffer bytes.Buffer
	if err := parsed.Execute(&buffer, data); err != nil {
		return "", Info{}, aierr.Internal(err, "templateId="+templateID)
	}
	return buffer.String(), info, nil
}
