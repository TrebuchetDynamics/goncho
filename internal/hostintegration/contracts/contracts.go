// Package contracts contains host-integration value types shared across host
// adapters and the root compatibility facade.
package contracts

import "strings"

// Input is the host-facing compatibility fixture input. It models the shared
// Honcho concepts used by current hosts without importing or running those
// hosts' plugins.
type Input struct {
	Host             string
	Workspace        string
	PeerName         string
	AIPeer           string
	SessionStrategy  string
	WorkingDirectory string
	Repository       string
	Branch           string
	HostSessionID    string
	ChatInstanceID   string
	CharacterName    string
	RecallMode       string
}

// Mapping is the internal Goncho interpretation of one host configuration.
type Mapping struct {
	Host              string
	WorkspaceID       string
	UserPeerID        string
	AIPeerID          string
	SessionStrategy   string
	SessionKey        string
	RecallMode        string
	InjectContext     bool
	ExposeTools       bool
	InternalService   string
	ExternalToolNames []string
	Unsupported       []UnsupportedMapping
}

// UnsupportedMapping explains a host compatibility input that Goncho cannot
// safely accept yet.
type UnsupportedMapping struct {
	Field  string
	Value  string
	Reason string
}

// AddUnsupported appends a structured unsupported-field diagnostic to a mapping.
func AddUnsupported(items *[]UnsupportedMapping, field, value, reason string) {
	*items = append(*items, UnsupportedMapping{
		Field:  field,
		Value:  value,
		Reason: reason,
	})
}

// NormalizeHost canonicalizes host identifiers used in config and session keys.
func NormalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.ReplaceAll(host, "-", "_")
	switch host {
	case "silly_tavern":
		return "sillytavern"
	default:
		return host
	}
}

// FirstNonBlank returns the first non-empty trimmed value from values.
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// ExternalCompatibility records the internal/external naming contract.
type ExternalCompatibility struct {
	InternalService   string
	ExternalToolNames []string
}

// HonchoExternalCompatibility returns the current public Honcho-compatible tool
// names while keeping the implementation service named Goncho.
func HonchoExternalCompatibility() ExternalCompatibility {
	return ExternalCompatibility{
		InternalService: "goncho",
		ExternalToolNames: []string{
			"honcho_profile",
			"honcho_search",
			"honcho_context",
			"honcho_chat",
			"honcho_reasoning",
			"honcho_conclude",
		},
	}
}

// ConfigDocument is the shared ~/.honcho/config.json shape needed for
// host-scoped config isolation fixtures.
type ConfigDocument struct {
	APIKey    string                   `json:"apiKey,omitempty"`
	BaseURL   string                   `json:"baseUrl,omitempty"`
	PeerName  string                   `json:"peerName,omitempty"`
	Workspace string                   `json:"workspace,omitempty"`
	Hosts     map[string]RuntimeConfig `json:"hosts,omitempty"`
}

// RuntimeConfig is one hosts.<name> block from the Honcho shared config.
type RuntimeConfig struct {
	Workspace       string `json:"workspace,omitempty"`
	AIPeer          string `json:"aiPeer,omitempty"`
	PeerName        string `json:"peerName,omitempty"`
	RecallMode      string `json:"recallMode,omitempty"`
	ObservationMode string `json:"observationMode,omitempty"`
	SessionStrategy string `json:"sessionStrategy,omitempty"`
}

// ConfigPatch updates only one hosts.<name> block.
type ConfigPatch struct {
	Workspace       *string
	AIPeer          *string
	PeerName        *string
	RecallMode      *string
	ObservationMode *string
	SessionStrategy *string
}

// SillyTavernInput models the Honcho SillyTavern panel decisions Goncho needs
// to preserve without importing the browser extension or Node plugin.
type SillyTavernInput struct {
	Workspace                string
	PeerMode                 string
	PeerName                 string
	PersonaName              string
	SessionNaming            string
	ChatInstanceID           string
	CharacterName            string
	CustomSessionName        string
	ExistingSessionKey       string
	ResetActiveSession       bool
	GroupCharacterNames      []string
	ExistingCharacterPeerIDs []string
	MessageCharacterName     string
	EnrichmentMode           string
	UnsupportedPanelKnobs    []string
}

// SillyTavernMapping is Goncho's fixture-level interpretation of the
// SillyTavern host contract.
type SillyTavernMapping struct {
	WorkspaceID               string
	UserPeerID                string
	SessionKey                string
	OrphanedSessionKey        string
	CharacterPeerIDs          []string
	LazyAddedCharacterPeerIDs []string
	InjectContext             bool
	UseReasoning              bool
	ReasoningToolName         string
	ExposeTools               bool
	ExternalToolNames         []string
	Unsupported               []UnsupportedMapping
}
