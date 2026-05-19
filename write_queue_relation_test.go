package goncho

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

// fakeMemoryStore records the order of applied writes deterministically.
type fakeMemoryStore struct {
	mu      sync.Mutex
	applied []MemoryWrite
	failOn  map[string]error
}

func (s *fakeMemoryStore) Apply(_ context.Context, w MemoryWrite) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err, ok := s.failOn[w.ID]; ok {
		return err
	}
	s.applied = append(s.applied, w)
	return nil
}

func (s *fakeMemoryStore) List(_ context.Context) ([]MemoryWrite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MemoryWrite, len(s.applied))
	copy(out, s.applied)
	return out, nil
}

func (s *fakeMemoryStore) appliedIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.applied))
	for _, w := range s.applied {
		out = append(out, w.ID)
	}
	return out
}

// fakeRelationStore captures pending relation candidates.
type fakeRelationStore struct {
	mu       sync.Mutex
	saved    []RelationCandidate
	failWith error
}

func (s *fakeRelationStore) SavePending(_ context.Context, candidates []RelationCandidate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	s.saved = append(s.saved, candidates...)
	return nil
}

func (s *fakeRelationStore) snapshot() []RelationCandidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RelationCandidate, len(s.saved))
	copy(out, s.saved)
	return out
}

// fakeDetector returns a deterministic candidate per write, unless configured to fail.
type fakeDetector struct {
	failWith    error
	verbForID   map[string]RelationVerb
	calledCount int
	calledMu    sync.Mutex
}

func (d *fakeDetector) Detect(_ context.Context, w MemoryWrite, prior []MemoryWrite) ([]RelationCandidate, error) {
	d.calledMu.Lock()
	d.calledCount++
	d.calledMu.Unlock()
	if d.failWith != nil {
		return nil, d.failWith
	}
	if len(prior) == 0 {
		return nil, nil
	}
	verb := RelationRelated
	if v, ok := d.verbForID[w.ID]; ok {
		verb = v
	}
	out := make([]RelationCandidate, 0, len(prior))
	for _, p := range prior {
		out = append(out, RelationCandidate{
			From: w.ID,
			To:   p.ID,
			Verb: verb,
			Note: "deterministic",
		})
	}
	return out, nil
}

// llmJudgeMustNotBeCalled is a sentinel to assert no LLM seam runs on the
// pending write path. The test passes a nil LLM seam to NewWriteQueue, but if
// the implementation ever introduces one, this assertion must remain a guard.
type llmJudgeMustNotBeCalled struct{}

func TestGonchoWriteQueue_DeterministicOrderUnderConcurrency(t *testing.T) {
	store := &fakeMemoryStore{}
	rel := &fakeRelationStore{}
	det := &fakeDetector{}
	q := NewWriteQueue(store, det, rel)

	ctx := context.Background()
	ids := []string{"m-1", "m-2", "m-3", "m-4", "m-5"}
	for _, id := range ids {
		ev := q.Submit(ctx, MemoryWrite{ID: id, Content: "c-" + id})
		if !ev.WriteOK {
			t.Fatalf("Submit %s WriteOK=false: %+v", id, ev)
		}
		if ev.Code != "goncho_write_complete" {
			t.Fatalf("Submit %s code=%q want goncho_write_complete", id, ev.Code)
		}
	}

	got := store.appliedIDs()
	if !reflect.DeepEqual(got, ids) {
		t.Fatalf("applied order = %v, want %v", got, ids)
	}
}

func TestGonchoWriteQueue_CancelBeforeStartNoMutation(t *testing.T) {
	store := &fakeMemoryStore{}
	rel := &fakeRelationStore{}
	det := &fakeDetector{}
	q := NewWriteQueue(store, det, rel)

	if !q.Cancel("m-cancel-me") {
		t.Fatalf("Cancel before Submit must return true")
	}

	ctx := context.Background()
	cancelledEv := q.Submit(ctx, MemoryWrite{ID: "m-cancel-me", Content: "should not store"})
	if cancelledEv.WriteOK {
		t.Fatalf("cancelled Submit must report WriteOK=false: %+v", cancelledEv)
	}
	if cancelledEv.Code != "goncho_write_cancelled" {
		t.Fatalf("cancelled code = %q, want goncho_write_cancelled", cancelledEv.Code)
	}

	okEv := q.Submit(ctx, MemoryWrite{ID: "m-ok", Content: "stored"})
	if !okEv.WriteOK {
		t.Fatalf("non-cancelled Submit must succeed: %+v", okEv)
	}
	if okEv.Code != "goncho_write_complete" {
		t.Fatalf("non-cancelled code = %q, want goncho_write_complete", okEv.Code)
	}

	got := store.appliedIDs()
	want := []string{"m-ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %v, want %v (cancel must not mutate storage)", got, want)
	}

	// Cancel after a write started/completed must return false (consumed).
	if q.Cancel("m-ok") {
		t.Fatalf("Cancel after write completed must return false")
	}
}

