// account 的 HTTP handler：解析参数、取当前用户、调用 service。路由登记在 internal/router/routes.go。

package account

import (
	"github.com/gin-gonic/gin"

	"caveclass/internal/framework/access"
	"caveclass/internal/platform/resp"
)

type Handler struct {
	service *Service
}

// NewHandler 全局只初始化一次：启动时在路由表里创建。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login POST /api/account/login（白名单，不需要登录）
func (handler *Handler) Login(ctx *gin.Context) (any, error) {
	var input LoginInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return handler.service.Login(ctx.Request.Context(), input)
}

// Logout POST /api/account/logout
func (handler *Handler) Logout(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	handler.service.Logout(ctx.Request.Context(), principal)
	return nil, nil
}

// GetCurrentUser GET /api/account/me：返回当前登录用户的资料和权限码
func (handler *Handler) GetCurrentUser(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return handler.service.GetCurrentUser(ctx.Request.Context(), principal)
}

// UpdateProfile PUT /api/account/profile
func (handler *Handler) UpdateProfile(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input UpdateProfileInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return nil, handler.service.UpdateProfile(ctx.Request.Context(), principal, input)
}

// ChangePassword PUT /api/account/password
func (handler *Handler) ChangePassword(ctx *gin.Context) (any, error) {
	principal, err := access.GetCurrentPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var input ChangePasswordInput
	if err := resp.BindJSON(ctx, &input); err != nil {
		return nil, err
	}
	return nil, handler.service.ChangePassword(ctx.Request.Context(), principal, input)
}
