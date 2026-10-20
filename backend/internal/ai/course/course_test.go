package course_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"caveclass/internal/ai/course"
)

func loadMiniCourse(t *testing.T) course.StandardCourse {
	t.Helper()
	data, err := os.ReadFile("testdata/mini_course.json")
	if err != nil {
		t.Fatal(err)
	}
	var loaded course.StandardCourse
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	return loaded
}

func requireProblem(t *testing.T, err error, field, messagePart string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error")
	}
	validationError, ok := errors.AsType[*course.ValidationError](err)
	if !ok {
		t.Fatalf("got %T (%v), want *course.ValidationError", err, err)
	}
	if len(validationError.Problems) != 1 {
		t.Fatalf("problems = %+v, want only %s", validationError.Problems, field)
	}
	problem := validationError.Problems[0]
	if problem.Field != field {
		t.Fatalf("field = %q, want %q", problem.Field, field)
	}
	if !strings.Contains(problem.Message, messagePart) {
		t.Fatalf("message = %q, want substring %q", problem.Message, messagePart)
	}
}

func TestValidateMiniCourse(t *testing.T) {
	loaded := loadMiniCourse(t)
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Introduction to Git for Beginners" || loaded.Audience != course.AudienceNonCS {
		t.Fatalf("unexpected course fields: %+v", loaded)
	}
	if loaded.EstimatedMinutes != 45 || len(loaded.Chapters) != 2 {
		t.Fatalf("chapters = %d, minutes = %d", len(loaded.Chapters), loaded.EstimatedMinutes)
	}
	if loaded.Chapters[0].Activities[0].Type != course.ActivityTypeRichText ||
		loaded.Chapters[0].Activities[1].Type != course.ActivityTypeQuizGame ||
		loaded.Chapters[1].Activities[0].Type != course.ActivityTypeCoding {
		t.Fatalf("unexpected activity types: %+v", loaded.Chapters)
	}

	// 再编码一轮，确认 json 标签能往返，且 payload 仍能通过校验。
	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip course.StandardCourse
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if err := roundTrip.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAllowsOptionalFieldsAndZeroMinutes(t *testing.T) {
	loaded := loadMiniCourse(t)
	loaded.Description = ""
	loaded.EstimatedMinutes = 0
	loaded.Chapters[0].Summary = "   "
	loaded.Chapters[0].Activities[0].Payload = json.RawMessage(`{"markdown":"只要正文"}`)
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAcceptsBoundaryLengths(t *testing.T) {
	loaded := loadMiniCourse(t)
	loaded.Title = strings.Repeat("课", course.MaxTitleRunes)
	loaded.Description = strings.Repeat("a", course.MaxDescriptionRunes)
	loaded.Chapters[0].Key = strings.Repeat("k", course.MaxKeyRunes)
	loaded.Chapters[0].Summary = strings.Repeat("s", course.MaxDescriptionRunes)
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAcceptsMockInterview(t *testing.T) {
	loaded := loadMiniCourse(t)
	loaded.Chapters[0].Activities[0].Type = course.ActivityTypeMockInterview
	loaded.Chapters[0].Activities[0].Payload = json.RawMessage(`{
		"role": "interviewer",
		"scenarioMarkdown": "介绍一个做过的项目",
		"questions": [{"question": "讲一下 Git", "followUpHints": ["分支怎么用"]}],
		"evaluationRubricMarkdown": "看是否提到版本控制"
	}`)
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsIncompleteOrIllegalCourse(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(*course.StandardCourse)
		field   string
		message string
	}{
		{name: "nil course", field: "course", message: "is nil"},
		{name: "missing schemaVersion", edit: func(item *course.StandardCourse) { item.SchemaVersion = "  " }, field: "schemaVersion", message: "required"},
		{name: "unknown schemaVersion", edit: func(item *course.StandardCourse) { item.SchemaVersion = "2.0" }, field: "schemaVersion", message: `must be "1.0"`},
		{name: "missing title", edit: func(item *course.StandardCourse) { item.Title = "" }, field: "title", message: "required"},
		{name: "title too long", edit: func(item *course.StandardCourse) { item.Title = strings.Repeat("课", course.MaxTitleRunes+1) }, field: "title", message: "at most 200"},
		{name: "description too long", edit: func(item *course.StandardCourse) {
			item.Description = strings.Repeat("a", course.MaxDescriptionRunes+1)
		}, field: "description", message: "at most 2000"},
		{name: "missing audience", edit: func(item *course.StandardCourse) { item.Audience = "" }, field: "audience", message: "required"},
		{name: "illegal audience", edit: func(item *course.StandardCourse) { item.Audience = "student" }, field: "audience", message: "must be one of CS, NON_CS, GENERAL"},
		{name: "missing language", edit: func(item *course.StandardCourse) { item.Language = "" }, field: "language", message: "required"},
		{name: "illegal language", edit: func(item *course.StandardCourse) { item.Language = "zh" }, field: "language", message: "must be one of zh-CN, en-US"},
		{name: "negative minutes", edit: func(item *course.StandardCourse) { item.EstimatedMinutes = -1 }, field: "estimatedMinutes", message: "greater than or equal to 0"},
		{name: "no chapters", edit: func(item *course.StandardCourse) { item.Chapters = nil }, field: "chapters", message: "at least one chapter"},
		{name: "chapter missing key", edit: func(item *course.StandardCourse) { item.Chapters[0].Key = "" }, field: "chapters[0].key", message: "required"},
		{name: "chapter key too long", edit: func(item *course.StandardCourse) { item.Chapters[0].Key = strings.Repeat("k", course.MaxKeyRunes+1) }, field: "chapters[0].key", message: "at most 128"},
		{name: "duplicate chapter key", edit: func(item *course.StandardCourse) { item.Chapters[1].Key = item.Chapters[0].Key }, field: "chapters[1].key", message: `duplicate key "ch-1"`},
		{name: "chapter without activities", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities = nil }, field: "chapters[0].activities", message: "at least one activity"},
		{name: "activity missing key", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Key = "" }, field: "chapters[0].activities[0].key", message: "required"},
		{name: "duplicate activity key", edit: func(item *course.StandardCourse) {
			item.Chapters[1].Activities[0].Key = item.Chapters[0].Activities[0].Key
		}, field: "chapters[1].activities[0].key", message: `duplicate key "act-1-1"`},
		{name: "activity missing title", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Title = " " }, field: "chapters[0].activities[0].title", message: "required"},
		{name: "illegal activity type", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Type = "video" }, field: "chapters[0].activities[0].type", message: "must be one of rich_text, coding, mock_interview, quiz_game"},
		{name: "missing payload", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Payload = nil }, field: "chapters[0].activities[0].payload", message: "required"},
		{name: "null payload", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Payload = json.RawMessage("null") }, field: "chapters[0].activities[0].payload", message: "required"},
		{name: "rich text payload is not object", edit: func(item *course.StandardCourse) { item.Chapters[0].Activities[0].Payload = json.RawMessage(`"text"`) }, field: "chapters[0].activities[0].payload", message: "invalid JSON"},
		{name: "rich text missing markdown", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[0].Payload = json.RawMessage(`{"highlights":["版本控制"]}`)
		}, field: "chapters[0].activities[0].payload.markdown", message: "required"},
		{name: "blank highlight", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[0].Payload = json.RawMessage(`{"markdown":"正文","highlights":[""]}`)
		}, field: "chapters[0].activities[0].payload.highlights[0]", message: "required"},
		{name: "illegal coding language", edit: func(item *course.StandardCourse) {
			item.Chapters[1].Activities[0].Payload = json.RawMessage(`{"language":"Python","promptMarkdown":"写命令","testCases":[{"input":"","expectedOutput":"git status"}]}`)
		}, field: "chapters[1].activities[0].payload.language", message: "must be one of python, javascript, java, go, other"},
		{name: "coding missing prompt", edit: func(item *course.StandardCourse) {
			item.Chapters[1].Activities[0].Payload = json.RawMessage(`{"language":"other","promptMarkdown":"  "}`)
		}, field: "chapters[1].activities[0].payload.promptMarkdown", message: "required"},
		{name: "coding case missing expected output", edit: func(item *course.StandardCourse) {
			item.Chapters[1].Activities[0].Payload = json.RawMessage(`{"language":"go","promptMarkdown":"写命令","testCases":[{"input":"","expectedOutput":""}]}`)
		}, field: "chapters[1].activities[0].payload.testCases[0].expectedOutput", message: "required"},
		{name: "quiz correctIndex out of range", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[1].Payload = json.RawMessage(`{"instructionsMarkdown":"选择","questions":[{"question":"Git 是？","options":["版本控制系统","数据库"],"correctIndex":2,"explanation":"因为是。"}]}`)
		}, field: "chapters[0].activities[1].payload.questions[0].correctIndex", message: "out of range"},
		{name: "quiz too few options", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[1].Payload = json.RawMessage(`{"instructionsMarkdown":"选择","questions":[{"question":"Git 是？","options":["只有一个"],"correctIndex":0,"explanation":"因为是。"}]}`)
		}, field: "chapters[0].activities[1].payload.questions[0].options", message: "at least 2 options"},
		{name: "mock interview without questions", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[0].Type = course.ActivityTypeMockInterview
			item.Chapters[0].Activities[0].Payload = json.RawMessage(`{"role":"面试官","scenarioMarkdown":"场景","questions":[],"evaluationRubricMarkdown":"标准"}`)
		}, field: "chapters[0].activities[0].payload.questions", message: "at least one question"},
		{name: "blank follow-up hint", edit: func(item *course.StandardCourse) {
			item.Chapters[0].Activities[0].Type = course.ActivityTypeMockInterview
			item.Chapters[0].Activities[0].Payload = json.RawMessage(`{"role":"面试官","scenarioMarkdown":"场景","questions":[{"question":"讲一下","followUpHints":[" "]}],"evaluationRubricMarkdown":"标准"}`)
		}, field: "chapters[0].activities[0].payload.questions[0].followUpHints[0]", message: "required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "nil course" {
				var loaded *course.StandardCourse
				requireProblem(t, loaded.Validate(), test.field, test.message)
				return
			}
			loaded := loadMiniCourse(t)
			test.edit(&loaded)
			requireProblem(t, loaded.Validate(), test.field, test.message)
		})
	}
}

