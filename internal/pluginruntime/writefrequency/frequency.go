package writefrequency

import (
	"strconv"
	"strings"
)

// Mode identifies when plugin memory writes should be flushed.
type Mode string

const (
	Invalid Mode = "invalid"
	Async   Mode = "async"
	Turn    Mode = "turn"
	Session Mode = "session"
	Every   Mode = "every"
)

// Frequency is the parsed plugin memory write cadence.
type Frequency struct {
	Mode  Mode
	Every int
	Raw   string
}

// Parse converts a plugin config value into a write frequency.
func Parse(raw any) Frequency {
	switch v := raw.(type) {
	case nil:
		return Frequency{Mode: Async, Raw: "async"}
	case int:
		if v > 0 {
			return Frequency{Mode: Every, Every: v, Raw: intToString(v)}
		}
	case int64:
		if v > 0 {
			return Frequency{Mode: Every, Every: int(v), Raw: intToString(int(v))}
		}
	case float64:
		if v == float64(int(v)) && v > 0 {
			return Frequency{Mode: Every, Every: int(v), Raw: intToString(int(v))}
		}
	case string:
		trimmed := stringsLowerTrim(v)
		switch trimmed {
		case "", "async":
			return Frequency{Mode: Async, Raw: "async"}
		case "turn":
			return Frequency{Mode: Turn, Raw: "turn"}
		case "session":
			return Frequency{Mode: Session, Raw: "session"}
		default:
			if n, ok := parsePositiveInt(trimmed); ok {
				return Frequency{Mode: Every, Every: n, Raw: trimmed}
			}
		}
	}
	return Frequency{Mode: Invalid, Raw: ""}
}

func stringsLowerTrim(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parsePositiveInt(value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func intToString(n int) string {
	return strconv.Itoa(n)
}
