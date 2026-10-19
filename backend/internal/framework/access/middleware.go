// 路由级权限中间件 RequirePermission / RequireLogin，以及在请求里存取当前用户。

package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"caveclass/internal/platform/errs"
)

// 当前用户在 gin.Context 里的键，只在本包内使用
const principalContextKey = "caveclass.principal"

// SetPrincipal 由认证中间件在校验通过后调用，将查询到的用户信息存入ctx供后续复用
func SetPrincipal(ctx *gin.Context, principal *Principal) {
	ctx.Set(principalContextKey, principal)
}

// GetCurrentPrincipal 取出当前请求的用户。没有登录时返回 401 错误，handler 直接把错误返回即可。
func GetCurrentPrincipal(ctx *gin.Context) (*Principal, error) {
	value, exists := ctx.Get(principalContextKey)
	if !exists {
		return nil, errs.ErrUnauthorized
	}
	principal, ok := value.(*Principal)
	if !ok {
		return nil, errs.ErrUnauthorized
	}
	return principal, nil
}

// RequirePermission 接口要求持有 required 权限码，否则返回 403。
// 权限码写错（格式非法）会在启动注册路由时直接 panic，而不是等到请求时才发现。
func RequirePermission(required string) gin.HandlerFunc {
	if !ValidCode(required) {
		panic(fmt.Sprintf("RequirePermission: invalid permission code %q", required))
	}
	return func(ctx *gin.Context) {
		principal, err := GetCurrentPrincipal(ctx)
		if err != nil {
			abortWithError(ctx, err)
			return
		}
		if !principal.HasPermission(required) {
			abortWithError(ctx, errs.Forbidden(fmt.Sprintf("missing permission: %s", required)))
			return
		}
		ctx.Next()
	}
}

// RequireLogin 接口只要求登录、不要求具体权限码，例如 me、改密码。
// 认证中间件已经拦住了未登录的请求，这里是在路由表里显式写明"这个接口需要登录"。
func RequireLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, err := GetCurrentPrincipal(ctx); err != nil {
			abortWithError(ctx, err)
			return
		}
		ctx.Next()
	}
}

// abortWithError 把错误交给 ErrorHandler 渲染，并停止后续处理。
func abortWithError(ctx *gin.Context, err error) {
	_ = ctx.Error(err)
	ctx.Abort()
}
