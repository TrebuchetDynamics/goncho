package goncho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func TestGonchoGoalMetaanalysisComplexSuiteCoversTrustPreservingContextArchitecture(t *testing.T) {
	docPath := "docs/opensource-memory-systems/analysis/METAANALYSIS-MEMORY-SYSTEMS.md"
	docRaw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read metaanalysis: %v", err)
	}
	doc := string(docRaw)
	for _, phrase := range []string{
		"trust-preserving context architecture",
		"Evidence before memory",
		"Claims, not chunks",
		"Small agent surface",
		"review_required",
		"quarantined",
		"Negative memory matters",
	} {
		if !strings.Contains(doc, phrase) {
			t.Fatalf("metaanalysis missing %q", phrase)
		}
	}
	docHash := sha256.Sum256(docRaw)

	proof := runGonchoFullLocalProofMatrix(t)
	reviewLoopVerified := runGonchoMetaanalysisReviewLoop(t)
	memoryToolLoopVerified := runGonchoMetaanalysisMemoryToolLoop(t)

	report := BuildGonchoMetaanalysisCoverageReport(GonchoMetaanalysisCoverageInput{
		SourceDocumentPath:       docPath,
		SourceDocumentSHA256:     hex.EncodeToString(docHash[:]),
		ProofMatrix:              proof,
		ReviewLoopVerified:       reviewLoopVerified,
		MemoryToolLoopVerified:   memoryToolLoopVerified,
		LocalOnlyEvaluator:       "go test ./...",
		DocsArchitectureKeywords: []string{"trust-preserving context architecture", "Evidence before memory", "Claims, not chunks"},
	})

	if report.Service != "goncho" || report.CoverageVersion != "goncho-metaanalysis-goal-v1" {
		t.Fatalf("report metadata = %+v", report)
	}
	if report.SourceDocumentPath != docPath || report.SourceDocumentSHA256 == "" {
		t.Fatalf("source evidence = %+v", report)
	}
	for _, principle := range []string{
		"evidence_before_memory",
		"claims_not_chunks",
		"hooks_over_manual_saves",
		"orientation_not_dumping",
		"negative_memory_matters",
		"small_agent_surface",
		"trust_is_the_moat",
	} {
		if !containsMetaanalysisValue(report.PrinciplesCovered, principle) {
			t.Fatalf("principles = %#v, missing %s", report.PrinciplesCovered, principle)
		}
	}
	for _, tool := range []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"} {
		if !containsMetaanalysisValue(report.PublicToolsVerified, tool) {
			t.Fatalf("public tools = %#v, missing %s", report.PublicToolsVerified, tool)
		}
	}
	for _, feature := range []string{
		"sqlite_local_first",
		"raw_observations_with_audit",
		"filesystem_watcher_connector_import",
		"profile_conclusion_message_context_pack",
		"workspace_scope_isolation",
		"tombstone_exclusion",
		"recall_trace_diagnostics_replay",
		"token_budget_warning",
		"review_required_context_warning",
		"goncho_review_tool_resolution",
		"negative_memory_tool_recall",
		"small_public_tool_surface",
	} {
		if !containsMetaanalysisValue(report.LocalFeaturesVerified, feature) {
			t.Fatalf("features = %#v, missing %s", report.LocalFeaturesVerified, feature)
		}
	}
	for _, evaluation := range []string{
		"exact_recall",
		"paraphrase_recall",
		"multi_hop_recall",
		"temporal_state",
		"conflict_adjudication",
		"stale_code_claim",
		"token_budget",
		"noise_resistance",
		"scope_isolation",
		"prompt_injection_persistence",
		"drift_prevention",
	} {
		if !containsMetaanalysisValue(report.CoreEvaluationsCovered, evaluation) {
			t.Fatalf("core evaluations = %#v, missing %s", report.CoreEvaluationsCovered, evaluation)
		}
	}
	for _, deferred := range []string{"full_cognitive_map_ui", "postgres_team_adapter", "cloud_embeddings_required", "dashboard_visualization"} {
		if !containsMetaanalysisValue(report.DeferredFeatures, deferred) {
			t.Fatalf("deferred = %#v, missing %s", report.DeferredFeatures, deferred)
		}
	}
	if !report.AllLocalEvaluatorChecksPassed {
		t.Fatalf("report evaluator state = %+v", report)
	}
	if report.CompletionCondition != "go test ./..." {
		t.Fatalf("completion condition = %q", report.CompletionCondition)
	}
}

