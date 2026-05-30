package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

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
