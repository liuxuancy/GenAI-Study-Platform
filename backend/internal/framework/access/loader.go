// DBLoader：权限缓存未命中时，从 sys_user 和 sys_role_permission 加载用户的权限快照。

package access

import (
	"context"

	"gorm.io/gorm"

	"caveclass/internal/framework/model"
	"caveclass/internal/platform/errs"
)

// DBLoader 是 PrincipalLoader 的正式实现。
type DBLoader struct {
	database *gorm.DB
}

var _ PrincipalLoader = (*DBLoader)(nil)

// NewDBLoader 全局只初始化一次：启动时创建，交给 NewCache 使用。
func NewDBLoader(database *gorm.DB) *DBLoader {
	return &DBLoader{database: database}
}

// LoadPrincipal 查出用户的状态、token 版本和角色持有的权限码。用户不存在时返回 401。
func (loader *DBLoader) LoadPrincipal(ctx context.Context, userID int64) (*Principal, error) {
	var user model.User
	result := loader.database.WithContext(ctx).Raw("SELECT * FROM sys_user WHERE id = ?", userID).Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.Unauthorized("user does not exist or has been deleted")
	}

	permissions := []string{}
	if user.Role != RoleAdmin { // 管理员不需要查授权
		// 查询当前角色(非管理员)所有权限码，放入permissions
		err := loader.database.WithContext(ctx).Raw(
			"SELECT permission_code FROM sys_role_permission WHERE role = ? ORDER BY permission_code",
			user.Role,
		).Scan(&permissions).Error
		if err != nil {
			return nil, err
		}
	}

	return &Principal{
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
		Active:       user.Status == model.UserStatusActive,
		TokenVersion: user.TokenVersion,
		Permissions:  permissions,
	}, nil
}
