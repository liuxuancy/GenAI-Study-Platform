package quality_test

import (
	"strings"
	"testing"

	"caveclass/internal/ai/course"
	"caveclass/internal/ai/quality"
)

func validReport() quality.Report {
	return quality.Report{
		OverallScore:    86,
		SummaryMarkdown: "结构清楚。",
		Dimensions: []quality.Dimension{
			{Name: quality.DimensionCoverage, Score: 84, Comment: "覆盖基本概念。"},
			{Name: quality.DimensionClarity, Score: 90, Comment: "说明短。"},
			{Name: quality.DimensionPractice, Score: 80, Comment: "有练习。"},
			{Name: quality.DimensionAudienceFit, Score: 88, Comment: "适合入门。"},
		},
		Suggestions: []string{"补一个例子。"},
	}
}

func TestValidateAcceptsReport(t *testing.T) {
	report := validReport()
	report.Suggestions = []string{}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsScoreAndMissingDimension(t *testing.T) {
	report := validReport()
	report.OverallScore = 101
	report.Dimensions = report.Dimensions[:3]
	err := report.Validate()
	validationError, ok := err.(*course.ValidationError)
	if !ok {
		t.Fatalf("got %v", err)
	}
	var sawScore, sawMissing bool
	for _, problem := range validationError.Problems {
		if problem.Field == "overallScore" && strings.Contains(problem.Message, "0 to 100") {
			sawScore = true
		}
		if problem.Field == "dimensions" && strings.Contains(problem.Message, "audience_fit") {
			sawMissing = true
		}
	}
	if !sawScore || !sawMissing {
		t.Fatalf("problems = %+v", validationError.Problems)
	}
}
