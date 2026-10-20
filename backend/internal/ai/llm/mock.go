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
	case "templateId: chapter_v1":
		return MockChapterText()
	case "templateId: activity_v1":
		return MockActivityText()
	default:
		return courseText
	}
}
