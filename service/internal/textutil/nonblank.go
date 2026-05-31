package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if value := trimmed.Space(value); value != "" {
			return value
		}
	}
	return ""
}
