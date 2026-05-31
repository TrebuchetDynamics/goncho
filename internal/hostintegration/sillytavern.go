package hostintegration

import (
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
	sillytavernadapter "github.com/TrebuchetDynamics/goncho/internal/hostintegration/sillytavern"
)

type SillyTavernInput = contracts.SillyTavernInput
type SillyTavernMapping = contracts.SillyTavernMapping

// MapSillyTavern maps the SillyTavern-specific Honcho integration controls
// into Goncho's host compatibility fixture surface.
func MapSillyTavern(input SillyTavernInput) SillyTavernMapping {
	return sillytavernadapter.MapSillyTavern(input)
}
