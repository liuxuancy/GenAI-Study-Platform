package course

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"caveclass/internal/ai"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

const conversationTitleRunes = 40

type AIConversationItem struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	CourseID         int64     `json:"courseId"`
	CourseTitle      string    `json:"courseTitle"`
	CourseAccessible bool      `json:"courseAccessible"`
	UpdatedAt        time.Time `json:"updatedAt"`
	MessageCount     int       `json:"messageCount"`
}

type AIConversationMessage struct {
	ID           int64     `json:"id"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	ChapterKey   string    `json:"chapterKey"`
	ActivityKey  string    `json:"activityKey"`
	SectionTitle string    `json:"sectionTitle"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AIConversationDetail struct {
	ID        int64                   `json:"id"`
	Title     string                  `json:"title"`
	UpdatedAt time.Time               `json:"updatedAt"`
	Messages  []AIConversationMessage `json:"messages"`
}

func (service *Service) requireStudentCourse(ctx context.Context, principal *access.Principal, courseID int64) error {
	if principal.Role != access.RoleStudent {
		return errs.Forbidden("only students can use course conversations")
	}
	var id int64
	found := service.database.WithContext(ctx).Raw(
		`SELECT c.id FROM course c WHERE c.id = ? AND `+studentVisibleCourseSQL("c"),
		append([]any{courseID}, studentVisibleCourseArgs(principal.UserID)...)...,
	).Scan(&id)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		return errs.NotFound("course not found")
	}
	return nil
}

func (service *Service) ListAIConversations(ctx context.Context, principal *access.Principal, courseID int64) ([]AIConversationItem, error) {
	if err := service.requireStudentCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	var items []AIConversationItem
	err := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title, c.updated_at, COUNT(m.id) AS message_count
		 FROM student_ai_conversation c
		 LEFT JOIN student_ai_message m ON m.conversation_id = c.id
		 WHERE c.user_id = ? AND c.course_id = ?
		 GROUP BY c.id, c.title, c.updated_at
		 ORDER BY c.updated_at DESC, c.id DESC`,
		principal.UserID, courseID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []AIConversationItem{}
	}
	return items, nil
}

func (service *Service) ListOwnedAIConversations(ctx context.Context, principal *access.Principal) ([]AIConversationItem, error) {
	if principal.Role != access.RoleStudent && principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only students and teachers can use the workspace")
	}
	var rows []struct {
		ID           int64
		Title        string
		CourseID     int64
		CourseTitle  string
		UpdatedAt    time.Time
		MessageCount int
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT c.id, c.title, c.course_id, COALESCE(co.title, '') AS course_title, c.updated_at,
		        COUNT(m.id) AS message_count
		 FROM student_ai_conversation c
		 LEFT JOIN course co ON co.id = c.course_id
		 LEFT JOIN student_ai_message m ON m.conversation_id = c.id
		 WHERE c.user_id = ?
		 GROUP BY c.id, c.title, c.course_id, co.title, c.updated_at
		 ORDER BY c.updated_at DESC, c.id DESC`,
		principal.UserID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	visible := map[int64]struct{}{}
	var visibleIDs []int64
	var visibleQuery string
	var visibleArgs []any
	if principal.Role == access.RoleTeacher {
		visibleQuery = `SELECT id FROM course WHERE owner_id = ?`
		visibleArgs = []any{principal.UserID}
	} else {
		visibleQuery = `SELECT c.id FROM course c WHERE ` + studentVisibleCourseSQL("c")
		visibleArgs = studentVisibleCourseArgs(principal.UserID)
	}
	if err := service.database.WithContext(ctx).Raw(visibleQuery, visibleArgs...).Scan(&visibleIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range visibleIDs {
		visible[id] = struct{}{}
	}
	items := make([]AIConversationItem, 0, len(rows))
	for _, row := range rows {
		_, accessible := visible[row.CourseID]
		title := row.CourseTitle
		if !accessible {
			title = ""
		}
		items = append(items, AIConversationItem{
			ID: row.ID, Title: row.Title, CourseID: row.CourseID, CourseTitle: title,
			CourseAccessible: accessible, UpdatedAt: row.UpdatedAt, MessageCount: row.MessageCount,
		})
	}
	return items, nil
}

func (service *Service) GetAIConversation(ctx context.Context, principal *access.Principal, courseID, conversationID int64) (*AIConversationDetail, error) {
	if err := service.requireStudentCourse(ctx, principal, courseID); err != nil {
		return nil, err
	}
	detail, err := service.ownedConversation(ctx, principal.UserID, courseID, conversationID)
	if err != nil {
		return nil, err
	}
	messages, err := service.conversationMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	detail.Messages = messages
	return detail, nil
}

func (service *Service) GetOwnedAIConversation(ctx context.Context, principal *access.Principal, conversationID int64) (*AIConversationDetail, error) {
	if principal.Role != access.RoleStudent && principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only students and teachers can use the workspace")
	}
	var row struct {
		ID        int64
		Title     string
		UserID    int64
		CourseID  *int64
		UpdatedAt time.Time
	}
	found := service.database.WithContext(ctx).Raw(
		`SELECT id, title, user_id, course_id, updated_at FROM student_ai_conversation WHERE id = ?`, conversationID,
	).Scan(&row)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 || row.UserID != principal.UserID {
		return nil, errs.NotFound("conversation not found")
	}
	messages, err := service.conversationMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	return &AIConversationDetail{ID: row.ID, Title: row.Title, UpdatedAt: row.UpdatedAt, Messages: messages}, nil
}

