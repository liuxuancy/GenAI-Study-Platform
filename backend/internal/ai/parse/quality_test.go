package parse_test

import (
	"encoding/json"
	"errors"
	"testing"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/llm"
	"caveclass/internal/ai/parse"
)

func TestParseQualityJSON(t *testing.T) {
	report, parsedReport, err := parse.ParseQualityJSON("```json\n"+llm.MockQualityText()+"\n```", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	if report.OverallScore != 86 || len(parsedReport.RepairedFields) != 0 {
		t.Fatalf("report = %+v repaired = %#v", report, parsedReport.RepairedFields)
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(llm.MockQualityText()), &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "suggestions")
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	repaired, repairReport, err := parse.ParseQualityJSON(string(raw), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(repaired.Suggestions) != 0 || len(repairReport.RepairedFields) != 1 || repairReport.RepairedFields[0] != "suggestions" {
		t.Fatalf("repaired = %+v fields = %#v", repaired, repairReport.RepairedFields)
	}

	_, _, err = parse.ParseQualityJSON(`{"overallScore":120,"summaryMarkdown":"x","dimensions":[],"suggestions":[]}`, true)
	if !errors.Is(err, aierr.ErrOutputSchemaFailed) {
		t.Fatalf("err = %v", err)
	}
}
