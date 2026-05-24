package goncho

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestContractContextParamsSummaryOptionsJSONShape(t *testing.T) {
	includeSummary := false

	raw, err := json.Marshal(ContextParams{
		Peer:        "telegram:6586915095",
		Tokens:      1500,
		SearchQuery: "coding preferences",
		Summary:     &includeSummary,
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	for _, want := range []string{
		`"tokens":1500`,
		`"search_query":"coding preferences"`,
		`"summary":false`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ContextParams JSON missing %s in %s", want, raw)
		}
	}
}

func TestService_ContextSummariesUseDefaultCadenceAndSlots(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	ids := seedSummaryContextTurns(t, ctx, svc, "sess-summary", 40, 3)
	includeSummary := true

	if _, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "sess-summary",
		Tokens:     120,
		Summary:    &includeSummary,
	}); err != nil {
		t.Fatal(err)
	}

	assertSummarySlot(t, ctx, svc, "sess-summary", "short", ids[39])
	assertNoSummarySlot(t, ctx, svc, "sess-summary", "long")

	ids = append(ids, seedSummaryContextTurns(t, ctx, svc, "sess-summary", 20, 3)...)
	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "sess-summary",
		Tokens:     120,
		Summary:    &includeSummary,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertSummarySlot(t, ctx, svc, "sess-summary", "short", ids[59])
	assertSummarySlot(t, ctx, svc, "sess-summary", "long", ids[59])
	if got.Summary == nil {
		t.Fatalf("Context Summary = nil, want the longest fitting summary; unavailable=%+v", got.Unavailable)
	}
	if got.Summary.SummaryType != "long" {
		t.Fatalf("Context Summary type = %q, want long", got.Summary.SummaryType)
	}
}

func TestService_ContextSummaryTrueSplitsTokenBudgetAndSkipsCoveredMessages(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	ids := seedSummaryContextTurns(t, ctx, svc, "sess-budget", 8, 3)
	upsertTestSessionSummary(t, ctx, svc, "sess-budget", "short", "compressed history note", ids[3], 4)
	includeSummary := true

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "sess-budget",
		Tokens:     10,
		Summary:    &includeSummary,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.Summary == nil || got.Summary.Content != "compressed history note" {
		t.Fatalf("Summary = %+v, want compressed history note; unavailable=%+v", got.Summary, got.Unavailable)
	}
	if len(got.RecentMessages) != 2 {
		t.Fatalf("RecentMessages len = %d, want 2 from the 60 percent message budget: %+v", len(got.RecentMessages), got.RecentMessages)
	}
	for _, msg := range got.RecentMessages {
		if strings.Contains(msg.Content, "turn-04") {
			t.Fatalf("RecentMessages double-billed covered message: %+v", got.RecentMessages)
		}
	}
	if got.RecentMessages[0].Content != "turn-07 word word" || got.RecentMessages[1].Content != "turn-08 word word" {
		t.Fatalf("RecentMessages = %+v, want newest two messages after summary coverage", got.RecentMessages)
	}
}

func TestService_ContextSummaryFalseUsesFullBudgetForRecentMessages(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	ids := seedSummaryContextTurns(t, ctx, svc, "sess-no-summary", 8, 3)
	upsertTestSessionSummary(t, ctx, svc, "sess-no-summary", "short", "compressed history note", ids[3], 4)
	includeSummary := false

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "sess-no-summary",
		Tokens:     10,
		Summary:    &includeSummary,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.Summary != nil {
		t.Fatalf("Summary = %+v, want nil when summary=false", got.Summary)
	}
	if len(got.RecentMessages) != 3 {
		t.Fatalf("RecentMessages len = %d, want 3 from the full token budget: %+v", len(got.RecentMessages), got.RecentMessages)
	}
	if got.RecentMessages[0].Content != "turn-06 word word" || got.RecentMessages[2].Content != "turn-08 word word" {
		t.Fatalf("RecentMessages = %+v, want newest three messages", got.RecentMessages)
	}
	if hasUnavailableField(got, "summary_absent") {
		t.Fatalf("Unavailable = %+v, want no summary_absent evidence when summary=false", got.Unavailable)
	}
}

