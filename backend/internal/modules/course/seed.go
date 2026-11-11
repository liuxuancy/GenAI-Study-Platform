// 课程种子：按标题补齐课程、班级关联与学习进度。不写入章节（content_json）或习题。

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

// seedClassName 示例班。已发布且属于 teacher1 的课会挂到这个班，否则学生列表在可见性收紧后会变空。
const seedClassName = "示例班"

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
	},
	{
		Title: "How a lecture is read", IsPublish: PublishOn, Owner: "teacher1", Major: seedMajor,
	},
	{
		Title: "Asking on a line", IsPublish: PublishOff, Owner: "student1", Major: seedMajor,
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
		return ensureSeedClass(tx)
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

func ensureSeedClass(tx *gorm.DB) error {
	classID, err := ensureClass(tx, seedClassName)
	if err != nil {
		return err
	}
	if err := ensureClassMember(tx, classID, "teacher1"); err != nil {
		return err
	}
	if err := ensureClassMember(tx, classID, "student1"); err != nil {
		return err
	}
	for _, item := range seedCourses {
		if item.Owner != "teacher1" || item.IsPublish != PublishOn {
			continue
		}
		if err := ensureTeacherCourseClass(tx, item.Title, classID); err != nil {
			return err
		}
	}
	return nil
}

func ensureClass(tx *gorm.DB, name string) (int64, error) {
	var classID int64
	found := tx.Raw("SELECT id FROM study_class WHERE name = ?", name).Scan(&classID)
	if found.Error != nil {
		return 0, found.Error
	}
	if found.RowsAffected > 0 {
		return classID, nil
	}
	now := time.Now().UTC()
	if err := tx.Exec(
		"INSERT INTO study_class (name, created_at, updated_at) VALUES (?, ?, ?)",
		name, now, now,
	).Error; err != nil {
		return 0, err
	}
	found = tx.Raw("SELECT id FROM study_class WHERE name = ?", name).Scan(&classID)
	if found.Error != nil {
		return 0, found.Error
	}
	if found.RowsAffected == 0 {
		return 0, fmt.Errorf("class %q was not inserted", name)
	}
	return classID, nil
}

func ensureClassMember(tx *gorm.DB, classID int64, username string) error {
	userID, err := getUserIDByUsername(tx, username)
	if err != nil {
		return err
	}
	var count int64
	err = tx.Raw(
		"SELECT COUNT(*) FROM study_class_member WHERE class_id = ? AND user_id = ?",
		classID, userID,
	).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Exec(
		"INSERT INTO study_class_member (class_id, user_id) VALUES (?, ?)",
		classID, userID,
	).Error
}

// ensureTeacherCourseClass 只给当前仍然已发布、且所有者是 teacher1 的课补发布关系。已撤回的不重新发布。
func ensureTeacherCourseClass(tx *gorm.DB, title string, classID int64) error {
	var courseID int64
	found := tx.Raw(
		`SELECT c.id FROM course c
		 JOIN sys_user owner ON owner.id = c.owner_id
		 WHERE c.title = ? AND c.is_publish = ? AND owner.username = ?`,
		title, PublishOn, "teacher1",
	).Scan(&courseID)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		return nil
	}
	var count int64
	err := tx.Raw(
		"SELECT COUNT(*) FROM course_class WHERE course_id = ? AND class_id = ?",
		courseID, classID,
	).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Exec(
		"INSERT INTO course_class (course_id, class_id) VALUES (?, ?)",
		courseID, classID,
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
	quiz, err := quizPercent(tx, userID, courseID)
	if err != nil {
		return err
	}
	var oldWeights int64
	if err := tx.Raw(
		`SELECT COUNT(*) FROM migration_log WHERE kind = ?
		 AND NOT EXISTS (SELECT 1 FROM migration_log WHERE kind = ?)`,
		affectionBackfillKind, affectionWeightsKind,
	).Scan(&oldWeights).Error; err != nil {
		return err
	}
	affection := affectionBase(percent, quiz)
	if oldWeights > 0 {
		affection = (percent + quiz) / 2
	}
	return tx.Exec(
		"INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, ?, ?)",
		userID, courseID, percent, affection,
	).Error
}