func (service *Service) DeleteAIConversation(ctx context.Context, principal *access.Principal, courseID, conversationID int64) error {
	if err := service.requireStudentCourse(ctx, principal, courseID); err != nil {
		return err
	}
	if _, err := service.ownedConversation(ctx, principal.UserID, courseID, conversationID); err != nil {
		return err
	}
	return service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM student_ai_message WHERE conversation_id = ?`, conversationID).Error; err != nil {
			return err
		}
		result := tx.Exec(
			`DELETE FROM student_ai_conversation WHERE id = ? AND user_id = ? AND course_id = ?`,
			conversationID, principal.UserID, courseID,
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errs.NotFound("conversation not found")
		}
		return nil
	})
}

func (service *Service) ownedConversation(ctx context.Context, userID, courseID, conversationID int64) (*AIConversationDetail, error) {
	var row struct {
		ID        int64
		Title     string
		UpdatedAt time.Time
	}
	found := service.database.WithContext(ctx).Raw(
		`SELECT id, title, updated_at FROM student_ai_conversation WHERE id = ? AND user_id = ? AND course_id = ?`,
		conversationID, userID, courseID,
	).Scan(&row)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("conversation not found")
	}
	return &AIConversationDetail{ID: row.ID, Title: row.Title, UpdatedAt: row.UpdatedAt, Messages: []AIConversationMessage{}}, nil
}

func (service *Service) conversationMessages(ctx context.Context, conversationID int64) ([]AIConversationMessage, error) {
	var messages []AIConversationMessage
	err := service.database.WithContext(ctx).Raw(
		`SELECT id, role, content, COALESCE(chapter_key, '') AS chapter_key,
		        COALESCE(activity_key, '') AS activity_key, COALESCE(section_title, '') AS section_title, created_at
		 FROM student_ai_message WHERE conversation_id = ? ORDER BY id ASC`,
		conversationID,
	).Scan(&messages).Error
	if err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []AIConversationMessage{}
	}
	return messages, nil
}

func (service *Service) recentConversationHistory(ctx context.Context, userID, courseID, conversationID int64) ([]ai.HistoryTurn, error) {
	if conversationID == 0 {
		return nil, nil
	}
	if _, err := service.ownedConversation(ctx, userID, courseID, conversationID); err != nil {
		return nil, err
	}
	var rows []AIConversationMessage
	err := service.database.WithContext(ctx).Raw(
		`SELECT role, content, COALESCE(section_title, '') AS section_title
		 FROM student_ai_message WHERE conversation_id = ?
		 ORDER BY id DESC LIMIT ?`,
		conversationID, 8,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	history := make([]ai.HistoryTurn, 0, len(rows))
	for index := len(rows) - 1; index >= 0; index-- {
		history = append(history, ai.HistoryTurn{
			Role:         rows[index].Role,
			Content:      rows[index].Content,
			SectionTitle: rows[index].SectionTitle,
		})
	}
	return history, nil
}

func (service *Service) saveChapterExchange(ctx context.Context, userID, courseID, conversationID int64, chapterKey, activityKey, sectionTitle, question, answer string) (int64, error) {
	now := time.Now().UTC()
	err := service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if conversationID == 0 {
			var created int64
			insert := tx.Raw(
				`INSERT INTO student_ai_conversation (user_id, course_id, title, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?) RETURNING id`,
				userID, courseID, conversationTitle(question), now, now,
			).Scan(&created)
			if insert.Error != nil {
				return insert.Error
			}
			conversationID = created
		} else {
			result := tx.Exec(
				`UPDATE student_ai_conversation SET updated_at = ? WHERE id = ? AND user_id = ? AND course_id = ?`,
				now, conversationID, userID, courseID,
			)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errs.NotFound("conversation not found")
			}
		}
		for _, message := range []struct{ role, content string }{
			{role: "user", content: question},
			{role: "assistant", content: answer},
		} {
			if err := tx.Exec(
				`INSERT INTO student_ai_message
				 (conversation_id, role, content, chapter_key, activity_key, section_title, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				conversationID, message.role, message.content, chapterKey, activityKey, sectionTitle, now,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return conversationID, nil
}

func conversationTitle(question string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(question)), " ")
	if text == "" {
		return "新对话"
	}
	if utf8.RuneCountInString(text) <= conversationTitleRunes {
		return text
	}
	return string([]rune(text)[:conversationTitleRunes]) + "…"
}

func sectionTitleFor(chapter *aicourse.Chapter, activityKey string) (string, string) {
	activityKey = strings.TrimSpace(activityKey)
	if chapter == nil || activityKey == "" {
		return "", ""
	}
	for _, activity := range chapter.Activities {
		if activity.Key == activityKey {
			return activity.Key, strings.TrimSpace(activity.Title)
		}
	}
	return "", ""
}

func ensureOptionalConversationCourse(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	var notNull int
	row := database.Raw(`SELECT "notnull" FROM pragma_table_info('student_ai_conversation') WHERE name = 'course_id'`).Row()
	if row == nil {
		return nil
	}
	if err := row.Scan(&notNull); err != nil || notNull == 0 {
		return nil
	}
	return database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TABLE student_ai_conversation_new (
			id integer PRIMARY KEY AUTOINCREMENT,
			user_id integer NOT NULL,
			course_id integer,
			title text NOT NULL,
			created_at datetime,
			updated_at datetime
		)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO student_ai_conversation_new (id, user_id, course_id, title, created_at, updated_at)
			SELECT id, user_id, course_id, title, created_at, updated_at FROM student_ai_conversation`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE student_ai_conversation`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`ALTER TABLE student_ai_conversation_new RENAME TO student_ai_conversation`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_ai_conv_user_course ON student_ai_conversation(user_id, course_id, updated_at)`).Error
	})
}
