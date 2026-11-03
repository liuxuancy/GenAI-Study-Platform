package course

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// syncQuizAttemptFromLegacy 根据 student_question_* 表 upsert quiz_attempt 与 quiz_answer。
func syncQuizAttemptFromLegacy(ctx context.Context, database *gorm.DB, userID, quizID int64) error {
	return writeQuizAttemptFromLegacy(ctx, database.WithContext(ctx), userID, quizID)
}

func writeQuizAttemptFromLegacy(ctx context.Context, tx *gorm.DB, userID, quizID int64) error {
	var totalQuestions int
	if err := tx.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM course_question_content WHERE question_id = ?`, quizID,
	).Scan(&totalQuestions).Error; err != nil {
		return err
	}
	if totalQuestions == 0 {
		return nil
	}

	var answered int
	if err := tx.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM student_question_progress
		 WHERE user_id = ? AND question_id = ? AND COALESCE(user_option, '') <> ''`,
		userID, quizID,
	).Scan(&answered).Error; err != nil {
		return err
	}

	var isSubmit int
	_ = tx.WithContext(ctx).Raw(
		`SELECT COALESCE(is_submit, 0) FROM student_question_record WHERE user_id = ? AND question_id = ?`,
		userID, quizID,
	).Scan(&isSubmit)

	status := AttemptStatusInProgress
	if isSubmit == SubmitOn {
		status = AttemptStatusSubmitted
	} else if answered == 0 {
		return nil
	}

	if isSubmit == SubmitOn && answered < totalQuestions {
		logMigrationIssue(tx, "submit_incomplete",
			fmt.Sprintf("user:%d:quiz:%d", userID, quizID),
			fmt.Sprintf("submitted with %d/%d answers", answered, totalQuestions))
	}

	var score *int
	if status == AttemptStatusSubmitted {
		var computed int
		if err := tx.WithContext(ctx).Raw(
			`SELECT CAST(ROUND(100.0 * SUM(CASE WHEN p.user_option = qc.correct_answer AND COALESCE(p.user_option, '') <> '' THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0)) AS SIGNED)
			 FROM course_question_content qc
			 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
			 WHERE qc.question_id = ?`,
			userID, quizID,
		).Scan(&computed).Error; err != nil {
			return err
		}
		score = &computed
	}

	now := time.Now().UTC()
	var attemptID int64
	found := tx.WithContext(ctx).Raw(
		`SELECT id FROM quiz_attempt WHERE quiz_id = ? AND student_id = ? AND attempt_no = 1`,
		quizID, userID,
	).Scan(&attemptID)
	if found.Error != nil {
		return found.Error
	}
	if found.RowsAffected == 0 {
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO quiz_attempt
			 (quiz_id, student_id, attempt_no, status, total_questions, answered_questions, score, started_at, submitted_at, created_at, updated_at)
			 VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?)`,
			quizID, userID, status, totalQuestions, answered, score,
			nullTimePtr(answered > 0, now), nullTimePtr(status == AttemptStatusSubmitted, now), now, now,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Raw(
			`SELECT id FROM quiz_attempt WHERE quiz_id = ? AND student_id = ? AND attempt_no = 1`,
			quizID, userID,
		).Scan(&attemptID).Error; err != nil {
			return err
		}
	} else if err := tx.WithContext(ctx).Exec(
		`UPDATE quiz_attempt
		 SET status = ?, total_questions = ?, answered_questions = ?, score = ?, updated_at = ?,
		     submitted_at = CASE WHEN ? = ? THEN COALESCE(submitted_at, ?) ELSE submitted_at END
		 WHERE id = ?`,
		status, totalQuestions, answered, score, now, status, AttemptStatusSubmitted, now, attemptID,
	).Error; err != nil {
		return err
	}

	var progressRows []struct {
		ContentID  int64
		UserOption string
		Correct    string
	}
	if err := tx.WithContext(ctx).Raw(
		`SELECT p.content_id, COALESCE(p.user_option, '') AS user_option, qc.correct_answer
		 FROM student_question_progress p
		 JOIN course_question_content qc ON qc.id = p.content_id
		 WHERE p.user_id = ? AND p.question_id = ? AND COALESCE(p.user_option, '') <> ''`,
		userID, quizID,
	).Scan(&progressRows).Error; err != nil {
		return err
	}
	for _, row := range progressRows {
		correctFlag := 0
		if row.UserOption == row.Correct {
			correctFlag = 1
		}
		perScore := 0
		if status == AttemptStatusSubmitted && correctFlag == 1 && totalQuestions > 0 {
			perScore = 100 / totalQuestions
		}
		isCorrect := correctFlag
		updated := tx.WithContext(ctx).Exec(
			`UPDATE quiz_answer SET answer = ?, is_correct = ?, score = ?, updated_at = ?
			 WHERE attempt_id = ? AND question_id = ?`,
			row.UserOption, isCorrect, nullableScore(status == AttemptStatusSubmitted, perScore, correctFlag == 1), now, attemptID, row.ContentID,
		)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected > 0 {
			continue
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO quiz_answer (attempt_id, question_id, answer, is_correct, score, answered_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			attemptID, row.ContentID, row.UserOption, isCorrect,
			nullableScore(status == AttemptStatusSubmitted, perScore, correctFlag == 1), now, now,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func nullTimePtr(ok bool, t time.Time) *time.Time {
	if !ok {
		return nil
	}
	return &t
}

func nullableScore(submitted bool, perQuestion int, correct bool) *int {
	if !submitted {
		return nil
	}
	if !correct {
		zero := 0
		return &zero
	}
	return &perQuestion
}
