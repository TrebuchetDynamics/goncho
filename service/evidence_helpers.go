package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func evidenceListKinds(items []EvidenceItem) []string {
	kinds := make([]string, 0, len(items))
	for _, item := range items {
		kinds = append(kinds, item.Kind)
	}
	return kinds
}

func evidenceListHas(items []EvidenceItem, kind, id string) bool {
	return sliceutil.ContainsFunc(items, func(item EvidenceItem) bool {
		return item.Kind == kind && item.ID == id
	})
}

func evidenceListHasKind(items []EvidenceItem, kind string) bool {
	return sliceutil.ContainsFunc(items, func(item EvidenceItem) bool {
		return item.Kind == kind
	})
}

func evidenceListHasKindNote(items []EvidenceItem, kind, note string) bool {
	return sliceutil.ContainsFunc(items, func(item EvidenceItem) bool {
		return item.Kind == kind && item.Note == note
	})
}

func evidenceListFindKindSourceScoreNoteContains(items []EvidenceItem, kind, source string, score float64, noteContains string) (EvidenceItem, bool) {
	for _, item := range items {
		if item.Kind != kind {
			continue
		}
		if source != "" && item.Source != source {
			continue
		}
		if score != 0 && item.Score != score {
			continue
		}
		if noteContains != "" && !strings.Contains(item.Note, noteContains) {
			continue
		}
		return item, true
	}
	return EvidenceItem{}, false
}
