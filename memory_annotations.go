package goncho

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/goncho/internal/memoryannotations"
)

const memoryAnnotationSourceConclusion = memoryannotations.SourceConclusion

var memoryAnnotationDDL = memoryannotations.DDL

type memoryFactAnnotation = memoryannotations.FactAnnotation

func conclusionFactAnnotations(content string) []string {
	return memoryannotations.ConclusionFacts(content)
}

func storeConclusionFactAnnotations(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string, conclusionID int64, facts []string) error {
	return memoryannotations.StoreConclusionFacts(ctx, db, workspaceID, profileID, observer, peer, conclusionID, facts)
}

func attachConclusionFactAnnotations(ctx context.Context, db *sql.DB, hits []SearchHit) ([]SearchHit, error) {
	ids := make([]int64, 0, len(hits))
	indexes := map[int64][]int{}
	for i, hit := range hits {
		if hit.Source != memoryAnnotationSourceConclusion || hit.ID <= 0 {
			continue
		}
		if _, ok := indexes[hit.ID]; !ok {
			ids = append(ids, hit.ID)
		}
		indexes[hit.ID] = append(indexes[hit.ID], i)
	}
	annotationsByID, err := memoryannotations.ConclusionFactsByMemoryID(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for memoryID, annotations := range annotationsByID {
		for _, index := range indexes[memoryID] {
			hits[index].factAnnotations = append(hits[index].factAnnotations, annotations...)
		}
	}
	return hits, nil
}