func TestService_ContextReturnsSummaryAbsentEvidenceWhenNoSummaryFits(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	ids := seedSummaryContextTurns(t, ctx, svc, "sess-degraded", 4, 2)
	upsertTestSessionSummary(t, ctx, svc, "sess-degraded", "short", strings.Repeat("large ", 50), ids[3], 50)
	includeSummary := true

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "sess-degraded",
		Tokens:     20,
		Summary:    &includeSummary,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.Summary != nil {
		t.Fatalf("Summary = %+v, want nil when no summary fits", got.Summary)
	}
	if len(got.RecentMessages) != 4 {
		t.Fatalf("RecentMessages len = %d, want recent messages under the full fallback budget", len(got.RecentMessages))
	}
	if !hasUnavailableField(got, "summary_absent") {
		t.Fatalf("Unavailable = %+v, missing summary_absent evidence", got.Unavailable)
	}
}

func seedSummaryContextTurns(t *testing.T, ctx context.Context, svc *Service, sessionKey string, count int, wordsPerMessage int) []int64 {
	t.Helper()

	if wordsPerMessage < 1 {
		wordsPerMessage = 1
	}
	var maxTS int64
	if err := svc.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(ts_unix), 0)
		FROM turns
		WHERE chat_id = ? OR session_id = ?
	`, sessionKey, sessionKey).Scan(&maxTS); err != nil {
		t.Fatalf("max turn timestamp: %v", err)
	}

	ids := make([]int64, 0, count)
	for i := 1; i <= count; i++ {
		var existing int
		if err := svc.db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM turns
			WHERE chat_id = ? OR session_id = ?
		`, sessionKey, sessionKey).Scan(&existing); err != nil {
			t.Fatalf("count existing turns: %v", err)
		}
		ordinal := existing + 1
		content := summaryTestMessageContent(ordinal, wordsPerMessage)
		res, err := svc.db.ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`,
			sessionKey, "user", content, maxTS+int64(i), sessionKey,
		)
		if err != nil {
			t.Fatalf("insert turn %d: %v", ordinal, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("last insert id: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func summaryTestMessageContent(ordinal int, wordsPerMessage int) string {
	words := []string{fmt.Sprintf("turn-%02d", ordinal)}
	for len(words) < wordsPerMessage {
		words = append(words, "word")
	}
	return strings.Join(words, " ")
}

func upsertTestSessionSummary(t *testing.T, ctx context.Context, svc *Service, sessionKey, summaryType, content string, messageID int64, tokenCount int) {
	t.Helper()

	if err := upsertSessionSummary(ctx, svc.db, sessionSummaryRow{
		WorkspaceID: svc.workspaceID,
		SessionKey:  sessionKey,
		SummaryType: summaryType,
		Content:     content,
		MessageID:   messageID,
		CreatedAt:   time.Now().Unix(),
		TokenCount:  tokenCount,
	}); err != nil {
		t.Fatalf("upsert session summary: %v", err)
	}
}

func assertSummarySlot(t *testing.T, ctx context.Context, svc *Service, sessionKey, summaryType string, wantMessageID int64) {
	t.Helper()

	var messageID int64
	var tokenCount int
	if err := svc.db.QueryRowContext(ctx, `
		SELECT message_id, token_count
		FROM goncho_session_summaries
		WHERE workspace_id = ? AND session_key = ? AND summary_type = ?
	`, svc.workspaceID, sessionKey, summaryType).Scan(&messageID, &tokenCount); err != nil {
		t.Fatalf("summary slot %s missing: %v", summaryType, err)
	}
	if messageID != wantMessageID {
		t.Fatalf("%s summary message_id = %d, want %d", summaryType, messageID, wantMessageID)
	}
	if tokenCount <= 0 {
		t.Fatalf("%s summary token_count = %d, want > 0", summaryType, tokenCount)
	}
}

func assertNoSummarySlot(t *testing.T, ctx context.Context, svc *Service, sessionKey, summaryType string) {
	t.Helper()

	var summaryTypes []string
	rows, err := svc.db.QueryContext(ctx, `
		SELECT summary_type
		FROM goncho_session_summaries
		WHERE workspace_id = ? AND session_key = ?
	`, svc.workspaceID, sessionKey)
	if err != nil {
		t.Fatalf("query summaries: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var got string
		if err := rows.Scan(&got); err != nil {
			t.Fatalf("scan summary type: %v", err)
		}
		summaryTypes = append(summaryTypes, got)
	}
	if slices.Contains(summaryTypes, summaryType) {
		t.Fatalf("summary slots = %v, want no %s slot", summaryTypes, summaryType)
	}
}

func hasUnavailableField(got ContextResult, field string) bool {
	for _, item := range got.Unavailable {
		if item.Field == field {
			return true
		}
	}
	return false
}
