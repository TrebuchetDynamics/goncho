package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocomoBackendComparisonMarksExternalBackendsNotComparable(t *testing.T) {
	_, reason, err := newLocomoBackend("agentmemory")
	if err != nil {
		t.Fatal(err)
	}
	if reason == "" {
		t.Fatal("agentmemory reason empty, want not-comparable explanation")
	}
	_, reason, err = newLocomoBackend("mem0")
	if err != nil {
		t.Fatal(err)
	}
	if reason == "" {
		t.Fatal("mem0 reason empty, want not-comparable explanation")
	}
}

func TestLocomoInMemoryBackendScopedSearchUsesConversationIndex(t *testing.T) {
	ctx := context.Background()
	backend := &bm25Backend{}
	if err := backend.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if err := backend.Insert(ctx, "m1", "orchid marker lives in c1", map[string]any{"conversation_id": "c1"}); err != nil {
		t.Fatal(err)
	}
	if err := backend.Insert(ctx, "m2", "orchid marker lives in c2", map[string]any{"conversation_id": "c2"}); err != nil {
		t.Fatal(err)
	}
	if got := len(backend.byConversation["c1"]); got != 1 {
		t.Fatalf("c1 indexed items = %d, want 1", got)
	}

	backend.items["poison"] = backendMemory{ID: "poison", Content: "orchid marker poison", ConversationID: "c1", Seq: 999}
	hits, err := backend.SearchScoped(ctx, "c1", "orchid marker", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range hits {
		if hit.MemoryID == "poison" {
			t.Fatalf("scoped search used all-items scan instead of conversation index: %+v", hits)
		}
	}
}

func TestGonchoBackendScopedSearchCapsStableIDFanoutToTopK(t *testing.T) {
	ctx := context.Background()
	backend, unsupported, err := newLocomoBackend("goncho")
	if err != nil {
		t.Fatal(err)
	}
	if unsupported != "" {
		t.Fatalf("goncho backend unsupported: %s", unsupported)
	}
	defer backend.Close(ctx)
	if err := backend.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"m1", "m2"} {
		if err := backend.Insert(ctx, id, "orchid marker duplicate", map[string]any{"conversation_id": "c1"}); err != nil {
			t.Fatal(err)
		}
	}
	hits, err := searchLocomoBackend(ctx, backend, locomoQuestionRow{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("goncho hits = %+v, want duplicate content fan-out capped to topK 1", hits)
	}
}

func TestLocomoBackendComparisonUsesStableMemoryIDs(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", Content: "Maya keeps the orchid marker in the archive cabinet.", TurnIndex: 1},
			{MemoryID: "m2", ConversationID: "c1", SessionID: "s1", Speaker: "Leo", Content: "Leo talked about unrelated dashboard notes.", TurnIndex: 2},
			{MemoryID: "m3", ConversationID: "c2", SessionID: "s1", Speaker: "Nia", Content: "Nia repeats orchid marker orchid marker orchid marker in another conversation.", TurnIndex: 1},
		},
		Questions: []locomoQuestionRow{
			{QuestionID: "q1", ConversationID: "c1", Question: "Where is the orchid marker?", GoldMemoryIDs: []string{"m1"}, Category: "single_hop_retrieval"},
		},
	}
	entry, err := evaluateLocomoBackend(ctx, data, "bm25", 10, config{})
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Comparable {
		t.Fatalf("bm25 comparable = false: %+v", entry)
	}
	if len(entry.QuestionsDetail) != 1 || entry.QuestionsDetail[0].RetrievedIDs[0] != "m1" {
		t.Fatalf("question detail = %+v, want stable memory id m1 first", entry.QuestionsDetail)
	}
	if entry.RecallAnyAt5 != 1 || entry.StrictRecallAt5 != 1 || entry.NDCGAt5 != 1 || entry.NDCGAt10 != 1 || entry.MRR != 1 {
		t.Fatalf("metrics = any5 %.2f strict5 %.2f ndcg5 %.2f ndcg10 %.2f mrr %.2f, want all 1", entry.RecallAnyAt5, entry.StrictRecallAt5, entry.NDCGAt5, entry.NDCGAt10, entry.MRR)
	}
}

