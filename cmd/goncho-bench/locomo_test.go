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

func TestCheckLocomoLeakageUsesConversationIndex(t *testing.T) {
	data := locomoDataset{
		Questions: []locomoQuestionRow{{
			QuestionID:     "q1",
			ConversationID: "c1",
			Question:       "What phrase should not appear?",
			GoldMemoryIDs:  []string{"gold-id"},
			Category:       "leakage_guard",
			AnswerHint:     "secret answer",
		}},
		memoriesByConversation: map[string][]locomoMemoryRow{
			"c1": {{MemoryID: "m1", ConversationID: "c1", Content: "The secret answer mentions gold-id and asks What phrase should not appear?"}},
		},
	}
	checks := checkLocomoLeakage(data)
	if checks.AnswerTextInMemoryContent != 1 || checks.GoldIDInMemoryContent != 1 || checks.QuestionTextInMemory != 1 {
		t.Fatalf("leakage checks = %+v, want one answer, gold ID, and question-text leak from indexed memories", checks)
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

func TestLoadLocomoDatasetRejectsDuplicateGoldStableIDs(t *testing.T) {
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"stable fact"}
`)
	writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"what fact?","gold_memory_ids":["m1","m1"],"category":"single_hop_retrieval"}
`)
	_, err := loadLocomoDataset(memories, questions)
	if err == nil || !strings.Contains(err.Error(), `duplicate gold_memory_id "m1"`) {
		t.Fatalf("load duplicate gold_memory_id error = %v, want duplicate gold stable ID error", err)
	}
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
	if got.NDCGAt5 != 0.6509 || got.NDCGAt10 != 0.6509 {
		t.Fatalf("ndcg@5/@10 = %f/%f, want 0.6509/0.6509", got.NDCGAt5, got.NDCGAt10)
	}
	got = scoreLocomoQuestion(q, []string{"m1", "m2", "m3"})
	if got.RecallAnyAt5 != 1 || got.StrictRecallAt5 != 0 {
		t.Fatalf("partial @5 any/strict = %f/%f, want 1/0", got.RecallAnyAt5, got.StrictRecallAt5)
	}
}

func TestLocomoCategoryMetricAggregation(t *testing.T) {
	report := summarizeLocomoSystem("test", []locomoQuestionResult{
		{Category: "latest_state_retrieval", RecallAnyAt5: 1, RecallAnyAt10: 1, StrictRecallAt5: 1, StrictRecallAt10: 1, MRR: 1, NDCGAt5: 1, NDCGAt10: 1},
		{Category: "latest_state_retrieval", RecallAnyAt5: 0, RecallAnyAt10: 1, StrictRecallAt5: 0, StrictRecallAt10: 1, MRR: 0.5, NDCGAt5: 0.5, NDCGAt10: 0.5},
		{Category: "speaker_attribution", RecallAnyAt5: 1, RecallAnyAt10: 1, StrictRecallAt5: 1, StrictRecallAt10: 1, MRR: 1, NDCGAt5: 1, NDCGAt10: 1},
	})
	if report.NDCGAt5 != 0.8333 || report.NDCGAt10 != 0.8333 {
		t.Fatalf("system ndcg@5/@10 = %f/%f, want .8333/.8333", report.NDCGAt5, report.NDCGAt10)
	}
	m := report.CategoryMetrics["latest_state_retrieval"]
	if m.Questions != 2 || m.RecallAnyAt5 != 0.5 || m.MRR != 0.75 || m.NDCGAt5 != 0.75 || m.NDCGAt10 != 0.75 {
		t.Fatalf("latest category metrics = %+v, want n=2 any5=.5 mrr=.75 ndcg=.75", m)
	}
}

func TestLocomoLatencyMetricAggregation(t *testing.T) {
	report := summarizeLocomoSystem("test", []locomoQuestionResult{
		{RetrievalLatencyMs: 4},
		{RetrievalLatencyMs: 1},
		{RetrievalLatencyMs: 9},
		{RetrievalLatencyMs: 2},
	})
	want := locomoLatencyStats{Min: 1, P50: 2, P95: 9, Max: 9}
	if report.LatencyMs != want {
		t.Fatalf("latency stats = %+v, want %+v", report.LatencyMs, want)
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
	data := locomoDataset{Memories: []locomoMemoryRow{{MemoryID: "gold", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 0, Content: "expected"}, {MemoryID: "m1", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 1, Content: "wrong"}}, Questions: []locomoQuestionRow{{QuestionID: "q", ConversationID: "c", Question: "question", GoldMemoryIDs: []string{"gold"}, Category: "true_retrieval_failure"}}}
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

func TestLocomoFailureJSONLNotesUseRetrievedWindow(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "gold", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 0, Content: "expected"},
			{MemoryID: "m1", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 1, Content: "wrong"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q", ConversationID: "c", Question: "question", GoldMemoryIDs: []string{"gold"}, Category: "true_retrieval_failure"}},
	}
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
	if row.Notes != "no gold memory ID appeared in top 1" {
		t.Fatalf("failure notes = %q, want configured retrieved top-K window", row.Notes)
	}
}

func TestWriteLocomoFailureAuditRejectsUnknownQuestionID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "missing-q", ConversationID: "c1", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"m1"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `unknown question_id "missing-q"`) {
		t.Fatalf("write failure audit error = %v, want unknown question ID error", err)
	}
}

