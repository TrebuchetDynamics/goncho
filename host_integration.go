package goncho

import (
	"fmt"
	"strings"
)

// HostIntegrationInput is the host-facing compatibility fixture input. It
// models the shared Honcho concepts used by current hosts without importing or
// running those hosts' plugins.
type HostIntegrationInput struct {
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

// HostIntegrationMapping is the internal Goncho interpretation of one host
// configuration.
type HostIntegrationMapping struct {
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
	Unsupported       []UnsupportedHostMapping
}

// UnsupportedHostMapping explains a host compatibility input that Goncho cannot
// safely accept yet.
type UnsupportedHostMapping struct {
	Field  string
	Value  string
	Reason string
}

// ExternalCompatibility records the internal/external naming contract.
type ExternalCompatibility struct {
	InternalService   string
	ExternalToolNames []string
}

// HostConfigDocument is the shared ~/.honcho/config.json shape needed for
// host-scoped config isolation fixtures.
type HostConfigDocument struct {
	APIKey    string                       `json:"apiKey,omitempty"`
	BaseURL   string                       `json:"baseUrl,omitempty"`
	PeerName  string                       `json:"peerName,omitempty"`
	Workspace string                       `json:"workspace,omitempty"`
	Hosts     map[string]HostRuntimeConfig `json:"hosts,omitempty"`
}

// HostRuntimeConfig is one hosts.<name> block from the Honcho shared config.
type HostRuntimeConfig struct {
	Workspace       string `json:"workspace,omitempty"`
	AIPeer          string `json:"aiPeer,omitempty"`
	PeerName        string `json:"peerName,omitempty"`
	RecallMode      string `json:"recallMode,omitempty"`
	ObservationMode string `json:"observationMode,omitempty"`
	SessionStrategy string `json:"sessionStrategy,omitempty"`
}

// HostConfigPatch updates only one hosts.<name> block.
type HostConfigPatch struct {
	Workspace       *string
	AIPeer          *string
	PeerName        *string
	RecallMode      *string
	ObservationMode *string
	SessionStrategy *string
}

type hostDefaults struct {
	workspace       string
	aiPeer          string
	sessionStrategy string
	recallMode      string
}

// MapHostIntegration translates host config concepts to the current internal
// Goncho service contract. Unsupported fields are returned as diagnostics
// instead of being silently widened or accepted.
func MapHostIntegration(input HostIntegrationInput) HostIntegrationMapping {
	host := normalizeHost(input.Host)
	defaults, ok := defaultsForHost(host)
	if !ok {
		defaults = hostDefaults{
			workspace:       "default",
			aiPeer:          "gormes",
			sessionStrategy: "per-session",
			recallMode:      "hybrid",
		}
	}

	compat := HonchoExternalCompatibility()
	out := HostIntegrationMapping{
		Host:              host,
		WorkspaceID:       firstNonBlank(input.Workspace, defaults.workspace, "default"),
		UserPeerID:        strings.TrimSpace(input.PeerName),
		AIPeerID:          firstNonBlank(input.AIPeer, defaults.aiPeer, "gormes"),
		InternalService:   compat.InternalService,
		ExternalToolNames: append([]string(nil), compat.ExternalToolNames...),
	}
	if !ok {
		out.Unsupported = append(out.Unsupported, UnsupportedHostMapping{
			Field:  "host",
			Value:  strings.TrimSpace(input.Host),
			Reason: "host has no Goncho compatibility defaults",
		})
	}
	if out.UserPeerID == "" {
		out.Unsupported = append(out.Unsupported, UnsupportedHostMapping{
			Field:  "peer_name",
			Value:  "",
			Reason: "host mappings require an explicit durable user peer",
		})
	}

	out.SessionStrategy = normalizeSessionStrategy(firstNonBlank(input.SessionStrategy, defaults.sessionStrategy))
	out.SessionKey = sessionKeyForStrategy(host, out.SessionStrategy, input, &out.Unsupported)

	recallMode, ok := normalizeRecallMode(firstNonBlank(input.RecallMode, defaults.recallMode))
	if !ok {
		out.Unsupported = append(out.Unsupported, UnsupportedHostMapping{
			Field:  "recall_mode",
			Value:  strings.TrimSpace(input.RecallMode),
			Reason: "supported recall modes are context, tools, and hybrid",
		})
	} else {
		out.RecallMode = recallMode
		out.InjectContext = recallMode == "context" || recallMode == "hybrid"
		out.ExposeTools = recallMode == "tools" || recallMode == "hybrid"
	}

	return out
}

// ApplyHostConfigPatch applies host-scoped config writes without mutating the
// input document or sibling host entries.
func ApplyHostConfigPatch(doc HostConfigDocument, host string, patch HostConfigPatch) (HostConfigDocument, error) {
	host = normalizeHost(host)
	if host == "" {
		return HostConfigDocument{}, fmt.Errorf("goncho: host is required")
	}

	out := doc
	out.Hosts = make(map[string]HostRuntimeConfig, len(doc.Hosts)+1)
	for key, value := range doc.Hosts {
		out.Hosts[normalizeHost(key)] = value
	}

	cfg := out.Hosts[host]
	if patch.Workspace != nil {
		cfg.Workspace = strings.TrimSpace(*patch.Workspace)
	}
	if patch.AIPeer != nil {
		cfg.AIPeer = strings.TrimSpace(*patch.AIPeer)
	}
	if patch.PeerName != nil {
		cfg.PeerName = strings.TrimSpace(*patch.PeerName)
	}
	if patch.RecallMode != nil {
		cfg.RecallMode = strings.TrimSpace(*patch.RecallMode)
	}
	if patch.ObservationMode != nil {
		cfg.ObservationMode = strings.TrimSpace(*patch.ObservationMode)
	}
	if patch.SessionStrategy != nil {
		cfg.SessionStrategy = strings.TrimSpace(*patch.SessionStrategy)
	}
	out.Hosts[host] = cfg
	return out, nil
}

// HonchoExternalCompatibility returns the current public Honcho-compatible
// tool names while keeping the implementation service named Goncho.
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

func defaultsForHost(host string) (hostDefaults, bool) {
	switch host {
	case "hermes":
		return hostDefaults{
			workspace:       "hermes",
			aiPeer:          "hermes",
			sessionStrategy: "per-directory",
			recallMode:      "hybrid",
		}, true
	case "opencode":
		return hostDefaults{
			workspace:       "opencode",
			aiPeer:          "opencode",
			sessionStrategy: "per-directory",
			recallMode:      "hybrid",
		}, true
	case "sillytavern":
		return hostDefaults{
			workspace:       "sillytavern",
			aiPeer:          "sillytavern",
			sessionStrategy: "chat-instance",
			recallMode:      "hybrid",
		}, true
	default:
		return hostDefaults{}, false
	}
}

func sessionKeyForStrategy(host, strategy string, input HostIntegrationInput, unsupported *[]UnsupportedHostMapping) string {
	switch strategy {
	case "per-directory":
		value := strings.TrimSpace(input.WorkingDirectory)
		if value == "" {
			addUnsupported(unsupported, "session_strategy", strategy, "per-directory requires working_directory")
			return ""
		}
		return host + ":dir:" + value
	case "per-repo":
		value := strings.TrimSpace(input.Repository)
		if value == "" {
			addUnsupported(unsupported, "session_strategy", strategy, "per-repo requires repository")
			return ""
		}
		return host + ":repo:" + value
	case "git-branch":
		repo := strings.TrimSpace(input.Repository)
		branch := strings.TrimSpace(input.Branch)
		if repo == "" || branch == "" {
			addUnsupported(unsupported, "session_strategy", strategy, "git-branch requires repository and branch")
			return ""
		}
		return host + ":branch:" + repo + ":" + branch
	case "per-session":
		value := firstNonBlank(input.HostSessionID, input.CharacterName)
		if value == "" {
			addUnsupported(unsupported, "session_strategy", strategy, "per-session requires host_session_id")
			return ""
		}
		return host + ":session:" + value
	case "chat-instance":
		value := strings.TrimSpace(input.ChatInstanceID)
		if value == "" {
			addUnsupported(unsupported, "session_strategy", strategy, "chat-instance requires chat_instance_id")
			return ""
		}
		return host + ":chat:" + value
	case "global":
		return host + ":global"
	default:
		addUnsupported(unsupported, "session_strategy", strategy, "unsupported session strategy")
		return ""
	}
}

func addUnsupported(items *[]UnsupportedHostMapping, field, value, reason string) {
	*items = append(*items, UnsupportedHostMapping{
		Field:  field,
		Value:  value,
		Reason: reason,
	})
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.ReplaceAll(host, "-", "_")
	switch host {
	case "silly_tavern":
		return "sillytavern"
	default:
		return host
	}
}

func normalizeSessionStrategy(strategy string) string {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "directory", "per_directory":
		return "per-directory"
	case "repo", "per_repo":
		return "per-repo"
	case "branch", "git_branch":
		return "git-branch"
	case "session", "per_session", "custom", "per-character", "per_character":
		return "per-session"
	case "chat", "per-chat", "per_chat", "chat_instance", "auto":
		return "chat-instance"
	default:
		return strings.ToLower(strings.TrimSpace(strategy))
	}
}

func normalizeRecallMode(mode string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "hybrid", "reasoning":
		return "hybrid", true
	case "context", "context-only", "context_only":
		return "context", true
	case "tools", "tool", "tool-only", "tool_only", "tool-call", "tool_call":
		return "tools", true
	default:
		return "", false
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
