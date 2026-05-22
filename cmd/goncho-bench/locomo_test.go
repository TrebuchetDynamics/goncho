package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLocomoDatasetParsesSchema(t *testing.T) {
	data, err := loadLocomoDataset(filepath.Join("testdata", "locomo-smoke", "memories.jsonl"), filepath.Join("testdata", "locomo-smoke", "questions.jsonl"))
	if err != nil {
		t.Fatalf("load LOCOMO fixture: %v", err)
	}
	if len(data.Memories) < 10 || len(data.Questions) < 8 {
		t.Fatalf("fixture sizes memories=%d questions=%d, want at least 10 memories and 8 questions", len(data.Memories), len(data.Questions))
	}
	if data.Memories[0].MemoryID == "" || data.Memories[0].Speaker == "" || data.Memories[0].TurnIndex == 0 {
		t.Fatalf("first memory missing schema fields: %+v", data.Memories[0])
	}
	if data.Questions[0].QuestionID == "" || data.Questions[0].AnswerHint == "" || len(data.Questions[0].GoldMemoryIDs) == 0 {
		t.Fatalf("first question missing schema fields: %+v", data.Questions[0])
	}
	firstConversation := data.Memories[0].ConversationID
	indexed := data.memoriesByConversation[firstConversation]
	if len(indexed) == 0 || indexed[0].MemoryID != data.Memories[0].MemoryID {
		t.Fatalf("conversation index did not preserve memory order for %q", firstConversation)
	}
}

func TestLoadLocomoDatasetRejectsDuplicateStableIDs(t *testing.T) {
	t.Run("memory_id", func(t *testing.T) {
		dir := t.TempDir()
		memories := filepath.Join(dir, "memories.jsonl")
		questions := filepath.Join(dir, "questions.jsonl")
		writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"first stable fact"}
{"memory_id":"m1","conversation_id":"c2","session_id":"s1","speaker":"Noah","turn_index":1,"content":"second stable fact"}
`)
		writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"what fact?","gold_memory_ids":["m1"],"category":"single_hop_retrieval"}
`)
		_, err := loadLocomoDataset(memories, questions)
		if err == nil || !strings.Contains(err.Error(), `duplicate memory_id "m1"`) {
			t.Fatalf("load duplicate memory_id error = %v, want duplicate stable ID error", err)
		}
	})
	t.Run("question_id", func(t *testing.T) {
		dir := t.TempDir()
		memories := filepath.Join(dir, "memories.jsonl")
		questions := filepath.Join(dir, "questions.jsonl")
		writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"stable fact"}
`)
		writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"what fact?","gold_memory_ids":["m1"],"category":"single_hop_retrieval"}
{"question_id":"q1","conversation_id":"c1","question":"what other fact?","gold_memory_ids":["m1"],"category":"single_hop_retrieval"}
`)
		_, err := loadLocomoDataset(memories, questions)
		if err == nil || !strings.Contains(err.Error(), `duplicate question_id "q1"`) {
			t.Fatalf("load duplicate question_id error = %v, want duplicate stable ID error", err)
		}
	})
}

func TestLoadLocomoDatasetRejectsInvalidGoldStableIDs(t *testing.T) {
	t.Run("unknown memory_id", func(t *testing.T) {
		dir := t.TempDir()
		memories := filepath.Join(dir, "memories.jsonl")
		questions := filepath.Join(dir, "questions.jsonl")
		writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"stable fact"}
`)
		writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"what fact?","gold_memory_ids":["missing"],"category":"single_hop_retrieval"}
`)
		_, err := loadLocomoDataset(memories, questions)
		if err == nil || !strings.Contains(err.Error(), `unknown gold_memory_id "missing"`) {
			t.Fatalf("load unknown gold_memory_id error = %v, want unknown stable ID error", err)
		}
	})
	t.Run("out of conversation memory_id", func(t *testing.T) {
		dir := t.TempDir()
		memories := filepath.Join(dir, "memories.jsonl")
		questions := filepath.Join(dir, "questions.jsonl")
		writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"first stable fact"}
{"memory_id":"m2","conversation_id":"c2","session_id":"s1","speaker":"Noah","turn_index":1,"content":"second stable fact"}
`)
		writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"what fact?","gold_memory_ids":["m2"],"category":"single_hop_retrieval"}
`)
		_, err := loadLocomoDataset(memories, questions)
		if err == nil || !strings.Contains(err.Error(), `out-of-conversation gold_memory_id "m2"`) {
			t.Fatalf("load out-of-conversation gold_memory_id error = %v, want scoped stable ID error", err)
		}
	})
}

