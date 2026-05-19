package goncho

import "time"

type ConflictResolution struct {
	Winner   *MemoryEntry
	Loser    *MemoryEntry
	Reason   string
	Resolved bool
}

func ResolveConflict(existing, incoming *MemoryEntry) ConflictResolution {
	if existing == nil {
		return ConflictResolution{Winner: incoming, Reason: "no_existing_entry", Resolved: true}
	}
	if incoming == nil {
		return ConflictResolution{Winner: existing, Reason: "no_incoming_entry", Resolved: true}
	}

	if existing.Type != incoming.Type {
		return ConflictResolution{Winner: existing, Reason: "type_mismatch_no_conflict", Resolved: false}
	}

	if incoming.Confidence > existing.Confidence {
		return ConflictResolution{
			Winner:   incoming,
			Loser:    existing,
			Reason:   "higher_confidence",
			Resolved: true,
		}
	}

	if incoming.Confidence < existing.Confidence {
		return ConflictResolution{
			Winner:   existing,
			Loser:    incoming,
			Reason:   "lower_confidence",
			Resolved: true,
		}
	}

	if incoming.UpdatedAt.After(existing.UpdatedAt) {
		return ConflictResolution{
			Winner:   incoming,
			Loser:    existing,
			Reason:   "equal_confidence_newer",
			Resolved: true,
		}
	}

	return ConflictResolution{
		Winner:   existing,
		Loser:    incoming,
		Reason:   "equal_confidence_older",
		Resolved: true,
	}
}

func (s *TypedMemoryStore) UpsertWithConflictResolution(entry *MemoryEntry) ConflictResolution {
	existing, ok := s.entries[entry.ID]
	if !ok {
		for _, e := range s.entries {
			if e.Type == entry.Type && e.Content == entry.Content {
				existing = e
				break
			}
		}
	}

	resolution := ResolveConflict(existing, entry)
	if resolution.Resolved && resolution.Winner == entry {
		s.Create(entry)
		if existing != nil && existing.ID != entry.ID {
			delete(s.entries, existing.ID)
		}
	}

	return resolution
}

func (s *TypedMemoryStore) GetStats() map[MemoryType]int {
	stats := make(map[MemoryType]int)
	for _, entry := range s.entries {
		stats[entry.Type]++
	}
	return stats
}

func (s *TypedMemoryStore) GetAverageConfidence(memType MemoryType) float64 {
	entries := s.ListByType(memType)
	if len(entries) == 0 {
		return 0
	}
	sum := 0.0
	for _, e := range entries {
		sum += e.Confidence
	}
	return sum / float64(len(entries))
}

func (s *TypedMemoryStore) GetPruningCandidates(now time.Time) []*MemoryEntry {
	var candidates []*MemoryEntry
	for _, entry := range s.entries {
		if shouldPrune(entry, now) {
			candidates = append(candidates, entry)
		}
	}
	return candidates
}
