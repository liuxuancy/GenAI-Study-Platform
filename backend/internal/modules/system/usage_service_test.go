package system

import (
	"context"
	"testing"
	"time"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/usage"
)

func TestUsageReportTotalsAndPeople(t *testing.T) {
	database := testutil.NewDB(t)
	if err := database.AutoMigrate(&usage.Usage{}); err != nil {
		t.Fatal(err)
	}
	studentID := testutil.GetUserID(t, database, "student1")
	adminID := testutil.GetUserID(t, database, "admin")
	earlier := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	later := earlier.Add(time.Hour)
	testutil.MustExec(t, database,
		`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, ?, 'mock', 4, 6, 10, ?, ?)`,
		studentID, usage.SceneGenerateCourse, usage.StatusSuccess, earlier,
	)
	testutil.MustExec(t, database,
		`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, ?, '', 0, 0, 0, ?, ?)`,
		studentID, usage.SceneGenerateActivity, usage.StatusFailed, later,
	)
	testutil.MustExec(t, database,
		`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
		 VALUES (?, ?, 'mock', 1, 2, 3, ?, ?)`,
		adminID, usage.SceneGenerateCourse, usage.StatusSuccess, later,
	)

	report, err := NewUsageService(database).GetUsageReport(context.Background(), UsageListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if report.AccountCount != 4 || report.CallCount != 3 || report.PromptTokens != 5 || report.CompletionTokens != 8 || report.TotalTokens != 13 {
		t.Fatalf("totals = %+v", report)
	}
	if report.Total != 2 || len(report.Users) != 2 {
		t.Fatalf("users = %+v", report.Users)
	}
	student := report.Users[0]
	if student.Username != "student1" || student.Role != access.RoleStudent || student.RoleName != "学生" {
		t.Fatalf("first user = %+v", student)
	}
	if student.CallCount != 2 || student.PromptTokens != 4 || student.CompletionTokens != 6 || student.TotalTokens != 10 {
		t.Fatalf("student totals = %+v", student)
	}
	if len(student.Calls) != 2 || student.Calls[0].Scene != usage.SceneGenerateActivity || student.Calls[0].CallStatus != usage.StatusFailed {
		t.Fatalf("calls = %+v", student.Calls)
	}
	if student.Calls[0].TotalTokens != 0 || student.Calls[1].Scene != usage.SceneGenerateCourse || !student.Calls[1].CreatedAt.Equal(earlier) {
		t.Fatalf("call order = %+v", student.Calls)
	}
	if report.Users[1].Username != "admin" || report.Users[1].Role != access.RoleAdmin {
		t.Fatalf("admin usage missing: %+v", report.Users[1])
	}
}

func TestUsageReportFiltersAndPagination(t *testing.T) {
	database := testutil.NewDB(t)
	if err := database.AutoMigrate(&usage.Usage{}); err != nil {
		t.Fatal(err)
	}
	studentID := testutil.GetUserID(t, database, "student1")
	teacherID := testutil.GetUserID(t, database, "teacher1")
	dayOne := time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC)
	dayTwo := dayOne.Add(2 * time.Hour)
	for _, row := range []struct {
		userID int64
		scene  string
		model  string
		status string
		at     time.Time
	}{
		{studentID, usage.SceneChapterQA, "model-a", usage.StatusSuccess, dayOne},
		{studentID, usage.SceneChapterQA, "model-b", usage.StatusFailed, dayTwo},
		{teacherID, usage.SceneChapterQA, "model-a", usage.StatusSuccess, dayOne},
	} {
		testutil.MustExec(t, database,
			`INSERT INTO ai_usage (user_id, scene, model, prompt_tokens, completion_tokens, total_tokens, call_status, created_at)
			 VALUES (?, ?, ?, 2, 3, 5, ?, ?)`,
			row.userID, row.scene, row.model, row.status, row.at,
		)
	}

	report, err := NewUsageService(database).GetUsageReport(context.Background(), UsageListQuery{
		From: "2026-10-01", To: "2026-10-01", User: "student",
		Role: access.RoleStudent, Scene: usage.SceneChapterQA, Model: "model-a",
		CallStatus: usage.StatusSuccess, Page: 1, PageSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.CallCount != 1 || report.TotalTokens != 5 || report.Total != 1 || len(report.Users) != 1 {
		t.Fatalf("filtered report = %+v", report)
	}
	if len(report.Users[0].Calls) != 1 || report.Users[0].Calls[0].Model != "model-a" {
		t.Fatalf("filtered calls = %+v", report.Users[0].Calls)
	}
}
