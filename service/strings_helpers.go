package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil"

func firstNonBlank(values ...string) string {
	return textutil.FirstNonBlank(values...)
}

func firstNonEmpty(values ...string) string {
	return firstNonBlank(values...)
}
