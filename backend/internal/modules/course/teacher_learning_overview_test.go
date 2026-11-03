package course

import (
	"context"
	"testing"

	"caveclass/internal/testutil"
)

func TestTeacherCourseLearningOverviewSeed(t *testing.T) {
	database := testutil.NewDB(t, Module{})
	seedCoursesAndLegacyQuestions(t, database)
	service := NewService(database)
	teacher := teacherPrincipal(t, database, "teacher1")
	gitID := courseIDByTitle(t, database, "Introduction to Git")

	overview, err := service.GetTeacherCourseLearningOverview(context.Background(), teacher, gitID)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if len(overview.Students) != 1 || overview.Students[0].Username != "student1" || overview.Students[0].Percent != 0 {
		t.Fatalf("students = %+v", overview.Students)
	}
	if len(overview.Quizzes) != 1 || overview.Quizzes[0].Title != "What is a commit?" {
		t.Fatalf("quizzes = %+v", overview.Quizzes)
	}
	if len(overview.Students[0].Quizzes) != 1 {
		t.Fatalf("student quizzes = %+v", overview.Students[0].Quizzes)
	}
	stat := overview.Students[0].Quizzes[0]
	if stat.Answered != 2 || stat.Total != 2 || stat.Submitted {
		t.Fatalf("quiz stat = %+v", stat)
	}
}