func TestValidateDoesNotMutateOrder(t *testing.T) {
	loaded := loadMiniCourse(t)
	loaded.Chapters[0].Order = 9
	loaded.Chapters[0].Activities[0].Order = 4
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
	if loaded.Chapters[0].Order != 9 || loaded.Chapters[0].Activities[0].Order != 4 {
		t.Fatalf("Validate changed order: chapter %d activity %d", loaded.Chapters[0].Order, loaded.Chapters[0].Activities[0].Order)
	}
}

func TestNormalizeRewritesOrderAndDropsIllegalActivities(t *testing.T) {
	richText := json.RawMessage(`{"markdown":"正文","highlights":["要点"]}`)
	quiz := json.RawMessage(`{"instructionsMarkdown":"选择","questions":[{"question":"问","options":["对","错"],"correctIndex":0,"explanation":"因为对"}]}`)
	loaded := course.StandardCourse{
		SchemaVersion: course.SchemaVersion,
		Title:         "Normalize",
		Audience:      course.AudienceCS,
		Language:      course.LanguageEnUS,
		Chapters: []course.Chapter{
			{
				Key: "ch-late", Title: "Later", Order: 5,
				Activities: []course.Activity{
					{Key: "act-b", Type: course.ActivityTypeRichText, Title: "B", Order: 4, Payload: richText},
					{Key: "act-bad", Type: "video", Title: "Dropped", Order: 1, Payload: json.RawMessage(`{}`)},
					{Key: "act-a", Type: course.ActivityTypeQuizGame, Title: "A", Order: 2, Payload: quiz},
				},
			},
			{
				Key: "ch-empty", Title: "Only illegal", Order: 1,
				Activities: []course.Activity{
					{Key: "act-x", Type: "slides", Title: "X", Order: 1, Payload: json.RawMessage(`{}`)},
				},
			},
			{
				Key: "ch-early", Title: "Earlier", Order: 2,
				Activities: []course.Activity{
					{Key: "act-c", Type: course.ActivityTypeRichText, Title: "C", Order: 8, Payload: richText},
				},
			},
		},
	}

	// 两处非法 type 都会让严格校验失败；Normalize 再把它们滤掉。
	validationError, ok := errors.AsType[*course.ValidationError](loaded.Validate())
	if !ok {
		t.Fatal("expected *course.ValidationError before normalize")
	}
	sawVideo, sawSlides := false, false
	for _, problem := range validationError.Problems {
		if problem.Field == "chapters[0].activities[1].type" {
			sawVideo = true
		}
		if problem.Field == "chapters[1].activities[0].type" {
			sawSlides = true
		}
	}
	if !sawVideo || !sawSlides {
		t.Fatalf("problems = %+v, want both illegal activity types", validationError.Problems)
	}

	report := loaded.Normalize()
	if len(report.DroppedActivities) != 2 ||
		report.DroppedActivities[0].Key != "act-bad" || report.DroppedActivities[0].Reason != course.DropReasonIllegalActivityType ||
		report.DroppedActivities[1].Key != "act-x" || report.DroppedActivities[1].Path != "chapters[1].activities[0]" {
		t.Fatalf("dropped activities = %+v", report.DroppedActivities)
	}
	if len(report.DroppedChapters) != 1 || report.DroppedChapters[0].Key != "ch-empty" || report.DroppedChapters[0].Reason != course.DropReasonEmptyChapter {
		t.Fatalf("dropped chapters = %+v", report.DroppedChapters)
	}
	if len(loaded.Chapters) != 2 {
		t.Fatalf("chapters left = %+v", loaded.Chapters)
	}
	if loaded.Chapters[0].Key != "ch-early" || loaded.Chapters[0].Order != 1 {
		t.Fatalf("first chapter = %+v", loaded.Chapters[0])
	}
	if loaded.Chapters[1].Key != "ch-late" || loaded.Chapters[1].Order != 2 {
		t.Fatalf("second chapter = %+v", loaded.Chapters[1])
	}
	if len(loaded.Chapters[1].Activities) != 2 ||
		loaded.Chapters[1].Activities[0].Key != "act-a" || loaded.Chapters[1].Activities[0].Order != 1 ||
		loaded.Chapters[1].Activities[1].Key != "act-b" || loaded.Chapters[1].Activities[1].Order != 2 {
		t.Fatalf("activities = %+v", loaded.Chapters[1].Activities)
	}
	if loaded.Chapters[0].Activities[0].Order != 1 {
		t.Fatalf("early activity order = %d", loaded.Chapters[0].Activities[0].Order)
	}
	if err := loaded.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeKeepsEqualOrderStable(t *testing.T) {
	payload := json.RawMessage(`{"markdown":"正文"}`)
	loaded := course.StandardCourse{
		SchemaVersion: course.SchemaVersion,
		Title:         "Stable",
		Audience:      course.AudienceGeneral,
		Language:      course.LanguageZhCN,
		Chapters: []course.Chapter{
			{Key: "ch-1", Title: "One", Order: 1, Activities: []course.Activity{
				{Key: "a", Type: course.ActivityTypeRichText, Title: "A", Order: 3, Payload: payload},
				{Key: "b", Type: course.ActivityTypeRichText, Title: "B", Order: 3, Payload: payload},
			}},
			{Key: "ch-2", Title: "Two", Order: 1, Activities: []course.Activity{
				{Key: "c", Type: course.ActivityTypeRichText, Title: "C", Order: 1, Payload: payload},
			}},
		},
	}
	report := loaded.Normalize()
	if len(report.DroppedActivities) != 0 || len(report.DroppedChapters) != 0 {
		t.Fatalf("unexpected drops: %+v", report)
	}
	if loaded.Chapters[0].Key != "ch-1" || loaded.Chapters[0].Order != 1 || loaded.Chapters[1].Key != "ch-2" || loaded.Chapters[1].Order != 2 {
		t.Fatalf("chapter order changed beyond renumber: %+v", loaded.Chapters)
	}
	if loaded.Chapters[0].Activities[0].Key != "a" || loaded.Chapters[0].Activities[0].Order != 1 ||
		loaded.Chapters[0].Activities[1].Key != "b" || loaded.Chapters[0].Activities[1].Order != 2 {
		t.Fatalf("activity order not stable: %+v", loaded.Chapters[0].Activities)
	}
}

func TestNormalizeIsIdempotentOnMiniCourse(t *testing.T) {
	loaded := loadMiniCourse(t)
	report := loaded.Normalize()
	if len(report.DroppedActivities) != 0 || len(report.DroppedChapters) != 0 {
		t.Fatalf("mini course should not drop anything: %+v", report)
	}
	if loaded.Chapters[0].Order != 1 || loaded.Chapters[0].Activities[1].Order != 2 || loaded.Chapters[1].Order != 2 {
		t.Fatalf("mini course order changed: %+v", loaded.Chapters)
	}
	again := loaded.Normalize()
	if len(again.DroppedActivities) != 0 || len(again.DroppedChapters) != 0 {
		t.Fatalf("second normalize dropped items: %+v", again)
	}
}

func TestNormalizeNilCourse(t *testing.T) {
	var loaded *course.StandardCourse
	report := loaded.Normalize()
	if len(report.DroppedActivities) != 0 || len(report.DroppedChapters) != 0 {
		t.Fatalf("nil normalize = %+v", report)
	}
}

func TestNormalizeDropsChapterThatBecomesEmptyThenValidateFails(t *testing.T) {
	loaded := course.StandardCourse{
		SchemaVersion: course.SchemaVersion,
		Title:         "Empty",
		Audience:      course.AudienceCS,
		Language:      course.LanguageZhCN,
		Chapters: []course.Chapter{{
			Key: "ch-1", Title: "Only bad", Order: 3,
			Activities: []course.Activity{{Key: "bad", Type: "lab", Title: "Bad", Order: 1}},
		}},
	}
	report := loaded.Normalize()
	if len(loaded.Chapters) != 0 || len(report.DroppedChapters) != 1 {
		t.Fatalf("course = %+v, report = %+v", loaded, report)
	}
	requireProblem(t, loaded.Validate(), "chapters", "at least one chapter")
}
