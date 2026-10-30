// account 业务逻辑：登录、读取当前用户、修改个人信息、修改密码。不依赖 HTTP。

package account

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
)

type Service struct {
	database    *gorm.DB
	tokenIssuer *identity.TokenIssuer
	cache       *access.Cache
}

// NewService 全局只初始化一次：启动时在路由表里创建。
func NewService(database *gorm.DB, tokenIssuer *identity.TokenIssuer, cache *access.Cache) *Service {
	return &Service{database: database, tokenIssuer: tokenIssuer, cache: cache}
}

type LoginInput struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=72"`
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// errWrongCredentials 用户不存在和密码错误返回同一句话，不让人借此探测用户名是否存在。
var errWrongCredentials = errs.Unauthorized("incorrect username or password")

// Login 校验用户名密码，签发 token。
func (service *Service) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	var user model.User
	result := service.database.WithContext(ctx).Raw("SELECT * FROM sys_user WHERE username = ?", username).Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		identity.CompareWithDummyHash(input.Password) // 耗时和密码错误时一致
		return nil, errWrongCredentials
	}
	if !identity.CheckPassword(user.PasswordHash, input.Password) {
		return nil, errWrongCredentials
	}
	if user.Status != model.UserStatusActive {
		return nil, errs.Unauthorized("account is disabled")
	}

	token, expiresAt, err := service.tokenIssuer.Issue(user.ID, user.TokenVersion)
	if err != nil {
		return nil, err
	}
	logs.CtxInfo(ctx, "login succeeded: user_id=%d username=%s", user.ID, user.Username)
	return &LoginResult{Token: token, ExpiresAt: expiresAt}, nil
}

// Logout token 是无状态的，后端只记一条日志，前端丢弃 token 即可。
func (service *Service) Logout(ctx context.Context, principal *access.Principal) {
	logs.CtxInfo(ctx, "logout: user_id=%d", principal.UserID)
}

// CurrentUserResult 当前用户的资料和权限，前端据此决定显示哪些菜单和按钮。
type CurrentUserResult struct {
	ID          int64    `json:"id"`
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Email       string   `json:"email"`
	Major       string   `json:"major"`
	Role        string   `json:"role"`
	RoleName    string   `json:"roleName"`
	IsAdmin     bool     `json:"isAdmin"`
	Permissions []string `json:"permissions"` // 管理员为空数组，前端按 isAdmin 全部放行
}

// GetCurrentUser 资料从数据库读最新的，权限用认证时已经拿到的快照。
func (service *Service) GetCurrentUser(ctx context.Context, principal *access.Principal) (*CurrentUserResult, error) {
	user, err := service.getUserByID(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}
	return &CurrentUserResult{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Email:       user.Email,
		Major:       user.Major,
		Role:        user.Role,
		RoleName:    access.GetRoleName(user.Role),
		IsAdmin:     principal.IsAdmin(),
		Permissions: principal.Permissions,
	}, nil
}

type UpdateProfileInput struct {
	Nickname string `json:"nickname" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
}

// UpdateProfile 只能改自己的昵称和邮箱。
func (service *Service) UpdateProfile(ctx context.Context, principal *access.Principal, input UpdateProfileInput) error {
	return service.database.WithContext(ctx).Exec(
		"UPDATE sys_user SET nickname = ?, email = ?, updated_at = ? WHERE id = ?",
		strings.TrimSpace(input.Nickname), strings.TrimSpace(input.Email), time.Now().UTC(), principal.UserID,
	).Error
}

type ChangePasswordInput struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=72"`
}

// ChangePassword 校验原密码后修改，并让 token_version 加一：包括当前设备在内的所有旧 token 立即失效，需要重新登录。
// 原密码错误返回 400 而不是 401，因为前端收到 401 会直接退出登录。
func (service *Service) ChangePassword(ctx context.Context, principal *access.Principal, input ChangePasswordInput) error {
	user, err := service.getUserByID(ctx, principal.UserID)
	if err != nil {
		return err
	}
	if !identity.CheckPassword(user.PasswordHash, input.OldPassword) {
		return errs.BadRequest("current password is incorrect")
	}
	if input.NewPassword == input.OldPassword {
		return errs.BadRequest("new password must be different from the current password")
	}
	hash, err := identity.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	err = service.database.WithContext(ctx).Exec(
		"UPDATE sys_user SET password_hash = ?, token_version = token_version + 1, updated_at = ? WHERE id = ?",
		hash, time.Now().UTC(), user.ID,
	).Error
	if err != nil {
		return err
	}
	service.cache.Reset() // 让新的 token_version 在下一个请求立即生效
	logs.CtxInfo(ctx, "password changed: user_id=%d", user.ID)
	return nil
}

func (service *Service) getUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	result := service.database.WithContext(ctx).Raw("SELECT * FROM sys_user WHERE id = ?", userID).Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.Unauthorized("user does not exist or has been deleted")
	}
	return &user, nil
}
