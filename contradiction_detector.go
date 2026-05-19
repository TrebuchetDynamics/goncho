package goncho

import (
	"strings"
	"unicode"
)

type MemoryContradiction struct {
	Existing MemoryToolEntry
	Incoming MemoryToolEntry
	Subject  string
	Relation string
	Reason   string
}

type memoryFact struct {
	subject  string
	relation string
	object   string
}

func DetectMemoryContradiction(existing, incoming MemoryToolEntry) (MemoryContradiction, bool) {
	oldFact, oldOK := parseMemoryFact(existing.Content)
	newFact, newOK := parseMemoryFact(incoming.Content)
	if !oldOK || !newOK {
		return MemoryContradiction{}, false
	}
	if oldFact.subject != newFact.subject || oldFact.relation != newFact.relation {
		return MemoryContradiction{}, false
	}
	if oldFact.object == newFact.object {
		return MemoryContradiction{}, false
	}
	if strings.Contains(newFact.object, oldFact.object) || strings.Contains(oldFact.object, newFact.object) {
		return MemoryContradiction{}, false
	}
	return MemoryContradiction{
		Existing: existing,
		Incoming: incoming,
		Subject:  oldFact.subject,
		Relation: oldFact.relation,
		Reason:   "same subject and relation with different objects",
	}, true
}

func parseMemoryFact(content string) (memoryFact, bool) {
	normalized := normalizeFactText(content)
	for _, relation := range []string{" is ", " are ", " uses ", " use ", " prefers ", " stores "} {
		before, after, ok := strings.Cut(normalized, relation)
		if !ok {
			continue
		}
		subject := strings.TrimSpace(before)
		object := strings.TrimSpace(after)
		if subject == "" || object == "" {
			continue
		}
		return memoryFact{
			subject:  subject,
			relation: strings.TrimSpace(relation),
			object:   object,
		}, true
	}
	return memoryFact{}, false
}

func normalizeFactText(content string) string {
	content = strings.ToLower(strings.TrimSpace(content))
	var b strings.Builder
	lastSpace := false
	for _, r := range content {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
