// Package course 定义标准课程 JSON（schema 1.0）以及校验、规范化。
// 本包不访问数据库，也不注册 HTTP 路由。
package course

import "encoding/json"

const SchemaVersion = "1.0"

const (
	MaxTitleRunes       = 200
	MaxDescriptionRunes = 2000
	MaxKeyRunes         = 128 // 规格未写上限；稳定标识按 128 个字符封顶
)

const (
	AudienceCS      = "CS"
	AudienceNonCS   = "NON_CS"
	AudienceGeneral = "GENERAL"

	LanguageZhCN = "zh-CN"
	LanguageEnUS = "en-US"

	ActivityTypeRichText      = "rich_text"
	ActivityTypeCoding        = "coding"
	ActivityTypeMockInterview = "mock_interview"
	ActivityTypeQuizGame      = "quiz_game"

	CodingLanguagePython     = "python"
	CodingLanguageJavaScript = "javascript"
	CodingLanguageJava       = "java"
	CodingLanguageGo         = "go"
	CodingLanguageOther      = "other"
)

// StandardCourse 是 AI 输出与校验使用的标准课程。
type StandardCourse struct {
	SchemaVersion    string    `json:"schemaVersion"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
	Audience         string    `json:"audience"`
	Language         string    `json:"language"`
	EstimatedMinutes int       `json:"estimatedMinutes"`
	Chapters         []Chapter `json:"chapters"`
}

// Chapter 是课程中的一章。Key 是稳定标识，供后续局部重生成合并。
type Chapter struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Summary    string     `json:"summary,omitempty"`
	Order      int        `json:"order"`
	Activities []Activity `json:"activities"`
}

// Activity 是章节中的一个学习活动。
// Payload 保留原始 JSON：非法 type 仍能反序列化，再由 Validate 拒绝或由 Normalize 丢弃。
type Activity struct {
	Key     string          `json:"key"`
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Order   int             `json:"order"`
	Payload json.RawMessage `json:"payload"`
}

// RichTextPayload 是 type=rich_text 的 payload。
type RichTextPayload struct {
	Markdown   string   `json:"markdown"`
	Highlights []string `json:"highlights,omitempty"`
}

// CodingPayload 是 type=coding 的 payload。
type CodingPayload struct {
	Language       string     `json:"language"`
	PromptMarkdown string     `json:"promptMarkdown"`
	StarterCode    string     `json:"starterCode,omitempty"`
	SolutionHint   string     `json:"solutionHint,omitempty"`
	TestCases      []TestCase `json:"testCases,omitempty"`
}

// TestCase 是编程活动的一条输入输出用例。
type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
	IsHidden       bool   `json:"isHidden"`
}

// MockInterviewPayload 是 type=mock_interview 的 payload。
type MockInterviewPayload struct {
	Role                     string              `json:"role"`
	ScenarioMarkdown         string              `json:"scenarioMarkdown"`
	Questions                []InterviewQuestion `json:"questions"`
	EvaluationRubricMarkdown string              `json:"evaluationRubricMarkdown"`
}

// InterviewQuestion 是模拟面试中的一道题。
type InterviewQuestion struct {
	Question      string   `json:"question"`
	FollowUpHints []string `json:"followUpHints,omitempty"`
}

// QuizGamePayload 是 type=quiz_game 的 payload。
type QuizGamePayload struct {
	InstructionsMarkdown string         `json:"instructionsMarkdown"`
	Questions            []QuizQuestion `json:"questions"`
}

// QuizQuestion 是测验中的一道选择题。CorrectIndex 从 0 开始。
type QuizQuestion struct {
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correctIndex"`
	Explanation  string   `json:"explanation"`
}

func validActivityType(activityType string) bool {
	switch activityType {
	case ActivityTypeRichText, ActivityTypeCoding, ActivityTypeMockInterview, ActivityTypeQuizGame:
		return true
	default:
		return false
	}
}
