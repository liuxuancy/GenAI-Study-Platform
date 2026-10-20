package parse

import (
	"encoding/json"
	"strings"

	"caveclass/internal/ai/aierr"
	"caveclass/internal/ai/course"
)

// Report 是 ParseCourseJSON 的附带结果。
// 补过的字段放在 RepairedFields，由调用方写入 meta.validation。
type Report struct {
	RepairedFields []string
	Warnings       []string
	Normalize      course.NormalizeReport
}

// ParseCourseJSON 把模型原文解析成标准课程。
// 顺序：提取 JSON、反序列化、repair 时补默认、Normalize、Validate。
// Normalize 放在最终校验之前，非法 activity 会被滤掉；滤完仍不合法则返回 schema 错误。
// repair=false 时不补默认。已有的非法值不会被覆盖，例如 schemaVersion="2.0"。
func ParseCourseJSON(rawText string, repair bool) (course.StandardCourse, Report, error) {
	extracted, err := ExtractJSON(rawText)
	if err != nil {
		return course.StandardCourse{}, Report{}, err
	}
	var parsed course.StandardCourse
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		return course.StandardCourse{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
	}

	report := Report{}
	if repair {
		fields, warnings, err := repairDefaults(&parsed, extracted)
		if err != nil {
			return course.StandardCourse{}, Report{}, aierr.OutputParseFailed(err, "invalid JSON")
		}
		report.RepairedFields = fields
		report.Warnings = warnings
	}
	report.Normalize = parsed.Normalize()
	if err := parsed.Validate(); err != nil {
		return course.StandardCourse{}, Report{}, schemaFailed(err)
	}
	return parsed, report, nil
}

// 只补缺失或空白的默认值。estimatedMinutes 只有键不存在时才记为修复。
func repairDefaults(parsed *course.StandardCourse, rawJSON string) ([]string, []string, error) {
	var present map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &present); err != nil {
		return nil, nil, err
	}
	var fields []string
	var warnings []string
	if strings.TrimSpace(parsed.SchemaVersion) == "" {
		parsed.SchemaVersion = course.SchemaVersion
		fields = append(fields, "schemaVersion")
	}
	if strings.TrimSpace(parsed.Language) == "" {
		parsed.Language = course.LanguageZhCN
		fields = append(fields, "language")
	}
	if _, ok := present["estimatedMinutes"]; !ok {
		parsed.EstimatedMinutes = 0
		fields = append(fields, "estimatedMinutes")
		warnings = append(warnings, "estimatedMinutes was missing and defaulted to 0")
	}
	return fields, warnings, nil
}

func schemaFailed(err error) error {
	validationError, ok := err.(*course.ValidationError)
	if !ok {
		return aierr.OutputSchemaFailed(err)
	}
	details := make([]string, len(validationError.Problems))
	for i, problem := range validationError.Problems {
		details[i] = problem.Field + ": " + problem.Message
	}
	return aierr.OutputSchemaFailed(err, details...)
}
