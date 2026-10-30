package course

import (
	"context"
	"testing"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/usage"
)

func TestGenerateCourseStoresJSONQuestionsAndUsage(t *testing.T) {
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
	student := studentPrincipal(t, database, "student1")
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Generated notes"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.GenerateOwnedCourse(context.Background(), student, created.ID, GenerateCourseInput{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.Meta.TemplateID == "" || result.Course == nil {
		t.Fatalf("result = %+v", result)
	}
	detail := result.Course.(*CourseDetail)
	if detail.Tree == nil || len(detail.Tree.Children) == 0 {
		t.Fatalf("tree = %+v", detail.Tree)
	}
	var quizChapters int
	for _, chapter := range detail.Tree.Children {
		if chapter.QuestionID != 0 {
			quizChapters++
		}
	}
	if quizChapters == 0 {
		t.Fatal("no chapter quiz was stored")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ? AND call_status = ?", 1, student.UserID, usage.SceneGenerateCourse, usage.StatusSuccess)
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 1, student.UserID, usage.SceneGenerateActivity)

	again, err := service.GenerateOwnedCourse(context.Background(), student, created.ID, GenerateCourseInput{Topic: "Git"})
	if err != nil {
		t.Fatalf("generate again: %v", err)
	}
	if again.Course.(*CourseDetail).Tree == nil {
		t.Fatal("second generate cleared the tree")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM course_question WHERE course_id = ? AND chapter_key IS NOT NULL AND chapter_key <> ''", int64(quizChapters), created.ID)

	teacher := teacherPrincipal(t, database, "teacher1")
	draft, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Generated lecture"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GenerateOwnedCourse(context.Background(), teacher, draft.ID, GenerateCourseInput{}); err != nil {
		t.Fatalf("teacher generate: %v", err)
	}
	if _, err := service.GenerateOwnedCourse(context.Background(), &access.Principal{UserID: student.UserID, Role: access.RoleStudent}, draft.ID, GenerateCourseInput{}); err == nil {
		t.Fatal("student generated a teacher course")
	}
}
