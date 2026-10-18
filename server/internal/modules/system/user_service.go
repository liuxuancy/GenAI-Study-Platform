// 用户管理的业务逻辑：列表、新建、编辑、启用禁用、重置密码、删除。不依赖 HTTP。

package system

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/framework/identity"
	"caveclass/internal/framework/model"
	"caveclass/internal/framework/module"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// 用户名：3 到 64 位小写字母、数字、下划线
var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,64}$`)

type UserService struct {
	database       *gorm.DB
	cache          *access.Cache
	deletionGuards []module.UserDeletionGuard
}

// NewUserService 全局只初始化一次：启动时在路由表里创建。
func NewUserService(database *gorm.DB, cache *access.Cache, deletionGuards []module.UserDeletionGuard) *UserService {
	return &UserService{database: database, cache: cache, deletionGuards: deletionGuards}
}

// UserItem 返回给前端的用户信息，不含密码哈希和 token 版本。
type UserItem struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	RoleName  string    `json:"roleName" gorm:"-"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserListQuery struct {
	Keyword  string `form:"keyword" binding:"max=64"`
	Role     string `form:"role"`
	Status   *int   `form:"status" binding:"omitempty,oneof=0 1"` // 不传表示不按状态筛选
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

type UserListResult struct {
	Items []UserItem `json:"items"`
	Total int64      `json:"total"`
}

// 用户列表的两条 SQL 都完整写死，不在代码里拼接。
// 每个筛选条件写成"(参数为空 OR 条件)"：没传这个参数时条件恒为真，相当于不筛选。
// 两条 SQL 的筛选参数相同，由 getUserListFilterArgs 按顺序给出。
const countUsersSQL = `
	SELECT COUNT(*)
	FROM sys_user
	WHERE (? = '' OR username LIKE ? ESCAPE '\' OR nickname LIKE ? ESCAPE '\')
	  AND (? = '' OR role = ?)
	  AND (? = -1 OR status = ?)`

const listUsersSQL = `
	SELECT id, username, nickname, email, role, status, created_at
	FROM sys_user
	WHERE (? = '' OR username LIKE ? ESCAPE '\' OR nickname LIKE ? ESCAPE '\')
	  AND (? = '' OR role = ?)
	  AND (? = -1 OR status = ?)
	ORDER BY id
	LIMIT ? OFFSET ?`

// 没传状态时用 -1 表示"不按状态筛选"
const noStatusFilter = -1

// GetUserList 按关键字（用户名或昵称）、角色、状态筛选，按 ID 排序分页。
func (service *UserService) GetUserList(ctx context.Context, query UserListQuery) (*UserListResult, error) {
	filterArgs := getUserListFilterArgs(query)

	var total int64
	if err := service.database.WithContext(ctx).Raw(countUsersSQL, filterArgs...).Scan(&total).Error; err != nil {
		return nil, err
	}

	page, pageSize := normalizePage(query.Page, query.PageSize)
	var items []UserItem
	pageArgs := append(filterArgs, pageSize, (page-1)*pageSize)
	if err := service.database.WithContext(ctx).Raw(listUsersSQL, pageArgs...).Scan(&items).Error; err != nil {
		return nil, err
	}
	for index := range items {
		items[index].RoleName = access.GetRoleName(items[index].Role)
	}
	return &UserListResult{Items: items, Total: total}, nil
}

type CreateUserInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Nickname string `json:"nickname" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Role     string `json:"role" binding:"required"` // 创建后不可修改
}

// CreateUser 新建用户。用户名统一转成小写，重复时返回 409。
func (service *UserService) CreateUser(ctx context.Context, input CreateUserInput) (*UserItem, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	if !usernamePattern.MatchString(username) {
		return nil, errs.BadRequest("用户名只能是 3 到 64 位小写字母、数字或下划线")
	}
	if !access.IsValidRole(input.Role) {
		return nil, errs.BadRequest(fmt.Sprintf("未知角色：%s", input.Role))
	}

	var count int64
	if err := service.database.WithContext(ctx).Raw("SELECT COUNT(*) FROM sys_user WHERE username = ?", username).Scan(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errs.Conflict(fmt.Sprintf("用户名 %s 已存在", username))
	}

	hash, err := identity.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = service.database.WithContext(ctx).Exec(
		`INSERT INTO sys_user (username, password_hash, nickname, email, role, status, token_version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		username, hash, strings.TrimSpace(input.Nickname), strings.TrimSpace(input.Email),
		input.Role, model.UserStatusActive, now, now,
	).Error
	if err != nil {
		return nil, err
	}
	logs.CtxInfo(ctx, "user created: username=%s role=%s", username, input.Role)
	return service.getUserItemByUsername(ctx, username)
}

