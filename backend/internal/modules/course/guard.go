package course

import (
	"context"

	"gorm.io/gorm"

	"caveclass/internal/platform/errs"
)

// CheckUserDeletion 用户还在班级里时不能删。先移出班级再删账号。
func (Module) CheckUserDeletion(ctx context.Context, database *gorm.DB, userID int64) error {
	var count int64
	err := database.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM study_class_member WHERE user_id = ?",
		userID,
	).Scan(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errs.Conflict("user is still in a class and cannot be deleted")
	}
	return nil
}
