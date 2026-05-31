// Package policy contains shared host mapping defaults and session-key rules.
package policy

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
)

// HostDefaults are host-specific fallback mapping decisions.
type HostDefaults struct {
	Workspace       string
	AIPeer          string
	SessionStrategy string
	RecallMode      string
}

// DefaultsForHost returns the documented defaults for a supported host.
func DefaultsForHost(host string) (HostDefaults, bool) {
	switch host {
	case "hermes":
		return HostDefaults{
			Workspace:       "hermes",
			AIPeer:          "hermes",
			SessionStrategy: "per-directory",
			RecallMode:      "hybrid",
		}, true
	case "opencode":
		return HostDefaults{
			Workspace:       "opencode",
			AIPeer:          "opencode",
			SessionStrategy: "per-directory",
			RecallMode:      "hybrid",
		}, true
	case "sillytavern":
		return HostDefaults{
			Workspace:       "sillytavern",
			AIPeer:          "sillytavern",
			SessionStrategy: "chat-instance",
			RecallMode:      "hybrid",
		}, true
	default:
		return HostDefaults{}, false
	}
}

// SessionKeyForStrategy builds a host-scoped durable session key or records why
// the selected strategy cannot be used safely.
func SessionKeyForStrategy(host, strategy string, input contracts.Input, unsupported *[]contracts.UnsupportedMapping) string {
	switch strategy {
	case "per-directory":
		value := strings.TrimSpace(input.WorkingDirectory)
		if value == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", strategy, "per-directory requires working_directory")
			return ""
		}
		return host + ":dir:" + value
	case "per-repo":
		value := strings.TrimSpace(input.Repository)
		if value == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", strategy, "per-repo requires repository")
			return ""
		}
		return host + ":repo:" + value
	case "git-branch":
		repo := strings.TrimSpace(input.Repository)
		branch := strings.TrimSpace(input.Branch)
		if repo == "" || branch == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", strategy, "git-branch requires repository and branch")
			return ""
		}
		return host + ":branch:" + repo + ":" + branch
	case "per-session":
		value := contracts.FirstNonBlank(input.HostSessionID, input.CharacterName)
		if value == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", strategy, "per-session requires host_session_id")
			return ""
		}
		return host + ":session:" + value
	case "chat-instance":
		value := strings.TrimSpace(input.ChatInstanceID)
		if value == "" {
			contracts.AddUnsupported(unsupported, "session_strategy", strategy, "chat-instance requires chat_instance_id")
			return ""
		}
		return host + ":chat:" + value
	case "global":
		return host + ":global"
	default:
		contracts.AddUnsupported(unsupported, "session_strategy", strategy, "unsupported session strategy")
		return ""
	}
}

// NormalizeSessionStrategy canonicalizes supported strategy labels.
func NormalizeSessionStrategy(strategy string) string {
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

// NormalizeRecallMode canonicalizes supported recall modes.
func NormalizeRecallMode(mode string) (string, bool) {
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
