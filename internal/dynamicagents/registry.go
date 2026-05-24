package dynamicagents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrAgentIDReserved is returned by Create when the requested name would
// normalize to an AgentID that is already claimed by the static config
// (passed via CreateAgentOptions.ReservedIDs). Operator-defined identity in
// config.toml must not be silently shadowed by a runtime spawn.
var ErrAgentIDReserved = errors.New("goncho: dynamic agent id reserved by static config")

// ErrAgentIDInvalid is returned when the normalized AgentID would not match
// the pattern accepted by config.AgentsCfg (^[a-z][a-z0-9_-]{0,63}$). The
// caller should report the underlying name back to the operator.
var ErrAgentIDInvalid = errors.New("goncho: dynamic agent name does not normalize to a valid id")

// dynamicAgentIDPattern mirrors config.AgentsCfg's agentIDPattern so dynamic
// records can be passed through code paths that accept the same string set
// without re-validating.
var dynamicAgentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// AgentRecord describes a runtime-spawned agent persisted in the dynamic
// registry. Static config.AgentCfg remains the operator-defined surface;
// AgentRecord is the runtime overlay layered on top of it.
type AgentRecord struct {
	ID        string
	Name      string
	Persona   string
	CreatedAt time.Time
}

// CreateAgentOptions parameterizes DynamicAgentRegistry.Create. Name is
// required; the registry normalizes it to an AgentID compatible with
// config.AgentsCfg. ReservedIDs (typically the set of static AgentCfg.IDs
// observed at the time of the call) prevents the runtime registry from
// silently shadowing an operator-defined identity.
type CreateAgentOptions struct {
	Name        string
	Persona     string
	ReservedIDs map[string]struct{}
}

// DynamicAgentRegistry persists runtime-spawned agents and their channel
// bindings in the Goncho SQLite database. The registry knows nothing about
// the gateway resolver or channel adapters; callers compose it with the
// existing config.AgentsCfg overlay at the gateway boundary.
type DynamicAgentRegistry struct {
	db  *sql.DB
	now func() time.Time
}

// NewDynamicAgentRegistry opens (or migrates) the dynamic agent tables and
// returns a registry bound to db. The DDL is idempotent — calling the
// constructor twice on the same database is safe.
func NewDynamicAgentRegistry(db *sql.DB) (*DynamicAgentRegistry, error) {
	if db == nil {
		return nil, errors.New("goncho: NewDynamicAgentRegistry requires a non-nil *sql.DB")
	}
	if err := ensureDynamicAgentTables(context.Background(), db); err != nil {
		return nil, err
	}
	return &DynamicAgentRegistry{db: db, now: time.Now}, nil
}

// Create inserts a new dynamic agent. Returns ErrAgentIDReserved if the
// normalized AgentID is present in opts.ReservedIDs, and ErrAgentIDInvalid
// if the name does not normalize to a config-compatible AgentID.
func (r *DynamicAgentRegistry) Create(ctx context.Context, opts CreateAgentOptions) (AgentRecord, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return AgentRecord{}, errors.New("goncho: dynamic agent name is required")
	}
	id := normalizeDynamicAgentID(name)
	if !dynamicAgentIDPattern.MatchString(id) {
		return AgentRecord{}, fmt.Errorf("%w: %q", ErrAgentIDInvalid, name)
	}
	if _, claimed := opts.ReservedIDs[id]; claimed {
		return AgentRecord{}, fmt.Errorf("%w: %q", ErrAgentIDReserved, id)
	}

	createdAt := r.now().UTC()
	persona := strings.TrimSpace(opts.Persona)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO goncho_dynamic_agents (id, name, persona, created_at)
		VALUES (?, ?, ?, ?)
	`, id, name, persona, createdAt.Unix())
	if err != nil {
		return AgentRecord{}, fmt.Errorf("goncho: insert dynamic agent: %w", err)
	}
	return AgentRecord{
		ID:        id,
		Name:      name,
		Persona:   persona,
		CreatedAt: createdAt,
	}, nil
}

// List returns every dynamic AgentRecord ordered by creation time, oldest
// first. Empty registries return an empty slice with no error.
func (r *DynamicAgentRegistry) List(ctx context.Context) ([]AgentRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, persona, created_at
		FROM goncho_dynamic_agents
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("goncho: list dynamic agents: %w", err)
	}
	defer rows.Close()

	var out []AgentRecord
	for rows.Next() {
		var (
			rec       AgentRecord
			createdAt int64
		)
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Persona, &createdAt); err != nil {
			return nil, fmt.Errorf("goncho: scan dynamic agent: %w", err)
		}
		rec.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: list dynamic agents iterate: %w", err)
	}
	return out, nil
}

// BindingMatch describes a (channel, peer) tuple that should resolve to a
// dynamic AgentID at runtime. ThreadID is optional and stored as an empty
// string when absent; matches are scoped exactly so the General topic of a
// Telegram forum and one of its named topics never share a binding row.
type BindingMatch struct {
	Channel  string
	PeerKind string
	PeerID   string
	ThreadID string
}

// Bind associates agentID with match. Re-binding the same tuple replaces
// the previous AgentID — the most recent runtime decision wins.
func (r *DynamicAgentRegistry) Bind(ctx context.Context, agentID string, match BindingMatch) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return errors.New("goncho: agent_id is required for Bind")
	}
	match = normalizeBindingMatch(match)
	if err := validateBindingMatch(match); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO goncho_dynamic_agent_bindings
			(channel, peer_kind, peer_id, thread_id, agent_id, bound_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(channel, peer_kind, peer_id, thread_id) DO UPDATE SET
			agent_id = excluded.agent_id,
			bound_at = excluded.bound_at
	`, match.Channel, match.PeerKind, match.PeerID, match.ThreadID, agentID, r.now().UTC().Unix())
	if err != nil {
		return fmt.Errorf("goncho: bind dynamic agent: %w", err)
	}
	return nil
}

