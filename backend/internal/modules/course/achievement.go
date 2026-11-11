package course

import (
	"context"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

const (
	achievementAffection50  = "affection_50"
	achievementAffection100 = "affection_100"
	achievementFive100      = "affection_five_100"
	achievementFirstAvatar  = "first_pixel_avatar"
)

type Achievement struct {
	Key        string    `json:"key"`
	UnlockedAt time.Time `json:"unlockedAt"`
}

type AchievementSync struct {
	Unlocked []Achievement `json:"unlocked"`
	Notify   []Achievement `json:"notify"`
}

func (service *Service) withAffectionAchievements(ctx context.Context, userID int64, update func(*Service) error) error {
	return service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txService := *service
		txService.database = tx
		if err := update(&txService); err != nil {
			return err
		}
		return txService.unlockAchievements(ctx, userID)
	})
}

func (service *Service) unlockAchievements(ctx context.Context, userID int64) error {
	var hasAvatar bool
	if err := service.database.WithContext(ctx).Raw(
		`SELECT EXISTS (
			SELECT 1 FROM course c
			JOIN course_pixel_avatar a ON a.course_id = c.id
			JOIN student_course_progress p ON p.course_id = c.id AND p.user_id = c.owner_id
			WHERE c.owner_id = ?
		)`, userID,
	).Scan(&hasAvatar).Error; err != nil {
		return err
	}
	var courses []struct {
		ID          int64
		HasProgress bool
		Affection   int
	}
	if err := service.database.WithContext(ctx).Raw(
		`SELECT c.id, p.user_id IS NOT NULL AS has_progress, COALESCE(p.affection, 0) AS affection
		 FROM course c LEFT JOIN student_course_progress p ON p.course_id = c.id AND p.user_id = ?
		 WHERE `+studentVisibleCourseSQL("c"),
		append([]any{userID}, studentVisibleCourseArgs(userID)...)...,
	).Scan(&courses).Error; err != nil {
		return err
	}
	missing := make([]int64, 0)
	for _, course := range courses {
		if !course.HasProgress {
			missing = append(missing, course.ID)
		}
	}
	quizPercents, err := service.quizPercentsForCourses(ctx, userID, missing)
	if err != nil {
		return err
	}
	count50, count100 := 0, 0
	for _, course := range courses {
		affection := course.Affection
		if !course.HasProgress {
			affection = affectionBase(0, quizPercents[course.ID])
		}
		if affection >= 50 {
			count50++
		}
		if affection >= 100 {
			count100++
		}
	}
	keys := make([]string, 0, 4)
	if hasAvatar {
		keys = append(keys, achievementFirstAvatar)
	}
	if count50 >= 1 {
		keys = append(keys, achievementAffection50)
	}
	if count100 >= 1 {
		keys = append(keys, achievementAffection100)
	}
	if count100 >= 5 {
		keys = append(keys, achievementFive100)
	}
	for _, key := range keys {
		if err := service.recordAchievement(ctx, userID, key); err != nil {
			return err
		}
	}
	return nil
}

func (service *Service) recordAchievement(ctx context.Context, userID int64, key string) error {
	return service.database.WithContext(ctx).Exec(
		`INSERT INTO student_achievement (user_id, achievement, unlocked_at)
		 VALUES (?, ?, ?) ON CONFLICT (user_id, achievement) DO NOTHING`,
		userID, key, time.Now().UTC(),
	).Error
}

// SyncAchievements also reconciles progress created before achievements existed.
// Claiming pending notifications in the same transaction prevents duplicate delivery.
func (service *Service) SyncAchievements(ctx context.Context, principal *access.Principal) (*AchievementSync, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can view achievements")
	}
	result := &AchievementSync{Unlocked: []Achievement{}, Notify: []Achievement{}}
	err := service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txService := *service
		txService.database = tx
		if err := txService.unlockAchievements(ctx, principal.UserID); err != nil {
			return err
		}
		var rows []StudentAchievement
		if err := tx.Where("user_id = ?", principal.UserID).Order("unlocked_at, achievement").Find(&rows).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, row := range rows {
			item := Achievement{Key: row.Achievement, UnlockedAt: row.UnlockedAt}
			result.Unlocked = append(result.Unlocked, item)
			if row.NotifiedAt == nil {
				result.Notify = append(result.Notify, item)
				if err := tx.Model(&StudentAchievement{}).
					Where("user_id = ? AND achievement = ? AND notified_at IS NULL", principal.UserID, row.Achievement).
					Update("notified_at", now).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
