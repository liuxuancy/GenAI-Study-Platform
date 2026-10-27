package quality

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"caveclass/internal/ai/course"
)

// RequiredDimensions 是报告必须各出现一次的维度，顺序稳定。
func RequiredDimensions() []string {
	return []string{DimensionCoverage, DimensionClarity, DimensionPractice, DimensionAudienceFit}
}

// Validate 检查质量报告。不修改报告。失败返回 *course.ValidationError，方便和课程输出共用 schema 错误。
func (report *Report) Validate() error {
	if report == nil {
		return &course.ValidationError{Problems: []course.Problem{{Field: "report", Message: "is nil"}}}
	}
	var problems []course.Problem
	add := func(field, message string) {
		problems = append(problems, course.Problem{Field: field, Message: message})
	}
	if report.OverallScore < 0 || report.OverallScore > 100 {
		add("overallScore", "must be from 0 to 100")
	}
	if strings.TrimSpace(report.SummaryMarkdown) == "" {
		add("summaryMarkdown", "required")
	} else if utf8.RuneCountInString(report.SummaryMarkdown) > MaxSummaryRunes {
		add("summaryMarkdown", fmt.Sprintf("must be at most %d characters", MaxSummaryRunes))
	}
	seen := map[string]struct{}{}
	if len(report.Dimensions) == 0 {
		add("dimensions", "coverage, clarity, practice, and audience_fit are required")
	}
	for index, dimension := range report.Dimensions {
		field := fmt.Sprintf("dimensions[%d]", index)
		if !knownDimension(dimension.Name) {
			add(field+".name", "must be one of coverage, clarity, practice, audience_fit")
		} else if _, ok := seen[dimension.Name]; ok {
			add(field+".name", "duplicate")
		} else {
			seen[dimension.Name] = struct{}{}
		}
		if dimension.Score < 0 || dimension.Score > 100 {
			add(field+".score", "must be from 0 to 100")
		}
		if strings.TrimSpace(dimension.Comment) == "" {
			add(field+".comment", "required")
		} else if utf8.RuneCountInString(dimension.Comment) > MaxCommentRunes {
			add(field+".comment", fmt.Sprintf("must be at most %d characters", MaxCommentRunes))
		}
	}
	if len(report.Dimensions) > 0 {
		for _, name := range RequiredDimensions() {
			if _, ok := seen[name]; !ok {
				add("dimensions", "missing "+name)
			}
		}
	}
	if report.Suggestions == nil {
		add("suggestions", "required")
	} else if len(report.Suggestions) > MaxSuggestions {
		add("suggestions", fmt.Sprintf("at most %d items", MaxSuggestions))
	}
	for index, suggestion := range report.Suggestions {
		if strings.TrimSpace(suggestion) == "" {
			add(fmt.Sprintf("suggestions[%d]", index), "required")
		} else if utf8.RuneCountInString(suggestion) > MaxSuggestionRunes {
			add(fmt.Sprintf("suggestions[%d]", index), fmt.Sprintf("must be at most %d characters", MaxSuggestionRunes))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return &course.ValidationError{Problems: problems}
}

func knownDimension(name string) bool {
	switch name {
	case DimensionCoverage, DimensionClarity, DimensionPractice, DimensionAudienceFit:
		return true
	default:
		return false
	}
}
