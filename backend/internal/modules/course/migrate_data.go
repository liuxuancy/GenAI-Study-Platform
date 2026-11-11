package course

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// RunStartupMigrations 幂等：结构同步、Quiz 元数据回填、Attempt/Answer 迁移。
func RunStartupMigrations(ctx context.Context, database *gorm.DB) error {
	if err := EnsureQuizIndexes(database); err != nil {
		return err
	}
	if err := deletePlaceholderQuestions(database); err != nil {
		return fmt.Errorf("drop placeholder questions: %w", err)
	}
	if err := SyncAllCourseStructures(ctx, database); err != nil {
		return fmt.Errorf("sync course structure: %w", err)
	}
	if err := backfillQuizMetadata(ctx, database); err != nil {
		return fmt.Errorf("backfill quiz metadata: %w", err)
	}
	if err := backfillQuestionContentMeta(ctx, database); err != nil {
		return fmt.Errorf("backfill question content: %w", err)
	}
	if err := migrateLegacyAttempts(ctx, database); err != nil {
		return fmt.Errorf("migrate attempts: %w", err)
	}
	if err := migrateAffectionWeights(ctx, database); err != nil {
		return fmt.Errorf("migrate affection weights: %w", err)
	}
	if err := backfillAffection(ctx, database); err != nil {
		return fmt.Errorf("backfill affection: %w", err)
	}
	return nil
}

func backfillQuizMetadata(ctx context.Context, tx *gorm.DB) error {
	var rows []struct {
		ID          int64
		Title       string
		IsPublish   int
		ChapterKey  string
		CreatorRole string
	}
	err := tx.WithContext(ctx).Raw(
		`SELECT q.id, q.title, q.is_publish, COALESCE(q.chapter_key, '') AS chapter_key, COALESCE(u.role, '') AS creator_role
		 FROM course_question q
		 JOIN sys_user u ON u.id = q.creator_id
		 WHERE COALESCE(q.publish_status, '') = '' OR COALESCE(q.creation_method, '') = ''`,
	).Scan(&rows).Error
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, row := range rows {
		status := publishStatusFromLegacy(row.IsPublish)
		method := inferCreationMethod(row.Title, row.ChapterKey, row.CreatorRole)
		if err := tx.Exec(
			`UPDATE course_question
			 SET publish_status = COALESCE(NULLIF(publish_status, ''), ?),
			     creation_method = COALESCE(NULLIF(creation_method, ''), ?),
			     is_publish = CASE WHEN ? = ? THEN 1 ELSE 0 END,
			     updated_at = COALESCE(updated_at, ?)
			 WHERE id = ?`,
			status, nullIfEmpty(method), status, PublishStatusPublished, now, row.ID,
		).Error; err != nil {
			return err
		}
	}
	return linkAllQuizChapterIDs(ctx, tx)
}

func linkAllQuizChapterIDs(ctx context.Context, tx *gorm.DB) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE course_question
		 SET chapter_id = (
		   SELECT cc.id FROM course_chapter cc
		   WHERE cc.course_id = course_question.course_id AND cc.chapter_key = course_question.chapter_key
		 )
		 WHERE chapter_key IS NOT NULL AND chapter_key <> ''`,
	).Error
}

func backfillQuestionContentMeta(ctx context.Context, tx *gorm.DB) error {
	now := time.Now().UTC()
	if err := tx.WithContext(ctx).Exec(
		`UPDATE course_question_content SET question_type = ? WHERE question_type IS NULL OR question_type = ''`,
		QuestionTypeSingleChoice,
	).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE course_question_content SET score = 1 WHERE score IS NULL OR score = 0`,
	).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`UPDATE course_question_content
		 SET sort_order = (
		   SELECT COUNT(*) FROM course_question_content qc2
		   WHERE qc2.question_id = course_question_content.question_id AND qc2.id <= course_question_content.id
		 ),
		 updated_at = COALESCE(updated_at, ?)
		 WHERE sort_order IS NULL OR sort_order = 0`,
		now,
	).Error
}

func migrateLegacyAttempts(ctx context.Context, tx *gorm.DB) error {
	var pairs []struct {
		UserID     int64
		QuestionID int64
	}
	err := tx.WithContext(ctx).Raw(
		`SELECT DISTINCT user_id, question_id FROM (
		   SELECT user_id, question_id FROM student_question_progress
		   UNION
		   SELECT user_id, question_id FROM student_question_record
		 )`,
	).Scan(&pairs).Error
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		if err := writeQuizAttemptFromLegacy(ctx, tx, pair.UserID, pair.QuestionID); err != nil {
			return err
		}
	}
	return nil
}

func logMigrationIssue(tx *gorm.DB, kind, refKey, detail string) {
	_ = tx.Exec(
		`INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, ?, ?, ?)`,
		kind, refKey, detail, time.Now().UTC(),
	)
}
