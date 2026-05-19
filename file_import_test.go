package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestService_ImportFileCreatesSessionMessagesWithFileMetadata(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	createdAt := time.Unix(1_714_558_400, 0).UTC()
	got, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-1",
		PeerID:      "telegram:6586915095",
		Filename:    "MEMORY.md",
		ContentType: "text/markdown",
		Content:     []byte("# Memory\n\nJuan prefers evidence-first reports."),
		Metadata: map[string]any{
			"source": "legacy-memory",
			"owner":  "juan",
		},
		Configuration: map[string]any{
			"reasoning": map[string]any{
				"observe": true,
			},
		},
		CreatedAt: &createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	if len(got.Unavailable) != 1 || got.Unavailable[0].Capability != "goncho_reasoning_queue" {
		t.Fatalf("Unavailable = %+v, want queue-unavailable evidence", got.Unavailable)
	}

	msg := got.Messages[0]
	if msg.SessionKey != "session-import-1" || msg.PeerID != "telegram:6586915095" || msg.Role != "user" {
		t.Fatalf("message identity = %+v, want ordinary user session message for required peer", msg)
	}
	if msg.Content != "# Memory\n\nJuan prefers evidence-first reports." {
		t.Fatalf("content = %q", msg.Content)
	}
	if msg.CreatedAt.Unix() != createdAt.Unix() {
		t.Fatalf("CreatedAt = %s, want %s", msg.CreatedAt, createdAt)
	}
	if msg.Metadata["source"] != "legacy-memory" || msg.Metadata["owner"] != "juan" {
		t.Fatalf("Metadata = %+v, want caller metadata preserved", msg.Metadata)
	}
	if !nestedBool(msg.Configuration, "reasoning", "observe") {
		t.Fatalf("Configuration = %+v, want caller configuration preserved", msg.Configuration)
	}
	if msg.File.FileID == "" {
		t.Fatal("FileID is empty")
	}
	wantFile := FileImportMetadata{
		FileID:           msg.File.FileID,
		Filename:         "MEMORY.md",
		ChunkIndex:       0,
		TotalChunks:      1,
		OriginalFileSize: int64(len("# Memory\n\nJuan prefers evidence-first reports.")),
		ContentType:      "text/markdown",
		ChunkCharacterRange: [2]int{
			0,
			len("# Memory\n\nJuan prefers evidence-first reports."),
		},
	}
	if msg.File != wantFile {
		t.Fatalf("File metadata = %+v, want %+v", msg.File, wantFile)
	}

	rows := loadImportedTurns(t, svc.db, "session-import-1")
	if len(rows) != 1 {
		t.Fatalf("turn rows len = %d, want 1", len(rows))
	}
	if rows[0].role != "user" || rows[0].chatID != "telegram:6586915095" || rows[0].content != msg.Content {
		t.Fatalf("turn row = %+v, want ordinary imported user turn", rows[0])
	}
	if rows[0].tsUnix != createdAt.Unix() {
		t.Fatalf("ts_unix = %d, want %d", rows[0].tsUnix, createdAt.Unix())
	}
	assertMetaValue(t, rows[0].meta, "file_id", msg.File.FileID)
	assertMetaValue(t, rows[0].meta, "filename", "MEMORY.md")
	assertMetaValue(t, rows[0].meta, "chunk_index", float64(0))
	assertMetaValue(t, rows[0].meta, "total_chunks", float64(1))
	assertMetaValue(t, rows[0].meta, "original_file_size", float64(len("# Memory\n\nJuan prefers evidence-first reports.")))
	assertMetaValue(t, rows[0].meta, "content_type", "text/markdown")
	assertMetaValue(t, rows[0].meta, "metadata.source", "legacy-memory")
	assertMetaValue(t, rows[0].meta, "configuration.reasoning.observe", true)

	ctx, err := svc.Context(context.Background(), ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "session-import-1",
		MaxTokens:  400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.RecentMessages) != 1 || ctx.RecentMessages[0].Content != msg.Content {
		t.Fatalf("RecentMessages = %+v, want imported chunk as normal session message", ctx.RecentMessages)
	}
}

func TestService_ImportFileSupportsTextMarkdownAndJSON(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	for _, tc := range []struct {
		name        string
		filename    string
		contentType string
		content     []byte
		assert      func(t *testing.T, content string)
	}{
		{
			name:        "plain text",
			filename:    "USER.txt",
			contentType: "text/plain",
			content:     []byte("Plain text memory."),
			assert: func(t *testing.T, content string) {
				t.Helper()
				if content != "Plain text memory." {
					t.Fatalf("content = %q, want decoded text", content)
				}
			},
		},
		{
			name:        "markdown",
			filename:    "SOUL.md",
			contentType: "text/markdown",
			content:     []byte("## Soul\n\nMarkdown memory."),
			assert: func(t *testing.T, content string) {
				t.Helper()
				if content != "## Soul\n\nMarkdown memory." {
					t.Fatalf("content = %q, want decoded markdown", content)
				}
			},
		},
		{
			name:        "json",
			filename:    "memory.json",
			contentType: "application/json",
			content:     []byte("{\n  \"prefers\": [\"evidence\", \"exactness\"],\n  \"active\": true\n}"),
			assert: func(t *testing.T, content string) {
				t.Helper()
				var decoded map[string]any
				if err := json.Unmarshal([]byte(content), &decoded); err != nil {
					t.Fatalf("content = %q, want valid JSON text: %v", content, err)
				}
				if decoded["active"] != true {
					t.Fatalf("decoded JSON = %+v, want active=true", decoded)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.ImportFile(context.Background(), ImportFileParams{
				SessionKey:  "session-" + strings.ReplaceAll(tc.name, " ", "-"),
				PeerID:      "telegram:6586915095",
				Filename:    tc.filename,
				ContentType: tc.contentType,
				Content:     tc.content,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Messages) != 1 {
				t.Fatalf("messages len = %d, want 1", len(got.Messages))
			}
			tc.assert(t, got.Messages[0].Content)
			if got.Messages[0].File.ContentType != tc.contentType {
				t.Fatalf("content_type metadata = %q, want %q", got.Messages[0].File.ContentType, tc.contentType)
			}
		})
	}
}

func TestService_ImportFileRejectsUnsupportedTypesBeforeWrites(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-unsupported",
		PeerID:      "telegram:6586915095",
		Filename:    "scan.pdf",
		ContentType: "application/pdf",
		Content:     []byte("%PDF original bytes"),
	})
	if err == nil {
		t.Fatal("expected unsupported content type error")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("error = %v, want unsupported content type evidence", err)
	}
	rows := loadImportedTurns(t, svc.db, "session-import-unsupported")
	if len(rows) != 0 {
		t.Fatalf("turn rows = %+v, want no writes for unsupported content type", rows)
	}
}

