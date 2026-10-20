// Package aierr 定义 AI 子系统可识别的错误（aiCode）。
// 独立成包，避免 parse 与后续 ai.Service 互相引用。
// 本阶段只有输出解析相关错误码，其余码在阶段 3 补齐。
package aierr

import (
	"strings"
)

const (
	CodeOutputParseFailed  = "AI_OUTPUT_PARSE_FAILED"
	CodeOutputSchemaFailed = "AI_OUTPUT_SCHEMA_FAILED"
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

var (
	ErrOutputParseFailed  = &Error{Code: CodeOutputParseFailed, Msg: "AI output JSON parse failed"}
	ErrOutputSchemaFailed = &Error{Code: CodeOutputSchemaFailed, Msg: "AI output failed schema validation"}
)

// OutputParseFailed 表示模型文本里没有可解析的 JSON。
func OutputParseFailed(cause error, details ...string) *Error {
	return &Error{Code: CodeOutputParseFailed, Msg: ErrOutputParseFailed.Msg, Details: details, cause: cause}
}

// OutputSchemaFailed 表示 JSON 已解析，但课程结构仍然不合法。
func OutputSchemaFailed(cause error, details ...string) *Error {
	return &Error{Code: CodeOutputSchemaFailed, Msg: ErrOutputSchemaFailed.Msg, Details: details, cause: cause}
}
