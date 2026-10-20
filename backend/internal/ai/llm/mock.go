package llm

import (
	"context"
	_ "embed"
	"strings"
)

//go:embed testdata/mock_course.json
var mockCourseJSON []byte

// MockText 是 mock 客户端固定返回的课程 JSON，内容与模块说明 §12 样例一致。
func MockText() string {
	return strings.TrimSpace(string(mockCourseJSON))
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
	return Response{Text: mock.text, Model: model}, nil
}
