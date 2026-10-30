// 课程种子：按标题补齐课程、进度和套题。已有行只补空的专业和难度。

package course

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type seedProgress struct {
	Username string
	Percent  int
}

const seedMajor = "Computer Science"

type seedCourse struct {
	Title     string
	IsPublish int
	Owner     string
	Major     string
	Progress  []seedProgress
}

var seedCourses = []seedCourse{
	{
		Title: "Introduction to Git", IsPublish: PublishOn, Owner: "teacher1", Major: seedMajor,
		Progress: []seedProgress{{Username: "student1", Percent: 35}},
	},
	{
		Title: "How a lecture is read", IsPublish: PublishOn, Owner: "teacher1", Major: seedMajor,
		Progress: []seedProgress{{Username: "student1", Percent: 70}},
	},
	{
		Title: "Asking on a line", IsPublish: PublishOff, Owner: "student1", Major: seedMajor,
		Progress: []seedProgress{{Username: "student1", Percent: 10}},
	},
	{
		Title: "Draft lecture", IsPublish: PublishOff, Owner: "teacher1", Major: seedMajor,
	},
}

// Seed 在用户种子之后调用。可重复执行。
func Seed(ctx context.Context, database *gorm.DB) error {
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dropContentUserOption(tx); err != nil {
			return err
		}
		for _, item := range seedCourses {
			if err := ensureCourse(tx, item); err != nil {
				return err
			}
		}
		for _, item := range seedQuestionSets {
			if err := ensureQuestionSet(tx, item); err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureCourse(tx *gorm.DB, item seedCourse) error {
	ownerID, err := getUserIDByUsername(tx, item.Owner)
	if err != nil {
		return fmt.Errorf("course %q: %w", item.Title, err)
	}

	courseID, exists, err := findCourseID(tx, item.Title)
	if err != nil {
		return err
	}
	if !exists {
		now := time.Now().UTC()
		err = tx.Exec(
			`INSERT INTO course (title, course_type, major, is_publish, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.Title, CourseTypeStructured, item.Major, item.IsPublish, ownerID, now, now,
		).Error
		if err != nil {
			return err
		}
		courseID, exists, err = findCourseID(tx, item.Title)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("course %q was not inserted", item.Title)
		}
	} else if err := fillEmptyCourseMajor(tx, courseID, item.Major); err != nil {
		return err
	}

	for _, progress := range item.Progress {
		userID, err := getUserIDByUsername(tx, progress.Username)
		if err != nil {
			return fmt.Errorf("course %q progress: %w", item.Title, err)
		}
		if err := ensureProgress(tx, userID, courseID, progress.Percent); err != nil {
			return err
		}
	}
	return nil
}

type seedQuestionItem struct {
	Content       string
	Options       string
	CorrectAnswer string
	UserOption    string
	Explanation   string
}

type seedQuestionSet struct {
	Title         string
	CourseTitle   string
	Creator       string
	QuestionCount int
	IsPublish     int
	Difficulty    string
	Items         []seedQuestionItem
}

var seedQuestionSets = []seedQuestionSet{
	{
		Title: "What is a commit?", CourseTitle: "Introduction to Git", Creator: "teacher1",
		QuestionCount: 2, IsPublish: PublishOn, Difficulty: "easy",
		Items: []seedQuestionItem{
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
		Items: []seedQuestionItem{
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
		Items: []seedQuestionItem{
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

func ensureQuestionSet(tx *gorm.DB, item seedQuestionSet) error {
	questionID, exists, err := findQuestionID(tx, item.Title)
	if err != nil {
		return err
	}
	if exists {
		return fillEmptyQuestionDifficulty(tx, questionID, item.Difficulty)
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

	err = tx.Exec(
		`INSERT INTO course_question (title, course_id, creator_id, question_count, is_publish, difficulty)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.Title, courseID, creatorID, item.QuestionCount, item.IsPublish, item.Difficulty,
	).Error
	if err != nil {
		return err
	}
	questionID, exists, err = findQuestionID(tx, item.Title)
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

	for _, content := range item.Items {
		err = tx.Exec(
			`INSERT INTO course_question_content
			 (question_id, content, options, correct_answer, explanation)
			 VALUES (?, ?, ?, ?, ?)`,
			questionID, content.Content, content.Options, content.CorrectAnswer, content.Explanation,
		).Error
		if err != nil {
			return err
		}
		if content.UserOption == "" {
			continue
		}
		if err := insertSeedAnswer(tx, studentID, questionID, content.Content, content.UserOption); err != nil {
			return err
		}
	}
	return nil
}

func insertSeedAnswer(tx *gorm.DB, studentID int64, questionID int64, content string, userOption string) error {
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

func dropContentUserOption(tx *gorm.DB) error {
	var count int64
	err := tx.Raw(
		`SELECT COUNT(*) FROM pragma_table_info('course_question_content') WHERE name = 'user_option'`,
	).Scan(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	return tx.Exec(`ALTER TABLE course_question_content DROP COLUMN user_option`).Error
}

func findQuestionID(tx *gorm.DB, title string) (int64, bool, error) {
	var questionID int64
	result := tx.Raw("SELECT id FROM course_question WHERE title = ?", title).Scan(&questionID)
	if result.Error != nil {
		return 0, false, result.Error
	}
	return questionID, result.RowsAffected > 0, nil
}

func findCourseID(tx *gorm.DB, title string) (int64, bool, error) {
	var courseID int64
	result := tx.Raw("SELECT id FROM course WHERE title = ?", title).Scan(&courseID)
	if result.Error != nil {
		return 0, false, result.Error
	}
	return courseID, result.RowsAffected > 0, nil
}

func getUserIDByUsername(tx *gorm.DB, username string) (int64, error) {
	var userID int64
	result := tx.Raw("SELECT id FROM sys_user WHERE username = ?", username).Scan(&userID)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, fmt.Errorf("user %s not found", username)
	}
	return userID, nil
}

func fillEmptyCourseMajor(tx *gorm.DB, courseID int64, major string) error {
	return tx.Exec(
		"UPDATE course SET major = ? WHERE id = ? AND (major IS NULL OR major = '')",
		major, courseID,
	).Error
}

func fillEmptyQuestionDifficulty(tx *gorm.DB, questionID int64, difficulty string) error {
	return tx.Exec(
		"UPDATE course_question SET difficulty = ? WHERE id = ? AND (difficulty IS NULL OR difficulty = '')",
		difficulty, questionID,
	).Error
}

func ensureProgress(tx *gorm.DB, userID int64, courseID int64, percent int) error {
	var count int64
	err := tx.Raw(
		"SELECT COUNT(*) FROM student_course_progress WHERE user_id = ? AND course_id = ?",
		userID, courseID,
	).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Exec(
		"INSERT INTO student_course_progress (user_id, course_id, percent) VALUES (?, ?, ?)",
		userID, courseID, percent,
	).Error
}
