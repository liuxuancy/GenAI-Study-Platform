// Package resp 统一响应结构和 handler 适配器。
package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caveclass/internal/platform/errs"
)

// Result 是所有接口的响应体，Code 为 0 表示成功。
type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// HandlerFunc 是 handler 的统一写法：返回数据或错误，不直接写响应。
type HandlerFunc func(ctx *gin.Context) (any, error)

// ToGinHandler 把我们的 HandlerFunc 转成 gin 要求的 handler：
// 成功时写出 {code: 0, msg, data}；失败时把错误交给 ErrorHandler 中间件统一渲染。
func ToGinHandler(handler HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		data, err := handler(ctx)
		if err != nil {
			_ = ctx.Error(err)
			ctx.Abort()
			return
		}
		ctx.JSON(http.StatusOK, Result{Code: errs.SystemOK, Msg: "ok", Data: data})
	}
}

// Fail 写出错误响应，只应由错误中间件调用。
func Fail(ctx *gin.Context, appError *errs.Error) {
	ctx.AbortWithStatusJSON(appError.Code, Result{Code: appError.Code, Msg: appError.Msg})
}

// BindJSON 解析并校验请求体，失败返回 400。
func BindJSON(ctx *gin.Context, target any) error {
	if err := ctx.ShouldBindJSON(target); err != nil {
		return errs.BadRequest("请求参数错误: " + err.Error())
	}
	return nil
}