func runGonchoMetaanalysisReviewLoop(t *testing.T) bool {
	t.Helper()
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		PeerID:      "peer-meta",
		SessionKey:  "session-meta",
		SubjectID:   "memory-new",
		RelatedID:   "memory-old",
		Reason:      "metaanalysis requires visible review stewardship",
		EvidenceIDs: []string{"obs-meta"},
		CreatedAt:   time.Date(2026, 5, 19, 16, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}
	before, err := svc.Context(ctx, ContextParams{Peer: "peer-meta", SessionKey: "session-meta"})
	if err != nil {
		t.Fatalf("Context before review resolve: %v", err)
	}
	if !contextUnavailableHasCapability(before.Unavailable, "review_required") {
		t.Fatalf("context unavailable before resolve = %+v, want review_required", before.Unavailable)
	}
	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"metaanalysis evidence checked"}`)
	if stringField(t, resolved, "status") != string(ReviewStatusResolved) {
		t.Fatalf("resolved output = %+v", resolved)
	}
	after, err := svc.Context(ctx, ContextParams{Peer: "peer-meta", SessionKey: "session-meta"})
	if err != nil {
		t.Fatalf("Context after review resolve: %v", err)
	}
	return !contextUnavailableHasCapability(after.Unavailable, "review_required")
}

func runGonchoMetaanalysisMemoryToolLoop(t *testing.T) bool {
	t.Helper()
	ctx := context.Background()
	store := newMockToolStore()
	storeTool := NewStoreMemoryTool(store)
	retrieveTool := NewRetrieveMemoryTool(store)
	updateTool := NewUpdateMemoryTool(store)
	summarizeTool := NewSummarizeMemoryTool(store)
	forgetTool := NewForgetMemoryTool(store)

	stored := executeMemoryTool(t, ctx, storeTool, `{"content":"Dead end: retrying stale Docker cache fix repeats a known failure.","tags":["negative","dead-end"],"importance":0.9}`)
	id := stringField(t, stored, "id")
	retrieved := executeMemoryTool(t, ctx, retrieveTool, `{"query":"dead-end","limit":5}`)
	if intField(t, retrieved, "count") != 1 {
		t.Fatalf("retrieve output = %+v, want one negative memory", retrieved)
	}
	updated := executeMemoryTool(t, ctx, updateTool, `{"id":"`+id+`","content":"Dead end: retrying stale Docker cache fix repeats a known failure; verify live state first."}`)
	if updated["success"] != true {
		t.Fatalf("update output = %+v", updated)
	}
	summary := executeMemoryTool(t, ctx, summarizeTool, `{"filter":"dead-end","max_items":5}`)
	if intField(t, summary, "summarized") == 0 {
		t.Fatalf("summary output = %+v, want summarized negative memory", summary)
	}
	forgotten := executeMemoryTool(t, ctx, forgetTool, `{"id":"`+id+`"}`)
	if forgotten["success"] != true {
		t.Fatalf("forget output = %+v", forgotten)
	}
	afterForget := executeMemoryTool(t, ctx, retrieveTool, `{"query":"dead-end","limit":5}`)
	return intField(t, afterForget, "count") == 0
}

func containsMetaanalysisValue(values []string, want string) bool {
	return sliceutil.Contains(values, want)
}
