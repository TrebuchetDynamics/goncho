package goncho

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type Kind string

const (
	KindConclusion Kind = "conclusion"
	KindMemory     Kind = "memory"
	KindProfile    Kind = "profile"
	KindSummary    Kind = "summary"
	KindPreference Kind = "preference"
	KindFact       Kind = "fact"
	KindDecision   Kind = "decision"
)

type Scope string

const (
	ScopePrivate   Scope = "private"
	ScopeWorkspace Scope = "workspace"
	ScopeGlobal    Scope = "global"
	ScopeProject   Scope = "project"
	ScopeTask      Scope = "task"
)

type GoalStatus string

const (
	GoalActive    GoalStatus = "active"
	GoalCompleted GoalStatus = "completed"
	GoalArchived  GoalStatus = "archived"
)

type Memory struct {
	ID           string
	Kind         Kind
	Content      string
	PeerID       string
	WorkspaceID  string
	Scope        Scope
	ContextID    string
	Importance   float64
	ValidFrom    time.Time
	ValidUntil   time.Time
	SupersedesID string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Checksum     string
}

func (m *Memory) IsExpired(now time.Time) bool {
	if m.ValidUntil.IsZero() {
		return false
	}
	return now.After(m.ValidUntil)
}

type Relation struct {
	ID           int64
	SourceID     string
	TargetEntity string
	RelationType string
	Confidence   float64
	CreatedAt    time.Time
}

type Goal struct {
	ID        string
	Name      string
	Status    GoalStatus
	ParentID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StoreParams struct {
	Content     string
	Kind        Kind
	PeerID      string
	WorkspaceID string
	Scope       Scope
	ContextID   string
	Importance  float64
	GoalID      string
}

type StoreResult struct {
	Memory Memory
}

type UpdateParams struct {
	ID         string
	Content    string
	Reason     string
	Importance float64
}

type UpdateResult struct {
	Memory    Memory
	Supersede Memory
}

type ForgetParams struct {
	Reason string
}

type RetrieveParams struct {
	Query       string
	PeerID      string
	WorkspaceID string
	ContextID   string
	Kinds       []Kind
	Scopes      []Scope
	Limit       int
}

type RetrieveResult struct {
	Memories []Memory
	Trace    RetrieveTrace
}

type RetrieveTrace struct {
	FTSHits          int
	GraphHits        int
	CandidatesScored int
	MMRDiversity     float64
	Warnings         []string
}

type ContextParams struct {
	PeerID      string
	WorkspaceID string
	ContextID   string
	MaxTokens   int
}

type ContextResult struct {
	Memories []Memory
	TokenEst int
}

type GoalParams struct {
	Name     string
	ParentID string
}

type GoalResult struct {
	Goal Goal
}

func checksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