func TestLocomoBackendComparisonConsumesExternalStableIDJSONL(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", Content: "duplicate"}, {MemoryID: "m2", ConversationID: "c2", Content: "duplicate"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "duplicate", GoldMemoryIDs: []string{"m1"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m1","score":0.9,"backend_raw_id":"raw-1","metadata":{"memory_id":"m1"}}]}
`)
	entry, err := evaluateLocomoBackend(ctx, data, "mem0", 10, config{LocomoMem0Results: path})
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Comparable || entry.RecallAnyAt5 != 1 || entry.MRR != 1 {
		t.Fatalf("entry = %+v, want comparable perfect stable-ID score", entry)
	}
}

func TestLocomoBackendComparisonLimitsExternalRowsToTopK(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", Content: "distractor marker"},
			{MemoryID: "m2", ConversationID: "c1", Content: "orchid marker"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m2"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m1","score":0.9},{"memory_id":"m2","score":0.8}]}
`)
	entry, err := evaluateLocomoBackend(ctx, data, "mem0", 1, config{LocomoMem0Results: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.QuestionsDetail) != 1 || strings.Join(entry.QuestionsDetail[0].RetrievedIDs, ",") != "m1" {
		t.Fatalf("retrieved ids = %+v, want only topK external hit m1", entry.QuestionsDetail)
	}
	if entry.RecallAnyAt5 != 0 || entry.MRR != 0 {
		t.Fatalf("metrics = any5 %.2f mrr %.2f, want miss when gold appears after topK", entry.RecallAnyAt5, entry.MRR)
	}
}

func TestLocomoBackendComparisonDuplicateExternalRowsDoNotExpandTopK(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", Content: "distractor marker"},
			{MemoryID: "m2", ConversationID: "c1", Content: "orchid marker"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m2"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m1","score":0.9},{"memory_id":"m1","score":0.8},{"memory_id":"m2","score":0.7}]}
`)
	entry, err := evaluateLocomoBackend(ctx, data, "mem0", 2, config{LocomoMem0Results: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.QuestionsDetail) != 1 || strings.Join(entry.QuestionsDetail[0].RetrievedIDs, ",") != "m1" {
		t.Fatalf("retrieved ids = %+v, want duplicate rows to consume the top-K window", entry.QuestionsDetail)
	}
	if entry.RecallAnyAt5 != 0 || entry.MRR != 0 {
		t.Fatalf("metrics = any5 %.2f mrr %.2f, want miss when gold appears after duplicate-expanded topK", entry.RecallAnyAt5, entry.MRR)
	}
}

func TestLocomoBackendComparisonRejectsExternalUnknownQuestionID(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", Content: "orchid marker"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m1"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m1","score":0.9}]}
{"backend":"mem0","question_id":"q2","comparable":true,"results":[{"memory_id":"m1","score":0.8}]}
`)
	_, err := evaluateLocomoBackend(ctx, data, "mem0", 10, config{LocomoMem0Results: path})
	if err == nil || !strings.Contains(err.Error(), `unknown question_id "q2"`) {
		t.Fatalf("err = %v, want unknown question_id rejection", err)
	}
}

func TestLocomoBackendComparisonRejectsExternalOutOfConversationMemoryID(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", Content: "orchid marker"},
			{MemoryID: "m2", ConversationID: "c2", Content: "orchid marker"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m1"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m2","score":0.9}]}
`)
	_, err := evaluateLocomoBackend(ctx, data, "mem0", 10, config{LocomoMem0Results: path})
	if err == nil || !strings.Contains(err.Error(), `out-of-conversation memory_id "m2"`) {
		t.Fatalf("err = %v, want out-of-conversation memory_id rejection", err)
	}
}

func TestLocomoBackendComparisonRejectsExternalUnknownMemoryID(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", Content: "orchid marker"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m1"}, Category: "single_hop_retrieval"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"mx","score":0.9}]}
`)
	_, err := evaluateLocomoBackend(ctx, data, "mem0", 10, config{LocomoMem0Results: path})
	if err == nil || !strings.Contains(err.Error(), `unknown memory_id "mx"`) {
		t.Fatalf("err = %v, want unknown memory_id rejection", err)
	}
}

func TestLocomoBackendComparisonConsumesExternalNotComparableJSONL(t *testing.T) {
	ctx := context.Background()
	data := locomoDataset{Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "q", GoldMemoryIDs: []string{"m1"}}}}
	dir := t.TempDir()
	path := filepath.Join(dir, "external.jsonl")
	writeTestFile(t, path, `{"backend":"agentmemory","comparable":false,"reason":"no stable ids"}
`)
	entry, err := evaluateLocomoBackend(ctx, data, "agentmemory", 10, config{LocomoAgentMemoryResults: path})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Comparable || !strings.Contains(entry.NotComparableReason, "stable ids") {
		t.Fatalf("entry = %+v, want not comparable reason", entry)
	}
}

