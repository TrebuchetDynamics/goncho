package goncho

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSessionEndSummary_ProducesStructuredCapsule(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	messages := []Message{
		{Role: "user", Content: "Let's fix the auth bug in internal/auth/handler.go"},
		{Role: "assistant", Content: "I decided to use JWT tokens instead of sessions. The file internal/auth/handler.go needs changes."},
		{Role: "user", Content: "What about the config.toml changes?"},
		{Role: "assistant", Content: "I also updated cmd/gormes/gateway.go and internal/config/auth.go. Next: we need to add tests."},
	}

	ctx := context.Background()
	if err := svc.OnSessionEnd(ctx, "sess-test", messages); err != nil {
		t.Fatalf("OnSessionEnd: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	result, err := getSessionSummary(ctx, svc.db, svc.workspaceID, "sess-test", "structured")
	if err != nil {
		t.Fatalf("getSessionSummary: %v", err)
	}
	if result == nil {
		t.Fatal("expected structured summary to be stored")
	}

	var summary StructuredSummary
	if err := json.Unmarshal([]byte(result.Content), &summary); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(summary.FilesModified) == 0 {
		t.Error("expected files_modified to be populated")
	}
	if len(summary.DecisionsMade) == 0 {
		t.Error("expected decisions_made to be populated")
	}
}

func TestSessionEndSummary_AsyncNonBlocking(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	messages := []Message{
		{Role: "user", Content: "hello"},
	}

	start := time.Now()
	if err := svc.OnSessionEnd(context.Background(), "sess-async", messages); err != nil {
		t.Fatalf("OnSessionEnd: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("OnSessionEnd took %v, expected non-blocking return", elapsed)
	}
}

func TestSessionEndSummary_StoredInSessionSummariesTable(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	messages := []Message{
		{Role: "assistant", Content: "I decided to use SQLite. Updated internal/goncho/sql.go."},
	}

	ctx := context.Background()
	if err := svc.OnSessionEnd(ctx, "sess-store", messages); err != nil {
		t.Fatalf("OnSessionEnd: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	var count int
	err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goncho_session_summaries WHERE workspace_id = ? AND session_key = ? AND summary_type = ?`,
		svc.workspaceID, "sess-store", "structured").Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 structured summary row, got %d", count)
	}
}

func TestSessionEndSummary_EmptyMessages(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.OnSessionEnd(ctx, "sess-empty", nil); err != nil {
		t.Fatalf("OnSessionEnd with nil messages: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	result, err := getSessionSummary(ctx, svc.db, svc.workspaceID, "sess-empty", "structured")
	if err != nil {
		t.Fatalf("getSessionSummary: %v", err)
	}
	if result == nil {
		t.Fatal("expected empty structured summary to be stored")
	}

	var summary StructuredSummary
	if err := json.Unmarshal([]byte(result.Content), &summary); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(summary.FilesModified) != 0 || len(summary.DecisionsMade) != 0 {
		t.Errorf("expected empty summary, got %+v", summary)
	}
}

func TestExtractStructuredSummary_FileDetection(t *testing.T) {
	messages := []Message{
		{Role: "assistant", Content: "Modified internal/goncho/sql.go and cmd/gormes/gateway.go"},
	}
	summary := extractStructuredSummary(messages)
	if len(summary.FilesModified) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(summary.FilesModified), summary.FilesModified)
	}
}

func TestExtractStructuredSummary_DecisionDetection(t *testing.T) {
	messages := []Message{
		{Role: "assistant", Content: "I decided to use SQLite instead of PostgreSQL"},
	}
	summary := extractStructuredSummary(messages)
	if len(summary.DecisionsMade) == 0 {
		t.Error("expected decisions to be detected")
	}
}

func TestExtractStructuredSummary_Deduplication(t *testing.T) {
	messages := []Message{
		{Role: "assistant", Content: "Updated internal/goncho/sql.go"},
		{Role: "user", Content: "Did you change internal/goncho/sql.go?"},
		{Role: "assistant", Content: "Yes, internal/goncho/sql.go was modified"},
	}
	summary := extractStructuredSummary(messages)
	if len(summary.FilesModified) != 1 {
		t.Errorf("expected 1 unique file, got %d: %v", len(summary.FilesModified), summary.FilesModified)
	}
}

func TestExtractStructuredSummary_QuestionDetection(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "What about the config.toml changes?"},
	}
	summary := extractStructuredSummary(messages)
	if len(summary.OpenQuestions) == 0 {
		t.Error("expected questions to be detected")
	}
}
