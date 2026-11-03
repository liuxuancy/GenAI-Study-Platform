package course

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	aicourse "caveclass/internal/ai/course"
)

// SyncCourseStructureFromJSON 把 content_json 中的章节与小节 upsert 到关系表。JSON 仍为权威来源。
func SyncCourseStructureFromJSON(ctx context.Context, database *gorm.DB, courseID int64, contentJSON string) error {
	contentJSON = strings.TrimSpace(contentJSON)
	if contentJSON == "" {
		return nil
	}
	var parsed aicourse.StandardCourse
	if err := json.Unmarshal([]byte(contentJSON), &parsed); err != nil {
		return err
	}
	chapters := append([]aicourse.Chapter(nil), parsed.Chapters...)
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Order < chapters[j].Order
	})
	now := time.Now().UTC()
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for chapterIndex, chapter := range chapters {
			chapterKey := strings.TrimSpace(chapter.Key)
			if chapterKey == "" {
				continue
			}
			sortOrder := chapter.Order
			if sortOrder <= 0 {
				sortOrder = chapterIndex + 1
			}
			description := strings.TrimSpace(chapter.Summary)
			if err := upsertCourseChapter(tx, courseID, chapterKey, chapter.Title, description, sortOrder, now); err != nil {
				return err
			}
			var chapterID int64
			if err := tx.Raw(
				`SELECT id FROM course_chapter WHERE course_id = ? AND chapter_key = ?`,
				courseID, chapterKey,
			).Scan(&chapterID).Error; err != nil {
				return err
			}
			activities := append([]aicourse.Activity(nil), chapter.Activities...)
			sort.SliceStable(activities, func(i, j int) bool {
				return activities[i].Order < activities[j].Order
			})
			for activityIndex, activity := range activities {
				sectionKey := strings.TrimSpace(activity.Key)
				if sectionKey == "" {
					continue
				}
				activityOrder := activity.Order
				if activityOrder <= 0 {
					activityOrder = activityIndex + 1
				}
				if err := upsertCourseSection(tx, courseID, chapterID, sectionKey, activity.Title, "", activityOrder, now); err != nil {
					return err
				}
			}
			if err := ensureSectionQuizzes(tx, courseID, chapterKey, activities, now); err != nil {
				return err
			}
		}
		return linkQuizNodes(tx, courseID)
	})
}

// EnsureQuizIndexes 章末题与测验小节题使用不同的唯一约束。
func EnsureQuizIndexes(database *gorm.DB) error {
	if err := database.Exec(`DROP INDEX IF EXISTS uidx_course_chapter`).Error; err != nil {
		return err
	}
	if err := database.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_chapter_only ON course_question(course_id, chapter_key) WHERE chapter_key IS NOT NULL AND chapter_key <> '' AND (section_key IS NULL OR section_key = '')`).Error; err != nil {
		return err
	}
	return database.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_section_quiz ON course_question(course_id, chapter_key, section_key) WHERE section_key IS NOT NULL AND section_key <> ''`).Error
}

