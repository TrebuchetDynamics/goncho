package goncho

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGonchoProofMatrixFullLocalReportFixture(t *testing.T) {
	report := runGonchoFullLocalProofMatrix(t)
	if report.Service != "goncho" || report.ProofVersion != gonchoProofMatrixVersion {
		t.Fatalf("report metadata = %+v, want goncho %s", report, gonchoProofMatrixVersion)
	}
	if !report.SQLiteRestartVerified || len(report.APIContractsVerified) < 6 {
		t.Fatalf("report storage/api proof = %+v", report)
	}
	if !report.ScopeIsolationVerified || !report.TombstoneExclusionVerified {
		t.Fatalf("report isolation/tombstone proof = %+v", report)
	}
	if report.TraceProjectionInvariant != "no_projection_without_recall_trace" {
		t.Fatalf("projection invariant = %q", report.TraceProjectionInvariant)
	}
	for _, code := range []string{
		RecallWarningSemanticUnavailable,
		RecallWarningScopeExcludedAllCandidates,
		RecallWarningTokenBudgetTruncated,
	} {
		if !gonchoProofContains(report.WarningCodesSeen, code) {
			t.Fatalf("warning codes = %v, missing %s", report.WarningCodesSeen, code)
		}
	}
	for _, control := range []string{"workspace:other", "tombstone:deleted-conclusion", "scope:other-memory"} {
		if !gonchoProofContains(report.NegativeControlsRejected, control) {
			t.Fatalf("negative controls = %v, missing %s", report.NegativeControlsRejected, control)
		}
	}
	if report.BenchmarkSummary.RecallAt5 != 1 || report.BenchmarkSummary.ContextHitRate != 1 || report.BenchmarkSummary.TokenBudgetPassRate != 1 {
		t.Fatalf("benchmark summary = %+v, want perfect deterministic local corpus", report.BenchmarkSummary)
	}
	if report.KernelE2EFixture != "internal/e2e/testdata/goncho_memory_turn/golden.json" {
		t.Fatalf("kernel e2e fixture = %q", report.KernelE2EFixture)
	}
	if len(report.StableTraceIDs) != 3 || report.StableJSONSHA256 == "" {
		t.Fatalf("trace stability evidence = ids %v hash %q", report.StableTraceIDs, report.StableJSONSHA256)
	}
	assertGonchoProofMatrixReportFixture(t, report)
}

func runGonchoFullLocalProofMatrix(t *testing.T) gonchoProofMatrixReport {
	t.Helper()

	sqliteRestartVerified, apiContractsVerified, scopeIsolationVerified, tombstoneExclusionVerified := runGonchoProofStorageMatrix(t)
	selectedTrace, scopeTrace, budgetTrace := runGonchoProofRecallTraces(t)
	benchmark := EvaluateRecallBenchmark([]RecallBenchmarkCase{
		{
			ID:              "proof-selected-trace",
			Trace:           selectedTrace,
			RelevantIDs:     []string{"proof-default"},
			ContextContains: []string{"full local proof matrix"},
			Latency:         7 * time.Millisecond,
		},
		{
			ID:              "proof-budget-trace",
			Trace:           budgetTrace,
			RelevantIDs:     []string{"proof-short"},
			ContextContains: []string{"short local proof"},
			Latency:         11 * time.Millisecond,
		},
	})

	report, err := buildGonchoProofMatrixReport(gonchoProofMatrixReportInput{
		SQLiteRestartVerified:      sqliteRestartVerified,
		APIContractsVerified:       apiContractsVerified,
		ScopeIsolationVerified:     scopeIsolationVerified,
		TombstoneExclusionVerified: tombstoneExclusionVerified,
		Traces:                     []RecallTrace{selectedTrace, scopeTrace, budgetTrace},
		Benchmark:                  benchmark,
		KernelE2EFixture:           "internal/e2e/testdata/goncho_memory_turn/golden.json",
		NegativeControlsRejected:   []string{"workspace:other", "tombstone:deleted-conclusion", "scope:other-memory"},
	})
	if err != nil {
		t.Fatalf("build proof report: %v", err)
	}
	return report
}

