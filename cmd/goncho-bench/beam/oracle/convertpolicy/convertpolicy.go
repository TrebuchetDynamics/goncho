package convertpolicy

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

const DefaultPeer = "beam"

func StableIDSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return shared.FirstNonEmptyTrimmed(strings.Trim(b.String(), "-"), "conversation")
}

var pythonLiteralBarewordPattern = regexp.MustCompile(`\b(True|False|None)\b`)

func PythonLiteralToJSONish(input string) string {
	var b strings.Builder
	inString := false
	var quote rune
	escaped := false
	for _, r := range input {
		if inString {
			if escaped {
				switch r {
				case '\'', '"':
					if r == '"' {
						b.WriteString(`\"`)
					} else {
						b.WriteRune(r)
					}
				case '\\':
					b.WriteString(`\\`)
				case 'n':
					b.WriteString(`\n`)
				case 't':
					b.WriteString(`\t`)
				default:
					b.WriteRune(r)
				}
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				b.WriteByte('"')
				inString = false
				continue
			}
			if r == '"' {
				b.WriteString(`\"`)
				continue
			}
			b.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			inString = true
			quote = r
			b.WriteByte('"')
			continue
		}
		b.WriteRune(r)
	}
	return pythonLiteralBarewordPattern.ReplaceAllStringFunc(b.String(), func(token string) string {
		switch token {
		case "True":
			return "true"
		case "False":
			return "false"
		case "None":
			return "null"
		default:
			return strconv.Quote(token)
		}
	})
}
