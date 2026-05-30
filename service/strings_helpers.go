package goncho

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/maputil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func firstNonBlank(values ...string) string {
	return textutil.FirstNonBlank(values...)
}

func firstNonEmpty(values ...string) string {
	return firstNonBlank(values...)
}

func cloneStringMap(in map[string]string) map[string]string {
	return maputil.CloneStringString(in)
}

func copyMetadata(in map[string]any) map[string]any {
	return maputil.CloneStringAny(in)
}

func stringMapToAny(in map[string]string) map[string]any {
	return maputil.StringStringToAny(in)
}

func cloneStrings(in []string) []string {
	return sliceutil.Clone(in)
}
