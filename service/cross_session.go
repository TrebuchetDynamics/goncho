package goncho

import (
	"context"
	"sort"
	"strings"
	"time"
)

type CrossSessionMemory struct {
	store MemoryToolStore
}

func NewCrossSessionMemory(store MemoryToolStore) *CrossSessionMemory {
	return &CrossSessionMemory{store: store}
}

func summarizeMemoryEntries(entries []MemoryToolEntry) string {
	if len(entries) == 0 {
		return "No matching memories."
	}
	var summary strings.Builder
	for _, entry := range entries {
		content := strings.TrimSpace(entry.Content)
		if content == "" {
			continue
		}
		if summary.Len() > 0 {
			summary.WriteByte('\n')
		}
		summary.WriteString("- ")
		if entry.ID != "" {
			summary.WriteString(entry.ID)
			summary.WriteString(": ")
		}
		summary.WriteString(content)
	}
	if summary.Len() == 0 {
		return "No matching memories."
	}
	return summary.String()
}

func (csm *CrossSessionMemory) LoadRelevant(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	return csm.store.Retrieve(ctx, query, limit)
}

type CrossSessionKnowledge struct {
	Query    string
	Memories []MemoryToolEntry
	Sessions map[string]int
	Summary  string
}

func (csm *CrossSessionMemory) QueryKnowledge(ctx context.Context, query string, limit int) (CrossSessionKnowledge, error) {
	entries, err := csm.LoadRelevant(ctx, query, limit)
	if err != nil {
		return CrossSessionKnowledge{Query: strings.TrimSpace(query)}, err
	}
	answer := CrossSessionKnowledge{
		Query:    strings.TrimSpace(query),
		Memories: entries,
		Sessions: make(map[string]int),
		Summary:  summarizeMemoryEntries(entries),
	}
	for _, entry := range entries {
		if sessionID := memoryEntrySessionID(entry); sessionID != "" {
			answer.Sessions[sessionID]++
		}
	}
	return answer, nil
}

func (csm *CrossSessionMemory) DetectContradictions(ctx context.Context, newEntry MemoryToolEntry) ([]MemoryToolEntry, error) {
	existing, err := csm.store.Retrieve(ctx, "", 50)
	if err != nil || len(existing) == 0 {
		return nil, err
	}
	var conflicts []MemoryToolEntry
	for _, e := range existing {
		if e.ID == newEntry.ID {
			continue
		}
		if _, ok := DetectMemoryContradiction(e, newEntry); ok {
			conflicts = append(conflicts, e)
		}
	}
	return conflicts, nil
}

func isContradictory(a, b string) bool {
	_, ok := DetectMemoryContradiction(MemoryToolEntry{Content: a}, MemoryToolEntry{Content: b})
	return ok
}

type SessionMemoryDeletionPlan struct {
	SessionID     string
	Preserve      bool
	Preserved     []MemoryToolEntry
	CascadeForget []MemoryToolEntry
	Unrelated     []MemoryToolEntry
}

func PlanSessionMemoryDeletion(entries []MemoryToolEntry, sessionID string, cascade bool) SessionMemoryDeletionPlan {
	sessionID = strings.TrimSpace(sessionID)
	plan := SessionMemoryDeletionPlan{SessionID: sessionID, Preserve: !cascade}
	for _, entry := range entries {
		if memoryEntrySessionID(entry) != sessionID {
			plan.Unrelated = append(plan.Unrelated, entry)
			continue
		}
		if cascade {
			plan.CascadeForget = append(plan.CascadeForget, entry)
		} else {
			plan.Preserved = append(plan.Preserved, entry)
		}
	}
	return plan
}

func memoryEntrySessionID(entry MemoryToolEntry) string {
	if sessionID := strings.TrimSpace(entry.SessionID); sessionID != "" {
		return sessionID
	}
	if entry.Metadata != nil {
		return strings.TrimSpace(entry.Metadata["session_id"])
	}
	return ""
}

func RecentEntries(entries []MemoryToolEntry, now time.Time, maxAge time.Duration) []MemoryToolEntry {
	var recent []MemoryToolEntry
	for _, e := range entries {
		if now.Sub(e.CreatedAt) <= maxAge {
			recent = append(recent, e)
		}
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].CreatedAt.After(recent[j].CreatedAt)
	})
	return recent
}
