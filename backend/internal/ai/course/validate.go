package course

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Problem 是校验失败的一个字段。Field 使用 JSON 路径，例如 chapters[0].activities[1].type。
type Problem struct {
	Field   string
	Message string
}

// ValidationError 收集标准课程结构中的全部问题。Validate 不修改课程。
type ValidationError struct {
	Problems []Problem
}

func (validationError *ValidationError) Error() string {
	if validationError == nil || len(validationError.Problems) == 0 {
		return "course validation failed"
	}
	parts := make([]string, len(validationError.Problems))
	for i, problem := range validationError.Problems {
		parts[i] = problem.Field + ": " + problem.Message
	}
	return "course validation failed: " + strings.Join(parts, "; ")
}

type checker struct {
	problems []Problem
}

func (checker *checker) add(field, message string) {
	checker.problems = append(checker.problems, Problem{Field: field, Message: message})
}

func (checker *checker) err() error {
	if len(checker.problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: checker.problems}
}

// Validate 按 schema 1.0 检查课程。
// 缺必填、非法枚举（含 activity type）、payload 与 type 不符、重复 key 时返回 *ValidationError。
// order 是否连续由 Normalize 负责，这里不要求从 1 连续编号。
func (course *StandardCourse) Validate() error {
	if course == nil {
		return &ValidationError{Problems: []Problem{{Field: "course", Message: "is nil"}}}
	}
	checker := &checker{}
	checker.requireSchemaVersion(course.SchemaVersion)
	checker.requireText("title", course.Title, MaxTitleRunes)
	checker.optionalText("description", course.Description, MaxDescriptionRunes)
	checker.oneOf("audience", course.Audience, AudienceCS, AudienceNonCS, AudienceGeneral)
	checker.oneOf("language", course.Language, LanguageZhCN, LanguageEnUS)
	if course.EstimatedMinutes < 0 {
		checker.add("estimatedMinutes", "must be greater than or equal to 0")
	}
	if len(course.Chapters) == 0 {
		checker.add("chapters", "at least one chapter is required")
	}

	chapterKeys := map[string]struct{}{}
	activityKeys := map[string]struct{}{}
	for chapterIndex, chapter := range course.Chapters {
		checker.chapter(chapterIndex, chapter, chapterKeys, activityKeys)
	}
	return checker.err()
}

func (checker *checker) chapter(chapterIndex int, chapter Chapter, chapterKeys, activityKeys map[string]struct{}) {
	prefix := fmt.Sprintf("chapters[%d]", chapterIndex)
	checker.requireText(prefix+".key", chapter.Key, MaxKeyRunes)
	checker.requireText(prefix+".title", chapter.Title, MaxTitleRunes)
	checker.optionalText(prefix+".summary", chapter.Summary, MaxDescriptionRunes)
	checker.uniqueKey(prefix+".key", chapter.Key, chapterKeys)
	if len(chapter.Activities) == 0 {
		checker.add(prefix+".activities", "at least one activity is required")
		return
	}
	for activityIndex, activity := range chapter.Activities {
		checker.activity(fmt.Sprintf("%s.activities[%d]", prefix, activityIndex), activity, activityKeys)
	}
}

func (checker *checker) activity(prefix string, activity Activity, activityKeys map[string]struct{}) {
	checker.requireText(prefix+".key", activity.Key, MaxKeyRunes)
	checker.uniqueKey(prefix+".key", activity.Key, activityKeys)
	checker.requireText(prefix+".title", activity.Title, MaxTitleRunes)
	if !checker.oneOf(prefix+".type", activity.Type,
		ActivityTypeRichText, ActivityTypeCoding, ActivityTypeMockInterview, ActivityTypeQuizGame) {
		return
	}
	checker.payload(prefix+".payload", activity)
}

func (checker *checker) payload(path string, activity Activity) {
	payloadText := bytes.TrimSpace(activity.Payload)
	if len(payloadText) == 0 || bytes.Equal(payloadText, []byte("null")) {
		checker.add(path, "required")
		return
	}
	switch activity.Type {
	case ActivityTypeRichText:
		checker.richText(path, activity.Payload)
	case ActivityTypeCoding:
		checker.coding(path, activity.Payload)
	case ActivityTypeMockInterview:
		checker.mockInterview(path, activity.Payload)
	case ActivityTypeQuizGame:
		checker.quizGame(path, activity.Payload)
	}
}