func TestService_ImportFileRequiresPeerIDBeforeWrites(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-missing-peer",
		Filename:    "USER.txt",
		ContentType: "text/plain",
		Content:     []byte("memory"),
	})
	if err == nil {
		t.Fatal("expected peer_id required error")
	}
	if !strings.Contains(err.Error(), "peer_id is required") {
		t.Fatalf("error = %v, want peer_id validation", err)
	}
	rows := loadImportedTurns(t, svc.db, "session-import-missing-peer")
	if len(rows) != 0 {
		t.Fatalf("turn rows = %+v, want no writes without peer_id", rows)
	}
}

func TestService_ImportFileChunksAtHonchoRuntimeLimit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	content := strings.Repeat("a", DefaultMaxMessageSize) + strings.Repeat("b", 10)
	got, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-chunked",
		PeerID:      "telegram:6586915095",
		Filename:    "long.txt",
		ContentType: "text/plain",
		Content:     []byte(content),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(got.Messages))
	}
	if len(got.Messages[0].Content) != DefaultMaxMessageSize {
		t.Fatalf("first chunk len = %d, want %d", len(got.Messages[0].Content), DefaultMaxMessageSize)
	}
	if got.Messages[0].File.ChunkCharacterRange != [2]int{0, DefaultMaxMessageSize} {
		t.Fatalf("first range = %+v", got.Messages[0].File.ChunkCharacterRange)
	}
	if got.Messages[1].Content != strings.Repeat("b", 10) {
		t.Fatalf("second chunk = %q", got.Messages[1].Content)
	}
	if got.Messages[1].File.ChunkCharacterRange != [2]int{DefaultMaxMessageSize, DefaultMaxMessageSize + 10} {
		t.Fatalf("second range = %+v", got.Messages[1].File.ChunkCharacterRange)
	}
	for i, msg := range got.Messages {
		if msg.File.ChunkIndex != i || msg.File.TotalChunks != 2 {
			t.Fatalf("message %d file metadata = %+v, want chunk index %d of 2", i, msg.File, i)
		}
	}
}

func TestService_ImportFileDoesNotPersistOriginalFileBytes(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	raw := "{\n  \"z\": 1,\n  \"legacy\": \"memory\"\n}\n"
	got, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-json",
		PeerID:      "telegram:6586915095",
		Filename:    "memory.json",
		ContentType: "application/json",
		Content:     []byte(raw),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].Content == raw {
		t.Fatalf("message content persisted raw upload bytes: %q", got.Messages[0].Content)
	}

	dump := dumpSessionRows(t, svc.db, "session-import-json")
	if strings.Contains(dump, raw) {
		t.Fatalf("database row dump persisted original file bytes %q in %q", raw, dump)
	}
	if !strings.Contains(dump, `"legacy"`) {
		t.Fatalf("database row dump = %q, want extracted JSON message content", dump)
	}
}

func nestedBool(root map[string]any, path ...string) bool {
	var current any = root
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current = m[key]
	}
	got, ok := current.(bool)
	return ok && got
}

type importedTurnRow struct {
	role    string
	chatID  string
	content string
	tsUnix  int64
	meta    map[string]any
}

func loadImportedTurns(t *testing.T, db *sql.DB, sessionKey string) []importedTurnRow {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `
		SELECT role, chat_id, content, ts_unix, COALESCE(meta_json, '{}')
		FROM turns
		WHERE session_id = ?
		ORDER BY id ASC
	`, sessionKey)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []importedTurnRow
	for rows.Next() {
		var row importedTurnRow
		var rawMeta string
		if err := rows.Scan(&row.role, &row.chatID, &row.content, &row.tsUnix, &rawMeta); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(rawMeta), &row.meta); err != nil {
			t.Fatalf("meta_json = %q: %v", rawMeta, err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertMetaValue(t *testing.T, meta map[string]any, dotted string, want any) {
	t.Helper()

	var got any = meta
	for _, part := range strings.Split(dotted, ".") {
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("meta path %q hit non-object %T in %+v", dotted, got, meta)
		}
		got = m[part]
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("meta[%s] = %#v (%T), want %#v (%T)", dotted, got, got, want, want)
	}
}

func dumpSessionRows(t *testing.T, db *sql.DB, sessionKey string) string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `
		SELECT content, COALESCE(meta_json, '')
		FROM turns
		WHERE session_id = ?
		ORDER BY id ASC
	`, sessionKey)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var b strings.Builder
	for rows.Next() {
		var content, meta string
		if err := rows.Scan(&content, &meta); err != nil {
			t.Fatal(err)
		}
		b.WriteString(content)
		b.WriteByte('\n')
		b.WriteString(meta)
		b.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
