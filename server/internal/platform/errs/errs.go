// Package errs 定义业务错误。错误中间件把它们渲染成 {code, msg, data}，
// Code 同时作为 HTTP 状态码。
package errs

import (
	"errors"
	"fmt"
)

const (
	SystemOK           = 0
	SystemBadRequest   = 400
	SystemUnauthorized = 401
	SystemForbidden    = 403
	SystemNotFound     = 404
	SystemConflict     = 409
	SystemInternal     = 500
)

type Error struct {
	Code  int
	Msg   string // 返回给用户
	cause error  // 只进日志，不出服务端
}

func (appError *Error) Error() string {
	if appError.cause != nil {
		return fmt.Sprintf("%d %s: %v", appError.Code, appError.Msg, appError.cause)
	}
	return fmt.Sprintf("%d %s", appError.Code, appError.Msg)
}

func (appError *Error) Unwrap() error { return appError.cause }

// Is 按 Code 比较，消息不同也能匹配：
// errors.Is(errs.Forbidden("不是你的课程"), errs.ErrForbidden) == true
func (appError *Error) Is(target error) bool {
	targetError, ok := target.(*Error)
	return ok && targetError.Code == appError.Code
}

// 哨兵错误：用于 errors.Is 判断，或直接返回默认消息
var (
	ErrBadRequest   = &Error{Code: SystemBadRequest, Msg: "请求参数错误"}
	ErrUnauthorized = &Error{Code: SystemUnauthorized, Msg: "未登录或登录已失效"}
	ErrForbidden    = &Error{Code: SystemForbidden, Msg: "没有权限"}
	ErrNotFound     = &Error{Code: SystemNotFound, Msg: "资源不存在"}
	ErrConflict     = &Error{Code: SystemConflict, Msg: "操作冲突"}
	ErrInternal     = &Error{Code: SystemInternal, Msg: "服务器内部错误"}
)

func BadRequest(msg string) *Error   { return &Error{Code: SystemBadRequest, Msg: msg} }
func Unauthorized(msg string) *Error { return &Error{Code: SystemUnauthorized, Msg: msg} }
func Forbidden(msg string) *Error    { return &Error{Code: SystemForbidden, Msg: msg} }
func NotFound(msg string) *Error     { return &Error{Code: SystemNotFound, Msg: msg} }
func Conflict(msg string) *Error     { return &Error{Code: SystemConflict, Msg: msg} }

// Internal 包装意外错误：用户只看到通用提示，cause 写日志。
func Internal(cause error) *Error {
	return &Error{Code: SystemInternal, Msg: ErrInternal.Msg, cause: cause}
}

// From 从错误链里取出 *Error，取不到就当作 Internal。
func From(err error) *Error {
	if appError, ok := errors.AsType[*Error](err); ok {
		return appError
	}
	return Internal(err)
}