func TestLocomoScoringStrictAnyAndMRR(t *testing.T) {
	q := locomoQuestionRow{QuestionID: "q", ConversationID: "c", Category: "gold_ambiguity", Question: "who", GoldMemoryIDs: []string{"m2", "m4"}}
	got := scoreLocomoQuestion(q, []string{"m1", "m2", "m3", "m4"})
	if got.Rank != 2 || got.MRR != 0.5 {
		t.Fatalf("rank/mrr = %d/%f, want 2/0.5", got.Rank, got.MRR)
	}
	if got.RecallAnyAt5 != 1 || got.StrictRecallAt5 != 1 {
		t.Fatalf("@5 any/strict = %f/%f, want 1/1", got.RecallAnyAt5, got.StrictRecallAt5)
	}
	got = scoreLocomoQuestion(q, []string{"m1", "m2", "m3"})
	if got.RecallAnyAt5 != 1 || got.StrictRecallAt5 != 0 {
		t.Fatalf("partial @5 any/strict = %f/%f, want 1/0", got.RecallAnyAt5, got.StrictRecallAt5)
	}
}

func TestLocomoCategoryMetricAggregation(t *testing.T) {
	report := summarizeLocomoSystem("test", []locomoQuestionResult{
		{Category: "latest_state_retrieval", RecallAnyAt5: 1, RecallAnyAt10: 1, StrictRecallAt5: 1, StrictRecallAt10: 1, MRR: 1},
		{Category: "latest_state_retrieval", RecallAnyAt5: 0, RecallAnyAt10: 1, StrictRecallAt5: 0, StrictRecallAt10: 1, MRR: 0.5},
		{Category: "speaker_attribution", RecallAnyAt5: 1, RecallAnyAt10: 1, StrictRecallAt5: 1, StrictRecallAt10: 1, MRR: 1},
	})
	m := report.CategoryMetrics["latest_state_retrieval"]
	if m.Questions != 2 || m.RecallAnyAt5 != 0.5 || m.MRR != 0.75 {
		t.Fatalf("latest category metrics = %+v, want n=2 any5=.5 mrr=.75", m)
	}
}

func TestLocomoRecencyBaselineOrdering(t *testing.T) {
	items := []locomoMemoryRow{
		{MemoryID: "old", Timestamp: "2026-05-20T10:00:00Z", TurnIndex: 1},
		{MemoryID: "new", Timestamp: "2026-05-21T10:00:00Z", TurnIndex: 1},
		{MemoryID: "newer-turn", Timestamp: "2026-05-21T10:00:00Z", TurnIndex: 2},
	}
	sortLocomoRecency(items)
	got := locomoFirstIDs(items, 3)
	want := []string{"newer-turn", "new", "old"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recency ids = %v, want %v", got, want)
		}
	}
}

