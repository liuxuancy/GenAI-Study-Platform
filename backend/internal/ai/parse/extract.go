package parse

import (
	"regexp"
	"strings"

	"caveclass/internal/ai/aierr"
)

// 语言标签只认 json。用词边界，避免把 ```javascript 当成 json。
var codeFence = regexp.MustCompile("(?is)```(?:json\\b)?[ \\t]*\\r?\\n?(.*?)```")

// ExtractJSON 从模型原文取出一个 JSON 对象。
// 支持纯 JSON，也支持前后有说明文字的 ```json 代码块。
// 正文以 { 开头时，字符串里的 markdown 代码块不会被当成外层围栏。
func ExtractJSON(rawText string) (string, error) {
	text := strings.TrimSpace(rawText)
	text = strings.TrimPrefix(text, "\uFEFF")
	if text == "" {
		return "", aierr.OutputParseFailed(nil, "empty input")
	}
	if strings.HasPrefix(text, "{") {
		return objectAt(text)
	}
	if fenced, ok := firstCodeFence(text); ok {
		text = strings.TrimSpace(fenced)
	}
	return objectAt(text)
}

func objectAt(text string) (string, error) {
	start := strings.Index(text, "{")
	if start < 0 {
		return "", aierr.OutputParseFailed(nil, "no JSON object")
	}
	end, ok := matchingObjectEnd(text, start)
	if !ok {
		return "", aierr.OutputParseFailed(nil, "unbalanced JSON object")
	}
	return text[start : end+1], nil
}

func firstCodeFence(text string) (string, bool) {
	match := codeFence.FindStringSubmatch(text)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// matchingObjectEnd 找与 start 处 '{' 配对的 '}'，忽略字符串里的括号。
func matchingObjectEnd(text string, start int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		char := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}
