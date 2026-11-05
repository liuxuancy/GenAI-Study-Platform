package course

import (
	"context"
	"strings"
	"time"

	"github.com/liuxuancy/Libs/logs"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// QuizDetail 规范化测验详情，与 QuestionDetail 字段对齐。
type QuizDetail = QuestionDetail

// QuizAttemptDetail 当前学生对某套题的唯一 Attempt（attempt_no = 1）。
type QuizAttemptDetail struct {
	ID                int64  `json:"id"`
	QuizID            int64  `json:"quizId"`
	Status            string `json:"status"`
	TotalQuestions    int    `json:"totalQuestions"`
	AnsweredQuestions int    `json:"answeredQuestions"`
	Score             *int   `json:"score,omitempty"`
}

// AttemptAnswerInput 保存单题答案。
type AttemptAnswerInput struct {
	Answer string `json:"answer"`
}

// ListCourseQuizzes 列出课程下学生可见的测验。
func (service *Service) ListCourseQuizzes(ctx context.Context, principal *access.Principal, courseID int64) ([]QuestionListItem, error) {
	all, err := service.GetQuestionList(ctx, principal)
	if err != nil {
		logs.CtxError(ctx, "[ListCourseQuizzes] failed, err: %v", err)
		return nil, err
	}
	items := make([]QuestionListItem, 0)
	for _, item := range all {
		if item.CourseID == courseID {
			items = append(items, item)
		}
	}
	logs.CtxInfo(ctx, "[ListCourseQuizzes] done: course_id=%d len(items)=%d", courseID, len(items))
	return items, nil
}

// GetQuizByID 与 GetQuestion 相同，保留旧 id 语义（course_question.id）。
func (service *Service) GetQuizByID(ctx context.Context, principal *access.Principal, quizID int64) (*QuizDetail, error) {
	output, err := service.GetQuestion(ctx, principal, quizID)
	if err != nil {
		logs.CtxError(ctx, "[GetQuizByID] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[GetQuizByID] done: id=%d title=%q question_count=%d percent=%d is_submit=%t score=%s", output.ID, output.Title, output.QuestionCount, output.Percent, output.IsSubmit, formatOptionalInt(output.Score))
	return output, nil
}

// GetQuizAttempt 返回当前学生对该测验的 Attempt。
func (service *Service) GetQuizAttempt(ctx context.Context, principal *access.Principal, quizID int64) (*QuizAttemptDetail, error) {
	if _, err := service.GetQuestion(ctx, principal, quizID); err != nil {
		logs.CtxError(ctx, "[GetQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	if err := syncQuizAttemptFromLegacy(ctx, service.database, principal.UserID, quizID); err != nil {
		logs.CtxError(ctx, "[GetQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	var row QuizAttemptDetail
	found := service.database.WithContext(ctx).Raw(
		`SELECT id, quiz_id, status, total_questions, answered_questions, score
		 FROM quiz_attempt WHERE quiz_id = ? AND student_id = ? AND attempt_no = 1`,
		quizID, principal.UserID,
	).Scan(&row)
	if found.Error != nil {
		logs.CtxError(ctx, "[GetQuizAttempt] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("attempt not found")
	}
	logs.CtxInfo(ctx, "[GetQuizAttempt] done: id=%d quiz_id=%d status=%s answered=%d/%d score=%s", row.ID, row.QuizID, row.Status, row.AnsweredQuestions, row.TotalQuestions, formatOptionalInt(row.Score))
	return &row, nil
}

// StartQuizAttempt 创建 attempt_no=1；若已有进度则同步后返回。
func (service *Service) StartQuizAttempt(ctx context.Context, principal *access.Principal, quizID int64) (*QuizAttemptDetail, error) {
	if _, err := service.GetQuestion(ctx, principal, quizID); err != nil {
		logs.CtxError(ctx, "[StartQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	now := time.Now().UTC()
	var total int
	if err := service.database.WithContext(ctx).Raw(
		`SELECT question_count FROM course_question WHERE id = ?`, quizID,
	).Scan(&total).Error; err != nil {
		logs.CtxError(ctx, "[StartQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	inserted := service.database.WithContext(ctx).Exec(
		`INSERT INTO quiz_attempt (quiz_id, student_id, attempt_no, status, total_questions, answered_questions, started_at, created_at, updated_at)
		 SELECT ?, ?, 1, ?, ?, 0, ?, ?, ?
		 WHERE NOT EXISTS (SELECT 1 FROM quiz_attempt WHERE quiz_id = ? AND student_id = ? AND attempt_no = 1)`,
		quizID, principal.UserID, AttemptStatusInProgress, total, now, now, now, quizID, principal.UserID,
	)
	if inserted.Error != nil {
		logs.CtxError(ctx, "[StartQuizAttempt] failed, err: %v", inserted.Error)
		return nil, inserted.Error
	}
	output, err := service.GetQuizAttempt(ctx, principal, quizID)
	if err != nil {
		logs.CtxError(ctx, "[StartQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[StartQuizAttempt] done: id=%d quiz_id=%d status=%s answered=%d/%d score=%s", output.ID, output.QuizID, output.Status, output.AnsweredQuestions, output.TotalQuestions, formatOptionalInt(output.Score))
	return output, nil
}

// UpdateAttemptAnswer 写入答案并同步旧表与 Attempt 统计。
func (service *Service) UpdateAttemptAnswer(ctx context.Context, principal *access.Principal, attemptID, questionContentID int64, answer string) (*QuizDetail, error) {
	var meta struct {
		QuizID int64
		Status string
	}
	found := service.database.WithContext(ctx).Raw(
		`SELECT quiz_id, status FROM quiz_attempt WHERE id = ? AND student_id = ?`,
		attemptID, principal.UserID,
	).Scan(&meta)
	if found.Error != nil {
		logs.CtxError(ctx, "[UpdateAttemptAnswer] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("attempt not found")
	}
	if meta.Status == AttemptStatusSubmitted {
		return nil, errs.BadRequest("attempt already submitted")
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil, errs.BadRequest("answer is required")
	}
	output, err := service.AnswerQuestion(ctx, principal, meta.QuizID, questionContentID, answer, false)
	if err != nil {
		logs.CtxError(ctx, "[UpdateAttemptAnswer] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[UpdateAttemptAnswer] done: attempt_id=%d question_content_id=%d answer=%q id=%d title=%q question_count=%d percent=%d is_submit=%t score=%s", attemptID, questionContentID, answer, output.ID, output.Title, output.QuestionCount, output.Percent, output.IsSubmit, formatOptionalInt(output.Score))
	return output, nil
}

// SubmitQuizAttempt 提交测验。
func (service *Service) SubmitQuizAttempt(ctx context.Context, principal *access.Principal, attemptID int64) (*QuizDetail, error) {
	var meta struct {
		QuizID int64
		Status string
	}
	found := service.database.WithContext(ctx).Raw(
		`SELECT quiz_id, status FROM quiz_attempt WHERE id = ? AND student_id = ?`,
		attemptID, principal.UserID,
	).Scan(&meta)
	if found.Error != nil {
		logs.CtxError(ctx, "[SubmitQuizAttempt] failed, err: %v", found.Error)
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("attempt not found")
	}
	if meta.Status == AttemptStatusSubmitted {
		return nil, errs.BadRequest("attempt already submitted")
	}
	if err := markQuestionSubmitted(service.database.WithContext(ctx), principal.UserID, meta.QuizID); err != nil {
		logs.CtxError(ctx, "[SubmitQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	if err := syncQuizAttemptFromLegacy(ctx, service.database, principal.UserID, meta.QuizID); err != nil {
		logs.CtxError(ctx, "[SubmitQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	service.recordQuizLearningEvents(ctx, principal.UserID, meta.QuizID, true)
	output, err := service.GetQuestion(ctx, principal, meta.QuizID)
	if err != nil {
		logs.CtxError(ctx, "[SubmitQuizAttempt] failed, err: %v", err)
		return nil, err
	}
	logs.CtxInfo(ctx, "[SubmitQuizAttempt] done: attempt_id=%d id=%d title=%q question_count=%d percent=%d is_submit=%t score=%s", attemptID, output.ID, output.Title, output.QuestionCount, output.Percent, output.IsSubmit, formatOptionalInt(output.Score))
	return output, nil
}