func TestLocomoFailureJSONLGeneration(t *testing.T) {
	data := locomoDataset{Memories: []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 1, Content: "wrong"}}, Questions: []locomoQuestionRow{{QuestionID: "q", ConversationID: "c", Question: "question", GoldMemoryIDs: []string{"gold"}, Category: "true_retrieval_failure"}}}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q", ConversationID: "c", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"gold"}, RetrievedIDs: []string{"m1"}, Rank: 0}}}
	path := filepath.Join(t.TempDir(), "failures.jsonl")
	if err := writeLocomoFailureAudit(path, data, []locomoSystemReport{report}); err != nil {
		t.Fatalf("write failures: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failures: %v", err)
	}
	var row locomoFailureRow
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &row); err != nil {
		t.Fatalf("decode failure row: %v", err)
	}
	if row.QuestionID != "q" || row.FailureCategory != "true_retrieval_failure" || len(row.TopHits) != 1 {
		t.Fatalf("failure row = %+v", row)
	}
}

func TestLocomoAnswerHintIsNotIndexedOrScored(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 1, Timestamp: "2026-05-20T10:00:00Z", Content: "The durable fact is hidden under code name orchid."}},
		Questions: []locomoQuestionRow{{QuestionID: "q", ConversationID: "c", Question: "What is the hidden durable fact?", GoldMemoryIDs: []string{"m1"}, Category: "lexical_miss", AnswerHint: "forbidden-answer-token"}},
	}
	if strings.Contains(locomoIndexableContent(data.Memories[0]), data.Questions[0].AnswerHint) {
		t.Fatalf("answer_hint leaked into indexable content")
	}
	got, err := retrieveLocomo(context.Background(), nil, data, data.Questions[0], "bm25", nil, 10)
	if err != nil {
		t.Fatalf("retrieve bm25: %v", err)
	}
	if len(got) != 1 || got[0] != "m1" {
		t.Fatalf("retrieved = %v, want gold by question/content only", got)
	}
}

func TestRunLocomoBenchmarkHonorsConfiguredLimit(t *testing.T) {
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	out := filepath.Join(dir, "locomo.json")
	writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"orchid marker primary"}
{"memory_id":"m2","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":2,"content":"orchid marker secondary"}
`)
	writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"orchid marker","gold_memory_ids":["m2"],"category":"single_hop_retrieval"}
`)
	if err := runLocomoBenchmark(context.Background(), config{LocomoMemoriesPath: memories, LocomoQuestionsPath: questions, OutPath: out, Limit: 1}); err != nil {
		t.Fatalf("run LOCOMO limit smoke: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report locomoReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	for _, system := range report.Systems {
		if len(system.QuestionsDetail) != 1 {
			t.Fatalf("%s question detail count = %d, want 1", system.System, len(system.QuestionsDetail))
		}
		if got := len(system.QuestionsDetail[0].RetrievedIDs); got > 1 {
			t.Fatalf("%s retrieved ids = %v, want configured limit 1", system.System, system.QuestionsDetail[0].RetrievedIDs)
		}
	}
}

func TestRunLocomoSmokeProducesReport(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "locomo.json")
	failures := filepath.Join(dir, "failures.jsonl")
	md := filepath.Join(dir, "report.md")
	if err := runLocomoBenchmark(context.Background(), config{LocomoMemoriesPath: filepath.Join("testdata", "locomo-smoke", "memories.jsonl"), LocomoQuestionsPath: filepath.Join("testdata", "locomo-smoke", "questions.jsonl"), OutPath: out, FailurePath: failures, LocomoMarkdownOut: md}); err != nil {
		t.Fatalf("run LOCOMO smoke: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report locomoReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Mode != "retrieval" || !report.NoLLMJudge || len(report.Systems) != 5 {
		t.Fatalf("report mode/judge/systems = %+v", report)
	}
	if _, err := os.Stat(failures); err != nil {
		t.Fatalf("failure JSONL missing: %v", err)
	}
	mdRaw, err := os.ReadFile(md)
	if err != nil {
		t.Fatalf("markdown missing: %v", err)
	}
	if !strings.Contains(string(mdRaw), "LOCOMO smoke validates the benchmark harness") || !strings.Contains(string(mdRaw), "No answer generation") {
		t.Fatalf("markdown missing required wording:\n%s", mdRaw)
	}
}
