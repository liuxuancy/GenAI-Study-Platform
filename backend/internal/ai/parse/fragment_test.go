package parse_test

import (
	"errors"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/llm"
	"caveclass/internal/ai/parse"
)

func TestParseChapterAndActivityFragments(t *testing.T) {
	chapter, chapterReport, err := parse.ParseChapterJSON("```json\n"+llm.MockChapterText()+"\n```", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := chapter.Validate(); err != nil {
		t.Fatal(err)
	}
	if chapter.Key != "ch-1" || len(chapter.Activities) != 2 || len(chapterReport.RepairedFields) != 0 {
		t.Fatalf("chapter = %+v report = %+v", chapter, chapterReport)
	}

	activity, activityReport, err := parse.ParseActivityJSON(llm.MockActivityText(), true)
	if err != nil {
		t.Fatal(err)
	}
	if activity.Type != course.ActivityTypeRichText || activity.Order != 1 || len(activityReport.RepairedFields) != 0 {
		t.Fatalf("activity = %+v report = %+v", activity, activityReport)
	}

	_, _, err = parse.ParseActivityJSON(`{"title":"missing key"}`, true)
	if !errors.Is(err, aierr.ErrOutputSchemaFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestParseChapterRecordsMissingOrder(t *testing.T) {
	raw := `{
		"key": "ch-1",
		"title": "What is Version Control",
		"activities": [{
			"key": "act-1-1",
			"type": "rich_text",
			"title": "Read",
			"payload": {"markdown": "Git"}
		}]
	}`
	chapter, report, err := parse.ParseChapterJSON(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if chapter.Order != 1 || chapter.Activities[0].Order != 1 {
		t.Fatalf("orders = %d %d", chapter.Order, chapter.Activities[0].Order)
	}
	if !hasField(report.RepairedFields, "order") || !hasField(report.RepairedFields, "activities[0].order") {
		t.Fatalf("repaired = %#v", report.RepairedFields)
	}
}

func hasField(fields []string, field string) bool {
	for _, item := range fields {
		if item == field {
			return true
		}
	}
	return false
}
