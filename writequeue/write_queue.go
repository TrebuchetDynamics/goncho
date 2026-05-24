package writequeue

import (
	"context"
	"sync"
)

// MemoryWrite is the Goncho-native shape submitted to the serialized write
// queue. It is intentionally narrower than the full Honcho message envelope:
// the queue only needs a stable ID and the content/tags the detector can
// inspect to surface relation candidates.
type MemoryWrite struct {
	ID      string
	Content string
	Tags    []string
}

// MemoryStore is the seam the write queue uses to persist memories. Apply
// must be safe to call from a single goroutine; the queue serializes calls.
type MemoryStore interface {
	Apply(ctx context.Context, w MemoryWrite) error
	List(ctx context.Context) ([]MemoryWrite, error)
}

// Write evidence codes returned by Submit. Keep these in sync with operator
// docs; they are part of the public contract.
const (
	// WriteEvidenceComplete signals the originating write succeeded and
	// (if applicable) any pending relation candidates were stored.
	WriteEvidenceComplete = "goncho_write_complete"
	// WriteEvidenceCancelled signals the submission was cancelled before the
	// write started; storage was not mutated.
	WriteEvidenceCancelled = "goncho_write_cancelled"
	// WriteEvidenceRelationFailed signals the originating write succeeded but
	// candidate detection failed; no relations were stored. The write is
	// authoritative and is not rolled back.
	WriteEvidenceRelationFailed = "goncho_relation_detection_failed"
	// WriteEvidenceWriteFailed signals the originating write itself failed;
	// no relations are recorded and the caller must retry or surface the
	// underlying error.
	WriteEvidenceWriteFailed = "goncho_write_failed"
)

// WriteEvidence is the Goncho-native operator evidence for one Submit call.
// It is the only return shape from the queue; callers MUST inspect Code and
// WriteOK before treating Candidates as authoritative.
type WriteEvidence struct {
	Code       string
	WriteOK    bool
	Candidates []RelationCandidate
	Err        error
}

// WriteQueue serializes memory writes and records pending relation candidates
// without blocking the originating write. The queue uses a single mutex to
// guarantee mutex-serialized order; concurrent submitters are safe but the
// deterministic acceptance test exercises sequential submission so the
// recorded order is reproducible.
type WriteQueue struct {
	mu        sync.Mutex
	store     MemoryStore
	detector  CandidateDetector
	relations RelationStore

	cancelMu  sync.Mutex
	cancelled map[string]struct{}
	consumed  map[string]struct{}
}

// NewWriteQueue constructs a queue from the three required seams. Pass nil
// for relations if relation candidate storage is intentionally disabled; in
// that case detected candidates are still attached to the returned evidence
// for the caller to inspect.
func NewWriteQueue(store MemoryStore, detector CandidateDetector, relations RelationStore) *WriteQueue {
	return &WriteQueue{
		store:     store,
		detector:  detector,
		relations: relations,
		cancelled: make(map[string]struct{}),
		consumed:  make(map[string]struct{}),
	}
}

// Cancel marks the given write ID as cancelled. Returns true if the
// cancellation was registered before the write started, false if the write
// has already begun, completed, or the ID was already consumed.
func (q *WriteQueue) Cancel(id string) bool {
	if id == "" {
		return false
	}
	q.cancelMu.Lock()
	defer q.cancelMu.Unlock()
	if _, ok := q.consumed[id]; ok {
		return false
	}
	if _, ok := q.cancelled[id]; ok {
		// Idempotent: already cancelled, still "before start", report true.
		return true
	}
	q.cancelled[id] = struct{}{}
	return true
}

