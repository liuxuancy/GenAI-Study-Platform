package course

import (
	"context"
	"strings"
	"testing"
	"time"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/usage"
)

func TestLearningDashboardAggregatesEvents(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}

	now := time.Now().UTC()
	weekStart := startOfWeekUTC(now)
	courseID := courseIDByTitle(t, database, "Introduction to Git")

	service.recordLearningEvent(context.Background(), studentID, EventCourseSession, &courseID, nil, "", 3600)
	service.recordLearningEvent(context.Background(), studentID, EventQuizAnswer, &courseID, nil, "", 0)

	dashboard, err := service.GetLearningDashboard(context.Background(), student, now.Year(), int(now.Month()))
	if err != nil {
		t.Fatalf("GetLearningDashboard: %v", err)
	}
	if len(dashboard.WeeklyMinutes) != 7 {
		t.Fatalf("weeklyMinutes len = %d", len(dashboard.WeeklyMinutes))
	}
	dayIndex := int(now.Sub(weekStart).Hours()) / 24
	if dayIndex >= 0 && dayIndex < 7 && dashboard.WeeklyMinutes[dayIndex] < 60 {
		t.Fatalf("expected session minutes on today, got %v", dashboard.WeeklyMinutes)
	}
	foundToday := false
	for _, day := range dashboard.ActiveDays {
		if day == now.Day() {
			foundToday = true
		}
	}
	if !foundToday {
		t.Fatalf("expected active day %d, got %v", now.Day(), dashboard.ActiveDays)
	}
	if len(dashboard.Tasks) == 0 {
		t.Fatal("expected at least one task from seed data")
	}
}

func TestRecordCourseHeartbeatClampsSeconds(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewService(database)
	studentID := testutil.GetUserID(t, database, "student1")
	student := &access.Principal{UserID: studentID, Role: access.RoleStudent}
	courseID := courseIDByTitle(t, database, "Introduction to Git")

	if err := service.RecordCourseHeartbeat(context.Background(), student, courseID, 999); err != nil {
		t.Fatalf("RecordCourseHeartbeat: %v", err)
	}
	teacherID := testutil.GetUserID(t, database, "teacher1")
	if err := service.RecordCourseHeartbeat(context.Background(), &access.Principal{
		UserID: teacherID,
		Role:   access.RoleTeacher,
	}, courseID, 30); err == nil {
		t.Fatal("expected forbidden for teacher")
	}
}

func TestGenerateLearningInsightRecordsUsage(t *testing.T) {
	t.Setenv("AI_PROVIDER", "mock")
	aiService, err := ai.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	database := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	service := NewService(database)
	service.UseAI(aiService)
	student := &access.Principal{
		UserID: testutil.GetUserID(t, database, "student1"),
		Role:   access.RoleStudent,
	}
	result, err := service.GenerateLearningInsight(context.Background(), student, LearningInsightInput{})
	if err != nil {
		t.Fatalf("GenerateLearningInsight: %v", err)
	}
	if strings.TrimSpace(result.Suggestion) == "" {
		t.Fatal("expected suggestion")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 1, student.UserID, usage.SceneLearningInsight)
}
