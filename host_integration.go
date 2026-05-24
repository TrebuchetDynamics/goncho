package goncho

import hostintegration "github.com/TrebuchetDynamics/goncho/internal/hostintegration"

// HostIntegrationInput is the host-facing compatibility fixture input. It
// models the shared Honcho concepts used by current hosts without importing or
// running those hosts' plugins.
type HostIntegrationInput = hostintegration.Input

// HostIntegrationMapping is the internal Goncho interpretation of one host
// configuration.
type HostIntegrationMapping = hostintegration.Mapping

// UnsupportedHostMapping explains a host compatibility input that Goncho cannot
// safely accept yet.
type UnsupportedHostMapping = hostintegration.UnsupportedMapping

// ExternalCompatibility records the internal/external naming contract.
type ExternalCompatibility = hostintegration.ExternalCompatibility

// HostConfigDocument is the shared ~/.honcho/config.json shape needed for
// host-scoped config isolation fixtures.
type HostConfigDocument = hostintegration.ConfigDocument

// HostRuntimeConfig is one hosts.<name> block from the Honcho shared config.
type HostRuntimeConfig = hostintegration.RuntimeConfig

// HostConfigPatch updates only one hosts.<name> block.
type HostConfigPatch = hostintegration.ConfigPatch

// MapHostIntegration translates host config concepts to the current internal
// Goncho service contract. Unsupported fields are returned as diagnostics
// instead of being silently widened or accepted.
func MapHostIntegration(input HostIntegrationInput) HostIntegrationMapping {
	return hostintegration.Map(input)
}

// ApplyHostConfigPatch applies host-scoped config writes without mutating the
// input document or sibling host entries.
func ApplyHostConfigPatch(doc HostConfigDocument, host string, patch HostConfigPatch) (HostConfigDocument, error) {
	return hostintegration.ApplyConfigPatch(doc, host, patch)
}

// HonchoExternalCompatibility returns the current public Honcho-compatible
// tool names while keeping the implementation service named Goncho.
func HonchoExternalCompatibility() ExternalCompatibility {
	return hostintegration.HonchoExternalCompatibility()
}