type UpdateUserInput struct {
	Nickname string `json:"nickname" binding:"max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Status   *int   `json:"status" binding:"required,oneof=0 1"` // 用指针才能区分"没传"和"传了 0（禁用）"
}

// UpdateUser 修改昵称、邮箱、状态。不能禁用自己；从启用改为禁用时 token 版本加一，该用户立即下线。
func (service *UserService) UpdateUser(ctx context.Context, operator *access.Principal, userID int64, input UpdateUserInput) error {
	user, err := service.getUserByID(ctx, userID)
	if err != nil {
		return err
	}
	newStatus := *input.Status
	if userID == operator.UserID && newStatus == model.UserStatusDisabled {
		return errs.BadRequest("不能禁用自己")
	}

	tokenVersionIncrement := 0
	if user.Status == model.UserStatusActive && newStatus == model.UserStatusDisabled {
		tokenVersionIncrement = 1
	}
	err = service.database.WithContext(ctx).Exec(
		"UPDATE sys_user SET nickname = ?, email = ?, status = ?, token_version = token_version + ?, updated_at = ? WHERE id = ?",
		strings.TrimSpace(input.Nickname), strings.TrimSpace(input.Email), newStatus, tokenVersionIncrement,
		time.Now().UTC(), userID,
	).Error
	if err != nil {
		return err
	}
	if user.Status != newStatus {
		service.cache.Reset() // 状态变了，下一个请求立即按新状态判断
		logs.CtxInfo(ctx, "user status changed: user_id=%d %d -> %d", userID, user.Status, newStatus)
	}
	return nil
}

type ResetPasswordInput struct {
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// ResetUserPassword 管理员重置别人的密码：token 版本加一，该用户所有旧 token 立即失效。
func (service *UserService) ResetUserPassword(ctx context.Context, userID int64, input ResetPasswordInput) error {
	if _, err := service.getUserByID(ctx, userID); err != nil {
		return err
	}
	hash, err := identity.HashPassword(input.Password)
	if err != nil {
		return err
	}
	err = service.database.WithContext(ctx).Exec(
		"UPDATE sys_user SET password_hash = ?, token_version = token_version + 1, updated_at = ? WHERE id = ?",
		hash, time.Now().UTC(), userID,
	).Error
	if err != nil {
		return err
	}
	service.cache.Reset()
	logs.CtxInfo(ctx, "password reset by admin: user_id=%d", userID)
	return nil
}

// DeleteUser 不能删除自己；先让各模块检查是否还有数据引用这个用户，有就返回 409。
func (service *UserService) DeleteUser(ctx context.Context, operator *access.Principal, userID int64) error {
	if userID == operator.UserID {
		return errs.BadRequest("不能删除自己")
	}
	if _, err := service.getUserByID(ctx, userID); err != nil {
		return err
	}
	for _, guard := range service.deletionGuards {
		if err := guard.CheckUserDeletion(ctx, service.database.WithContext(ctx), userID); err != nil {
			return err
		}
	}
	if err := service.database.WithContext(ctx).Exec("DELETE FROM sys_user WHERE id = ?", userID).Error; err != nil {
		return err
	}
	service.cache.Reset()
	logs.CtxInfo(ctx, "user deleted: user_id=%d", userID)
	return nil
}

func (service *UserService) getUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	result := service.database.WithContext(ctx).Raw("SELECT * FROM sys_user WHERE id = ?", userID).Scan(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errs.NotFound("用户不存在")
	}
	return &user, nil
}

func (service *UserService) getUserItemByUsername(ctx context.Context, username string) (*UserItem, error) {
	var item UserItem
	err := service.database.WithContext(ctx).Raw(
		"SELECT id, username, nickname, email, role, status, created_at FROM sys_user WHERE username = ?", username,
	).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	item.RoleName = access.GetRoleName(item.Role)
	return &item, nil
}

// getUserListFilterArgs 按 countUsersSQL / listUsersSQL 里 ? 的顺序给出筛选参数：
// 关键字原文、关键字匹配模式 ×2、角色 ×2、状态 ×2。
func getUserListFilterArgs(query UserListQuery) []any {
	keyword := strings.TrimSpace(query.Keyword)
	pattern := "%" + escapeLikePattern(keyword) + "%"
	status := noStatusFilter
	if query.Status != nil {
		status = *query.Status
	}
	return []any{keyword, pattern, pattern, query.Role, query.Role, status, status}
}

// normalizePage 页码默认 1，每页默认 20 条、最多 100 条。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// escapeLikePattern 转义 LIKE 里的特殊字符，让用户输入的 % 和 _ 按普通字符匹配。
func escapeLikePattern(keyword string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(keyword)
}
