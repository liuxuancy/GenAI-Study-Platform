package course

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

type legacySeedQuestionItem struct {
	Content       string
	Options       string
	CorrectAnswer string
	UserOption    string
	Explanation   string
}

type legacySeedQuestionSet struct {
	Title         string
	CourseTitle   string
	Creator       string
	QuestionCount int
	IsPublish     int
	Difficulty    string
	Items         []legacySeedQuestionItem
}

var legacySeedQuestionSets = []legacySeedQuestionSet{
	{
		Title: "What is a commit?", CourseTitle: "Introduction to Git", Creator: "teacher1",
		QuestionCount: 2, IsPublish: PublishOn, Difficulty: "easy",
		Items: []legacySeedQuestionItem{
			{
				Content: "What does a commit store?", Options: "A snapshot of the work|A deleted file",
				CorrectAnswer: "A snapshot of the work", UserOption: "A snapshot of the work",
				Explanation: "A commit stores a snapshot of the files at that moment.",
			},
			{
				Content: "Who can read a commit after it is made?", Options: "Anyone with the repository|Only the computer that created it",
				CorrectAnswer: "Anyone with the repository", UserOption: "Anyone with the repository",
				Explanation: "A commit stays in the repository history for everyone who can read it.",
			},
		},
	},
	{
		Title: "Which paragraph comes first?", CourseTitle: "How a lecture is read", Creator: "teacher1",
		QuestionCount: 2, IsPublish: PublishOn, Difficulty: "medium",
		Items: []legacySeedQuestionItem{
			{
				Content: "Which paragraph comes first?", Options: "The opening|The closing",
				CorrectAnswer: "The opening", UserOption: "The opening",
				Explanation: "A lecture starts with the opening paragraph.",
			},
			{
				Content: "What follows the opening?", Options: "The body|The title",
				CorrectAnswer: "The body", UserOption: "",
				Explanation: "The body comes after the opening.",
			},
		},
	},
	{
		Title: "Where do you ask?", CourseTitle: "Asking on a line", Creator: "student1",
		QuestionCount: 2, IsPublish: PublishOff, Difficulty: "easy",
		Items: []legacySeedQuestionItem{
			{
				Content: "Where do you ask a question?", Options: "On the line|In a separate room",
				CorrectAnswer: "On the line", UserOption: "",
				Explanation: "Ask on the line so the next person can see it.",
			},
			{
				Content: "When do you ask?", Options: "After you have tried|Before reading",
				CorrectAnswer: "After you have tried", UserOption: "",
				Explanation: "Try the step first, then ask.",
			},
		},
	},
}

// seedLegacyQuestionSets 仅用于测试习题 API；生产种子不再写入套题。
func seedLegacyQuestionSets(t *testing.T, database *gorm.DB) {
	t.Helper()
	if err := database.Transaction(func(tx *gorm.DB) error {
		for _, item := range legacySeedQuestionSets {
			if err := ensureLegacyQuestionSet(tx, item); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("legacy question sets: %v", err)
	}
}

// seedLegacyQuestionSetsAgain 模拟旧种子对已有套题只补空难度的行为。
func seedLegacyQuestionSetsAgain(t *testing.T, database *gorm.DB) {
	t.Helper()
	if err := database.Transaction(func(tx *gorm.DB) error {
		for _, item := range legacySeedQuestionSets {
			questionID, exists, err := findQuestionIDByTitle(tx, item.Title)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("question %q missing", item.Title)
			}
			if err := fillEmptyLegacyQuestionDifficulty(tx, questionID, item.Difficulty); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("legacy question sets again: %v", err)
	}
}

func ensureLegacyQuestionSet(tx *gorm.DB, item legacySeedQuestionSet) error {
	questionID, exists, err := findQuestionIDByTitle(tx, item.Title)
	if err != nil {
		return err
	}
	if exists {
		return fillEmptyLegacyQuestionDifficulty(tx, questionID, item.Difficulty)
	}

	courseID, courseExists, err := findCourseID(tx, item.CourseTitle)
	if err != nil {
		return err
	}
	if !courseExists {
		return fmt.Errorf("question %q: course %q not found", item.Title, item.CourseTitle)
	}
	creatorID, err := getUserIDByUsername(tx, item.Creator)
	if err != nil {
		return fmt.Errorf("question %q: %w", item.Title, err)
	}

	now := time.Now().UTC()
	status := publishStatusFromLegacy(item.IsPublish)
	err = tx.Exec(
		`INSERT INTO course_question
		 (title, course_id, creator_id, question_count, is_publish, difficulty, publish_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Title, courseID, creatorID, item.QuestionCount, item.IsPublish, item.Difficulty, status, now, now,
	).Error
	if err != nil {
		return err
	}
	questionID, exists, err = findQuestionIDByTitle(tx, item.Title)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("question %q was not inserted", item.Title)
	}

	studentID, err := getUserIDByUsername(tx, "student1")
	if err != nil {
		return fmt.Errorf("question %q: %w", item.Title, err)
	}

	for index, content := range item.Items {
		err = tx.Exec(
			`INSERT INTO course_question_content
			 (question_id, content, options, correct_answer, explanation, question_type, sort_order, score, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
			questionID, content.Content, content.Options, content.CorrectAnswer, content.Explanation,
			QuestionTypeSingleChoice, index+1, now, now,
		).Error
		if err != nil {
			return err
		}
		if content.UserOption == "" {
			continue
		}
		if err := insertLegacySeedAnswer(tx, studentID, questionID, content.Content, content.UserOption); err != nil {
			return err
		}
	}
	return nil
}

func insertLegacySeedAnswer(tx *gorm.DB, studentID int64, questionID int64, content string, userOption string) error {
	var contentID int64
	found := tx.Raw(
		`SELECT id FROM course_question_content WHERE question_id = ? AND content = ?`,
		questionID, content,
	).Scan(&contentID)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		return fmt.Errorf("question content %q was not inserted", content)
	}
	return tx.Exec(
		`INSERT INTO student_question_progress (user_id, content_id, question_id, user_option) VALUES (?, ?, ?, ?)`,
		studentID, contentID, questionID, userOption,
	).Error
}

func findQuestionIDByTitle(tx *gorm.DB, title string) (int64, bool, error) {
	var questionID int64
	result := tx.Raw("SELECT id FROM course_question WHERE title = ?", title).Scan(&questionID)
	if result.Error != nil {
		return 0, false, result.Error
	}
	return questionID, result.RowsAffected > 0, nil
}

func fillEmptyLegacyQuestionDifficulty(tx *gorm.DB, questionID int64, difficulty string) error {
	return tx.Exec(
		"UPDATE course_question SET difficulty = ? WHERE id = ? AND (difficulty IS NULL OR difficulty = '')",
		difficulty, questionID,
	).Error
}

func seedCoursesAndLegacyQuestions(t *testing.T, database *gorm.DB) {
	t.Helper()
	if err := Seed(context.Background(), database); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	seedLegacyQuestionSets(t, database)
}
