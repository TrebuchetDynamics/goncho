package goncho

import (
	"database/sql"

	dynamicagents "github.com/TrebuchetDynamics/goncho/internal/dynamicagents"
)

// ErrAgentIDReserved is returned by Create when the requested name would
// normalize to an AgentID that is already claimed by the static config
// (passed via CreateAgentOptions.ReservedIDs). Operator-defined identity in
// config.toml must not be silently shadowed by a runtime spawn.
var ErrAgentIDReserved = dynamicagents.ErrAgentIDReserved

// ErrAgentIDInvalid is returned when the normalized AgentID would not match
// the pattern accepted by config.AgentsCfg (^[a-z][a-z0-9_-]{0,63}$). The
// caller should report the underlying name back to the operator.
var ErrAgentIDInvalid = dynamicagents.ErrAgentIDInvalid

// AgentRecord describes a runtime-spawned agent persisted in the dynamic
// registry. Static config.AgentCfg remains the operator-defined surface;
// AgentRecord is the runtime overlay layered on top of it.
type AgentRecord = dynamicagents.AgentRecord

// CreateAgentOptions parameterizes DynamicAgentRegistry.Create. Name is
// required; the registry normalizes it to an AgentID compatible with
// config.AgentsCfg. ReservedIDs (typically the set of static AgentCfg.IDs
// observed at the time of the call) prevents the runtime registry from
// silently shadowing an operator-defined identity.
type CreateAgentOptions = dynamicagents.CreateAgentOptions

// DynamicAgentRegistry persists runtime-spawned agents and their channel
// bindings in the Goncho SQLite database. The registry knows nothing about
// the gateway resolver or channel adapters; callers compose it with the
// existing config.AgentsCfg overlay at the gateway seam.
type DynamicAgentRegistry = dynamicagents.DynamicAgentRegistry

// BindingMatch describes a (channel, peer) tuple that should resolve to a
// dynamic AgentID at runtime. ThreadID is optional and stored as an empty
// string when absent; matches are scoped exactly so the General topic of a
// Telegram forum and one of its named topics never share a binding row.
type BindingMatch = dynamicagents.BindingMatch

// NewDynamicAgentRegistry opens (or migrates) the dynamic agent tables and
// returns a registry bound to db. The DDL is idempotent — calling the
// constructor twice on the same database is safe.
func NewDynamicAgentRegistry(db *sql.DB) (*DynamicAgentRegistry, error) {
	return dynamicagents.NewDynamicAgentRegistry(db)
}