func TestRunLocomoBackendComparisonHonorsConfiguredLimitForExternalRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	mem0Results := filepath.Join(dir, "mem0.jsonl")
	jsonOut := filepath.Join(dir, "locomo-backend-comparison.json")
	writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"distractor marker"}
{"memory_id":"m2","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":2,"content":"orchid marker"}
`)
	writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"orchid marker","gold_memory_ids":["m2"],"category":"single_hop_retrieval"}
`)
	writeTestFile(t, mem0Results, `{"backend":"mem0","question_id":"q1","comparable":true,"results":[{"memory_id":"m1","score":0.9},{"memory_id":"m2","score":0.8}]}
`)

	err := runLocomoBackendComparison(ctx, config{LocomoMemoriesPath: memories, LocomoQuestionsPath: questions, LocomoMem0Results: mem0Results, LocomoBackendComparisonJSON: jsonOut, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatal(err)
	}
	var rawReport map[string]any
	if err := json.Unmarshal(raw, &rawReport); err != nil {
		t.Fatal(err)
	}
	if rawReport["top_k"] != float64(1) {
		t.Fatalf("report top_k = %v, want configured limit 1", rawReport["top_k"])
	}
	var report locomoBackendComparisonReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	for _, backend := range report.Backends {
		if backend.Backend != "mem0" {
			continue
		}
		if len(backend.QuestionsDetail) != 1 || strings.Join(backend.QuestionsDetail[0].RetrievedIDs, ",") != "m1" {
			t.Fatalf("mem0 retrieved ids = %+v, want configured limit to keep only m1", backend.QuestionsDetail)
		}
		return
	}
	t.Fatal("mem0 backend missing from report")
}

func TestRunLocomoBackendComparisonWritesJSONAndMarkdown(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	memoriesRaw := `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"Maya keeps the orchid marker in the archive cabinet."}
{"memory_id":"m2","conversation_id":"c1","session_id":"s1","speaker":"Leo","turn_index":2,"content":"Leo talked about unrelated dashboard notes."}
`
	questionsRaw := `{"question_id":"q1","conversation_id":"c1","question":"Where is the orchid marker?","gold_memory_ids":["m1"],"category":"single_hop_retrieval","answer_hint":"orchid marker"}
`
	writeTestFile(t, memories, memoriesRaw)
	writeTestFile(t, questions, questionsRaw)
	jsonOut := filepath.Join(dir, "locomo-backend-comparison.json")
	mdOut := filepath.Join(dir, "locomo-backend-comparison.md")
	failuresOut := filepath.Join(dir, "locomo-backend-comparison.jsonl")
	if err := runLocomoBackendComparison(ctx, config{LocomoMemoriesPath: memories, LocomoQuestionsPath: questions, LocomoBackendComparisonJSON: jsonOut, LocomoBackendComparisonMD: mdOut, LocomoBackendComparisonFailures: failuresOut}); err != nil {
		t.Fatal(err)
	}
	rawReportBytes, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("read backend comparison report: %v", err)
	}
	var rawReport map[string]any
	if err := json.Unmarshal(rawReportBytes, &rawReport); err != nil {
		t.Fatalf("decode backend comparison report: %v", err)
	}
	if rawReport["memory_token_estimate"] != float64(15) {
		t.Fatalf("backend comparison memory_token_estimate = %v, want deterministic content token estimate 15", rawReport["memory_token_estimate"])
	}
	wantDatabaseSize := float64(len(memoriesRaw) + len(questionsRaw))
	if rawReport["database_size_bytes"] != wantDatabaseSize {
		t.Fatalf("backend comparison database_size_bytes = %v, want fixture byte size %.0f", rawReport["database_size_bytes"], wantDatabaseSize)
	}
	leakage, ok := rawReport["leakage_checks"].(map[string]any)
	if !ok || leakage["answer_text_in_memory_content"] != float64(1) {
		t.Fatalf("backend comparison leakage_checks = %#v, want one answer-text leakage count", rawReport["leakage_checks"])
	}
	var report locomoBackendComparisonReport
	if err := json.Unmarshal(rawReportBytes, &report); err != nil {
		t.Fatalf("decode typed backend comparison report: %v", err)
	}
	foundBM25 := false
	for _, backend := range report.Backends {
		if backend.Backend != "bm25" {
			continue
		}
		foundBM25 = true
		if backend.NDCGAt5 != 1 || backend.NDCGAt10 != 1 {
			t.Fatalf("bm25 ndcg@5/@10 = %.2f/%.2f, want 1/1", backend.NDCGAt5, backend.NDCGAt10)
		}
	}
	if !foundBM25 {
		t.Fatal("bm25 backend missing from report")
	}
	assertBenchFileContains(t, jsonOut, `"backend": "goncho"`)
	assertBenchFileContains(t, jsonOut, `"backend": "agentmemory"`)
	assertBenchFileContains(t, jsonOut, `"comparable": false`)
	assertBenchFileNotContains(t, jsonOut, `"backend": "random"`)
	assertBenchFileNotContains(t, jsonOut, `"backend": "recency"`)
	assertBenchFileContains(t, failuresOut, `"backend":"agentmemory"`)
	assertBenchFileContains(t, failuresOut, `"failure_category":"not_comparable"`)
	assertBenchFileContains(t, mdOut, "benchmark adapter suite")
	assertBenchFileContains(t, mdOut, "Failure JSONL")
	assertBenchFileContains(t, mdOut, "- Top-K: `10`")
	assertBenchFileContains(t, mdOut, "- Memory token estimate: `15`")
	assertBenchFileContains(t, mdOut, "- Database size bytes:")
	assertBenchFileContains(t, mdOut, "## Leakage checks")
	assertBenchFileContains(t, mdOut, "NDCG@5")
	assertBenchFileContains(t, mdOut, "NDCG@10")
	assertBenchFileContains(t, jsonOut, `"latency_ms"`)
	assertBenchFileContains(t, mdOut, "Insert latency ms")
	assertBenchFileContains(t, mdOut, "Latency p50 ms")
	assertBenchFileContains(t, mdOut, "RSS bytes")
}

