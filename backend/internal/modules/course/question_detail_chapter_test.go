package course

import (
	"context"
	"testing"

	"caveclass/internal/framework/testutil"
)

func TestQuestionDetailChapterNameMatchesList(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewDB(t, Module{})
	service := NewService(database)
	student := studentPrincipal(t, database, "student1")
	course := Course{Title: "Chapter title course", OwnerID: student.UserID, ContentJSON: sampleContent}
	if err := database.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	question := CourseQuestion{Title: "Quiz · ch-1 · 1 · 2026-11-10 10:30:25", CourseID: course.ID, CreatorID: student.UserID, QuestionCount: 1, ChapterKey: "ch-1"}
	if err := database.Create(&question).Error; err != nil {
		t.Fatal(err)
	}
	content := CourseQuestionContent{QuestionID: question.ID, Content: "Question", Options: "A|B", CorrectAnswer: "A", Explanation: "Answer"}
	if err := database.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	list, err := service.GetQuestionList(ctx, student)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.GetQuestion(ctx, student, question.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ChapterName != "Opening" || detail.ChapterName != list[0].ChapterName {
		t.Fatalf("list = %+v, detail = %+v", list, detail)
	}
	answered, err := service.AnswerQuestion(ctx, student, question.ID, content.ID, "A", false)
	if err != nil || answered.ChapterName != "Opening" {
		t.Fatalf("answered = %+v, error = %v", answered, err)
	}
}