func (checker *checker) richText(path string, raw json.RawMessage) {
	var payload RichTextPayload
	if !checker.decodePayload(path, raw, &payload) {
		return
	}
	checker.requireText(path+".markdown", payload.Markdown, 0)
	for i, highlight := range payload.Highlights {
		checker.requireText(fmt.Sprintf("%s.highlights[%d]", path, i), highlight, 0)
	}
}

func (checker *checker) coding(path string, raw json.RawMessage) {
	var payload CodingPayload
	if !checker.decodePayload(path, raw, &payload) {
		return
	}
	checker.oneOf(path+".language", payload.Language,
		CodingLanguagePython, CodingLanguageJavaScript, CodingLanguageJava, CodingLanguageGo, CodingLanguageOther)
	checker.requireText(path+".promptMarkdown", payload.PromptMarkdown, 0)
	for i, testCase := range payload.TestCases {
		checker.requireText(fmt.Sprintf("%s.testCases[%d].expectedOutput", path, i), testCase.ExpectedOutput, 0)
	}
}

func (checker *checker) mockInterview(path string, raw json.RawMessage) {
	var payload MockInterviewPayload
	if !checker.decodePayload(path, raw, &payload) {
		return
	}
	checker.requireText(path+".role", payload.Role, 0)
	checker.requireText(path+".scenarioMarkdown", payload.ScenarioMarkdown, 0)
	checker.requireText(path+".evaluationRubricMarkdown", payload.EvaluationRubricMarkdown, 0)
	if len(payload.Questions) == 0 {
		checker.add(path+".questions", "at least one question is required")
		return
	}
	for i, question := range payload.Questions {
		questionPath := fmt.Sprintf("%s.questions[%d]", path, i)
		checker.requireText(questionPath+".question", question.Question, 0)
		for j, hint := range question.FollowUpHints {
			checker.requireText(fmt.Sprintf("%s.followUpHints[%d]", questionPath, j), hint, 0)
		}
	}
}

func (checker *checker) quizGame(path string, raw json.RawMessage) {
	var payload QuizGamePayload
	if !checker.decodePayload(path, raw, &payload) {
		return
	}
	checker.requireText(path+".instructionsMarkdown", payload.InstructionsMarkdown, 0)
	if len(payload.Questions) == 0 {
		checker.add(path+".questions", "at least one question is required")
		return
	}
	for i, question := range payload.Questions {
		questionPath := fmt.Sprintf("%s.questions[%d]", path, i)
		checker.requireText(questionPath+".question", question.Question, 0)
		if len(question.Options) < 2 {
			checker.add(questionPath+".options", "at least 2 options are required")
		} else {
			for j, option := range question.Options {
				checker.requireText(fmt.Sprintf("%s.options[%d]", questionPath, j), option, 0)
			}
			if question.CorrectIndex < 0 || question.CorrectIndex >= len(question.Options) {
				checker.add(questionPath+".correctIndex", "out of range")
			}
		}
		checker.requireText(questionPath+".explanation", question.Explanation, 0)
	}
}

func (checker *checker) decodePayload(path string, raw json.RawMessage, dest any) bool {
	if err := json.Unmarshal(raw, dest); err != nil {
		checker.add(path, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func (checker *checker) requireSchemaVersion(value string) {
	if strings.TrimSpace(value) == "" {
		checker.add("schemaVersion", "required")
		return
	}
	if value != SchemaVersion {
		checker.add("schemaVersion", fmt.Sprintf("must be %q", SchemaVersion))
	}
}

// oneOf 在空值或非法枚举时记录问题。返回值表示值是否在允许列表中。
func (checker *checker) oneOf(field, value string, allowed ...string) bool {
	if strings.TrimSpace(value) == "" {
		checker.add(field, "required")
		return false
	}
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	checker.add(field, "must be one of "+strings.Join(allowed, ", "))
	return false
}

func (checker *checker) requireText(field, value string, maxRunes int) {
	if strings.TrimSpace(value) == "" {
		checker.add(field, "required")
		return
	}
	if maxRunes > 0 && utf8.RuneCountInString(value) > maxRunes {
		checker.add(field, fmt.Sprintf("must be at most %d characters", maxRunes))
	}
}

func (checker *checker) optionalText(field, value string, maxRunes int) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if maxRunes > 0 && utf8.RuneCountInString(value) > maxRunes {
		checker.add(field, fmt.Sprintf("must be at most %d characters", maxRunes))
	}
}

func (checker *checker) uniqueKey(field, key string, seen map[string]struct{}) {
	if strings.TrimSpace(key) == "" {
		return
	}
	if _, exists := seen[key]; exists {
		checker.add(field, fmt.Sprintf("duplicate key %q", key))
		return
	}
	seen[key] = struct{}{}
}