// Submit serializes the write through the queue. The originating memory
// write is authoritative: relation detection runs after a successful write
// and its failure is recorded as evidence without rolling the write back.
func (q *WriteQueue) Submit(ctx context.Context, w MemoryWrite) WriteEvidence {
	// Mutex-serialized: only one Submit progresses past this lock at a time.
	q.mu.Lock()
	defer q.mu.Unlock()

	// Cancellation check: must happen BEFORE any mutation. If cancelled,
	// mark consumed so a later Cancel for the same ID returns false.
	if q.takeCancelled(w.ID) {
		return WriteEvidence{
			Code:    WriteEvidenceCancelled,
			WriteOK: false,
		}
	}

	// Originating memory write. If this fails, we record the failure and do
	// NOT run relation detection; relations only apply to successful writes.
	if err := q.store.Apply(ctx, w); err != nil {
		q.markConsumed(w.ID)
		return WriteEvidence{
			Code:    WriteEvidenceWriteFailed,
			WriteOK: false,
			Err:     err,
		}
	}
	q.markConsumed(w.ID)

	// From here on, the write is authoritative; any relation work is
	// best-effort and must not flip WriteOK back to false.
	prior, err := q.priorMemoriesExcluding(ctx, w.ID)
	if err != nil {
		return WriteEvidence{
			Code:    WriteEvidenceRelationFailed,
			WriteOK: true,
			Err:     err,
		}
	}

	if q.detector == nil {
		return WriteEvidence{Code: WriteEvidenceComplete, WriteOK: true}
	}

	candidates, err := q.detector.Detect(ctx, w, prior)
	if err != nil {
		return WriteEvidence{
			Code:    WriteEvidenceRelationFailed,
			WriteOK: true,
			Err:     err,
		}
	}
	if len(candidates) == 0 {
		return WriteEvidence{Code: WriteEvidenceComplete, WriteOK: true}
	}
	candidates = withDefaultRelationEvidenceIDs(candidates)
	if q.relations != nil {
		if err := q.relations.SavePending(ctx, candidates); err != nil {
			return WriteEvidence{
				Code:       WriteEvidenceRelationFailed,
				WriteOK:    true,
				Candidates: nil,
				Err:        err,
			}
		}
	}
	return WriteEvidence{
		Code:       WriteEvidenceComplete,
		WriteOK:    true,
		Candidates: candidates,
	}
}

// priorMemoriesExcluding returns prior memories that are not the just-written
// one, so the detector compares the new write against existing context.
func (q *WriteQueue) priorMemoriesExcluding(ctx context.Context, id string) ([]MemoryWrite, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MemoryWrite, 0, len(all))
	for _, m := range all {
		if m.ID == id {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func withDefaultRelationEvidenceIDs(candidates []RelationCandidate) []RelationCandidate {
	out := make([]RelationCandidate, len(candidates))
	copy(out, candidates)
	for i := range out {
		if len(out[i].EvidenceIDs) != 0 {
			out[i].EvidenceIDs = append([]string(nil), out[i].EvidenceIDs...)
			continue
		}
		out[i].EvidenceIDs = relationMemoryEvidenceIDs(out[i].From, out[i].To)
	}
	return out
}

func relationMemoryEvidenceIDs(ids ...string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		evidenceID := "memory:" + id
		if _, ok := seen[evidenceID]; ok {
			continue
		}
		seen[evidenceID] = struct{}{}
		out = append(out, evidenceID)
	}
	return out
}

// takeCancelled atomically checks-and-clears a pending cancellation,
// returning true if the caller should treat the submission as cancelled.
// Once consumed, the same ID cannot be cancelled again.
func (q *WriteQueue) takeCancelled(id string) bool {
	q.cancelMu.Lock()
	defer q.cancelMu.Unlock()
	if _, ok := q.cancelled[id]; ok {
		delete(q.cancelled, id)
		q.consumed[id] = struct{}{}
		return true
	}
	return false
}

// markConsumed records that the given ID has progressed past the cancellation
// check; subsequent Cancel calls for it must return false.
func (q *WriteQueue) markConsumed(id string) {
	q.cancelMu.Lock()
	defer q.cancelMu.Unlock()
	q.consumed[id] = struct{}{}
}
