package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func recallCandidateStableMemoryID(candidate RecallCandidate) (string, bool) {
	memoryID := strings.TrimSpace(candidate.MemoryID)
	return memoryID, memoryID != ""
}

func recallCandidateIndexByStableMemoryID(candidates []RecallCandidate) map[string]int {
	return sliceutil.IndexBy(candidates, recallCandidateStableMemoryID)
}

func lookupRecallCandidateStableIndex(candidates []RecallCandidate, indexByID map[string]int, memoryID string) (int, bool) {
	if idx, exists := indexByID[memoryID]; exists {
		return idx, true
	}
	for idx, candidate := range candidates {
		candidateMemoryID, stable := recallCandidateStableMemoryID(candidate)
		if stable && candidateMemoryID == memoryID {
			indexByID[memoryID] = idx
			return idx, true
		}
	}
	return 0, false
}