func runGonchoProofStorageMatrix(t *testing.T) (bool, []string, bool, bool) {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "goncho-proof.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	svc := NewService(store.DB(), Config{WorkspaceID: "default", ObserverPeerID: "gormes", RecentMessages: 4}, nil)
	other := NewService(store.DB(), Config{WorkspaceID: "other", ObserverPeerID: "gormes", RecentMessages: 4}, nil)

	if err := svc.SetProfile(ctx, "user-juan", []string{"Prefers deterministic Goncho proof reports"}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-proof",
		Messages: []CreateMessage{{
			Peer:      "user-juan",
			Role:      "user",
			Content:   "Please prove Goncho with a full local proof matrix.",
			CreatedAt: time.Unix(1700000100, 0).UTC(),
		}},
	}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "user-juan",
		Conclusion: "Goncho full proof codename orchid lives in default workspace.",
		SessionKey: "sess-proof",
	}); err != nil {
		t.Fatalf("Conclude default: %v", err)
	}
	if _, err := other.Conclude(ctx, ConcludeParams{
		Peer:       "user-juan",
		Conclusion: "Goncho full proof codename orchid must not leak from other workspace.",
		SessionKey: "sess-proof",
	}); err != nil {
		t.Fatalf("Conclude other workspace: %v", err)
	}
	tombstone, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "user-juan",
		Conclusion: "Goncho tombstone codename obsidian must not be recalled.",
		SessionKey: "sess-proof",
	})
	if err != nil {
		t.Fatalf("Conclude tombstone candidate: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "user-juan", DeleteID: tombstone.ID}); err != nil {
		t.Fatalf("Delete tombstone conclusion: %v", err)
	}

	search, err := svc.Search(ctx, SearchParams{Peer: "user-juan", Query: "codename orchid", SessionKey: "sess-proof", MaxTokens: 200})
	if err != nil {
		t.Fatalf("Search orchid: %v", err)
	}
	scopeIsolationVerified := len(search.Results) == 1 &&
		strings.Contains(search.Results[0].Content, "default workspace") &&
		!strings.Contains(search.Results[0].Content, "other workspace")
	if !scopeIsolationVerified {
		t.Fatalf("search results = %+v, want default workspace only", search.Results)
	}
	contextResult, err := svc.Context(ctx, ContextParams{Peer: "user-juan", Query: "codename orchid", SessionKey: "sess-proof", MaxTokens: 400})
	if err != nil {
		t.Fatalf("Context orchid: %v", err)
	}
	if len(contextResult.PeerCard) != 1 || len(contextResult.Conclusions) != 1 || len(contextResult.RecentMessages) != 1 || contextResult.Representation == "" {
		t.Fatalf("context = %+v, want peer card, conclusion, recent message, representation", contextResult)
	}
	tombstoneSearch, err := svc.Search(ctx, SearchParams{Peer: "user-juan", Query: "codename obsidian", SessionKey: "sess-proof", MaxTokens: 200})
	if err != nil {
		t.Fatalf("Search tombstone: %v", err)
	}
	tombstoneExclusionVerified := true
	for _, hit := range tombstoneSearch.Results {
		if strings.Contains(hit.Content, "obsidian") {
			tombstoneExclusionVerified = false
		}
	}
	if !tombstoneExclusionVerified {
		t.Fatalf("tombstone search results = %+v, want no deleted obsidian content", tombstoneSearch.Results)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close first store: %v", err)
	}

	reopened, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer func() {
		if err := reopened.Close(ctx); err != nil {
			t.Fatalf("Close reopened store: %v", err)
		}
	}()
	reopenedSvc := NewService(reopened.DB(), Config{WorkspaceID: "default", ObserverPeerID: "gormes", RecentMessages: 4}, nil)
	reopenedSearch, err := reopenedSvc.Search(ctx, SearchParams{Peer: "user-juan", Query: "codename orchid", SessionKey: "sess-proof", MaxTokens: 200})
	if err != nil {
		t.Fatalf("Search after reopen: %v", err)
	}
	sqliteRestartVerified := len(reopenedSearch.Results) == 1 && strings.Contains(reopenedSearch.Results[0].Content, "default workspace")
	if !sqliteRestartVerified {
		t.Fatalf("reopened search results = %+v, want persisted default workspace conclusion", reopenedSearch.Results)
	}
	return sqliteRestartVerified, []string{
		"set_profile",
		"create_messages",
		"conclude",
		"search",
		"context",
		"delete_conclusion",
		"sqlite_restart_search",
	}, scopeIsolationVerified, tombstoneExclusionVerified
}

