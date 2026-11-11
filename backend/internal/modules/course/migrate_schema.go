package course

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsurePreAutoMigrate 在 GORM AutoMigrate 之前处理 SQLite 无法直接加 NOT NULL 无默认列的情况。
func EnsurePreAutoMigrate(database *gorm.DB) error {
	if err := ensureStudentCourseProgressAffection(database); err != nil {
		return fmt.Errorf("student_course_progress.affection: %w", err)
	}
	return nil
}

func ensureStudentCourseProgressAffection(database *gorm.DB) error {
	var tables int64
	if err := database.Raw(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'student_course_progress'`,
	).Scan(&tables).Error; err != nil {
		return err
	}
	if tables == 0 {
		return nil
	}
	var cols int64
	if err := database.Raw(
		`SELECT COUNT(*) FROM pragma_table_info('student_course_progress') WHERE name = 'affection'`,
	).Scan(&cols).Error; err != nil {
		return err
	}
	if cols > 0 {
		return nil
	}
	return database.Exec(
		`ALTER TABLE student_course_progress ADD COLUMN affection INTEGER NOT NULL DEFAULT 0`,
	).Error
}
