package goncho

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPortableJSONLExportImportRoundTripPreservesIDsStateAndTombstones(t *testing.T) {
	ctx := context.Background()
	src, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(src.db); err != nil {
		t.Fatalf("RunMigrations src: %v", err)
	}
	seedPortableExportFixture(t, ctx, src)

	exported, err := src.ExportPortableJSONL(ctx, PortableExportParams{Peer: "peer-portable", SessionKey: "sess-portable", IncludeSnapshots: true})
	if err != nil {
		t.Fatalf("ExportPortableJSONL: %v", err)
	}
	if exported.Manifest.SchemaVersion != PortableExportSchemaVersion || exported.Manifest.Checksum == "" || exported.Manifest.Counts["observations"] == 0 || exported.Manifest.Counts["snapshots"] == 0 {
		t.Fatalf("manifest = %+v, want schema/checksum/counts", exported.Manifest)
	}

	dst, dstCleanup := newTestService(t)
	defer dstCleanup()
	if err := RunMigrations(dst.db); err != nil {
		t.Fatalf("RunMigrations dst: %v", err)
	}
	preview, err := dst.PreviewPortableImport(ctx, exported.JSONL)
	if err != nil {
		t.Fatalf("PreviewPortableImport: %v", err)
	}
	if preview.Mutates || !preview.SafeToApply || preview.Counts["observations"] != 1 || preview.Counts["messages"] != 2 || preview.Counts["conclusions"] != 2 || preview.Counts["review_items"] != 1 || preview.Counts["memory_slots"] != 1 || preview.Counts["snapshots"] != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	applied, err := dst.ImportPortableJSONL(ctx, PortableImportParams{JSONL: exported.JSONL, Apply: true})
	if err != nil {
		t.Fatalf("ImportPortableJSONL apply: %v", err)
	}
	if !applied.Mutates || applied.Applied["conclusions"] != 2 || applied.ManifestChecksum != exported.Manifest.Checksum {
		t.Fatalf("applied = %+v", applied)
	}
	obs, err := dst.ListObservations(ctx, ObservationQuery{PeerID: "peer-portable", SessionKey: "sess-portable", Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if len(obs.Observations) != 1 || obs.Observations[0].ID != "obs-portable-1" || !obs.Observations[0].Redacted {
		t.Fatalf("observations = %+v, want preserved id/redaction", obs.Observations)
	}
	if countExportRows(t, dst, `SELECT COUNT(*) FROM turns WHERE session_id = 'sess-portable'`) != 2 {
		t.Fatalf("messages not imported")
	}
	if got := exportConclusionStatus(t, dst, "Portable archived memory should stay tombstoned."); got != "archived" {
		t.Fatalf("archived conclusion status = %q", got)
	}
	open, err := dst.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-portable", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(open.Items) != 1 || open.Items[0].ID != "review-portable-1" || len(open.Items[0].EvidenceIDs) == 0 {
		t.Fatalf("review items = %+v, want preserved id/evidence", open.Items)
	}
	if countExportRows(t, dst, `SELECT COUNT(*) FROM goncho_memory_slots WHERE peer_id = 'peer-portable' AND name = 'deprecated_pref' AND deleted = 1`) != 1 {
		t.Fatalf("deleted memory slot tombstone not preserved")
	}
	trace, err := dst.Recall(ctx, RecallQuery{Peer: "peer-portable", SessionKey: "sess-portable", Query: "tombstoned", Limit: 5})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	for _, selected := range trace.Selected {
		if strings.Contains(selected.Candidate.Content, "archived memory") {
			t.Fatalf("archived memory recalled: %+v", trace.Selected)
		}
	}
}

func TestPortableMarkdownExportIsDeterministicWithBacklinksProvenanceAndWarnings(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	seedPortableExportFixture(t, ctx, svc)

	first, err := svc.ExportPortableMarkdown(ctx, PortableExportParams{Peer: "peer-portable", SessionKey: "sess-portable", IncludeSnapshots: true})
	if err != nil {
		t.Fatalf("ExportPortableMarkdown first: %v", err)
	}
	second, err := svc.ExportPortableMarkdown(ctx, PortableExportParams{Peer: "peer-portable", SessionKey: "sess-portable", IncludeSnapshots: true})
	if err != nil {
		t.Fatalf("ExportPortableMarkdown second: %v", err)
	}
	if first != second {
		t.Fatalf("markdown export not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	for _, want := range []string{"# Goncho Portable Memory Export", "[[session:sess-portable]]", "provenance:", "review_status: open", "stale_warning:", "obs-portable-1", "Portable active memory survives export."} {
		if !strings.Contains(first, want) {
			t.Fatalf("markdown missing %q:\n%s", want, first)
		}
	}
}

func TestPortableImportPreviewDetectsStableIDConflictsAndRedactionSummary(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	seedPortableExportFixture(t, ctx, svc)
	exported, err := svc.ExportPortableJSONL(ctx, PortableExportParams{Peer: "peer-portable", SessionKey: "sess-portable", RedactionPolicy: "safe"})
	if err != nil {
		t.Fatalf("ExportPortableJSONL: %v", err)
	}
	preview, err := svc.PreviewPortableImport(ctx, exported.JSONL)
	if err != nil {
		t.Fatalf("PreviewPortableImport conflict: %v", err)
	}
	if preview.SafeToApply || len(preview.Conflicts) == 0 || !strings.Contains(preview.Conflicts[0].StableID, "obs-portable-1") {
		t.Fatalf("preview = %+v, want fail-closed stable ID conflict", preview)
	}
	if preview.Redaction.RedactedObservations == 0 || preview.Redaction.Policy != "safe" {
		t.Fatalf("redaction summary = %+v, want exported redaction evidence", preview.Redaction)
	}
	if _, err := svc.ImportPortableJSONL(ctx, PortableImportParams{JSONL: exported.JSONL, Apply: false}); err == nil || !strings.Contains(err.Error(), "apply") {
		t.Fatalf("dry import err = %v, want explicit apply requirement", err)
	}
}

func seedPortableExportFixture(t *testing.T, ctx context.Context, svc *Service) {
	t.Helper()
	observedAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	if _, err := svc.Observe(ctx, ObservationParams{ID: "obs-portable-1", Kind: ObservationKindUserPrompt, PeerID: "peer-portable", SessionKey: "sess-portable", Input: "redacted input", Output: "redacted output", Metadata: map[string]string{"redacted": "true"}, ObservedAt: observedAt}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_observations SET redacted = 1, redaction_count = 1 WHERE id = 'obs-portable-1'`); err != nil {
		t.Fatalf("mark observation redacted: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-portable", Messages: []CreateMessage{
		{Peer: "peer-portable", Role: "user", Content: "portable user message", CreatedAt: observedAt},
		{Peer: "peer-portable", Role: "assistant", Content: "portable assistant message", CreatedAt: observedAt.Add(time.Second)},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	active, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-portable", SessionKey: "sess-portable", Conclusion: "Portable active memory survives export."})
	if err != nil {
		t.Fatalf("Conclude active: %v", err)
	}
	archived, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-portable", SessionKey: "sess-portable", Conclusion: "Portable archived memory should stay tombstoned."})
	if err != nil {
		t.Fatalf("Conclude archived: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_conclusions SET id = ?, created_at = ?, updated_at = ? WHERE id = ?`, 91001, observedAt.Unix(), observedAt.Unix(), active.ID); err != nil {
		t.Fatalf("stabilize active id: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_conclusions SET id = ?, status = 'archived', created_at = ?, updated_at = ? WHERE id = ?`, 91002, observedAt.Unix(), observedAt.Unix(), archived.ID); err != nil {
		t.Fatalf("stabilize archived id: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO goncho_review_items(id, kind, status, workspace_id, peer_id, session_key, subject_id, related_id, reason, evidence_ids_json, created_at, resolution, resolved_by, resolution_reason, resolved_at) VALUES('review-portable-1', 'stale', 'open', ?, 'peer-portable', 'sess-portable', 'conclusion:91002', '', 'archived memory needs stale warning in markdown', '["obs-portable-1"]', ?, '', '', '', NULL)`, svc.workspaceID, observedAt.UnixNano()); err != nil {
		t.Fatalf("insert review: %v", err)
	}
	if _, err := svc.CreateMemorySlot(ctx, MemorySlotParams{Peer: "peer-portable", Scope: MemoryScopeWorkspace, Name: "deprecated_pref", Kind: "preference", Value: "old value"}); err != nil {
		t.Fatalf("CreateMemorySlot: %v", err)
	}
	if _, err := svc.DeleteMemorySlot(ctx, MemorySlotQuery{Peer: "peer-portable", Scope: MemoryScopeWorkspace, Name: "deprecated_pref"}); err != nil {
		t.Fatalf("DeleteMemorySlot: %v", err)
	}
}

func countExportRows(t *testing.T, svc *Service, query string) int {
	t.Helper()
	var count int
	if err := svc.db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func exportConclusionStatus(t *testing.T, svc *Service, content string) string {
	t.Helper()
	var status string
	if err := svc.db.QueryRow(`SELECT status FROM goncho_conclusions WHERE content = ?`, content).Scan(&status); err != nil {
		t.Fatalf("conclusion status: %v", err)
	}
	return status
}

func decodePortableLines(t *testing.T, jsonl []byte) []PortableExportRecord {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(jsonl), []byte("\n"))
	out := make([]PortableExportRecord, 0, len(lines))
	for _, line := range lines {
		var record PortableExportRecord
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("decode line: %v\n%s", err, line)
		}
		out = append(out, record)
	}
	return out
}