func runGonchoProofRecallTraces(t *testing.T) (RecallTrace, RecallTrace, RecallTrace) {
	t.Helper()

	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	selectedEngine := newRecallPipelineEngine(staticRecallGenerator{
		candidates: []RecallCandidate{
			{
				MemoryID:   "proof-default",
				SourceType: "conclusion",
				Content:    "Goncho full local proof matrix selected this default-scope memory.",
				SessionID:  "sess-proof",
				ScopeID:    "default",
				CreatedAt:  now,
				Importance: 0.9,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}, {Kind: "scope", Score: 1}},
			},
			{
				MemoryID:   "proof-cross-scope",
				SourceType: "conclusion",
				Content:    "Goncho full local proof matrix negative control from another scope.",
				SessionID:  "sess-proof",
				ScopeID:    "other",
				CreatedAt:  now,
				Importance: 0.8,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.95}, {Kind: "scope", Score: 0}},
			},
		},
		warnings: []RecallWarning{{
			Code:     RecallWarningSemanticUnavailable,
			Stage:    RecallStageGenerate,
			Severity: RecallWarningDegraded,
			Message:  "semantic proof fixture intentionally unavailable",
		}},
	}, recallPipelineOptions{
		pipelineVersion: "goncho-proof-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "proof-selected-v1",
			Weights:       map[string]float64{"keyword": 0.8, "scope": 0.2},
			RRFK:          60,
			MMRLambda:     1,
			DiversityKeys: []string{"scope_id"},
			TokenBudget:   200,
		},
		now: func() time.Time { return now },
	})
	selectedTrace, err := selectedEngine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "full local proof matrix",
		SessionKey:  "sess-proof",
		ScopeID:     "default",
		Limit:       2,
		MaxTokens:   200,
	})
	if err != nil {
		t.Fatalf("selected trace: %v", err)
	}

	scopeEngine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{{
		MemoryID:   "proof-scope-only",
		SourceType: "conclusion",
		Content:    "This memory is outside the query scope.",
		SessionID:  "sess-proof",
		ScopeID:    "other",
		CreatedAt:  now,
		Importance: 0.5,
		Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
	}}}, recallPipelineOptions{
		pipelineVersion: "goncho-proof-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "proof-scope-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return now },
	})
	scopeTrace, err := scopeEngine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "outside scope",
		SessionKey:  "sess-proof",
		ScopeID:     "default",
		Limit:       5,
		MaxTokens:   200,
	})
	if err != nil {
		t.Fatalf("scope trace: %v", err)
	}

	budgetEngine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "proof-short",
			SourceType: "conclusion",
			Content:    "short local proof",
			SessionID:  "sess-proof",
			ScopeID:    "default",
			CreatedAt:  now,
			Importance: 0.7,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
		},
		{
			MemoryID:   "proof-too-long",
			SourceType: "conclusion",
			Content:    "this local proof candidate is intentionally too long for the tiny token budget",
			SessionID:  "sess-proof",
			ScopeID:    "default",
			CreatedAt:  now,
			Importance: 0.6,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.9}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "goncho-proof-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "proof-budget-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 4,
		},
		now: func() time.Time { return now },
	})
	budgetTrace, err := budgetEngine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "local proof",
		SessionKey:  "sess-proof",
		ScopeID:     "default",
		Limit:       2,
		MaxTokens:   4,
	})
	if err != nil {
		t.Fatalf("budget trace: %v", err)
	}
	if !traceHasWarning(scopeTrace, RecallWarningScopeExcludedAllCandidates) || !traceHasWarning(budgetTrace, RecallWarningTokenBudgetTruncated) {
		t.Fatalf("trace warnings missing: scope=%+v budget=%+v", scopeTrace.Warnings, budgetTrace.Warnings)
	}
	return selectedTrace, scopeTrace, budgetTrace
}

