// Package aierr 定义 AI 子系统可识别的错误（aiCode）。
// 独立成包，避免 parse、llm 与 ai.Service 互相引用。
// 给前端的 msg 不含 details；aiCode 和 details 只进日志。
package aierr

import (
	"errors"
	"strings"

	"caveclass/internal/platform/errs"
)

const (
	CodeInvalidInput           = "AI_INVALID_INPUT"
	CodeInputTooLong           = "AI_INPUT_TOO_LONG"
	CodePromptInjectionSuspect = "AI_PROMPT_INJECTION_SUSPECT"
	CodeModelUnavailable       = "AI_MODEL_UNAVAILABLE"
	CodeTimeout                = "AI_TIMEOUT"
	CodeOutputParseFailed      = "AI_OUTPUT_PARSE_FAILED"
	CodeOutputSchemaFailed     = "AI_OUTPUT_SCHEMA_FAILED"
	CodeInternal               = "AI_INTERNAL"
)

// Error 是带 aiCode 的 AI 错误。errors.Is 按 Code 匹配。
type Error struct {
	Code    string
	Msg     string
	Details []string
	cause   error
}

func (aiError *Error) Error() string {
	if aiError == nil {
		return ""
	}
	message := aiError.Code
	if aiError.Msg != "" {
		message += ": " + aiError.Msg
	}
	if len(aiError.Details) > 0 {
		message += ": " + strings.Join(aiError.Details, "; ")
	}
	if aiError.cause != nil {
		message += ": " + aiError.cause.Error()
	}
	return message
}

func (aiError *Error) Unwrap() error { return aiError.cause }

// Is 按 Code 比较，消息不同也能匹配。
func (aiError *Error) Is(target error) bool {
	targetError, ok := target.(*Error)
	return ok && targetError.Code == aiError.Code
}

// Status 是建议的 HTTP 状态：入参和输出问题为 400，模型与内部故障为 500。
func (aiError *Error) Status() int {
	if aiError == nil {
		return errs.SystemInternal
	}
	switch aiError.Code {
	case CodeInvalidInput, CodeInputTooLong, CodePromptInjectionSuspect, CodeOutputParseFailed, CodeOutputSchemaFailed:
		return errs.SystemBadRequest
	default:
		return errs.SystemInternal
	}
}

// As 从错误链里取出 *Error。
func As(err error) (*Error, bool) {
	return errors.AsType[*Error](err)
}

var (
	ErrInvalidInput           = &Error{Code: CodeInvalidInput, Msg: "invalid AI input"}
	ErrInputTooLong           = &Error{Code: CodeInputTooLong, Msg: "AI input is too long"}
	ErrPromptInjectionSuspect = &Error{Code: CodePromptInjectionSuspect, Msg: "AI input was rejected"}
	ErrModelUnavailable       = &Error{Code: CodeModelUnavailable, Msg: "AI model is unavailable"}
	ErrTimeout                = &Error{Code: CodeTimeout, Msg: "AI request timed out"}
	ErrOutputParseFailed      = &Error{Code: CodeOutputParseFailed, Msg: "AI output JSON parse failed"}
	ErrOutputSchemaFailed     = &Error{Code: CodeOutputSchemaFailed, Msg: "AI output failed schema validation"}
	ErrInternal               = &Error{Code: CodeInternal, Msg: "AI request failed"}
)

func InvalidInput(cause error, details ...string) *Error {
	return newError(ErrInvalidInput, cause, details)
}

func InputTooLong(cause error, details ...string) *Error {
	return newError(ErrInputTooLong, cause, details)
}

func PromptInjectionSuspect(cause error, details ...string) *Error {
	return newError(ErrPromptInjectionSuspect, cause, details)
}

func ModelUnavailable(cause error, details ...string) *Error {
	return newError(ErrModelUnavailable, cause, details)
}

func Timeout(cause error, details ...string) *Error {
	return newError(ErrTimeout, cause, details)
}

// OutputParseFailed 表示模型文本里没有可解析的 JSON。
func OutputParseFailed(cause error, details ...string) *Error {
	return newError(ErrOutputParseFailed, cause, details)
}

// OutputSchemaFailed 表示 JSON 已解析，但课程结构仍然不合法。
func OutputSchemaFailed(cause error, details ...string) *Error {
	return newError(ErrOutputSchemaFailed, cause, details)
}

func Internal(cause error, details ...string) *Error {
	return newError(ErrInternal, cause, details)
}

func newError(sentinel *Error, cause error, details []string) *Error {
	return &Error{Code: sentinel.Code, Msg: sentinel.Msg, Details: details, cause: cause}
}

// ToAppError 转成业务层的 errs 错误。
// 400 只保留给用户看的 Msg，不附带 details。500 用 errs.Internal，原始错误留在 cause 里。
// errors.Is 请用转换前的 AI 错误；400 转换后不再携带 aiCode。
func ToAppError(err error) *errs.Error {
	if err == nil {
		return nil
	}
	aiError, ok := As(err)
	if !ok {
		return errs.Internal(err)
	}
	if aiError.Status() == errs.SystemBadRequest {
		return errs.BadRequest(aiError.Msg)
	}
	return errs.Internal(aiError)
}
