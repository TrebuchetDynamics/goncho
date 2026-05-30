package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

func contextUnavailableHasCapability(values []ContextUnavailableEvidence, capability string) bool {
	return sliceutil.ContainsFunc(values, func(value ContextUnavailableEvidence) bool {
		return value.Capability == capability
	})
}
