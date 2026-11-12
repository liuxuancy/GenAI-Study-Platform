package course

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/ai"
	"caveclass/internal/ai/llm"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
)

type quizTestClient struct {
	response string
	err      error
	prompt   string
	onCall   func()
}

func (client *quizTestClient) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	client.prompt = req.Prompt
	if client.onCall != nil {
		client.onCall()
	}
	return llm.Response{Text: client.response, Model: "test"}, client.err
}

const quizTestContent = `{"schemaVersion":"1.0","title":"Course","language":"en-US","chapters":[{"key":"one","title":"First","activities":[{"key":"a","type":"rich_text","title":"A","payload":{"markdown":"First chapter knowledge"}}]},{"key":"two","title":"Second","activities":[{"key":"b","type":"rich_text","title":"B","payload":{"markdown":"Second chapter knowledge"}}]}]}`
const quizTestResponse = `{"questions":[{"question":"Q1","options":["A","B"],"correctIndex":0,"explanation":"Explanation 1"},{"question":"Q2","options":["A","B"],"correctIndex":1,"explanation":"Explanation 2"}]}`

func TestQuizChapterInfo(t *testing.T) {
	tests := []struct {
		name, chapterKey, personalKey, personalTitle, wantKey, wantTitle string
		wantMissing                                                      bool
	}{
		{"unlinked", "", "", "", "", "", false},
		{"built-in chapter", "one", "", "", "one", "First", false},
		{"missing built-in chapter", "old", "", "", "old", "", false},
		{"personal chapter", "", "one", "Old title", "one", "First", false},
		{"missing personal chapter", "", "old", "Old title", "old", "Old title", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, title, missing := quizChapterInfo(quizTestContent, tt.chapterKey, tt.personalKey, tt.personalTitle)
			if key != tt.wantKey || title != tt.wantTitle || missing != tt.wantMissing {
				t.Fatalf("got (%q, %q, %t), want (%q, %q, %t)", key, title, missing, tt.wantKey, tt.wantTitle, tt.wantMissing)
			}
		})
	}
}

