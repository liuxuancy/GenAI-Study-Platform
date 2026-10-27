package parse

import (
	"encoding/json"
	"fmt"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
)

// ParseChapterJSON 解析单个章节片段。repair=true 时记录缺失的 order，再交给 Normalize 重排。
func ParseChapterJSON(rawText string, repair bool) (course.Chapter, Report, error) {
	extracted, err := ExtractJSON(rawText)
	if err != nil {
		return course.Chapter{}, Report{}, err
	}
	var parsed course.Chapter
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		return course.Chapter{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
	}
	report := Report{}
	if repair {
		fields, err := missingOrders(extracted, true)
		if err != nil {
			return course.Chapter{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
		}
		report.RepairedFields = fields
	}
	report.Normalize = parsed.Normalize()
	if err := parsed.Validate(); err != nil {
		return course.Chapter{}, Report{}, schemaFailed(err)
	}
	return parsed, report, nil
}

// ParseActivityJSON 解析单个 activity 片段。单个 activity 的 order 规范为 1。
func ParseActivityJSON(rawText string, repair bool) (course.Activity, Report, error) {
	extracted, err := ExtractJSON(rawText)
	if err != nil {
		return course.Activity{}, Report{}, err
	}
	var parsed course.Activity
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		return course.Activity{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
	}
	report := Report{}
	if repair {
		fields, err := missingOrders(extracted, false)
		if err != nil {
			return course.Activity{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
		}
		report.RepairedFields = fields
	}
	parsed.Order = 1
	if err := parsed.Validate(); err != nil {
		return course.Activity{}, Report{}, schemaFailed(err)
	}
	return parsed, report, nil
}

func missingOrders(rawJSON string, includeActivities bool) ([]string, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &object); err != nil {
		return nil, err
	}
	var fields []string
	if _, ok := object["order"]; !ok {
		fields = append(fields, "order")
	}
	if !includeActivities {
		return fields, nil
	}
	rawActivities, ok := object["activities"]
	if !ok {
		return fields, nil
	}
	var activities []map[string]json.RawMessage
	if err := json.Unmarshal(rawActivities, &activities); err != nil {
		return nil, err
	}
	for index, activity := range activities {
		if _, ok := activity["order"]; !ok {
			fields = append(fields, fmt.Sprintf("activities[%d].order", index))
		}
	}
	return fields, nil
}
