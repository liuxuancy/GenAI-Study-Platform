package parse

import (
	"bytes"
	"encoding/json"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
	"caveclass/internal/ai/quality"
)

// ParseQualityJSON 解析质量报告。repair=true 时把缺失或 null 的 suggestions 补成空数组。
func ParseQualityJSON(rawText string, repair bool) (quality.Report, Report, error) {
	extracted, err := ExtractJSON(rawText)
	if err != nil {
		return quality.Report{}, Report{}, err
	}
	var parsed quality.Report
	if err := unmarshalQualityJSON(extracted, &parsed); err != nil {
		return quality.Report{}, Report{}, err
	}
	if !json.Valid([]byte(extracted)) {
		extracted = RelaxJSON(extracted)
	}
	var present map[string]json.RawMessage
	if err := json.Unmarshal([]byte(extracted), &present); err != nil {
		return quality.Report{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
	}
	report := Report{}
	suggestions, hasSuggestions := present["suggestions"]
	if repair && (!hasSuggestions || bytes.Equal(bytes.TrimSpace(suggestions), []byte("null"))) {
		parsed.Suggestions = []string{}
		report.RepairedFields = []string{"suggestions"}
	}
	problems := qualityProblems(parsed, present)
	if len(problems) > 0 {
		return quality.Report{}, Report{}, schemaFailed(&course.ValidationError{Problems: problems})
	}
	return parsed, report, nil
}

func unmarshalQualityJSON(extracted string, parsed *quality.Report) error {
	err := json.Unmarshal([]byte(extracted), parsed)
	if err == nil {
		return nil
	}
	relaxed := RelaxJSON(extracted)
	if relaxed != extracted {
		if err2 := json.Unmarshal([]byte(relaxed), parsed); err2 == nil {
			return nil
		}
	}
	return aierr.OutputParseFailed(err, "invalid JSON")
}

func qualityProblems(parsed quality.Report, present map[string]json.RawMessage) []course.Problem {
	var problems []course.Problem
	if _, ok := present["overallScore"]; !ok {
		problems = append(problems, course.Problem{Field: "overallScore", Message: "required"})
	}
	if err := parsed.Validate(); err != nil {
		validationError, ok := err.(*course.ValidationError)
		if !ok {
			return []course.Problem{{Field: "report", Message: err.Error()}}
		}
		for _, problem := range validationError.Problems {
			if problem.Field == "overallScore" && problem.Message == "must be from 0 to 100" {
				if _, ok := present["overallScore"]; !ok {
					continue
				}
			}
			problems = append(problems, problem)
		}
	}
	return problems
}
