package course

import (
	"context"
	"time"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
)

// CreateCourseInput 学生或教师新建一门还没放出的课。
type CreateCourseInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
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

// CreateStudentCourse 学生创建私有课。不写 course_class。
func (service *Service) CreateStudentCourse(ctx context.Context, principal *access.Principal, input CreateCourseInput) (*CourseListItem, error) {
	if principal.Role != access.RoleStudent {
		return nil, errs.Forbidden("only students can create courses")
	}
	return service.createCourse(ctx, principal.UserID, input)
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
