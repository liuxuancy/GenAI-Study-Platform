// Package quality 定义内容质量评估报告。维度名在与前端对齐前固定为这四项。
package quality

// 四个维度都会出现在 content_quality_v1 的输出里，各评一次。
const (
	DimensionCoverage    = "coverage"
	DimensionClarity     = "clarity"
	DimensionPractice    = "practice"
	DimensionAudienceFit = "audience_fit"

	MaxSummaryRunes    = 4000
	MaxCommentRunes    = 2000
	MaxSuggestionRunes = 500
	MaxSuggestions     = 20
)

// Report 是 EvaluateCourse 的结果。分数都是 0 到 100 的整数。
type Report struct {
	OverallScore    int         `json:"overallScore"`
	SummaryMarkdown string      `json:"summaryMarkdown"`
	Dimensions      []Dimension `json:"dimensions"`
	Suggestions     []string    `json:"suggestions"`
}

// Dimension 是一个评分维度。
type Dimension struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}