func TestWriteLocomoFailureAuditRejectsQuestionConversationMismatch(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q1", ConversationID: "c2", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"m2"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `conversation_id "c2" does not match fixture conversation_id "c1"`) {
		t.Fatalf("write failure audit error = %v, want question conversation mismatch error", err)
	}
}

func TestWriteLocomoFailureAuditRejectsUnknownGoldMemoryID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q1", ConversationID: "c1", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"missing-gold"}, RetrievedIDs: []string{"m1"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `unknown gold_memory_id "missing-gold"`) {
		t.Fatalf("write failure audit error = %v, want unknown gold stable ID error", err)
	}
}

func TestWriteLocomoFailureAuditRejectsOutOfConversationGoldMemoryID(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q1", ConversationID: "c1", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"m2"}, RetrievedIDs: []string{"m1"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `out-of-conversation gold_memory_id "m2"`) {
		t.Fatalf("write failure audit error = %v, want out-of-conversation gold stable ID error", err)
	}
}

func TestWriteLocomoFailureAuditRejectsUnknownRetrievedID(t *testing.T) {
	data := locomoDataset{
		Memories:  []locomoMemoryRow{{MemoryID: "m1", ConversationID: "c", SessionID: "s", Speaker: "user", TurnIndex: 1, Content: "known memory"}},
		Questions: []locomoQuestionRow{{QuestionID: "q", ConversationID: "c", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q", ConversationID: "c", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"missing"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `unknown retrieved memory_id "missing"`) {
		t.Fatalf("write failure audit error = %v, want unknown retrieved stable ID error", err)
	}
}

func TestWriteLocomoFailureAuditRejectsOutOfConversationRetrievedID(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "known memory"},
			{MemoryID: "m2", ConversationID: "c2", SessionID: "s2", Speaker: "Leo", TurnIndex: 1, Content: "wrong conversation memory"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "question", GoldMemoryIDs: []string{"m1"}, Category: "true_retrieval_failure"}},
	}
	report := locomoSystemReport{System: "goncho", QuestionsDetail: []locomoQuestionResult{{QuestionID: "q1", ConversationID: "c1", Category: "true_retrieval_failure", Question: "question", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"m2"}, Rank: 0}}}
	err := writeLocomoFailureAudit(filepath.Join(t.TempDir(), "failures.jsonl"), data, []locomoSystemReport{report})
	if err == nil || !strings.Contains(err.Error(), `out-of-conversation retrieved memory_id "m2"`) {
		t.Fatalf("write failure audit error = %v, want out-of-conversation retrieved stable ID error", err)
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

func TestRetrieveLocomoReturnsNoIDsForNonPositiveLimits(t *testing.T) {
	data := locomoDataset{
		Memories: []locomoMemoryRow{
			{MemoryID: "m1", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 1, Content: "orchid marker primary"},
			{MemoryID: "m2", ConversationID: "c1", SessionID: "s1", Speaker: "Maya", TurnIndex: 2, Content: "orchid marker secondary"},
		},
		Questions: []locomoQuestionRow{{QuestionID: "q1", ConversationID: "c1", Question: "orchid marker", GoldMemoryIDs: []string{"m2"}, Category: "single_hop_retrieval"}},
	}
	for _, system := range []string{"random", "recency", "bm25", "sqlite-fts5", "goncho"} {
		for _, limit := range []int{0, -1} {
			t.Run(system+"/limit", func(t *testing.T) {
				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("retrieve %s with limit %d panicked: %v", system, limit, recovered)
					}
				}()
				got, err := retrieveLocomo(context.Background(), nil, data, data.Questions[0], system, nil, limit)
				if err != nil {
					t.Fatalf("retrieve %s with limit %d: %v", system, limit, err)
				}
				if len(got) != 0 {
					t.Fatalf("retrieve %s with limit %d = %v, want no IDs", system, limit, got)
				}
			})
		}
	}
}