// Unbind removes the binding for match. Unbinding an unknown tuple is a
// no-op so callers can call Unbind defensively without checking Resolve
// first. The associated AgentRecord is preserved — Unbind only releases
// the (channel, peer, thread) -> agent mapping.
func (r *DynamicAgentRegistry) Unbind(ctx context.Context, match BindingMatch) error {
	match = normalizeBindingMatch(match)
	if err := validateBindingMatch(match); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM goncho_dynamic_agent_bindings
		WHERE channel = ? AND peer_kind = ? AND peer_id = ? AND thread_id = ?
	`, match.Channel, match.PeerKind, match.PeerID, match.ThreadID)
	if err != nil {
		return fmt.Errorf("goncho: unbind dynamic agent: %w", err)
	}
	return nil
}

// Resolve returns the dynamic AgentID bound to match, if any. The second
// return value is false (no error) when no binding exists; callers should
// then fall back to static config.AgentBindingCfg at the gateway boundary.
func (r *DynamicAgentRegistry) Resolve(ctx context.Context, match BindingMatch) (string, bool, error) {
	match = normalizeBindingMatch(match)
	if err := validateBindingMatch(match); err != nil {
		return "", false, err
	}
	var agentID string
	err := r.db.QueryRowContext(ctx, `
		SELECT agent_id FROM goncho_dynamic_agent_bindings
		WHERE channel = ? AND peer_kind = ? AND peer_id = ? AND thread_id = ?
	`, match.Channel, match.PeerKind, match.PeerID, match.ThreadID).Scan(&agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("goncho: resolve dynamic agent binding: %w", err)
	}
	return agentID, true, nil
}

func normalizeBindingMatch(m BindingMatch) BindingMatch {
	m.Channel = strings.ToLower(strings.TrimSpace(m.Channel))
	m.PeerKind = strings.ToLower(strings.TrimSpace(m.PeerKind))
	m.PeerID = strings.TrimSpace(m.PeerID)
	m.ThreadID = strings.TrimSpace(m.ThreadID)
	return m
}

func validateBindingMatch(m BindingMatch) error {
	if m.Channel == "" {
		return errors.New("goncho: binding match channel is required")
	}
	if m.PeerKind == "" {
		return errors.New("goncho: binding match peer_kind is required")
	}
	if m.PeerID == "" {
		return errors.New("goncho: binding match peer_id is required")
	}
	return nil
}

// Get returns the AgentRecord for id, if any. The second return value is
// false (no error) when the id is unknown.
func (r *DynamicAgentRegistry) Get(ctx context.Context, id string) (AgentRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return AgentRecord{}, false, nil
	}
	var (
		name      string
		persona   string
		createdAt int64
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT name, persona, created_at
		FROM goncho_dynamic_agents
		WHERE id = ?
	`, id).Scan(&name, &persona, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentRecord{}, false, nil
	}
	if err != nil {
		return AgentRecord{}, false, fmt.Errorf("goncho: get dynamic agent: %w", err)
	}
	return AgentRecord{
		ID:        id,
		Name:      name,
		Persona:   persona,
		CreatedAt: time.Unix(createdAt, 0).UTC(),
	}, true, nil
}

// normalizeDynamicAgentID lowercases, replaces unsupported characters with
// '-', collapses runs, trims edge dashes, and caps to 64 chars so the result
// satisfies dynamicAgentIDPattern when the input contains at least one
// ASCII letter or digit.
func normalizeDynamicAgentID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == ' ':
			b.WriteRune('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	if len(out) > 64 {
		out = strings.Trim(out[:64], "-")
	}
	return out
}

func ensureDynamicAgentTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS goncho_dynamic_agents (
			id         TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 64),
			name       TEXT NOT NULL,
			persona    TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_goncho_dynamic_agents_created
			ON goncho_dynamic_agents(created_at);
		CREATE TABLE IF NOT EXISTS goncho_dynamic_agent_bindings (
			channel    TEXT NOT NULL,
			peer_kind  TEXT NOT NULL,
			peer_id    TEXT NOT NULL,
			thread_id  TEXT NOT NULL DEFAULT '',
			agent_id   TEXT NOT NULL,
			bound_at   INTEGER NOT NULL,
			PRIMARY KEY (channel, peer_kind, peer_id, thread_id)
		);
		CREATE INDEX IF NOT EXISTS idx_goncho_dynamic_agent_bindings_agent
			ON goncho_dynamic_agent_bindings(agent_id);
	`)
	if err != nil {
		return fmt.Errorf("goncho: ensure dynamic agent tables: %w", err)
	}
	return nil
}
