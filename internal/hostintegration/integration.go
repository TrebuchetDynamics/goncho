package hostintegration

import (
	configpatch "github.com/TrebuchetDynamics/goncho/internal/hostintegration/config"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/mapping"
)

type Input = contracts.Input
type Mapping = contracts.Mapping
type UnsupportedMapping = contracts.UnsupportedMapping
type ExternalCompatibility = contracts.ExternalCompatibility
type ConfigDocument = contracts.ConfigDocument
type RuntimeConfig = contracts.RuntimeConfig
type ConfigPatch = contracts.ConfigPatch

// Map translates host config concepts to the current internal Goncho service
// contract. Unsupported fields are returned as diagnostics instead of being
// silently widened or accepted.
func Map(input Input) Mapping {
	return mapping.Map(input)
}

// ApplyConfigPatch applies host-scoped config writes without mutating the
// input document or sibling host entries.
func ApplyConfigPatch(doc ConfigDocument, host string, patch ConfigPatch) (ConfigDocument, error) {
	return configpatch.ApplyPatch(doc, host, patch)
}

// HonchoExternalCompatibility returns the current public Honcho-compatible
// tool names while keeping the implementation service named Goncho.
func HonchoExternalCompatibility() ExternalCompatibility {
	return contracts.HonchoExternalCompatibility()
}
