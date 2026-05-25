package goncho

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRetentionPreviewListsCandidatesWithoutWrites(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	old := seedRetentionConclusion(t, ctx, svc, "peer-retention", "sess-old", "Old retention candidate should be archived.", time.Now().Add(-90*24*time.Hour))
	seedRetentionConclusion(t, ctx, svc, "peer-retention", "sess-new", "Fresh retention memory must stay active.", time.Now())

	preview, err := svc.PreviewRetention(ctx, RetentionPolicy{MaxAge: 30 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("PreviewRetention: %v", err)
	}
	if preview.Mutates || len(preview.Candidates) != 1 || preview.Candidates[0].StableID != "conclusion:"+itoa64(old.ID) {
		t.Fatalf("preview = %+v, want one non-mutating old conclusion candidate", preview)
	}
	if !strings.Contains(strings.Join(preview.Candidates[0].Reasons, "\n"), "max_age") {
		t.Fatalf("candidate reasons = %+v, want max_age", preview.Candidates[0].Reasons)
	}
	active := countRetentionRows(t, svc, "goncho_conclusions", "status = 'processed'")
	audit := countRetentionRows(t, svc, "goncho_retention_audit", "1=1")
	if active != 2 || audit != 0 {
		t.Fatalf("after preview active=%d audit=%d, want no writes", active, audit)
	}
}

func TestRetentionApplyArchivesCandidatesPreservesAuditAndRecallExcludes(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	old := seedRetentionConclusion(t, ctx, svc, "peer-apply", "sess-apply", "Retention tombstone codename amber must disappear from recall.", time.Now().Add(-90*24*time.Hour))

	applied, err := svc.ApplyRetention(ctx, RetentionPolicy{MaxAge: 30 * 24 * time.Hour, ArchiveBeforeEvict: true}, "test:retention")
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if !applied.Mutates || len(applied.Applied) != 1 || applied.Applied[0].StableID != "conclusion:"+itoa64(old.ID) {
		t.Fatalf("applied = %+v, want archived old conclusion", applied)
	}
	status := retentionConclusionStatus(t, svc, old.ID)
	if status != "archived" {
		t.Fatalf("conclusion status = %q, want archived tombstone", status)
	}
	audit, err := svc.RetentionAudit(ctx, RetentionAuditQuery{StableID: "conclusion:" + itoa64(old.ID)})
	if err != nil {
		t.Fatalf("RetentionAudit: %v", err)
	}
	if len(audit.Events) != 1 || audit.Events[0].Action != RetentionActionArchive || audit.Events[0].StableID != "conclusion:"+itoa64(old.ID) {
		t.Fatalf("audit = %+v, want archive event preserving stable id", audit.Events)
	}
	trace, err := svc.Recall(ctx, RecallQuery{Peer: "peer-apply", SessionKey: "sess-apply", Query: "amber", Limit: 5})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	for _, selected := range trace.Selected {
		if strings.Contains(selected.Candidate.Content, "amber") {
			t.Fatalf("selected = %+v, want archived amber memory excluded", trace.Selected)
		}
	}
}

func TestRetentionApplyArchivesImageRefsWhenImageBudgetExceeded(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	dir := t.TempDir()
	imageDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "img.bin"), []byte("image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	image, err := svc.StoreImageMemory(ctx, ImageMemoryParams{Peer: "peer-image-retention", SessionKey: "sess-image", ImageRef: "file://img.bin", Checksum: "sha256:image-retention", AltText: "retention image"})
	if err != nil {
		t.Fatalf("StoreImageMemory: %v", err)
	}

	applied, err := svc.ApplyRetention(ctx, RetentionPolicy{ImageDir: imageDir, MaxImageBytes: 1, ArchiveBeforeEvict: true}, "test:image-retention")
	if err != nil {
		t.Fatalf("ApplyRetention image: %v", err)
	}
	if applied.ImagesArchived != 1 || len(applied.Applied) != 1 || applied.Applied[0].StableID != "image:"+itoa64(image.ID) {
		t.Fatalf("applied = %+v, want archived image ref", applied)
	}
	listed, err := svc.SearchImageMemories(ctx, ImageMemoryQuery{Peer: "peer-image-retention", Query: "retention", Limit: 5})
	if err != nil {
		t.Fatalf("SearchImageMemories: %v", err)
	}
	if len(listed.Images) != 0 {
		t.Fatalf("images = %+v, want archived image excluded", listed.Images)
	}
}

func TestDiskUsageDiagnosticsIncludeDBImageAndVectorDirs(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	dir := t.TempDir()
	imageDir := filepath.Join(dir, "images")
	vectorDir := filepath.Join(dir, "vectors")
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vectorDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "img.bin"), []byte("image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vectorDir, "vec.bin"), []byte("vector-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	usage, err := svc.DiskUsage(ctx, RetentionPolicy{ImageDir: imageDir, VectorDir: vectorDir, MaxDBBytes: 1, MaxImageBytes: 1, MaxVectorBytes: 1})
	if err != nil {
		t.Fatalf("DiskUsage: %v", err)
	}
	if usage.DB.Bytes == 0 || usage.Images.Bytes == 0 || usage.Vectors.Bytes == 0 {
		t.Fatalf("usage = %+v, want DB/image/vector byte counts", usage)
	}
	if !usage.DB.OverBudget || !usage.Images.OverBudget || !usage.Vectors.OverBudget {
		t.Fatalf("usage = %+v, want all over budget with 1-byte limits", usage)
	}
}

func seedRetentionConclusion(t *testing.T, ctx context.Context, svc *Service, peer, session, content string, createdAt time.Time) ConcludeResult {
	t.Helper()
	out, err := svc.Conclude(ctx, ConcludeParams{Peer: peer, SessionKey: session, Conclusion: content})
	if err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_conclusions SET created_at = ?, updated_at = ? WHERE id = ?`, createdAt.Unix(), createdAt.Unix(), out.ID); err != nil {
		t.Fatalf("age conclusion: %v", err)
	}
	return out
}

func countRetentionRows(t *testing.T, svc *Service, table, where string) int {
	t.Helper()
	var count int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE ` + where).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func retentionConclusionStatus(t *testing.T, svc *Service, id int64) string {
	t.Helper()
	var status string
	if err := svc.db.QueryRow(`SELECT status FROM goncho_conclusions WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("status: %v", err)
	}
	return status
}

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }
