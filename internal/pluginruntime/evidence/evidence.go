package evidence

// Config records a config-policy evidence code plus optional provenance.
type Config struct {
	Code    string `json:"code"`
	Source  string `json:"source,omitempty"`
	Message string `json:"message,omitempty"`
}

// HasConfig reports whether items contains a config evidence item with code.
func HasConfig(items []Config, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

// Has reports whether items contains code.
func Has(items []string, code string) bool {
	for _, item := range items {
		if item == code {
			return true
		}
	}
	return false
}

// Append adds code to items unless it is already present.
func Append(items []string, code string) []string {
	if Has(items, code) {
		return items
	}
	return append(items, code)
}
