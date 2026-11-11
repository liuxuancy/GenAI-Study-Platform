package llm

import (
	"context"
	_ "embed"
	"strings"
)

//go:embed testdata/mock_course.json
var mockCourseJSON []byte

//go:embed testdata/mock_chapter.json
var mockChapterJSON []byte

//go:embed testdata/mock_activity.json
var mockActivityJSON []byte

//go:embed testdata/mock_outline.json
var mockOutlineJSON []byte

//go:embed testdata/mock_quality.json
var mockQualityJSON []byte

//go:embed testdata/mock_gal_script.json
var mockGalScriptJSON []byte

// MockText 是 mock 客户端固定返回的课程 JSON，内容与模块说明 §12 样例一致。
func MockText() string {
	return strings.TrimSpace(string(mockCourseJSON))
}

// MockChapterText 是单章生成时 mock 返回的章节 JSON。
func MockChapterText() string {
	return strings.TrimSpace(string(mockChapterJSON))
}

// MockActivityText 是单 activity 生成时 mock 返回的 JSON。
func MockActivityText() string {
	return strings.TrimSpace(string(mockActivityJSON))
}

func MockOutlineText() string {
	return strings.TrimSpace(string(mockOutlineJSON))
}

// MockQualityText 是质量评估时 mock 返回的报告 JSON。
func MockQualityText() string {
	return strings.TrimSpace(string(mockQualityJSON))
}

func MockGalScriptText() string {
	return strings.TrimSpace(string(mockGalScriptJSON))
}

type mockClient struct {
	model string
	text  string
}

func (mock *mockClient) Complete(ctx context.Context, request Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	model := request.Model
	if model == "" {
		model = mock.model
	}
	return Response{Text: mockTextForPrompt(request.Prompt, mock.text), Model: model}, nil
}

// 模板第一行标明 templateId。用户内容在后面，不能靠正文里的同名字符串切换返回值。
func mockTextForPrompt(promptText, courseText string) string {
	line, _, _ := strings.Cut(promptText, "\n")
	switch strings.TrimSpace(line) {
	case "templateId: course_outline_v1":
		return MockOutlineText()
	case "templateId: chapter_v1":
		return MockChapterText()
	case "templateId: activity_v1":
		return MockActivityText()
	case "templateId: content_quality_v1":
		return MockQualityText()
	case "templateId: chapter_qa_v1":
		return "这是仅根据当前章节内容生成的回答。"
	case "templateId: learning_insight_v1":
		return `{"suggestion":"根据你的进度，建议先完成进行中的课程章节，再处理待办测验。","recommendedAction":{"title":"继续学习","detail":"打开当前进度最高的课程，完成下一节内容。"}}`
	case "templateId: gal_script_v1":
		return MockGalScriptText()
	default:
		return courseText
	}
}
