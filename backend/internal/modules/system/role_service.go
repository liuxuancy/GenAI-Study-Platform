// 角色权限的业务逻辑：查看 3 个固定角色的授权、整体替换某个角色的授权、返回权限码目录。

package system

import (
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/errs"
	"caveclass/internal/platform/logs"
)

type RoleService struct {
	database *gorm.DB
	cache    *access.Cache
	catalog  *access.Catalog
}

// NewRoleService 全局只初始化一次：启动时在路由表里创建。
func NewRoleService(database *gorm.DB, cache *access.Cache, catalog *access.Catalog) *RoleService {
	return &RoleService{database: database, cache: cache, catalog: catalog}
}

// RoleItem 一个角色及其授权。
type RoleItem struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	Editable         bool     `json:"editable"`         // ROLE_ADMIN 在代码里直接放行，不能配置
	Permissions      []string `json:"permissions"`      // 当前有效的授权
	StalePermissions []string `json:"stalePermissions"` // 已失效：代码里已经删掉的权限码，不再匹配任何接口，可以清理
}

type rolePermissionRow struct {
	Role           string
	PermissionCode string
}

// GetRoleList 返回 3 个固定角色和各自持有的权限码。
func (service *RoleService) GetRoleList(ctx context.Context) ([]RoleItem, error) {
	// 角色权限不多，全部取出再分组
	var rows []rolePermissionRow
	err := service.database.WithContext(ctx).Raw(
		"SELECT role, permission_code FROM sys_role_permission ORDER BY role, permission_code",
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var items []RoleItem
	for _, role := range access.GetRoles() {
		item := RoleItem{
			Code:             role.Code,
			Name:             role.Name,
			Editable:         role.Code != access.RoleAdmin,
			Permissions:      []string{}, // 有效权限
			StalePermissions: []string{}, // 失效权限
		}
		// 认领自己角色的权限
		for _, row := range rows {
			if row.Role != role.Code {
				// 不是自己角色的权限
				continue
			}
			if service.catalog.ValidateGrant(row.PermissionCode) == nil {
				item.Permissions = append(item.Permissions, row.PermissionCode)
			} else {
				item.StalePermissions = append(item.StalePermissions, row.PermissionCode)
			}
		}
		items = append(items, item)
	}
	return items, nil
}

type AssignPermissionsInput struct {
	Permissions *[]string `json:"permissions" binding:"required"` // 用指针才能区分"没传"和"传了空数组（清空授权）"
}

// AssignRolePermissions 用提交的权限码整体替换角色的授权，提交后立即对该角色的所有用户生效。
func (service *RoleService) AssignRolePermissions(ctx context.Context, roleCode string, input AssignPermissionsInput) ([]string, error) {
	if !access.IsValidRole(roleCode) {
		return nil, errs.NotFound(fmt.Sprintf("角色不存在：%s", roleCode))
	}
	if roleCode == access.RoleAdmin {
		return nil, errs.BadRequest("管理员在代码里直接放行，不能配置权限")
	}

	codes := []string{}
	for _, code := range *input.Permissions {
		if err := service.catalog.ValidateGrant(code); err != nil {
			return nil, err
		}
		if !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}
	slices.Sort(codes)

	replacement := &permissionReplacement{role: roleCode, codes: codes}
	if err := service.database.WithContext(ctx).Transaction(replacement.run); err != nil {
		return nil, err
	}
	service.cache.Reset() // 授权变了，该角色所有用户的下一个请求立即按新授权判断
	logs.CtxInfo(ctx, "role permissions replaced: role=%s permissions=%v", roleCode, codes)
	return codes, nil
}

// permissionReplacement 在一个事务里先删掉角色的全部授权，再写入新的一组，不会出现中间状态。
type permissionReplacement struct {
	role  string
	codes []string
}

func (replacement *permissionReplacement) run(tx *gorm.DB) error {
	if err := tx.Exec("DELETE FROM sys_role_permission WHERE role = ?", replacement.role).Error; err != nil {
		return err
	}
	for _, code := range replacement.codes {
		err := tx.Exec("INSERT INTO sys_role_permission (role, permission_code) VALUES (?, ?)", replacement.role, code).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPermissionCatalog 返回所有模块声明的权限码，供角色权限页展示勾选树。
func (service *RoleService) GetPermissionCatalog() []access.PermissionDef {
	return service.catalog.GetAllDefinitions()
}