func TestRetrieveLocomoSQLiteFTSSkipsStoreForTokenlessQuery(t *testing.T) {
	dir := t.TempDir()
	badTemp := filepath.Join(dir, "not-dir")
	if err := os.WriteFile(badTemp, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create bad temp path: %v", err)
	}
	t.Setenv("TMPDIR", badTemp)

	items := []locomoMemoryRow{
		{MemoryID: "m1", ConversationID: "c1", TurnIndex: 1, Content: "older memory"},
		{MemoryID: "m2", ConversationID: "c1", TurnIndex: 2, Content: "newer memory"},
	}
	q := locomoQuestionRow{QuestionID: "q1", ConversationID: "c1", Question: "the and what"}
	got, err := retrieveLocomoSQLiteFTS(context.Background(), items, q, 1)
	if err != nil {
		t.Fatalf("tokenless SQLite FTS retrieval: %v", err)
	}
	if strings.Join(got, ",") != "m2" {
		t.Fatalf("tokenless SQLite FTS retrieval = %v, want recency fallback without temp SQLite store", got)
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
	var rawReport map[string]any
	if err := json.Unmarshal(raw, &rawReport); err != nil {
		t.Fatalf("decode raw report: %v", err)
	}
	if rawReport["top_k"] != float64(1) {
		t.Fatalf("report top_k = %v, want configured limit 1", rawReport["top_k"])
	}
	if rawReport["memory_token_estimate"] != float64(6) {
		t.Fatalf("report memory_token_estimate = %v, want deterministic content token estimate 6", rawReport["memory_token_estimate"])
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

func TestRunLocomoBenchmarkCapsGonchoStableIDFanoutToLimit(t *testing.T) {
	dir := t.TempDir()
	memories := filepath.Join(dir, "memories.jsonl")
	questions := filepath.Join(dir, "questions.jsonl")
	out := filepath.Join(dir, "locomo.json")
	writeTestFile(t, memories, `{"memory_id":"m1","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"orchid marker duplicate"}
{"memory_id":"m2","conversation_id":"c1","session_id":"s1","speaker":"Maya","turn_index":1,"content":"orchid marker duplicate"}
`)
	writeTestFile(t, questions, `{"question_id":"q1","conversation_id":"c1","question":"orchid marker","gold_memory_ids":["m1"],"category":"single_hop_retrieval"}
`)
	if err := runLocomoBenchmark(context.Background(), config{LocomoMemoriesPath: memories, LocomoQuestionsPath: questions, OutPath: out, Limit: 1}); err != nil {
		t.Fatalf("run LOCOMO duplicate fan-out smoke: %v", err)
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
		if system.System != "goncho" {
			continue
		}
		if len(system.QuestionsDetail) != 1 {
			t.Fatalf("goncho question detail count = %d, want 1", len(system.QuestionsDetail))
		}
		if got := len(system.QuestionsDetail[0].RetrievedIDs); got != 1 {
			t.Fatalf("goncho retrieved ids = %v, want duplicate content fan-out capped to limit 1", system.QuestionsDetail[0].RetrievedIDs)
		}
		return
	}
	t.Fatal("goncho system missing from report")
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
	var rawReport map[string]any
	if err := json.Unmarshal(raw, &rawReport); err != nil {
		t.Fatalf("decode raw report: %v", err)
	}
	systemsRaw, ok := rawReport["systems"].([]any)
	if !ok || len(systemsRaw) == 0 {
		t.Fatalf("raw systems = %#v, want non-empty system list", rawReport["systems"])
	}
	for _, systemRaw := range systemsRaw {
		system, ok := systemRaw.(map[string]any)
		if !ok {
			t.Fatalf("raw system = %#v, want object", systemRaw)
		}
		if _, ok := system["search_latency_ms"]; !ok {
			t.Fatalf("raw system %v missing search_latency_ms", system["system"])
		}
		if _, ok := system["rss_bytes"]; !ok {
			t.Fatalf("raw system %v missing rss_bytes", system["system"])
		}
		if _, ok := system["ndcg_at_5"]; !ok {
			t.Fatalf("raw system %v missing ndcg_at_5", system["system"])
		}
		if _, ok := system["ndcg_at_10"]; !ok {
			t.Fatalf("raw system %v missing ndcg_at_10", system["system"])
		}
		failureCategories, ok := system["failure_categories"].(map[string]any)
		if !ok || len(failureCategories) == 0 {
			t.Fatalf("raw system %v failure_categories = %#v, want non-empty metric counts", system["system"], system["failure_categories"])
		}
		latency, ok := system["latency_ms"].(map[string]any)
		if !ok || latency["min"] == nil || latency["p50"] == nil || latency["p95"] == nil || latency["max"] == nil {
			t.Fatalf("raw system %v latency_ms = %#v, want min/p50/p95/max", system["system"], system["latency_ms"])
		}
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
	if !strings.Contains(string(mdRaw), "- Top-K: `10`") {
		t.Fatalf("markdown missing effective top-K provenance:\n%s", mdRaw)
	}
	if !strings.Contains(string(mdRaw), "Memory token estimate") {
		t.Fatalf("markdown missing memory token estimate:\n%s", mdRaw)
	}
	if !strings.Contains(string(mdRaw), "Search latency ms") || !strings.Contains(string(mdRaw), "RSS bytes") {
		t.Fatalf("markdown missing resource metric columns:\n%s", mdRaw)
	}
	if !strings.Contains(string(mdRaw), "NDCG@5") || !strings.Contains(string(mdRaw), "NDCG@10") {
		t.Fatalf("markdown missing NDCG metric columns:\n%s", mdRaw)
	}
	if !strings.Contains(string(mdRaw), "## Failure categories") {
		t.Fatalf("markdown missing failure-category summary:\n%s", mdRaw)
	}
	if !strings.Contains(string(mdRaw), "Latency p95 ms") {
		t.Fatalf("markdown missing latency distribution columns:\n%s", mdRaw)
	}
}
