package course

import (
	"context"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/platform/errs"
)

const (
	affectionBackfillKind = "affection_backfill"
	affectionWeightsKind  = "affection_weights_80_20"
)

func affectionBase(coursePercent, quizPercent int) int {
	return (coursePercent*80 + quizPercent*20) / 100
}

func clampAffection(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

type storedAffection struct {
	Percent   int
	Affection int
	Exists    bool
}

func (service *Service) loadAffection(ctx context.Context, userID, courseID int64) (storedAffection, error) {
	return loadAffection(service.database.WithContext(ctx), userID, courseID)
}

func loadAffection(database *gorm.DB, userID, courseID int64) (storedAffection, error) {
	var row storedAffection
	result := database.Raw(
		`SELECT percent, affection FROM student_course_progress WHERE user_id = ? AND course_id = ?`,
		userID, courseID,
	).Scan(&row)
	if result.Error != nil {
		return storedAffection{}, result.Error
	}
	row.Exists = result.RowsAffected > 0
	return row, nil
}

func (service *Service) quizPercent(ctx context.Context, userID, courseID int64) (int, error) {
	return quizPercent(service.database.WithContext(ctx), userID, courseID)
}

func quizPercent(database *gorm.DB, userID, courseID int64) (int, error) {
	var row struct {
		Total    int
		Answered int
	}
	err := database.Raw(
		`SELECT COUNT(qc.id) AS total,
		        COALESCE(SUM(CASE WHEN COALESCE(p.user_option, '') <> '' THEN 1 ELSE 0 END), 0) AS answered
		 FROM course_question q
		 JOIN course_question_content qc ON qc.question_id = q.id
		 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
		 WHERE q.course_id = ? AND `+studentVisibleQuizFilterSQL,
		userID, courseID, userID,
	).Scan(&row).Error
	if err != nil {
		return 0, err
	}
	return answerPercent(row.Answered, row.Total), nil
}

func (service *Service) quizPercentsForCourses(ctx context.Context, userID int64, courseIDs []int64) (map[int64]int, error) {
	percents := map[int64]int{}
	if len(courseIDs) == 0 {
		return percents, nil
	}
	placeholders := ""
	args := make([]any, 0, len(courseIDs)+2)
	args = append(args, userID)
	for index, id := range courseIDs {
		if index > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}
	args = append(args, userID)
	var rows []struct {
		CourseID int64
		Total    int
		Answered int
	}
	err := service.database.WithContext(ctx).Raw(
		`SELECT q.course_id AS course_id, COUNT(qc.id) AS total,
		        COALESCE(SUM(CASE WHEN COALESCE(p.user_option, '') <> '' THEN 1 ELSE 0 END), 0) AS answered
		 FROM course_question q
		 JOIN course_question_content qc ON qc.question_id = q.id
		 LEFT JOIN student_question_progress p ON p.content_id = qc.id AND p.user_id = ?
		 WHERE q.course_id IN (`+placeholders+`) AND `+studentVisibleQuizFilterSQL+`
		 GROUP BY q.course_id`,
		args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		percents[row.CourseID] = answerPercent(row.Answered, row.Total)
	}
	return percents, nil
}

// studentAffection 有进度行时返回已保存的好感度；没有行时按当前基础值计算，不插入。
func (service *Service) studentAffection(ctx context.Context, userID, courseID int64) (int, error) {
	row, err := service.loadAffection(ctx, userID, courseID)
	if err != nil {
		return 0, err
	}
	if row.Exists {
		return row.Affection, nil
	}
	quiz, err := service.quizPercent(ctx, userID, courseID)
	if err != nil {
		return 0, err
	}
	return affectionBase(0, quiz), nil
}

func (service *Service) insertOwnerProgress(ctx context.Context, userID, courseID int64) error {
	return service.database.WithContext(ctx).Exec(
		`INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 0, 0)`,
		userID, courseID,
	).Error
}

// syncAffectionOnCoursePercent 在课程进度变化时保留已经发生的行为加减。
func (service *Service) syncAffectionOnCoursePercent(ctx context.Context, userID, courseID int64, newPercent int) error {
	return service.withAffectionAchievements(ctx, userID, func(txService *Service) error {
		return txService.updateAffectionOnCoursePercent(ctx, userID, courseID, newPercent)
	})
}

func (service *Service) updateAffectionOnCoursePercent(ctx context.Context, userID, courseID int64, newPercent int) error {
	row, err := service.loadAffection(ctx, userID, courseID)
	if err != nil {
		return err
	}
	quiz, err := service.quizPercent(ctx, userID, courseID)
	if err != nil {
		return err
	}
	newBase := affectionBase(newPercent, quiz)
	if !row.Exists {
		return service.database.WithContext(ctx).Exec(
			`INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, ?, ?)`,
			userID, courseID, newPercent, newBase,
		).Error
	}
	affection := clampAffection(row.Affection - affectionBase(row.Percent, quiz) + newBase)
	return service.database.WithContext(ctx).Exec(
		`UPDATE student_course_progress SET percent = ?, affection = ? WHERE user_id = ? AND course_id = ?`,
		newPercent, affection, userID, courseID,
	).Error
}

// updateAffectionOnQuizChange 在习题作答变化时保留已经发生的行为加减。没有进度行时按新基础值插入。
func (service *Service) updateAffectionOnQuizChange(ctx context.Context, userID, courseID int64, oldQuiz, newQuiz int) error {
	row, err := service.loadAffection(ctx, userID, courseID)
	if err != nil {
		return err
	}
	coursePercent := 0
	if row.Exists {
		coursePercent = row.Percent
	}
	newBase := affectionBase(coursePercent, newQuiz)
	if !row.Exists {
		return service.database.WithContext(ctx).Exec(
			`INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 0, ?)`,
			userID, courseID, newBase,
		).Error
	}
	affection := clampAffection(row.Affection - affectionBase(coursePercent, oldQuiz) + newBase)
	return service.database.WithContext(ctx).Exec(
		`UPDATE student_course_progress SET affection = ? WHERE user_id = ? AND course_id = ?`,
		affection, userID, courseID,
	).Error
}

// AdjustAffection 给后续行为加减好感度。这次没有调用方。结果夹在 0 到 100。
func (service *Service) AdjustAffection(ctx context.Context, userID, courseID int64, delta int) error {
	return service.withAffectionAchievements(ctx, userID, func(txService *Service) error {
		return txService.adjustAffection(ctx, userID, courseID, delta)
	})
}

func (service *Service) adjustAffection(ctx context.Context, userID, courseID int64, delta int) error {
	var courseExists int64
	found := service.database.WithContext(ctx).Raw(`SELECT COUNT(*) FROM course WHERE id = ?`, courseID).Scan(&courseExists)
	if found.Error != nil {
		return found.Error
	}
	if courseExists == 0 {
		return errs.NotFound("course not found")
	}
	row, err := service.loadAffection(ctx, userID, courseID)
	if err != nil {
		return err
	}
	quiz, err := service.quizPercent(ctx, userID, courseID)
	if err != nil {
		return err
	}
	coursePercent := 0
	if row.Exists {
		coursePercent = row.Percent
	}
	base := affectionBase(coursePercent, quiz)
	if !row.Exists {
		return service.database.WithContext(ctx).Exec(
			`INSERT INTO student_course_progress (user_id, course_id, percent, affection) VALUES (?, ?, 0, ?)`,
			userID, courseID, clampAffection(base+delta),
		).Error
	}
	return service.database.WithContext(ctx).Exec(
		`UPDATE student_course_progress SET affection = ? WHERE user_id = ? AND course_id = ?`,
		clampAffection(row.Affection+delta), userID, courseID,
	).Error
}

func backfillAffection(ctx context.Context, database *gorm.DB) error {
	var count int64
	if err := database.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM migration_log WHERE kind = ?`, affectionBackfillKind,
	).Scan(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}
	var rows []struct {
		UserID   int64
		CourseID int64
		Percent  int
	}
	if err := database.WithContext(ctx).Raw(
		`SELECT user_id, course_id, percent FROM student_course_progress`,
	).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		quiz, err := quizPercent(database.WithContext(ctx), row.UserID, row.CourseID)
		if err != nil {
			return err
		}
		if err := database.WithContext(ctx).Exec(
			`UPDATE student_course_progress SET affection = ? WHERE user_id = ? AND course_id = ?`,
			affectionBase(row.Percent, quiz), row.UserID, row.CourseID,
		).Error; err != nil {
			return err
		}
	}
	return database.WithContext(ctx).Exec(
		`INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, '', 'backfill affection from course and quiz progress', ?)`,
		affectionBackfillKind, time.Now().UTC(),
	).Error
}

func migrateAffectionWeights(ctx context.Context, database *gorm.DB) error {
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var applied int64
		if err := tx.Raw(`SELECT COUNT(*) FROM migration_log WHERE kind = ?`, affectionWeightsKind).Scan(&applied).Error; err != nil {
			return err
		}
		if applied > 0 {
			return nil
		}
		var backfilled int64
		if err := tx.Raw(`SELECT COUNT(*) FROM migration_log WHERE kind = ?`, affectionBackfillKind).Scan(&backfilled).Error; err != nil {
			return err
		}
		if backfilled > 0 {
			var rows []struct {
				UserID    int64
				CourseID  int64
				Percent   int
				Affection int
			}
			if err := tx.Raw(`SELECT user_id, course_id, percent, affection FROM student_course_progress`).Scan(&rows).Error; err != nil {
				return err
			}
			for _, row := range rows {
				quiz, err := quizPercent(tx, row.UserID, row.CourseID)
				if err != nil {
					return err
				}
				oldBase := (row.Percent + quiz) / 2
				updated := clampAffection(row.Affection - oldBase + affectionBase(row.Percent, quiz))
				if err := tx.Exec(`UPDATE student_course_progress SET affection = ? WHERE user_id = ? AND course_id = ?`,
					updated, row.UserID, row.CourseID).Error; err != nil {
					return err
				}
			}
		}
		return tx.Exec(
			`INSERT INTO migration_log (kind, ref_key, detail, created_at) VALUES (?, '', 'adjust affection weights to 80/20', ?)`,
			affectionWeightsKind, time.Now().UTC(),
		).Error
	})
}
