package course

import (
	"context"
	"time"

	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// PublishQuiz 将测验设为已发布。
func (service *Service) PublishQuiz(ctx context.Context, principal *access.Principal, quizID int64) error {
	if err := service.setQuizPublishStatus(ctx, principal, quizID, PublishStatusPublished, PublishOn); err != nil {
		logs.CtxError(ctx, "[PublishQuiz] failed, err: %v", err)
		return err
	}
	logs.CtxInfo(ctx, "[PublishQuiz] done: quiz_id=%d", quizID)
	return nil
}

// ArchiveQuiz 将测验归档；学生侧不可见（创建者除外由列表逻辑处理）。
func (service *Service) ArchiveQuiz(ctx context.Context, principal *access.Principal, quizID int64) error {
	if err := service.setQuizPublishStatus(ctx, principal, quizID, PublishStatusArchived, PublishOff); err != nil {
		logs.CtxError(ctx, "[ArchiveQuiz] failed, err: %v", err)
		return err
	}
	logs.CtxInfo(ctx, "[ArchiveQuiz] done: quiz_id=%d", quizID)
	return nil
}

func (service *Service) setQuizPublishStatus(ctx context.Context, principal *access.Principal, quizID int64, status string, legacyPublish int) error {
	if principal.Role != access.RoleTeacher {
		return errs.Forbidden("only teachers can manage quizzes")
	}
	var row struct {
		CourseID  int64
		CreatorID int64
	}
	found := service.database.WithContext(ctx).Raw(
		`SELECT course_id, creator_id FROM course_question WHERE id = ?`, quizID,
	).Scan(&row)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		return errs.NotFound("quiz not found")
	}
	if err := service.requireTeacherCourse(ctx, principal, row.CourseID); err != nil {
		return err
	}
	if row.CreatorID != principal.UserID {
		return errs.Forbidden("only the quiz creator can change publish status")
	}
	now := time.Now().UTC()
	var publishedAt any
	if status == PublishStatusPublished {
		publishedAt = now
	} else {
		publishedAt = nil
	}
	result := service.database.WithContext(ctx).Exec(
		`UPDATE course_question
		 SET publish_status = ?, is_publish = ?, published_at = ?, updated_at = ?
		 WHERE id = ?`,
		status, legacyPublish, publishedAt, now, quizID,
	)
	return result.Error
}
