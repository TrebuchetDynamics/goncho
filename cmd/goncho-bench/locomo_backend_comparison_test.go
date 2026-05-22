package main

import (
	"context"
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
	if entry.RecallAnyAt5 != 1 || entry.StrictRecallAt5 != 1 || entry.MRR != 1 {
		t.Fatalf("metrics = any5 %.2f strict5 %.2f mrr %.2f, want all 1", entry.RecallAnyAt5, entry.StrictRecallAt5, entry.MRR)
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

func TestRunLocomoBackendComparisonWritesJSONAndMarkdown(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"Maya keeps the orchid marker in the archive cabinet."}
{"memory_id":"m2","conversation_id":"c1","session_id":"s1","speaker":"Leo","turn_index":2,"content":"Leo talked about unrelated dashboard notes."}
`)
	writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"Where is the orchid marker?","gold_memory_ids":["m1"],"category":"single_hop_retrieval"}
`)
	jsonOut := filepath.Join(dir, "locomo-backend-comparison.json")
	mdOut := filepath.Join(dir, "locomo-backend-comparison.md")
	failuresOut := filepath.Join(dir, "locomo-backend-comparison.jsonl")
	if err := runLocomoBackendComparison(ctx, config{LocomoMemoriesPath: memories, LocomoQuestionsPath: questions, LocomoBackendComparisonJSON: jsonOut, LocomoBackendComparisonMD: mdOut, LocomoBackendComparisonFailures: failuresOut}); err != nil {
		t.Fatal(err)
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
