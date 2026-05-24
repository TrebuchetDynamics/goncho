package goncho

import hostintegration "github.com/TrebuchetDynamics/goncho/internal/hostintegration"

// SillyTavernIntegrationInput models the Honcho SillyTavern panel decisions
// Goncho needs to preserve without importing the browser extension or Node
// plugin.
type SillyTavernIntegrationInput = hostintegration.SillyTavernInput

// SillyTavernIntegrationMapping is Goncho's fixture-level interpretation of
// the SillyTavern host contract.
type SillyTavernIntegrationMapping = hostintegration.SillyTavernMapping

// MapSillyTavernIntegration maps the SillyTavern-specific Honcho integration
// controls into Goncho's host compatibility fixture surface.
func MapSillyTavernIntegration(input SillyTavernIntegrationInput) SillyTavernIntegrationMapping {
	return hostintegration.MapSillyTavern(input)
}
