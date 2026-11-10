package course

import (
	"context"
	"errors"
	"fmt"
	"time"

	"caveclass/internal/ai"
	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
	"caveclass/internal/usage"

	"gorm.io/gorm"
)

// CreateCourseInput 学生或教师新建一门还没放出的课。
type CreateCourseInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
}

// SharingInput 学生切换同班公开。省略 shared 时拒绝，避免误把课收回。
type SharingInput struct {
	Shared *bool `json:"shared"`
}

// SharingResult 告诉页面现在是否同班可见，以及还有几位同班学生能看见。
type SharingResult struct {
	Shared         bool `json:"shared"`
	ClassmateCount int  `json:"classmateCount"`
}

// CreateStudentCourse 学生创建私有课，并生成该课唯一的像素小人。不写 course_class。
func (service *Service) CreateStudentCourse(ctx context.Context, principal *access.Principal, input CreateCourseInput) (*CourseListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can create courses")
	}
	return service.createStudentCourse(ctx, principal, input)
}

// CreateTeacherCourse 教师创建未发布的课。发布时再写 course_class。
func (service *Service) CreateTeacherCourse(ctx context.Context, principal *access.Principal, input CreateCourseInput) (*TeacherCourseDetail, error) {
	if principal.Role != access.RoleTeacher {
		return nil, errs.Forbidden("only teachers can create courses")
	}
	created, err := service.createCourse(ctx, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return service.GetTeacherCourse(ctx, principal, created.ID)
}

func (service *Service) createCourse(ctx context.Context, ownerID int64, input CreateCourseInput) (*CourseListItem, error) {
	title, description, err := normalizeCourseText(input.Title, input.Description)
	if err != nil {
		return nil, err
	}
	courseID, err := service.insertCourse(service.database.WithContext(ctx), ownerID, title, description)
	if err != nil {
		return nil, err
	}
	return &CourseListItem{
		ID:          courseID,
		Title:       title,
		CourseType:  CourseTypeStructured,
		Description: description,
		Percent:     0,
		Visibility:  VisibilityPrivate,
	}, nil
}

func (service *Service) createStudentCourse(ctx context.Context, principal *access.Principal, input CreateCourseInput) (_ *CourseListItem, resultErr error) {
	title, description, err := normalizeCourseText(input.Title, input.Description)
	if err != nil {
		return nil, err
	}
	aiService := service.ai
	if aiService == nil {
		return nil, errs.Internal(nil)
	}
	if input.Provider != "" {
		aiService, err = service.aiForGenerate(input.Provider)
		if err != nil {
			return nil, err
		}
	}
	var courseID int64
	err = service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var insertErr error
		courseID, insertErr = service.insertCourse(tx, principal.UserID, title, description)
		return insertErr
	})
	if err != nil {
		return nil, err
	}
	kept := false
	defer func() {
		if !kept {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if cleanupErr := service.deleteCreatedCourse(cleanupCtx, courseID); cleanupErr != nil {
				logs.CtxError(cleanupCtx, "cleanup failed student course %d: %v", courseID, cleanupErr)
				resultErr = errs.Internal(errors.Join(resultErr, fmt.Errorf("cleanup course %d: %w", courseID, cleanupErr)))
			}
		}
	}()
	var avatar *CourseAvatar
	for attempt := 0; attempt < pixelAvatarAttempts; attempt++ {
		used, err := service.loadUsedAppearances(ctx)
		if err != nil {
			return nil, err
		}
		traits, err := drawPixelTraits(used)
		if errors.Is(err, errPixelAppearancesExhausted) {
			return nil, errs.Conflict("pixel avatar appearances are exhausted")
		}
		if errors.Is(err, errPixelCandidatesExhausted) {
			return nil, errs.Conflict("could not find an unused pixel avatar appearance; retry creation")
		}
		if err != nil {
			return nil, err
		}
		generated, meta, callErr := aiService.GeneratePixelSprite(ctx, ai.PixelSpriteInput{
			CourseID:      courseID,
			Title:         title,
			RoleType:      traits.RoleType,
			BodyType:      traits.BodyType,
			HairColor:     traits.HairColor,
			HairStyle:     traits.HairStyle,
			FacialFeature: traits.FacialFeature,
			Accessory:     traits.Accessory,
			SilhouetteKey: traits.SilhouetteKey,
			Personality:   traits.Personality,
		})
		service.recordUsage(ctx, principal.UserID, usage.SceneGeneratePixelSprite, meta, callErr)
		if callErr != nil {
			ai.LogError(ctx, callErr, meta, principal.UserID)
			return nil, businessAIError(callErr)
		}
		err = service.insertPixelAvatar(ctx, courseID, traits, generated)
		if duplicateKey(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := service.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txService := *service
			txService.database = tx
			if err := txService.insertOwnerProgress(ctx, principal.UserID, courseID); err != nil {
				return err
			}
			return txService.recordAchievement(ctx, principal.UserID, achievementFirstAvatar)
		}); err != nil {
			return nil, err
		}
		avatar = avatarFromGenerated(traits, generated)
		kept = true
		break
	}
	if !kept {
		return nil, errs.Conflict("pixel avatar appearance already exists")
	}
	return &CourseListItem{
		ID:          courseID,
		Title:       title,
		CourseType:  CourseTypeStructured,
		Description: description,
		Percent:     0,
		Visibility:  VisibilityPrivate,
		Affection:   0,
		Avatar:      avatar,
	}, nil
}

// SetCourseSharing 学生把自己的课改成同班可见或收回。学生课不保留 course_class 行。
func (service *Service) SetCourseSharing(ctx context.Context, principal *access.Principal, courseID int64, shared bool) (*SharingResult, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can share courses")
	}
	database := service.database.WithContext(ctx)
	var ownerID int64
	found := database.Raw(
		"SELECT owner_id FROM course WHERE id = ? AND owner_id = ?",
		courseID, principal.UserID,
	).Scan(&ownerID)
	if found.Error != nil {
		return nil, found.Error
	}
	if found.RowsAffected == 0 {
		return nil, errs.NotFound("course not found")
	}

	publish := PublishOff
	if shared {
		publish = PublishOn
	}
	tx := database.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	updated := tx.Exec(
		"UPDATE course SET is_publish = ?, updated_at = ? WHERE id = ? AND owner_id = ?",
		publish, time.Now().UTC(), courseID, principal.UserID,
	)
	if updated.Error != nil {
		tx.Rollback()
		return nil, updated.Error
	}
	cleared := tx.Exec("DELETE FROM course_class WHERE course_id = ?", courseID)
	if cleared.Error != nil {
		tx.Rollback()
		return nil, cleared.Error
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	classmates, err := service.countClassmates(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}
	return &SharingResult{Shared: shared, ClassmateCount: classmates}, nil
}