func TestStudentQuizGenerationAndChapterPreservation(t *testing.T) {
	db := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	student := studentPrincipal(t, db, "student1")
	id := courseIDByTitle(t, db, "Introduction to Git")
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", quizTestContent, id)
	client := &quizTestClient{response: quizTestResponse}
	aiService, err := ai.New(llm.Config{Provider: llm.ProviderMock, Timeout: time.Second}, client)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db)
	service.UseAI(aiService)
	input := GenerateStudentQuizInput{ChapterKey: "one", Count: 2, Difficulty: "easy", Provider: "mock"}
	first, err := service.GenerateStudentQuiz(context.Background(), student, id, input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(client.prompt, "First chapter knowledge") || strings.Contains(client.prompt, "Second chapter knowledge") {
		t.Fatal("chapter context is not isolated")
	}
	second, err := service.GenerateStudentQuiz(context.Background(), student, id, input)
	if err != nil || second.ID == first.ID {
		t.Fatalf("second quiz: %+v %v", second, err)
	}
	input.ChapterKey = ""
	third, err := service.GenerateStudentQuiz(context.Background(), student, id, input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(client.prompt, "First chapter knowledge") || !strings.Contains(client.prompt, "Second chapter knowledge") {
		t.Fatal("whole course context is incomplete")
	}
	assertCount(t, db, "SELECT COUNT(*) FROM student_quiz_chapter WHERE question_id IN (?, ?, ?)", 2, first.ID, second.ID, third.ID)
	assertCount(t, db, "SELECT COUNT(*) FROM course_question WHERE id IN (?, ?, ?) AND creation_method = 'ai' AND COALESCE(chapter_key, '') = ''", 3, first.ID, second.ID, third.ID)
	items, err := service.GetQuestionList(context.Background(), student)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range items {
		if item.ID == first.ID {
			found = item.ChapterKey == "one" && item.ChapterName == "First" && !item.ChapterMissing
		}
	}
	if !found {
		t.Fatal("chapter mapping missing")
	}
	testutil.MustExec(t, db, "INSERT INTO student_question_record (user_id, question_id, is_submit) VALUES (?, ?, ?)", student.UserID, first.ID, 1)
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", strings.Replace(quizTestContent, `"key":"one"`, `"key":"other"`, 1), id)
	detail, err := service.GetQuestion(context.Background(), student, first.ID)
	if err != nil || !detail.ChapterMissing || detail.ChapterName != "First" {
		t.Fatalf("missing chapter detail: %+v %v", detail, err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return deleteChapterQuizzes(tx, id) }); err != nil {
		t.Fatal(err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM course_question WHERE id IN (?, ?, ?)", 3, first.ID, second.ID, third.ID)
	assertCount(t, db, "SELECT COUNT(*) FROM student_question_record WHERE question_id = ? AND user_id = ?", 1, first.ID, student.UserID)
}

func TestStudentQuizFailuresNeverSavePartialQuiz(t *testing.T) {
	db := testutil.NewDB(t, Module{})
	if err := Seed(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	student := studentPrincipal(t, db, "student1")
	id := courseIDByTitle(t, db, "Introduction to Git")
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", quizTestContent, id)
	client := &quizTestClient{response: quizTestResponse}
	aiService, err := ai.New(llm.Config{Provider: llm.ProviderMock, Timeout: time.Second}, client)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db)
	service.UseAI(aiService)
	input := GenerateStudentQuizInput{ChapterKey: "one", Count: 2, Difficulty: "easy", Provider: "mock"}
	for _, tc := range []struct {
		name, response string
		modelErr       error
	}{
		{"model failure", "", errors.New("model unavailable")},
		{"invalid JSON", "bad", nil},
		{"wrong count", `{"questions":[]}`, nil},
		{"missing answer", `{"questions":[{"question":"Q","options":["A","B"],"explanation":"E"},{"question":"Q","options":["A","B"],"explanation":"E"}]}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client.response, client.err = tc.response, tc.modelErr
			if _, err := service.GenerateStudentQuiz(context.Background(), student, id, input); err == nil {
				t.Fatal("expected error")
			}
			assertCount(t, db, "SELECT COUNT(*) FROM course_question WHERE course_id = ? AND creator_id = ?", 0, id, student.UserID)
			assertCount(t, db, "SELECT COUNT(*) FROM course_question_content", 0)
			assertCount(t, db, "SELECT COUNT(*) FROM student_quiz_chapter", 0)
		})
	}
	client.response, client.err = quizTestResponse, nil
	if _, err := service.GenerateStudentQuiz(context.Background(), &access.Principal{UserID: student.UserID, Role: access.RoleTeacher}, id, input); err == nil {
		t.Fatal("teacher allowed")
	}
	input.ChapterKey = "missing"
	if _, err := service.GenerateStudentQuiz(context.Background(), student, id, input); err == nil {
		t.Fatal("unknown chapter allowed")
	}
	outsider := insertPrincipal(t, db, "student2", access.RoleStudent)
	client.prompt = ""
	input.ChapterKey = "one"
	if _, err := service.GenerateStudentQuiz(context.Background(), outsider, id, input); err == nil || client.prompt != "" {
		t.Fatal("invisible course reached the model")
	}
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", `{"chapters":[{"key":"one","activities":[{"type":"rich_text","payload":{"markdown":"（待生成）"}}]}]}`, id)
	if _, err := service.GenerateStudentQuiz(context.Background(), student, id, input); err == nil {
		t.Fatal("placeholder course allowed")
	}
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", quizTestContent, id)
	input.ChapterKey = "one"
	client.onCall = func() {
		testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", quizTestContent+" ", id)
	}
	if _, err := service.GenerateStudentQuiz(context.Background(), student, id, input); err == nil {
		t.Fatal("changed course allowed")
	}
	client.onCall = nil
	testutil.MustExec(t, db, "UPDATE course SET content_json = ? WHERE id = ?", quizTestContent, id)
	testutil.MustExec(t, db, "CREATE TRIGGER reject_quiz_chapter BEFORE INSERT ON student_quiz_chapter BEGIN SELECT RAISE(ABORT, 'test failure'); END")
	if _, err := service.GenerateStudentQuiz(context.Background(), student, id, input); err == nil {
		t.Fatal("expected transaction failure")
	}
	assertCount(t, db, "SELECT COUNT(*) FROM course_question WHERE course_id = ? AND creator_id = ?", 0, id, student.UserID)
	assertCount(t, db, "SELECT COUNT(*) FROM course_question_content", 0)
	assertCount(t, db, "SELECT COUNT(*) FROM student_quiz_chapter", 0)
}