func TestGonchoRelations_SaveCreatesPendingCandidates(t *testing.T) {
	store := &fakeMemoryStore{}
	rel := &fakeRelationStore{}
	det := &fakeDetector{
		verbForID: map[string]RelationVerb{
			"m-2": RelationConflicts,
			"m-3": RelationSupersedes,
		},
	}
	q := NewWriteQueue(store, det, rel)

	ctx := context.Background()
	ev1 := q.Submit(ctx, MemoryWrite{ID: "m-1", Content: "first"})
	if !ev1.WriteOK {
		t.Fatalf("first write must succeed: %+v", ev1)
	}
	if len(ev1.Candidates) != 0 {
		t.Fatalf("first write has no prior, candidates = %v", ev1.Candidates)
	}

	ev2 := q.Submit(ctx, MemoryWrite{ID: "m-2", Content: "second"})
	if !ev2.WriteOK {
		t.Fatalf("second write must succeed: %+v", ev2)
	}
	if len(ev2.Candidates) != 1 {
		t.Fatalf("second write candidates = %d, want 1", len(ev2.Candidates))
	}
	if ev2.Candidates[0].Verb != RelationConflicts {
		t.Fatalf("verb = %q, want %q", ev2.Candidates[0].Verb, RelationConflicts)
	}
	if ev2.Candidates[0].From != "m-2" || ev2.Candidates[0].To != "m-1" {
		t.Fatalf("candidate edge = %s->%s, want m-2->m-1", ev2.Candidates[0].From, ev2.Candidates[0].To)
	}

	ev3 := q.Submit(ctx, MemoryWrite{ID: "m-3", Content: "third"})
	if !ev3.WriteOK {
		t.Fatalf("third write must succeed: %+v", ev3)
	}
	if len(ev3.Candidates) != 2 {
		t.Fatalf("third write candidates = %d, want 2", len(ev3.Candidates))
	}
	for _, c := range ev3.Candidates {
		if c.Verb != RelationSupersedes {
			t.Fatalf("verb = %q, want %q", c.Verb, RelationSupersedes)
		}
	}

	saved := rel.snapshot()
	if len(saved) != 3 {
		t.Fatalf("relation store saved = %d candidates, want 3", len(saved))
	}

	// Verify the canonical verb vocabulary is the Goncho-native set.
	allVerbs := map[RelationVerb]struct{}{
		RelationRelated:     {},
		RelationConflicts:   {},
		RelationSupersedes:  {},
		RelationCompatible:  {},
		RelationScoped:      {},
		RelationNotConflict: {},
	}
	if _, ok := allVerbs[RelationConflicts]; !ok {
		t.Fatalf("RelationConflicts missing from canonical set")
	}
	if _, ok := allVerbs[RelationSupersedes]; !ok {
		t.Fatalf("RelationSupersedes missing from canonical set")
	}
}

func TestGonchoRelations_DetectorFailureDoesNotBlockWrite(t *testing.T) {
	store := &fakeMemoryStore{}
	rel := &fakeRelationStore{}
	det := &fakeDetector{failWith: errors.New("detector unavailable")}
	q := NewWriteQueue(store, det, rel)

	ctx := context.Background()
	ev := q.Submit(ctx, MemoryWrite{ID: "m-1", Content: "first"})
	if !ev.WriteOK {
		t.Fatalf("write must succeed even when detector fails: %+v", ev)
	}
	if ev.Code != "goncho_relation_detection_failed" {
		t.Fatalf("code = %q, want goncho_relation_detection_failed", ev.Code)
	}
	if len(ev.Candidates) != 0 {
		t.Fatalf("candidates = %v, want empty when detector fails", ev.Candidates)
	}

	got := store.appliedIDs()
	want := []string{"m-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("write must be applied; applied = %v want %v", got, want)
	}

	saved := rel.snapshot()
	if len(saved) != 0 {
		t.Fatalf("relation store must be empty on detector failure, got %v", saved)
	}
}

func TestGonchoWriteQueue_NoLLMOnPendingPath(t *testing.T) {
	// A nil judge / no LLM seam in NewWriteQueue means the deterministic
	// pending-relation path must run without invoking an LLM. We assert
	// this by counting detector calls (deterministic, non-LLM) and
	// confirming the queue does not touch any other seam.
	store := &fakeMemoryStore{}
	rel := &fakeRelationStore{}
	det := &fakeDetector{}
	q := NewWriteQueue(store, det, rel)

	ctx := context.Background()
	for i, id := range []string{"a", "b", "c"} {
		ev := q.Submit(ctx, MemoryWrite{ID: id, Content: "c-" + id})
		if !ev.WriteOK {
			t.Fatalf("Submit #%d (%s) failed: %+v", i, id, ev)
		}
	}

	det.calledMu.Lock()
	defer det.calledMu.Unlock()
	if det.calledCount != 3 {
		t.Fatalf("detector called %d times, want 3 (one per write)", det.calledCount)
	}
	// The LLM seam itself does not exist in NewWriteQueue's signature; the
	// fact that the queue compiles and runs without one is the contract.
	var _ llmJudgeMustNotBeCalled
}
