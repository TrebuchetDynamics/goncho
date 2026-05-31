// Package config applies host-scoped host integration config changes.
package config

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/hostintegration/contracts"
)

// ApplyPatch applies host-scoped config writes without mutating the input
// document or sibling host entries.
func ApplyPatch(doc contracts.ConfigDocument, host string, patch contracts.ConfigPatch) (contracts.ConfigDocument, error) {
	host = contracts.NormalizeHost(host)
	if host == "" {
		return contracts.ConfigDocument{}, fmt.Errorf("goncho: host is required")
	}

	out := doc
	out.Hosts = make(map[string]contracts.RuntimeConfig, len(doc.Hosts)+1)
	for key, value := range doc.Hosts {
		out.Hosts[contracts.NormalizeHost(key)] = value
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
