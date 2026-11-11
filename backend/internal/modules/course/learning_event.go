package course

import (
	"context"
	"time"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
)

const (
	estimatedProgressMinutes = 5
	estimatedQuizAnswerMinutes = 3
	estimatedChapterQAMinutes  = 5
	estimatedStorySessionMinutes = 5
)

// HeartbeatInput 课程页上报学习时长。
type HeartbeatInput struct {
	Seconds int `json:"seconds"`
}

func (service *Service) RecordCourseHeartbeat(ctx context.Context, principal *access.Principal, courseID int64, seconds int) error {
	if principal.Role != access.RoleStudent {
		return errs.Forbidden("only students can record learning time")
	}
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 120 {
		seconds = 120
	}
	if _, err := service.GetCourse(ctx, principal, courseID); err != nil {
		return err
	}
	service.recordLearningEvent(ctx, principal.UserID, EventCourseSession, &courseID, nil, "", seconds)
	return nil
}

func (service *Service) recordLearningEvent(
	ctx context.Context,
	userID int64,
	eventType string,
	courseID *int64,
	quizID *int64,
	refKey string,
	durationSeconds int,
) {
	if durationSeconds < 0 {
		durationSeconds = 0
	}
	now := time.Now().UTC()
	err := service.database.WithContext(ctx).Exec(
		`INSERT INTO student_learning_event
		 (user_id, event_type, course_id, quiz_id, ref_key, duration_seconds, occurred_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, eventType, courseID, quizID, refKey, durationSeconds, now,
	).Error
	if err != nil {
		logs.CtxError(ctx, "record learning event: %v", err)
	}
}

func effectiveEventMinutes(eventType string, durationSeconds int) int {
	switch eventType {
	case EventCourseSession, EventQuizSubmit:
		minutes := durationSeconds / 60
		if durationSeconds%60 > 0 {
			minutes++
		}
		return minutes
	case EventCourseProgress:
		return estimatedProgressMinutes
	case EventQuizAnswer:
		return estimatedQuizAnswerMinutes
	case EventChapterQA:
		return estimatedChapterQAMinutes
	case EventStorySession:
		return estimatedStorySessionMinutes
	default:
		return 0
	}
}

func (service *Service) recordQuizLearningEvents(ctx context.Context, userID int64, quizID int64, includeSubmit bool) {
	var meta struct {
		CourseID int64
	}
	if err := service.database.WithContext(ctx).Raw(
		`SELECT course_id FROM course_question WHERE id = ?`, quizID,
	).Scan(&meta).Error; err != nil || meta.CourseID == 0 {
		return
	}
	courseID := meta.CourseID
	quiz := quizID
	service.recordLearningEvent(ctx, userID, EventQuizAnswer, &courseID, &quiz, "", 0)

	if !includeSubmit {
		return
	}
	var attempt struct {
		StartedAt   *time.Time
		SubmittedAt *time.Time
	}
	_ = service.database.WithContext(ctx).Raw(
		`SELECT started_at, submitted_at FROM quiz_attempt
		 WHERE quiz_id = ? AND student_id = ? AND attempt_no = 1`,
		quizID, userID,
	).Scan(&attempt)
	duration := 0
	if attempt.StartedAt != nil && attempt.SubmittedAt != nil {
		seconds := int(attempt.SubmittedAt.Sub(*attempt.StartedAt).Seconds())
		if seconds > 0 {
			duration = seconds
		}
	}
	service.recordLearningEvent(ctx, userID, EventQuizSubmit, &courseID, &quiz, "", duration)
}
