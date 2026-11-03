package parse

import "regexp"

var trailingCommaBeforeClose = regexp.MustCompile(`,(\s*[}\]])`)

// RelaxJSON applies safe fixes for common LLM JSON mistakes (e.g. trailing commas).
func RelaxJSON(raw string) string {
	return trailingCommaBeforeClose.ReplaceAllString(raw, "$1")
}
