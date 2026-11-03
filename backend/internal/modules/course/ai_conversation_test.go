package course

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/framework/testutil"
	"caveclass/internal/platform/errs"
)

const conversationCourseJSON = `{
  "schemaVersion": "1.0",
  "title": "会话课",
  "description": "用于会话测试",
  "audience": "GENERAL",
  "language": "zh-CN",
  "estimatedMinutes": 10,
  "chapters": [{
    "key": "ch-1",
    "title": "提问",
    "order": 1,
    "activities": [{
      "key": "act-1",
      "type": "rich_text",
      "title": "锚点",
      "order": 1,
      "payload": {"markdown": "一个好问题要有锚点。", "highlights": []}
    }]
  }]
}`

func TestAIConversationIsolationAndHistory(t *testing.T) {
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
	owner := studentPrincipal(t, database, "student1")
	other := insertPrincipal(t, database, "student-other", access.RoleStudent)

	created, err := service.CreateStudentCourse(context.Background(), owner, CreateCourseInput{Title: "会话隔离课"})
	if err != nil {
		t.Fatal(err)
	}
	testutil.MustExec(t, database, "UPDATE course SET content_json = ? WHERE id = ?", conversationCourseJSON, created.ID)

	if _, err := service.AskChapterQuestion(context.Background(), owner, created.ID, "ch-1", ChapterQAInput{Question: ""}); err == nil {
		t.Fatal("empty question should fail before saving")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_conversation WHERE course_id = ?", 0, created.ID)

	longQuestion := strings.Repeat("问", 50)
	first, err := service.AskChapterQuestion(context.Background(), owner, created.ID, "ch-1", ChapterQAInput{
		Question: longQuestion, ActivityKey: "act-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ConversationID == 0 || first.Answer == "" {
		t.Fatalf("first = %+v", first)
	}
	var title string
	if err := database.Raw("SELECT title FROM student_ai_conversation WHERE id = ?", first.ConversationID).Scan(&title).Error; err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(title) != conversationTitleRunes+1 || !strings.HasSuffix(title, "…") {
		t.Fatalf("title = %q", title)
	}

	second, err := service.AskChapterQuestion(context.Background(), owner, created.ID, "ch-1", ChapterQAInput{
		Question: "能给我一个反例吗？", ConversationID: first.ConversationID, ActivityKey: "act-1",
		History: []ChapterQAMessage{{Role: "user", Content: "这段前端历史不应被采用"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ConversationID != first.ConversationID {
		t.Fatalf("conversation = %d, want %d", second.ConversationID, first.ConversationID)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_message WHERE conversation_id = ?", 4, first.ConversationID)
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_message WHERE content = ?", 0, "这段前端历史不应被采用")

	items, err := service.ListAIConversations(context.Background(), owner, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].MessageCount != 4 || items[0].ID != first.ConversationID {
		t.Fatalf("items = %+v", items)
	}
	detail, err := service.GetAIConversation(context.Background(), owner, created.ID, first.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Messages) != 4 || detail.Messages[0].SectionTitle != "锚点" || detail.Messages[0].ChapterKey != "ch-1" {
		t.Fatalf("detail = %+v", detail)
	}

	if _, err := service.ListAIConversations(context.Background(), other, created.ID); err == nil || (!errors.Is(err, errs.ErrNotFound) && !errors.Is(err, errs.ErrForbidden)) {
		t.Fatalf("other list err = %v", err)
	}
	if _, err := service.GetAIConversation(context.Background(), other, created.ID, first.ConversationID); err == nil {
		t.Fatal("other student read the conversation")
	}
	if err := service.DeleteAIConversation(context.Background(), other, created.ID, first.ConversationID); err == nil {
		t.Fatal("other student deleted the conversation")
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_conversation WHERE id = ?", 1, first.ConversationID)

	testutil.MustExec(t, database,
		`INSERT INTO student_ai_conversation (user_id, course_id, title, created_at, updated_at) VALUES (?, ?, '别人的', datetime('now'), datetime('now'))`,
		other.UserID, created.ID,
	)
	ownerItems, err := service.ListAIConversations(context.Background(), owner, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerItems) != 1 {
		t.Fatalf("owner items = %+v", ownerItems)
	}

	if err := service.DeleteAIConversation(context.Background(), owner, created.ID, first.ConversationID); err != nil {
		t.Fatal(err)
	}
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_message WHERE conversation_id = ?", 0, first.ConversationID)
	assertCount(t, database, "SELECT COUNT(*) FROM student_ai_conversation WHERE id = ?", 0, first.ConversationID)
}
