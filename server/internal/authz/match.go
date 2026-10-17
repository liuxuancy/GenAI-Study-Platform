// 权限码匹配规则和格式校验。

package authz

import "strings"

const (
	codeSeparator = ":"
	wildcard      = "*"
)

// Match 判断授予的权限码 granted 是否满足要求的权限码 required。
// 规则与前端共用，用例见 docs/permission-cases.json。
func Match(granted, required string) bool {
	grantedSegments, ok := parseCode(granted)
	if !ok {
		return false
	}
	requiredSegments, ok := parseCode(required)
	if !ok {
		return false
	}

	for index, segment := range grantedSegments {
		if segment == wildcard {
			return len(requiredSegments) > index // '*' 至少要匹配一段
		}
		if index >= len(requiredSegments) || requiredSegments[index] != segment {
			return false
		}
	}
	return len(grantedSegments) == len(requiredSegments)
}

// ValidCode 判断权限码格式是否合法。
func ValidCode(code string) bool {
	_, ok := parseCode(code)
	return ok
}

// IsWildcard 判断合法权限码是否以 '*' 结尾。
func IsWildcard(code string) bool {
	return strings.HasSuffix(code, codeSeparator+wildcard) || code == wildcard
}

// parseCode 按 ':' 切分；任一段为空，或 '*' 不是完整的最后一段，都视为非法。
func parseCode(code string) ([]string, bool) {
	if code == "" {
		return nil, false
	}
	segments := strings.Split(code, codeSeparator)
	lastIndex := len(segments) - 1
	for index, segment := range segments {
		if segment == "" {
			return nil, false
		}
		if strings.Contains(segment, wildcard) && (segment != wildcard || index != lastIndex) {
			return nil, false
		}
	}
	return segments, true
}
