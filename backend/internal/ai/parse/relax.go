package parse

// RelaxJSON applies safe fixes for common LLM JSON mistakes (e.g. trailing commas).
func RelaxJSON(raw string) string {
	var result []byte
	inString, escaped := false, false
	for index := 0; index < len(raw); index++ {
		char := raw[index]
		if inString {
			result = append(result, char)
			if escaped {
				escaped = false
			} else if char == '\\' {
				escaped = true
			} else if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
		} else if char == ',' {
			next := index + 1
			for next < len(raw) && (raw[next] == ' ' || raw[next] == '\t' || raw[next] == '\r' || raw[next] == '\n') {
				next++
			}
			if next < len(raw) && (raw[next] == '}' || raw[next] == ']') {
				continue
			}
		}
		result = append(result, char)
	}
	return string(result)
}