func upsertCourseChapter(tx *gorm.DB, courseID int64, chapterKey, title, description string, sortOrder int, now time.Time) error {
	updated := tx.Exec(
		`UPDATE course_chapter
		 SET title = ?, description = ?, sort_order = ?, updated_at = ?
		 WHERE course_id = ? AND chapter_key = ?`,
		title, nullIfEmpty(description), sortOrder, now, courseID, chapterKey,
	)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected > 0 {
		return nil
	}
	return tx.Exec(
		`INSERT INTO course_chapter (course_id, chapter_key, title, description, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		courseID, chapterKey, title, nullIfEmpty(description), sortOrder, now, now,
	).Error
}

func upsertCourseSection(tx *gorm.DB, courseID, chapterID int64, sectionKey, title, description string, sortOrder int, now time.Time) error {
	updated := tx.Exec(
		`UPDATE course_section
		 SET title = ?, description = ?, sort_order = ?, updated_at = ?
		 WHERE chapter_id = ? AND section_key = ?`,
		title, nullIfEmpty(description), sortOrder, now, chapterID, sectionKey,
	)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected > 0 {
		return nil
	}
	return tx.Exec(
		`INSERT INTO course_section (course_id, chapter_id, section_key, title, description, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		courseID, chapterID, sectionKey, title, nullIfEmpty(description), sortOrder, now, now,
	).Error
}

func linkQuizChapterIDs(tx *gorm.DB, courseID int64) error {
	return linkQuizNodes(tx, courseID)
}

func linkQuizNodes(tx *gorm.DB, courseID int64) error {
	if err := tx.Exec(
		`UPDATE course_question
		 SET chapter_id = (
		   SELECT cc.id FROM course_chapter cc
		   WHERE cc.course_id = course_question.course_id AND cc.chapter_key = course_question.chapter_key
		 )
		 WHERE course_id = ? AND chapter_key IS NOT NULL AND chapter_key <> ''`,
		courseID,
	).Error; err != nil {
		return err
	}
	return tx.Exec(
		`UPDATE course_question
		 SET section_id = (
		   SELECT cs.id FROM course_section cs
		   JOIN course_chapter cc ON cc.id = cs.chapter_id
		   WHERE cc.course_id = course_question.course_id
		     AND cc.chapter_key = course_question.chapter_key
		     AND cs.section_key = course_question.section_key
		 )
		 WHERE course_id = ? AND section_key IS NOT NULL AND section_key <> ''`,
		courseID,
	).Error
}

func ensureSectionQuizzes(tx *gorm.DB, courseID int64, chapterKey string, activities []aicourse.Activity, now time.Time) error {
	quizActivities := make([]aicourse.Activity, 0)
	for _, activity := range activities {
		if activity.Type == aicourse.ActivityTypeQuizGame && strings.TrimSpace(activity.Key) != "" {
			quizActivities = append(quizActivities, activity)
		}
	}
	if len(quizActivities) == 0 {
		return nil
	}
	var ownerID int64
	if err := tx.Raw(`SELECT owner_id FROM course WHERE id = ?`, courseID).Scan(&ownerID).Error; err != nil {
		return err
	}
	var legacyIDs []int64
	if err := tx.Raw(
		`SELECT id FROM course_question
		 WHERE course_id = ? AND chapter_key = ? AND (section_key IS NULL OR section_key = '')
		 ORDER BY id`,
		courseID, chapterKey,
	).Scan(&legacyIDs).Error; err != nil {
		return err
	}
	legacyIndex := 0
	for _, activity := range quizActivities {
		sectionKey := strings.TrimSpace(activity.Key)
		var existing int64
		found := tx.Raw(
			`SELECT id FROM course_question WHERE course_id = ? AND chapter_key = ? AND section_key = ?`,
			courseID, chapterKey, sectionKey,
		).Scan(&existing)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected > 0 {
			continue
		}
		if legacyIndex < len(legacyIDs) {
			if err := tx.Exec(
				`UPDATE course_question SET section_key = ?, updated_at = ? WHERE id = ?`,
				sectionKey, now, legacyIDs[legacyIndex],
			).Error; err != nil {
				return err
			}
			legacyIndex++
			continue
		}
		title := sectionQuizTitle(courseID, activity.Title, sectionKey)
		if err := tx.Exec(
			`INSERT INTO course_question
			 (title, course_id, creator_id, question_count, is_publish, difficulty, chapter_key, section_key, creation_method, publish_status, created_at, updated_at)
			 VALUES (?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?)`,
			title, courseID, ownerID, PublishOff, "medium", chapterKey, sectionKey, CreationMethodManual, PublishStatusDraft, now, now,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func sectionQuizTitle(courseID int64, sectionTitle, sectionKey string) string {
	title := strings.TrimSpace(sectionTitle)
	if title == "" {
		title = sectionKey
	}
	full := fmt.Sprintf("%s · %d · %s", title, courseID, sectionKey)
	runes := []rune(full)
	if len(runes) > 200 {
		return string(runes[:200])
	}
	return full
}

// SyncAllCourseStructures 迁移或启动时同步全部课程的章节表。
func SyncAllCourseStructures(ctx context.Context, database *gorm.DB) error {
	var rows []struct {
		ID          int64
		ContentJSON string
	}
	if err := database.WithContext(ctx).Raw(
		`SELECT id, COALESCE(content_json, '') AS content_json FROM course WHERE content_json IS NOT NULL AND content_json <> ''`,
	).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := SyncCourseStructureFromJSON(ctx, database, row.ID, row.ContentJSON); err != nil {
			return err
		}
	}
	return nil
}
