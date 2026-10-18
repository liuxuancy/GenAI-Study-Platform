// Package resp 统一响应结构和 handler 适配器。
package resp

import (
	"net/http"
	"strconv"

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

// BindQuery 解析并校验 URL 查询参数（?page=1&role=...），失败返回 400。
func BindQuery(ctx *gin.Context, target any) error {
	if err := ctx.ShouldBindQuery(target); err != nil {
		return errs.BadRequest("请求参数错误: " + err.Error())
	}
	return nil
}

// GetInt64PathParam 取出路径参数并转成整数，例如 /users/:id 里的 id，格式不对返回 400。
func GetInt64PathParam(ctx *gin.Context, name string) (int64, error) {
	value, err := strconv.ParseInt(ctx.Param(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, errs.BadRequest("路径参数 " + name + " 必须是正整数")
	}
	return value, nil
}