func assertGonchoProofMatrixReportFixture(t *testing.T, report gonchoProofMatrixReport) {
	t.Helper()
	path := filepath.Join("testdata", "proof_matrix", "report.golden.json")
	gotRaw, err := marshalStableJSON(report)
	if err != nil {
		t.Fatalf("marshal proof report: %v", err)
	}
	againRaw, err := marshalStableJSON(report)
	if err != nil {
		t.Fatalf("marshal proof report again: %v", err)
	}
	if !bytes.Equal(gotRaw, againRaw) {
		t.Fatalf("proof report JSON is nondeterministic:\n%s\n---\n%s", gotRaw, againRaw)
	}
	if os.Getenv("GORMES_UPDATE_GONCHO_PROOF_MATRIX") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create proof fixture dir: %v", err)
		}
		if err := os.WriteFile(path, gotRaw, 0o644); err != nil {
			t.Fatalf("write proof fixture: %v", err)
		}
		return
	}
	wantRaw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture_missing at $: %s", path)
	}
	if err != nil {
		t.Fatalf("read proof fixture: %v", err)
	}
	if err := compareGoldenJSON(wantRaw, gotRaw); err != nil {
		var diff gonchoJSONDiff
		if errors.As(err, &diff) {
			t.Fatalf("goncho_proof_report_mismatch at %s: %s", diff.Path, diff.Message)
		}
		t.Fatalf("goncho_proof_report_mismatch: %v", err)
	}
}

func gonchoProofContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestGonchoProofMatrix_StorageRetrievalTraceAndOperatorEvidence(t *testing.T) {
	t.Run("storage_retrieval_context_and_workspace_boundaries", func(t *testing.T) {
		svc, cleanup := newTestService(t)
		defer cleanup()
		other := NewService(svc.db, Config{WorkspaceID: "other", ObserverPeerID: "gormes"}, nil)

		ctx := context.Background()
		if err := svc.SetProfile(ctx, "user-juan", []string{"Prefers evidence-first proof reports"}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateMessages(ctx, CreateMessagesParams{
			SessionKey: "sess-proof",
			Messages: []CreateMessage{{
				Peer:      "user-juan",
				Role:      "user",
				Content:   "Please prove Goncho with deterministic fixtures.",
				CreatedAt: time.Unix(1700000100, 0).UTC(),
			}},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Conclude(ctx, ConcludeParams{
			Peer:       "user-juan",
			Conclusion: "Goncho proof matrix codename orchid lives in default workspace.",
			SessionKey: "sess-proof",
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := other.Conclude(ctx, ConcludeParams{
			Peer:       "user-juan",
			Conclusion: "Goncho proof matrix codename orchid must not leak from other workspace.",
			SessionKey: "sess-proof",
		}); err != nil {
			t.Fatal(err)
		}

		search, err := svc.Search(ctx, SearchParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  200,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(search.Results) != 1 || !strings.Contains(search.Results[0].Content, "default workspace") {
			t.Fatalf("search results = %+v, want default workspace proof conclusion", search.Results)
		}
		if strings.Contains(search.Results[0].Content, "other workspace") {
			t.Fatalf("search leaked cross-workspace content: %+v", search.Results)
		}

		contextResult, err := svc.Context(ctx, ContextParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  400,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(contextResult.PeerCard) != 1 || len(contextResult.Conclusions) != 1 || len(contextResult.RecentMessages) != 1 {
			t.Fatalf("context = %+v, want peer card, one conclusion, one recent message", contextResult)
		}
		if contextResult.Representation == "" {
			t.Fatal("context representation is empty")
		}

		deleted, err := svc.DeleteSession(ctx, "sess-proof")
		if err != nil {
			t.Fatal(err)
		}
		if deleted.MessagesDeleted != 1 || deleted.ConclusionsDeleted != 1 {
			t.Fatalf("DeleteSession = %+v, want one message and one session conclusion deleted", deleted)
		}
		afterDelete, err := svc.Search(ctx, SearchParams{
			Peer:       "user-juan",
			Query:      "codename orchid",
			SessionKey: "sess-proof",
			MaxTokens:  200,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(afterDelete.Results) != 0 {
			t.Fatalf("session search after delete = %+v, want no results", afterDelete.Results)
		}
		profile, err := svc.Profile(ctx, "user-juan")
		if err != nil {
			t.Fatal(err)
		}
		if len(profile.Card) != 1 {
			t.Fatalf("profile card = %+v, want peer card preserved outside session delete", profile.Card)
		}
	})

	t.Run("recall_trace_diagnostics_replay_and_projection", func(t *testing.T) {
		now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
		engine := newRecallPipelineEngine(staticRecallGenerator{
			candidates: []RecallCandidate{{
				MemoryID:   "101",
				SourceType: "conclusion",
				Content:    "Goncho uses durable RecallTrace before projection.",
				SessionID:  "sess-proof",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.8,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
			}},
			warnings: []RecallWarning{{
				Code:     RecallWarningSemanticUnavailable,
				Stage:    RecallStageGenerate,
				Severity: RecallWarningDegraded,
				Message:  "semantic proof fixture intentionally unavailable",
			}},
		}, recallPipelineOptions{
			pipelineVersion: "proof-pipeline",
			scoringConfig: RecallScoringConfig{
				Version:     "proof-v1",
				Weights:     map[string]float64{"keyword": 1},
				RRFK:        60,
				MMRLambda:   1,
				TokenBudget: 200,
			},
			now: func() time.Time { return now },
		})

		trace, err := engine.Run(context.Background(), RecallQuery{
			WorkspaceID: "default",
			Peer:        "user-juan",
			Query:       "RecallTrace projection",
			SessionKey:  "sess-proof",
			ScopeID:     "team",
			Limit:       1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if trace.TraceID == "" || trace.ScoringConfig.Version != "proof-v1" || len(trace.Selected) != 1 {
			t.Fatalf("trace = %+v, want stable id, scoring config, and selected candidate", trace)
		}
		if !traceHasWarning(trace, RecallWarningSemanticUnavailable) {
			t.Fatalf("trace warnings = %+v, want semantic_unavailable", trace.Warnings)
		}
		raw1, err := trace.StableJSON()
		if err != nil {
			t.Fatal(err)
		}
		raw2, err := trace.StableJSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(raw1, raw2) {
			t.Fatalf("StableJSON changed between calls:\n%s\n---\n%s", raw1, raw2)
		}

		diagnostics := BuildRecallDiagnostics(trace)
		if diagnostics.Status != "degraded" || diagnostics.ProjectionInvariant != "no_projection_without_recall_trace" {
			t.Fatalf("diagnostics = %+v, want degraded trace-only invariant", diagnostics)
		}
		replay := BuildRecallReplay(trace)
		if replay.ReplayContract != "deterministic_replay_from_recall_trace" || !recallReplayHasWarning(replay, RecallWarningSemanticUnavailable) {
			t.Fatalf("replay = %+v, want deterministic replay with semantic warning", replay)
		}

		projected := (&RecallProjector{}).ProjectSearch(trace)
		if projected.WorkspaceID != "default" || projected.Peer != "user-juan" || len(projected.Results) != 1 {
			t.Fatalf("projected search = %+v, want trace-derived search result", projected)
		}
	})
}

func recallReplayHasWarning(replay RecallReplay, code string) bool {
	for _, event := range replay.Events {
		if event.WarningCode == code {
			return true
		}
	}
	return false
}
