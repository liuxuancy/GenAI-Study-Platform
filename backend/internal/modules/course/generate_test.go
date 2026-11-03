package course

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"caveclass/internal/ai"
	aicourse "caveclass/internal/ai/course"
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
	assertCount(t, database, "SELECT COUNT(*) FROM course WHERE id = ? AND is_publish = ?", 1, draft.ID, PublishOff)
	assertCount(t, database, "SELECT COUNT(*) FROM course_question WHERE course_id = ? AND is_publish = ?", 1, draft.ID, PublishOff)
	if _, err := service.GenerateOwnedCourse(context.Background(), &access.Principal{UserID: student.UserID, Role: access.RoleStudent}, draft.ID, GenerateCourseInput{}); err == nil {
		t.Fatal("student generated a teacher course")
	}
}

func TestGenerateFragmentsMergeContentAndRecordUsage(t *testing.T) {
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
	created, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Local generation"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GenerateOwnedCourse(context.Background(), student, created.ID, GenerateCourseInput{}); err != nil {
		t.Fatal(err)
	}

	var contentJSON string
	if err := database.Raw("SELECT content_json FROM course WHERE id = ?", created.ID).Scan(&contentJSON).Error; err != nil {
		t.Fatal(err)
	}
	var content aicourse.StandardCourse
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		t.Fatal(err)
	}
	chapterKey := content.Chapters[0].Key
	if _, err := service.GenerateOwnedChapter(context.Background(), student, created.ID, GenerateChapterInput{
		TargetChapterKey: chapterKey,
	}); err != nil {
		t.Fatalf("chapter: %v", err)
	}
	activityKey := content.Chapters[0].Activities[0].Key
	if _, err := service.GenerateOwnedActivity(context.Background(), student, created.ID, GenerateActivityInput{
		TargetChapterKey: chapterKey, TargetActivityKey: activityKey,
	}); err != nil {
		t.Fatalf("activity: %v", err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 1, student.UserID, usage.SceneGenerateChapter)
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 2, student.UserID, usage.SceneGenerateActivity)
}

func TestChapterQAQualityReportAndMermaid(t *testing.T) {
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
	studentCourse, err := service.CreateStudentCourse(context.Background(), student, CreateCourseInput{Title: "Chapter QA"})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := service.GenerateOwnedCourse(context.Background(), student, studentCourse.ID, GenerateCourseInput{})
	if err != nil {
		t.Fatal(err)
	}
	chapterKey := generated.Course.(*CourseDetail).Tree.Children[0].Key
	answer, err := service.AskChapterQuestion(context.Background(), student, studentCourse.ID, chapterKey, ChapterQAInput{Question: "本章讲了什么？"})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Answer == "" {
		t.Fatal("empty chapter answer")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 1, student.UserID, usage.SceneChapterQA)

	teacher := teacherPrincipal(t, database, "teacher1")
	teacherCourse, err := service.CreateTeacherCourse(context.Background(), teacher, CreateCourseInput{Title: "Quality course"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GenerateOwnedCourse(context.Background(), teacher, teacherCourse.ID, GenerateCourseInput{}); err != nil {
		t.Fatal(err)
	}
	report, err := service.GenerateCourseQualityReport(context.Background(), teacher, teacherCourse.ID, QualityReportInput{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Report.OverallScore <= 0 || report.Meta == nil {
		t.Fatalf("report = %+v", report)
	}
	stored, err := service.GetCourseQualityReport(context.Background(), teacher, teacherCourse.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Report.OverallScore != report.Report.OverallScore || stored.Meta != nil {
		t.Fatalf("stored report = %+v", stored)
	}
	graph, err := service.GetCourseMermaid(context.Background(), teacher, teacherCourse.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(graph.Source, "flowchart TD") || !strings.Contains(graph.Source, "-->") {
		t.Fatalf("mermaid = %q", graph.Source)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND scene = ?", 1, teacher.UserID, usage.SceneEvaluateCourse)
}
