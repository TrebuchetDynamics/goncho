package goncho

import "context"

// RelationVerb names a Goncho-native relation between two memories. The
// vocabulary is intentionally Honcho-compatible and does not surface
// Engram-specific identifiers in any public Gormes/Honcho API.
type RelationVerb string

// Canonical Goncho relation vocabulary. New verbs must be added here so the
// pending-relation path remains deterministic and reviewable.
const (
	// RelationRelated is the catch-all "see also" link between two memories.
	RelationRelated RelationVerb = "related"
	// RelationConflicts marks a contradiction that may need human or judge
	// resolution; the original write still succeeds and the candidate is
	// stored as PENDING.
	RelationConflicts RelationVerb = "conflicts_with"
	// RelationSupersedes marks the new memory as replacing an older one.
	RelationSupersedes RelationVerb = "supersedes"
	// RelationCompatible marks two memories as not in conflict and reinforcing.
	RelationCompatible RelationVerb = "compatible"
	// RelationScoped marks a relation that only holds inside a narrower scope
	// (e.g. one session or one peer card slot).
	RelationScoped RelationVerb = "scoped"
	// RelationNotConflict explicitly records that a candidate inspected by the
	// detector was found to be not in conflict, useful for audit trails.
	RelationNotConflict RelationVerb = "not_conflict"
)

// RelationCandidate is one PENDING relation between two memories awaiting
// later judgment. The detector returns these synchronously after a write
// completes; storage must not block the originating write.
type RelationCandidate struct {
	From string       // memory ID the relation originates from
	To   string       // memory ID the relation targets
	Verb RelationVerb // canonical Goncho relation vocabulary
	Note string       // free-form note for later human/LLM judgment
}

// RelationStore persists pending relation candidates. Implementations must
// treat the candidates as PENDING — no judgment, no LLM, no resolution.
type RelationStore interface {
	SavePending(ctx context.Context, candidates []RelationCandidate) error
}

// CandidateDetector inspects a write against existing memories and returns
// pending relation candidates. Returning an error must not cause the
// originating write to fail; callers record the failure as evidence and
// proceed.
type CandidateDetector interface {
	Detect(ctx context.Context, w MemoryWrite, prior []MemoryWrite) ([]RelationCandidate, error)
}
