package hostintegration

import (
	"strings"

	configpatch "github.com/TrebuchetDynamics/goncho/internal/hostintegration/config"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/policy"
)

type Input = contracts.Input
type Mapping = contracts.Mapping
type UnsupportedMapping = contracts.UnsupportedMapping
type ExternalCompatibility = contracts.ExternalCompatibility
type ConfigDocument = contracts.ConfigDocument
type RuntimeConfig = contracts.RuntimeConfig
type ConfigPatch = contracts.ConfigPatch

// Map translates host config concepts to the current internal
// Goncho service contract. Unsupported fields are returned as diagnostics
// instead of being silently widened or accepted.
func Map(input Input) Mapping {
	host := contracts.NormalizeHost(input.Host)
	defaults, ok := policy.DefaultsForHost(host)
	if !ok {
		defaults = policy.HostDefaults{
			Workspace:       "default",
			AIPeer:          "gormes",
			SessionStrategy: "per-session",
			RecallMode:      "hybrid",
		}
	}

	compat := HonchoExternalCompatibility()
	out := Mapping{
		Host:              host,
		WorkspaceID:       contracts.FirstNonBlank(input.Workspace, defaults.Workspace, "default"),
		UserPeerID:        strings.TrimSpace(input.PeerName),
		AIPeerID:          contracts.FirstNonBlank(input.AIPeer, defaults.AIPeer, "gormes"),
		InternalService:   compat.InternalService,
		ExternalToolNames: append([]string(nil), compat.ExternalToolNames...),
	}
	if !ok {
		out.Unsupported = append(out.Unsupported, UnsupportedMapping{
			Field:  "host",
			Value:  strings.TrimSpace(input.Host),
			Reason: "host has no Goncho compatibility defaults",
		})
	}
	if out.UserPeerID == "" {
		out.Unsupported = append(out.Unsupported, UnsupportedMapping{
			Field:  "peer_name",
			Value:  "",
			Reason: "host mappings require an explicit durable user peer",
		})
	}

	out.SessionStrategy = policy.NormalizeSessionStrategy(contracts.FirstNonBlank(input.SessionStrategy, defaults.SessionStrategy))
	out.SessionKey = policy.SessionKeyForStrategy(host, out.SessionStrategy, input, &out.Unsupported)

	recallMode, ok := policy.NormalizeRecallMode(contracts.FirstNonBlank(input.RecallMode, defaults.RecallMode))
	if !ok {
		out.Unsupported = append(out.Unsupported, UnsupportedMapping{
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
