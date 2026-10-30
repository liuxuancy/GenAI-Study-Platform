// system 的 HTTP handler：解析参数、取当前用户、调用 service。路由登记在 internal/router/routes.go。

package system

import (
	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/resp"
)

type Handler struct {
	userService  *UserService
	roleService  *RoleService
	usageService *UsageService
}

// NewHandler 全局只初始化一次：启动时在路由表里创建。
func NewHandler(userService *UserService, roleService *RoleService, usageService *UsageService) *Handler {
	return &Handler{userService: userService, roleService: roleService, usageService: usageService}
}

// GetUserList GET /api/system/users?keyword=&role=&status=&page=&pageSize=
func (handler *Handler) GetUserList(ctx *gin.Context) (any, error) {
	var query UserListQuery
	if err := resp.BindQuery(ctx, &query); err != nil {
		return nil, err
	}
	return handler.userService.GetUserList(ctx.Request.Context(), query)
}

// CreateUser POST /api/system/users
func (handler *Handler) CreateUser(ctx *gin.Context) (any, error) {
	var input CreateUserInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.userService.CreateUser(ctx.Request.Context(), input)
}

// UpdateUser PUT /api/system/users/:id
func (handler *Handler) UpdateUser(ctx *gin.Context) (any, error) {
	operator, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input UpdateUserInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return nil, handler.userService.UpdateUser(ctx.Request.Context(), operator, userID, input)
}

// ResetUserPassword PUT /api/system/users/:id/password
func (handler *Handler) ResetUserPassword(ctx *gin.Context) (any, error) {
	userID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	var input ResetPasswordInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return nil, handler.userService.ResetUserPassword(ctx.Request.Context(), userID, input)
}

// DeleteUser DELETE /api/system/users/:id
func (handler *Handler) DeleteUser(ctx *gin.Context) (any, error) {
	operator, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := resp.GetInt64PathParam(ctx, "id")
	if err != nil {
		return nil, err
	}
	return nil, handler.userService.DeleteUser(ctx.Request.Context(), operator, userID)
}

// GetRoleList GET /api/system/roles
func (handler *Handler) GetRoleList(ctx *gin.Context) (any, error) {
	return handler.roleService.GetRoleList(ctx.Request.Context())
}

// AssignRolePermissions PUT /api/system/roles/:code/permissions
func (handler *Handler) AssignRolePermissions(ctx *gin.Context) (any, error) {
	var input AssignPermissionsInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.roleService.AssignRolePermissions(ctx.Request.Context(), ctx.Param("code"), input)
}

// GetPermissionCatalog GET /api/system/permissions
func (handler *Handler) GetPermissionCatalog(*gin.Context) (any, error) {
	return handler.roleService.GetPermissionCatalog(), nil
}

// GetUsageReport GET /api/system/usage
func (handler *Handler) GetUsageReport(ctx *gin.Context) (any, error) {
	return handler.usageService.GetUsageReport(ctx.Request.Context())
}
