package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

func FirstNonBlank(values ...string) string {
	return trimmed.FirstNonBlank(values...)
}
