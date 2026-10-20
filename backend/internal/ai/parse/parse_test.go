package parse_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/parse"
)

func miniCourseJSON(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../course/testdata/mini_course.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func miniCourseWithout(t *testing.T, keys ...string) string {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(miniCourseJSON(t)), &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		delete(fields, key)
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestParsePureJSONAndCodeFence(t *testing.T) {
	raw := miniCourseJSON(t)
	fenced := "下面是课程：\n注意 {这不是 JSON}\n```json\n" + raw + "\n```\n请查收"
	plainFence := "```\n" + raw + "\n```"

	for _, test := range []struct {
		name string
		text string
	}{
		{name: "pure json", text: raw},
		{name: "json fence", text: fenced},
		{name: "plain fence", text: plainFence},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, report, err := parse.ParseCourseJSON(test.text, false)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Title != "Introduction to Git for Beginners" || len(parsed.Chapters) != 2 {
				t.Fatalf("parsed course = %+v", parsed)
			}
			if len(report.RepairedFields) != 0 || len(report.Warnings) != 0 {
				t.Fatalf("complete course should not repair: %+v", report)
			}
			if parsed.Chapters[0].Order != 1 || parsed.Chapters[1].Activities[0].Order != 1 {
				t.Fatalf("order not normalized: %+v", parsed.Chapters)
			}
		})
	}
}

func TestExtractJSONKeepsBracesInsideStrings(t *testing.T) {
	raw := "```JSON\n" + `{"title":"use {git}","ok":true}` + "\n```"
	got, err := parse.ExtractJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(got), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["title"] != "use {git}" {
		t.Fatalf("title = %#v", fields["title"])
	}
}

func TestParseRepairFillsDefaults(t *testing.T) {
	raw := miniCourseWithout(t, "schemaVersion", "language", "estimatedMinutes")
	parsed, report, err := parse.ParseCourseJSON(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.SchemaVersion != course.SchemaVersion || parsed.Language != course.LanguageZhCN || parsed.EstimatedMinutes != 0 {
		t.Fatalf("defaults = version %q language %q minutes %d", parsed.SchemaVersion, parsed.Language, parsed.EstimatedMinutes)
	}
	if strings.Join(report.RepairedFields, ",") != "schemaVersion,language,estimatedMinutes" {
		t.Fatalf("repaired = %#v", report.RepairedFields)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "estimatedMinutes") {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
}

func TestParseWithoutRepairDoesNotFillDefaults(t *testing.T) {
	raw := miniCourseWithout(t, "schemaVersion", "language")
	_, _, err := parse.ParseCourseJSON(raw, false)
	if !errors.Is(err, aierr.ErrOutputSchemaFailed) {
		t.Fatalf("err = %v, want schema failure", err)
	}
	if errors.Is(err, aierr.ErrOutputParseFailed) {
		t.Fatal("schema failure must not match parse failure")
	}
	aiError, ok := errors.AsType[*aierr.Error](err)
	if !ok || !strings.Contains(strings.Join(aiError.Details, ";"), "schemaVersion") {
		t.Fatalf("details = %+v", aiError)
	}
}

func TestParseRepairDoesNotOverwriteInvalidValues(t *testing.T) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(miniCourseJSON(t)), &fields); err != nil {
		t.Fatal(err)
	}
	fields["schemaVersion"] = "2.0"
	fields["estimatedMinutes"] = -3
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	_, _, parseErr := parse.ParseCourseJSON(string(encoded), true)
	if !errors.Is(parseErr, aierr.ErrOutputSchemaFailed) {
		t.Fatalf("err = %v, want schema failure", parseErr)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		detail string
	}{
		{name: "empty", text: "  ", detail: "empty input"},
		{name: "prose", text: "这里没有课程", detail: "no JSON object"},
		{name: "unbalanced", text: "```json\n{\"title\":\"x\"\n```", detail: "unbalanced JSON object"},
		{name: "broken json", text: `{"title":}`, detail: "invalid JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parse.ParseCourseJSON(test.text, true)
			if !errors.Is(err, aierr.ErrOutputParseFailed) {
				t.Fatalf("err = %v, want parse failure", err)
			}
			aiError, ok := errors.AsType[*aierr.Error](err)
			if !ok || aiError.Code != aierr.CodeOutputParseFailed {
				t.Fatalf("error = %+v", aiError)
			}
			if !strings.Contains(aiError.Error(), test.detail) {
				t.Fatalf("error = %q, want detail %q", aiError.Error(), test.detail)
			}
		})
	}
}

func TestParseNormalizeDropsIllegalActivity(t *testing.T) {
	var parsed course.StandardCourse
	if err := json.Unmarshal([]byte(miniCourseJSON(t)), &parsed); err != nil {
		t.Fatal(err)
	}
	parsed.Chapters[0].Order = 5
	parsed.Chapters[1].Order = 2
	parsed.Chapters[0].Activities = append(parsed.Chapters[0].Activities, course.Activity{
		Key: "act-bad", Type: "video", Title: "Nope", Order: 1, Payload: json.RawMessage(`{}`),
	})
	encoded, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}
	got, report, err := parse.ParseCourseJSON("模型输出：\n```json\n"+string(encoded)+"\n```", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Normalize.DroppedActivities) != 1 || report.Normalize.DroppedActivities[0].Key != "act-bad" {
		t.Fatalf("dropped = %+v", report.Normalize.DroppedActivities)
	}
	if got.Chapters[0].Key != "ch-2" || got.Chapters[0].Order != 1 || got.Chapters[1].Key != "ch-1" || got.Chapters[1].Order != 2 {
		t.Fatalf("chapters = %+v", got.Chapters)
	}
	for _, activity := range got.Chapters[1].Activities {
		if activity.Type == "video" {
			t.Fatal("illegal activity was kept")
		}
	}
}
