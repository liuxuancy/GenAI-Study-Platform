package course

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"caveclass/internal/ai/aierr"
	aicourse "caveclass/internal/ai/course"
	"caveclass/internal/ai/parse"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

const maxClassNameRunes = 64

func (service *Service) countClassmates(ctx context.Context, userID int64) (int, error) {
	var count int
	err := service.database.WithContext(ctx).Raw(classmateCountSQL, classmateRole(), userID).Scan(&count).Error
	return count, err
}

func (service *Service) chapterQuestions(ctx context.Context, courseID int64) (map[string]int64, error) {
	var rows []struct {
		ChapterKey string
		ID         int64
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT chapter_key, id FROM course_question
		 WHERE course_id = ? AND chapter_key IS NOT NULL AND chapter_key <> ''`,
		courseID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make(map[string]int64, len(rows))
	for _, row := range rows {
		ids[row.ChapterKey] = row.ID
	}
	return ids, nil
}

// SaveCourseContent 把正文入库。必须先通过 parse.ParseCourseJSON，不改课程名、所有者和是否放出。
func (service *Service) SaveCourseContent(ctx context.Context, principal *access.Principal, courseID int64, raw string) error {
	if principal.Role != access.RoleStudent && principal.Role != access.RoleTeacher {
		return errs.Forbidden("only the course owner can save course content")
	}
	parsed, _, err := parse.ParseCourseJSON(raw, false)
	if err != nil {
		return invalidContent(err)
	}
	encoded, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	result := service.database.WithContext(ctx).Exec(
		`UPDATE course
		 SET description = ?, audience = ?, language = ?, content_json = ?, updated_at = ?
		 WHERE id = ? AND owner_id = ?`,
		nullIfEmpty(parsed.Description), parsed.Audience, parsed.Language, string(encoded), time.Now().UTC(), courseID, principal.UserID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.NotFound("course not found")
	}
	return nil
}

func invalidContent(err error) error {
	if aiError, ok := aierr.As(err); ok && aiError.Status() == errs.SystemBadRequest {
		return errs.BadRequest("course content is not valid")
	}
	return errs.Internal(err)
}

func requireText(value string, maxRunes int, emptyMessage string, longMessage string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errs.BadRequest(emptyMessage)
	}
	if utf8.RuneCountInString(trimmed) > maxRunes {
		return "", errs.BadRequest(longMessage)
	}
	return trimmed, nil
}

func optionalText(value string, maxRunes int, longMessage string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if utf8.RuneCountInString(trimmed) > maxRunes {
		return "", errs.BadRequest(longMessage)
	}
	return trimmed, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func duplicateKey(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func uniqueIDs(ids []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, errs.BadRequest("id must be a positive integer")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func (service *Service) insertCourse(tx *gorm.DB, ownerID int64, title string, description string) (int64, error) {
	var major string
	if err := tx.Raw("SELECT COALESCE(major, '') FROM sys_user WHERE id = ?", ownerID).Scan(&major).Error; err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	err := tx.Exec(
		`INSERT INTO course (title, course_type, major, description, is_publish, owner_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		title, CourseTypeStructured, nullIfEmpty(major), nullIfEmpty(description), PublishOff, ownerID, now, now,
	).Error
	if duplicateKey(err) {
		return 0, errs.Conflict("course title already exists")
	}
	if err != nil {
		return 0, err
	}
	courseID, exists, err := findCourseID(tx, title)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errs.NotFound("course not found")
	}
	return courseID, nil
}

// 课程名上限与标准课程 JSON 一致，避免生成后再改名时对不上。
func normalizeCourseText(title string, description string) (string, string, error) {
	normalizedTitle, err := requireText(title, aicourse.MaxTitleRunes, "title is required", "title is too long")
	if err != nil {
		return "", "", err
	}
	normalizedDescription, err := optionalText(description, aicourse.MaxDescriptionRunes, "description is too long")
	if err != nil {
		return "", "", err
	}
	return normalizedTitle, normalizedDescription, nil
}
