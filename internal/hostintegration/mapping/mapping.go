// Package mapping translates generic host config concepts into Goncho's
// internal host-integration contract.
package mapping

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/policy"
)

// Map translates host config concepts to the current internal Goncho service
// contract. Unsupported fields are returned as diagnostics instead of being
// silently widened or accepted.
func Map(input contracts.Input) contracts.Mapping {
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

	compat := contracts.HonchoExternalCompatibility()
	out := contracts.Mapping{
		Host:              host,
		WorkspaceID:       contracts.FirstNonBlank(input.Workspace, defaults.Workspace, "default"),
		UserPeerID:        strings.TrimSpace(input.PeerName),
		AIPeerID:          contracts.FirstNonBlank(input.AIPeer, defaults.AIPeer, "gormes"),
		InternalService:   compat.InternalService,
		ExternalToolNames: append([]string(nil), compat.ExternalToolNames...),
	}
	if !ok {
		contracts.AddUnsupported(&out.Unsupported, "host", strings.TrimSpace(input.Host), "host has no Goncho compatibility defaults")
	}
	if out.UserPeerID == "" {
		contracts.AddUnsupported(&out.Unsupported, "peer_name", "", "host mappings require an explicit durable user peer")
	}

	out.SessionStrategy = policy.NormalizeSessionStrategy(contracts.FirstNonBlank(input.SessionStrategy, defaults.SessionStrategy))
	out.SessionKey = policy.SessionKeyForStrategy(host, out.SessionStrategy, input, &out.Unsupported)

	recallMode, ok := policy.NormalizeRecallMode(contracts.FirstNonBlank(input.RecallMode, defaults.RecallMode))
	if !ok {
		contracts.AddUnsupported(&out.Unsupported, "recall_mode", strings.TrimSpace(input.RecallMode), "supported recall modes are context, tools, and hybrid")
	} else {
		out.RecallMode = recallMode
		out.InjectContext = recallMode == "context" || recallMode == "hybrid"
		out.ExposeTools = recallMode == "tools" || recallMode == "hybrid"
	}

	return out
}
