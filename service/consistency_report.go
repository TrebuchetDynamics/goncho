package goncho

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/timeutil"
)

const (
	ConsistencyDuplicate = "duplicate"
	ConsistencyConflict  = "conflict"
)

type ConsistencyReportParams struct {
	ProfileID string `json:"profile_id,omitempty"`
	Peer      string `json:"peer,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ConsistencyReport struct {
	Status      string             `json:"status"`
	Mutates     bool               `json:"mutates"`
	WorkspaceID string             `json:"workspace_id"`
	GeneratedAt time.Time          `json:"generated_at"`
	Counts      map[string]int     `json:"counts"`
	Groups      []ConsistencyGroup `json:"groups"`
}

type ConsistencyGroup struct {
	Kind     string              `json:"kind"`
	Scope    string              `json:"scope"`
	ScopeID  string              `json:"scope_id"`
	Peer     string              `json:"peer"`
	Subject  string              `json:"subject,omitempty"`
	Relation string              `json:"relation,omitempty"`
	Members  []ConsistencyMember `json:"members"`
}

type ConsistencyMember struct {
	ID         int64     `json:"id"`
	StableID   string    `json:"stable_id"`
	ProfileID  string    `json:"profile_id,omitempty"`
	Peer       string    `json:"peer"`
	SessionKey string    `json:"session_key,omitempty"`
	Scope      string    `json:"scope"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	FactObject string    `json:"fact_object,omitempty"`
}

type consistencyConflictGroup struct {
	group   ConsistencyGroup
	objects map[string]struct{}
}

// ConsistencyReport adapts the read-only stewardship scans used by typed memory
// systems: it finds exact duplicate claims and deterministic fact conflicts
// without promoting, superseding, or deleting memory.
func (s *Service) ConsistencyReport(ctx context.Context, params ConsistencyReportParams) (ConsistencyReport, error) {
	if err := ctx.Err(); err != nil {
		return ConsistencyReport{}, err
	}
	if s == nil || s.db == nil {
		return ConsistencyReport{}, fmt.Errorf("goncho: nil service")
	}
	query := `SELECT id, profile_id, peer_id, COALESCE(session_key,''), content, scope, created_at FROM goncho_conclusions WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed','active')`
	args := []any{s.workspaceID, s.observer}
	if profileID := strings.TrimSpace(params.ProfileID); profileID != "" {
		query += ` AND profile_id = ?`
		args = append(args, profileID)
	}
	if peer := strings.TrimSpace(params.Peer); peer != "" {
		query += ` AND peer_id = ?`
		args = append(args, peer)
	}
	query += ` ORDER BY created_at ASC, id ASC LIMIT ?`
	args = append(args, limitutil.DefaultClamped(params.Limit, 500, 5000))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ConsistencyReport{}, fmt.Errorf("goncho: consistency report conclusions: %w", err)
	}
	defer rows.Close()

	duplicates := map[string][]ConsistencyMember{}
	conflicts := map[string]*consistencyConflictGroup{}
	for rows.Next() {
		var member ConsistencyMember
		var created int64
		if err := rows.Scan(&member.ID, &member.ProfileID, &member.Peer, &member.SessionKey, &member.Content, &member.Scope, &created); err != nil {
			return ConsistencyReport{}, fmt.Errorf("goncho: scan consistency report conclusion: %w", err)
		}
		member.StableID = fmt.Sprintf("conclusion:%d", member.ID)
		member.CreatedAt = time.Unix(created, 0).UTC()
		scopeID := consistencyScopeID(s.workspaceID, member)
		baseKey := member.Peer + "\x00" + scopeID
		if normalized := normalizeFactText(member.Content); normalized != "" {
			duplicates[baseKey+"\x00"+normalized] = append(duplicates[baseKey+"\x00"+normalized], member)
		}
		fact, ok := parseMemoryFact(member.Content)
		if !ok {
			continue
		}
		key := baseKey + "\x00" + fact.subject + "\x00" + fact.relation
		group := conflicts[key]
		if group == nil {
			group = &consistencyConflictGroup{group: ConsistencyGroup{Kind: ConsistencyConflict, Scope: member.Scope, ScopeID: scopeID, Peer: member.Peer, Subject: fact.subject, Relation: fact.relation}, objects: map[string]struct{}{}}
			conflicts[key] = group
		}
		member.FactObject = fact.object
		group.group.Members = append(group.group.Members, member)
		group.objects[fact.object] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return ConsistencyReport{}, fmt.Errorf("goncho: iterate consistency report conclusions: %w", err)
	}

	groups := []ConsistencyGroup{}
	for _, members := range duplicates {
		if len(members) < 2 {
			continue
		}
		groups = append(groups, ConsistencyGroup{Kind: ConsistencyDuplicate, Scope: members[0].Scope, ScopeID: consistencyScopeID(s.workspaceID, members[0]), Peer: members[0].Peer, Members: members})
	}
	for _, conflict := range conflicts {
		if len(conflict.objects) > 1 {
			groups = append(groups, conflict.group)
		}
	}
	for i := range groups {
		sort.Slice(groups[i].Members, func(a, b int) bool { return groups[i].Members[a].ID < groups[i].Members[b].ID })
	}
	sort.Slice(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		return strings.Join([]string{a.Kind, a.ScopeID, a.Peer, a.Subject, a.Relation}, "\x00") < strings.Join([]string{b.Kind, b.ScopeID, b.Peer, b.Subject, b.Relation}, "\x00")
	})
	counts := map[string]int{ConsistencyDuplicate: 0, ConsistencyConflict: 0}
	for _, group := range groups {
		counts[group.Kind]++
	}
	return ConsistencyReport{Status: "ok", Mutates: false, WorkspaceID: s.workspaceID, GeneratedAt: timeutil.NowUTC(), Counts: counts, Groups: groups}, nil
}

func consistencyScopeID(workspaceID string, member ConsistencyMember) string {
	switch normalizeMemoryScope(member.Scope, member.ProfileID) {
	case MemoryScopeProfile:
		return workspaceID + "/profile/" + member.ProfileID
	case MemoryScopeSession:
		return workspaceID + "/profile/" + member.ProfileID + "/session/" + member.SessionKey
	case MemoryScopeGlobal:
		return MemoryScopeGlobal
	default:
		return workspaceID + "/" + normalizeMemoryScope(member.Scope, member.ProfileID)
	}
}
