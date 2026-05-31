package facttext

import "strings"

// Sentences splits fact-like text into replayable sentence candidates without
// treating decimal points inside numeric measurements as sentence boundaries.
func Sentences(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	out := []string{}
	start := 0
	for i := 0; i < len(content); i++ {
		switch content[i] {
		case '.', '!', '?':
			if content[i] == '.' && byteIsDigit(content, i-1) && byteIsDigit(content, i+1) {
				continue
			}
			if sentence := strings.TrimSpace(content[start : i+1]); sentence != "" {
				out = append(out, sentence)
			}
			start = i + 1
		}
	}
	if sentence := strings.TrimSpace(content[start:]); sentence != "" {
		out = append(out, sentence)
	}
	return out
}

func byteIsDigit(content string, idx int) bool {
	return idx >= 0 && idx < len(content) && content[idx] >= '0' && content[idx] <= '9'
}
