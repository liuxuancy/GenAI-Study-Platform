// Package prompt 加载版本化的 Prompt 模板。本阶段只有整课模板 full_course_v1。
package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"caveclass/internal/ai/aierr"
)

const (
	FullCourseV1 = "full_course_v1"
	Version1     = "1"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var fullCourseTemplate = template.Must(template.ParseFS(templateFiles, "templates/full_course_v1.tmpl"))

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

// Render 渲染指定模板。未知 templateId 返回 AI_INVALID_INPUT。
func Render(templateID string, data FullCourseData) (string, Info, error) {
	info := Info{ID: FullCourseV1, Version: Version1}
	if templateID != FullCourseV1 {
		return "", Info{}, aierr.InvalidInput(fmt.Errorf("unknown template %q", templateID), "templateId")
	}
	var buffer bytes.Buffer
	if err := fullCourseTemplate.Execute(&buffer, data); err != nil {
		return "", Info{}, aierr.Internal(err, "templateId="+templateID)
	}
	return buffer.String(), info, nil
}
