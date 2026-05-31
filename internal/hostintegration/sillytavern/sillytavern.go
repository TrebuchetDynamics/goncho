package sillytavern

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/policy"
)

type SillyTavernInput = contracts.SillyTavernInput
type SillyTavernMapping = contracts.SillyTavernMapping

// MapSillyTavern maps the SillyTavern-specific Honcho integration
// controls into Goncho's host compatibility fixture surface.
func MapSillyTavern(input SillyTavernInput) SillyTavernMapping {
	defaults, _ := policy.DefaultsForHost("sillytavern")
	compat := contracts.HonchoExternalCompatibility()

	out := SillyTavernMapping{
		WorkspaceID:       contracts.FirstNonBlank(input.Workspace, defaults.Workspace, "sillytavern"),
		ExternalToolNames: append([]string(nil), compat.ExternalToolNames...),
	}
	out.UserPeerID = sillyTavernUserPeerID(input, &out.Unsupported)
	out.SessionKey = sillyTavernSessionKey(input, &out.Unsupported)
	if input.ResetActiveSession {
		out.OrphanedSessionKey = strings.TrimSpace(input.ExistingSessionKey)
	}
	out.CharacterPeerIDs, out.LazyAddedCharacterPeerIDs = sillyTavernCharacterPeerIDs(input)
	sillyTavernEnrichment(input.EnrichmentMode, &out)

	for _, knob := range input.UnsupportedPanelKnobs {
		knob = strings.TrimSpace(knob)
		if knob == "" {
			continue
		}
		contracts.AddUnsupported(&out.Unsupported, "sillytavern_panel_knob", knob, "panel knob is not mapped by Goncho host fixtures")
	}

	return out
}

func sillyTavernUserPeerID(input SillyTavernInput, unsupported *[]contracts.UnsupportedMapping) string {
	peer := strings.TrimSpace(input.PeerName)
	if peer == "" {
		contracts.AddUnsupported(unsupported, "peer_name", "", "SillyTavern peer mapping requires Your peer name")
		return ""
	}

	switch normalizeSillyTavernLabel(input.PeerMode) {
	case "", "single peer for all personas", "single", "shared", "shared peer":
		return peer
	case "separate peer per persona", "separate", "per persona", "per-persona":
		persona := slugSillyTavernID(input.PersonaName)
		if persona == "" {
			contracts.AddUnsupported(unsupported, "persona_name", "", "separate peer per persona requires a SillyTavern persona name")
			return ""
		}
		return peer + ":persona:" + persona
	default:
		contracts.AddUnsupported(unsupported, "peer_mode", strings.TrimSpace(input.PeerMode), "supported peer modes are single peer for all personas and separate peer per persona")
		return ""
	}
}

func sillyTavernSessionKey(input SillyTavernInput, unsupported *[]contracts.UnsupportedMapping) string {
	existing := strings.TrimSpace(input.ExistingSessionKey)
	if existing != "" && !input.ResetActiveSession {
		return existing
	}
	if input.ResetActiveSession {
		if existing == "" {
			contracts.AddUnsupported(unsupported, "active_session", "", "reset requires an active session key to orphan")
		}
	}

	switch normalizeSillyTavernLabel(input.SessionNaming) {
	case "", "auto", "per chat", "per-chat", "chat":
		return policy.SessionKeyForStrategy("sillytavern", "chat-instance", contracts.Input{
			ChatInstanceID: input.ChatInstanceID,
		}, unsupported)
	case "per character", "per-character", "character":
		character := slugSillyTavernID(input.CharacterName)
		if character == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", "per-character", "per-character session naming requires character_name")
			return ""
		}
		return "sillytavern:session:character:" + character
	case "custom":
		custom := slugSillyTavernID(input.CustomSessionName)
		if custom == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", "custom", "custom session naming requires custom_session_name")
			return ""
		}
		return "sillytavern:session:custom:" + custom
	default:
		contracts.AddUnsupported(unsupported, "session_naming", strings.TrimSpace(input.SessionNaming), "supported session naming modes are auto, per character, and custom")
		return ""
	}
}

func sillyTavernCharacterPeerIDs(input SillyTavernInput) ([]string, []string) {
	existing := make(map[string]bool, len(input.ExistingCharacterPeerIDs))
	for _, peerID := range input.ExistingCharacterPeerIDs {
		peerID = strings.TrimSpace(peerID)
		if peerID != "" {
			existing[peerID] = true
		}
	}

	seen := make(map[string]bool, len(input.GroupCharacterNames)+1)
	var peers []string
	var lazy []string
	addCharacter := func(name string) {
		peerID := sillyTavernCharacterPeerID(name)
		if peerID == "" || seen[peerID] {
			return
		}
		seen[peerID] = true
		peers = append(peers, peerID)
		if !existing[peerID] {
			lazy = append(lazy, peerID)
		}
	}

	for _, name := range input.GroupCharacterNames {
		addCharacter(name)
	}
	addCharacter(input.MessageCharacterName)
	return peers, lazy
}

func sillyTavernEnrichment(mode string, out *SillyTavernMapping) {
	switch normalizeSillyTavernLabel(mode) {
	case "", "reasoning":
		out.InjectContext = true
		out.UseReasoning = true
		out.ReasoningToolName = "honcho_chat"
	case "context only", "context-only", "context":
		out.InjectContext = true
	case "tool call", "tool-call", "tool", "tools":
		out.InjectContext = true
		out.ExposeTools = true
	default:
		contracts.AddUnsupported(&out.Unsupported, "enrichment_mode", strings.TrimSpace(mode), "supported enrichment modes are context only, reasoning, and tool call")
	}
}

func sillyTavernCharacterPeerID(name string) string {
	slug := slugSillyTavernID(name)
	if slug == "" {
		return ""
	}
	return "sillytavern:character:" + slug
}

func normalizeSillyTavernLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func slugSillyTavernID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