func TestWriteLocomoBackendComparisonFailuresRejectsUnknownQuestionID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "missing-q",
			ConversationID: "c1",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"m1"},
			RetrievedIDs:   []string{"m1"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `unknown question_id "missing-q"`) {
		t.Fatalf("write backend comparison failures error = %v, want unknown question ID error", err)
	}
}

func TestWriteLocomoBackendComparisonFailuresRejectsQuestionConversationMismatch(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "q1",
			ConversationID: "c2",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"m1"},
			RetrievedIDs:   []string{"m2"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `conversation_id "c2" does not match fixture conversation_id "c1"`) {
		t.Fatalf("write backend comparison failures error = %v, want question conversation mismatch error", err)
	}
}

func TestWriteLocomoBackendComparisonFailuresRejectsUnknownGoldMemoryID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "q1",
			ConversationID: "c1",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"missing-gold"},
			RetrievedIDs:   []string{"m1"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `unknown gold_memory_id "missing-gold"`) {
		t.Fatalf("write backend comparison failures error = %v, want unknown gold stable ID error", err)
	}
}

func TestWriteLocomoBackendComparisonFailuresRejectsOutOfConversationGoldMemoryID(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "q1",
			ConversationID: "c1",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"m2"},
			RetrievedIDs:   []string{"m1"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `out-of-conversation gold_memory_id "m2"`) {
		t.Fatalf("write backend comparison failures error = %v, want out-of-conversation gold stable ID error", err)
	}
}

func TestWriteLocomoBackendComparisonFailuresRejectsUnknownRetrievedID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "q1",
			ConversationID: "c1",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"m1"},
			RetrievedIDs:   []string{"missing"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `unknown retrieved memory_id "missing"`) {
		t.Fatalf("write backend comparison failures error = %v, want unknown retrieved stable ID error", err)
	}
}

func TestWriteLocomoBackendComparisonFailuresRejectsOutOfConversationRetrievedID(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoBackendComparisonReport{Backends: []locomoBackendComparisonEntry{{
		Backend:    "mem0",
		Comparable: true,
		QuestionsDetail: []locomoQuestionResult{{
			QuestionID:     "q1",
			ConversationID: "c1",
			Category:       "true_retrieval_failure",
			Question:       "question",
			GoldMemoryIDs:  []string{"m1"},
			RetrievedIDs:   []string{"m2"},
			Rank:           0,
		}},
	}}}
	err := writeLocomoBackendComparisonFailures(filepath.Join(t.TempDir(), "failures.jsonl"), data, report)
	if err == nil || !strings.Contains(err.Error(), `out-of-conversation retrieved memory_id "m2"`) {
		t.Fatalf("write backend comparison failures error = %v, want out-of-conversation retrieved stable ID error", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertBenchFileContains(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), want) {
		t.Fatalf("%s missing %q\n%s", path, want, raw)
	}
}

func assertBenchFileNotContains(t *testing.T, path, unwanted string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(raw), unwanted) {
		t.Fatalf("%s unexpectedly contains %q\n%s", path, unwanted, raw)
	}
}
