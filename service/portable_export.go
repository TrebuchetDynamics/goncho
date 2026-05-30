package goncho

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/dbscan"
	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
)

const PortableExportSchemaVersion = "goncho-portable-v1"

type PortableExportParams struct {
	WorkspaceID      string    `json:"workspace_id,omitempty"`
	ProfileID        string    `json:"profile_id,omitempty"`
	Peer             string    `json:"peer_id,omitempty"`
	SessionKey       string    `json:"session_key,omitempty"`
	Since            time.Time `json:"since,omitempty"`
	Until            time.Time `json:"until,omitempty"`
	RedactionPolicy  string    `json:"redaction_policy,omitempty"`
	IncludeSnapshots bool      `json:"include_snapshots,omitempty"`
}

type PortableExportResult struct {
	Manifest PortableExportManifest `json:"manifest"`
	JSONL    []byte                 `json:"-"`
}

type PortableExportManifest struct {
	SchemaVersion   string         `json:"schema_version"`
	WorkspaceID     string         `json:"workspace_id"`
	ProfileID       string         `json:"profile_id,omitempty"`
	Peer            string         `json:"peer_id,omitempty"`
	SessionKey      string         `json:"session_key,omitempty"`
	RedactionPolicy string         `json:"redaction_policy,omitempty"`
	Counts          map[string]int `json:"counts"`
	Checksum        string         `json:"checksum"`
}

type PortableExportRecord struct {
	Type     string          `json:"type"`
	StableID string          `json:"stable_id"`
	Data     json.RawMessage `json:"data"`
}

type PortableImportParams struct {
	JSONL []byte `json:"-"`
	Apply bool   `json:"apply"`
}

type PortableImportPreview struct {
	SchemaVersion    string                   `json:"schema_version"`
	ManifestChecksum string                   `json:"manifest_checksum"`
	Mutates          bool                     `json:"mutates"`
	SafeToApply      bool                     `json:"safe_to_apply"`
	Counts           map[string]int           `json:"counts"`
	Conflicts        []PortableImportConflict `json:"conflicts,omitempty"`
	Redaction        PortableRedactionSummary `json:"redaction"`
}

type PortableImportResult struct {
	ManifestChecksum string         `json:"manifest_checksum"`
	Mutates          bool           `json:"mutates"`
	Applied          map[string]int `json:"applied"`
}

type PortableImportConflict struct {
	Type     string `json:"type"`
	StableID string `json:"stable_id"`
	Reason   string `json:"reason"`
}

type PortableRedactionSummary struct {
	Policy               string `json:"policy,omitempty"`
	RedactedObservations int    `json:"redacted_observations,omitempty"`
	RedactionCount       int    `json:"redaction_count,omitempty"`
}

type portableObservation struct {
	ID                  string            `json:"id"`
	Kind                ObservationKind   `json:"kind"`
	WorkspaceID         string            `json:"workspace_id"`
	ProfileID           string            `json:"profile_id,omitempty"`
	PeerID              string            `json:"peer_id,omitempty"`
	SessionKey          string            `json:"session_key,omitempty"`
	ContextID           string            `json:"context_id,omitempty"`
	Input               string            `json:"input"`
	Output              string            `json:"output"`
	Success             *bool             `json:"success,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	InputTruncated      bool              `json:"input_truncated"`
	OutputTruncated     bool              `json:"output_truncated"`
	InputOriginalBytes  int               `json:"input_original_bytes"`
	OutputOriginalBytes int               `json:"output_original_bytes"`
	Redacted            bool              `json:"redacted"`
	RedactionCount      int               `json:"redaction_count"`
	Checksum            string            `json:"checksum"`
	ObservedAt          time.Time         `json:"observed_at"`
}

type portableMessage struct {
	ID          int64          `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	SessionKey  string         `json:"session_key"`
	Peer        string         `json:"peer_id"`
	Role        string         `json:"role"`
	Content     string         `json:"content"`
	Sequence    int            `json:"seq_in_session"`
	CreatedAt   int64          `json:"created_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type portableConclusion struct {
	ID           int64  `json:"id"`
	WorkspaceID  string `json:"workspace_id"`
	ProfileID    string `json:"profile_id,omitempty"`
	Observer     string `json:"observer_peer_id"`
	Peer         string `json:"peer_id"`
	SessionKey   string `json:"session_key,omitempty"`
	Content      string `json:"content"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	Idempotency  string `json:"idempotency_key"`
	EvidenceJSON string `json:"evidence_json"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

func (s *Service) ExportPortableJSONL(ctx context.Context, params PortableExportParams) (PortableExportResult, error) {
	records, manifest, err := s.portableExportRecords(ctx, params)
	if err != nil {
		return PortableExportResult{}, err
	}
	data, checksum, err := marshalPortableRecords(records)
	if err != nil {
		return PortableExportResult{}, err
	}
	manifest.Checksum = checksum
	manifestRecord, err := portableRecord("manifest", "manifest:"+manifest.SchemaVersion, manifest)
	if err != nil {
		return PortableExportResult{}, err
	}
	manifestLine, err := json.Marshal(manifestRecord)
	if err != nil {
		return PortableExportResult{}, err
	}
	jsonl := append(append([]byte{}, manifestLine...), '\n')
	jsonl = append(jsonl, data...)
	return PortableExportResult{Manifest: manifest, JSONL: jsonl}, nil
}

func (s *Service) portableExportRecords(ctx context.Context, params PortableExportParams) ([]PortableExportRecord, PortableExportManifest, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(params.ProfileID)
	peer := strings.TrimSpace(params.Peer)
	sessionKey := strings.TrimSpace(params.SessionKey)
	counts := map[string]int{}
	records := []PortableExportRecord{}
	add := func(kind, stableID string, data any) error {
		record, err := portableRecord(kind, stableID, data)
		if err != nil {
			return err
		}
		records = append(records, record)
		counts[kind+"s"]++
		return nil
	}
	observations, err := s.exportPortableObservations(ctx, workspaceID, profileID, peer, sessionKey, params.Since, params.Until)
	if err != nil {
		return nil, PortableExportManifest{}, err
	}
	for _, obs := range observations {
		if err := add("observation", "observation:"+obs.ID, obs); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	messages, err := exportPortableMessages(ctx, s.db, workspaceID, peer, sessionKey, params.Since, params.Until)
	if err != nil {
		return nil, PortableExportManifest{}, err
	}
	for _, msg := range messages {
		if err := add("message", fmt.Sprintf("message:%d", msg.ID), msg); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	conclusions, err := exportPortableConclusions(ctx, s.db, workspaceID, profileID, s.observer, peer, sessionKey, params.Since, params.Until)
	if err != nil {
		return nil, PortableExportManifest{}, err
	}
	for _, c := range conclusions {
		if err := add("conclusion", fmt.Sprintf("conclusion:%d", c.ID), c); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	reviews, err := s.ListReviewItems(ctx, ReviewQuery{WorkspaceID: workspaceID, PeerID: peer, SessionKey: sessionKey, Limit: 500})
	if err != nil {
		return nil, PortableExportManifest{}, err
	}
	for _, item := range reviews.Items {
		if err := add("review_item", "review_item:"+item.ID, item); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	slots, err := exportPortableMemorySlots(ctx, s.db, workspaceID, profileID, peer)
	if err != nil {
		return nil, PortableExportManifest{}, err
	}
	for _, slot := range slots {
		if err := add("memory_slot", "memory_slot:"+slot.WorkspaceID+":"+slot.ProfileID+":"+slot.Peer+":"+slot.Scope+":"+slot.Name, slot); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	if params.IncludeSnapshots && peer != "" {
		snapshot, err := s.ExportSnapshotManifest(ctx, SnapshotParams{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer})
		if err != nil {
			return nil, PortableExportManifest{}, err
		}
		if err := add("snapshot", "snapshot:"+snapshot.SnapshotID, snapshot); err != nil {
			return nil, PortableExportManifest{}, err
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		left, right := portableTypeOrder(records[i].Type), portableTypeOrder(records[j].Type)
		if left == right {
			return records[i].StableID < records[j].StableID
		}
		return left < right
	})
	return records, PortableExportManifest{SchemaVersion: PortableExportSchemaVersion, WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, SessionKey: sessionKey, RedactionPolicy: strings.TrimSpace(params.RedactionPolicy), Counts: counts}, nil
}

func (s *Service) PreviewPortableImport(ctx context.Context, jsonl []byte) (PortableImportPreview, error) {
	manifest, records, err := parsePortableJSONL(jsonl)
	if err != nil {
		return PortableImportPreview{}, err
	}
	preview := PortableImportPreview{SchemaVersion: manifest.SchemaVersion, ManifestChecksum: manifest.Checksum, Mutates: false, SafeToApply: true, Counts: map[string]int{}, Redaction: PortableRedactionSummary{Policy: manifest.RedactionPolicy}}
	for _, record := range records {
		preview.Counts[record.Type+"s"]++
		if record.Type == "observation" {
			var obs portableObservation
			_ = json.Unmarshal(record.Data, &obs)
			if obs.Redacted {
				preview.Redaction.RedactedObservations++
			}
			preview.Redaction.RedactionCount += obs.RedactionCount
		}
		conflict, err := s.portableRecordExists(ctx, record)
		if err != nil {
			return PortableImportPreview{}, err
		}
		if conflict {
			preview.Conflicts = append(preview.Conflicts, PortableImportConflict{Type: record.Type, StableID: record.StableID, Reason: "stable_id already exists in target workspace"})
			preview.SafeToApply = false
		}
	}
	return preview, nil
}

func (s *Service) ImportPortableJSONL(ctx context.Context, params PortableImportParams) (PortableImportResult, error) {
	if !params.Apply {
		return PortableImportResult{}, fmt.Errorf("goncho: portable import requires apply=true after preview")
	}
	preview, err := s.PreviewPortableImport(ctx, params.JSONL)
	if err != nil {
		return PortableImportResult{}, err
	}
	if !preview.SafeToApply {
		return PortableImportResult{}, fmt.Errorf("goncho: portable import preview is not safe to apply: %d conflicts", len(preview.Conflicts))
	}
	_, records, err := parsePortableJSONL(params.JSONL)
	if err != nil {
		return PortableImportResult{}, err
	}
	out := PortableImportResult{ManifestChecksum: preview.ManifestChecksum, Mutates: true, Applied: map[string]int{}}
	for _, record := range records {
		if err := s.importPortableRecord(ctx, record); err != nil {
			return PortableImportResult{}, err
		}
		out.Applied[record.Type+"s"]++
	}
	return out, nil
}

func (s *Service) ExportPortableMarkdown(ctx context.Context, params PortableExportParams) (string, error) {
	exported, err := s.ExportPortableJSONL(ctx, params)
	if err != nil {
		return "", err
	}
	_, records, err := parsePortableJSONL(exported.JSONL)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Goncho Portable Memory Export\n\nschema: %s\nchecksum: %s\nworkspace: %s\npeer: %s\nsession: [[session:%s]]\n\n", exported.Manifest.SchemaVersion, exported.Manifest.Checksum, exported.Manifest.WorkspaceID, exported.Manifest.Peer, exported.Manifest.SessionKey)
	for _, record := range records {
		fmt.Fprintf(&b, "## %s `%s`\n\n", record.Type, record.StableID)
		fmt.Fprintf(&b, "provenance:\n- stable_id: %s\n- export_checksum: %s\n", record.StableID, exported.Manifest.Checksum)
		switch record.Type {
		case "conclusion":
			var c portableConclusion
			_ = json.Unmarshal(record.Data, &c)
			fmt.Fprintf(&b, "- backlink: [[session:%s]]\n- review_status: %s\n", c.SessionKey, portableReviewStatus(records, "conclusion:"+strconv.FormatInt(c.ID, 10)))
			if c.Status != "processed" && c.Status != "active" {
				fmt.Fprintf(&b, "- stale_warning: status=%s\n", c.Status)
			}
			fmt.Fprintf(&b, "\n%s\n\n", c.Content)
		case "observation":
			var obs portableObservation
			_ = json.Unmarshal(record.Data, &obs)
			fmt.Fprintf(&b, "- backlink: [[session:%s]]\n- observed_at: %s\n\ninput: %s\noutput: %s\n\n", obs.SessionKey, obs.ObservedAt.UTC().Format(time.RFC3339), obs.Input, obs.Output)
		case "review_item":
			var item ReviewItem
			_ = json.Unmarshal(record.Data, &item)
			fmt.Fprintf(&b, "- review_status: %s\n- subject: %s\n\n%s\n\n", item.Status, item.SubjectID, item.Reason)
		}
	}
	return b.String(), nil
}

func portableTypeOrder(kind string) int {
	switch kind {
	case "observation":
		return 1
	case "message":
		return 2
	case "conclusion":
		return 3
	case "review_item":
		return 4
	case "memory_slot":
		return 5
	case "snapshot":
		return 6
	default:
		return 100
	}
}

func portableRecord(kind, stableID string, data any) (PortableExportRecord, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return PortableExportRecord{}, err
	}
	return PortableExportRecord{Type: kind, StableID: stableID, Data: raw}, nil
}

func marshalPortableRecords(records []PortableExportRecord) ([]byte, string, error) {
	var b bytes.Buffer
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			return nil, "", err
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return b.Bytes(), hashutil.SHA256Hex(b.Bytes()), nil
}

func parsePortableJSONL(data []byte) (PortableExportManifest, []PortableExportRecord, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1024*1024*16)
	manifest := PortableExportManifest{}
	records := []PortableExportRecord{}
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec PortableExportRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return manifest, nil, err
		}
		if rec.Type == "manifest" {
			if err := json.Unmarshal(rec.Data, &manifest); err != nil {
				return manifest, nil, err
			}
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return manifest, nil, err
	}
	if manifest.SchemaVersion != PortableExportSchemaVersion {
		return manifest, nil, fmt.Errorf("goncho: unsupported portable schema %q", manifest.SchemaVersion)
	}
	return manifest, records, nil
}

func (s *Service) exportPortableObservations(ctx context.Context, workspaceID, profileID, peer, sessionKey string, since, until time.Time) ([]portableObservation, error) {
	query := `SELECT id, kind, workspace_id, profile_id, peer_id, session_key, context_id, input, output, success, metadata_json, input_truncated, output_truncated, input_original_bytes, output_original_bytes, redacted, redaction_count, checksum, observed_at FROM goncho_observations WHERE workspace_id = ?`
	args := []any{workspaceID}
	query = appendPortableFilters(query, &args, profileID, peer, sessionKey, since, until, "observed_at") + ` ORDER BY observed_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []portableObservation{}
	for rows.Next() {
		var o portableObservation
		var kind string
		var success sql.NullInt64
		var meta string
		var inTr, outTr, red int
		var observed int64
		if err := rows.Scan(&o.ID, &kind, &o.WorkspaceID, &o.ProfileID, &o.PeerID, &o.SessionKey, &o.ContextID, &o.Input, &o.Output, &success, &meta, &inTr, &outTr, &o.InputOriginalBytes, &o.OutputOriginalBytes, &red, &o.RedactionCount, &o.Checksum, &observed); err != nil {
			return nil, err
		}
		o.Kind = ObservationKind(kind)
		if success.Valid {
			v := success.Int64 == 1
			o.Success = &v
		}
		_ = json.Unmarshal([]byte(meta), &o.Metadata)
		o.InputTruncated = inTr == 1
		o.OutputTruncated = outTr == 1
		o.Redacted = red == 1
		o.ObservedAt = time.Unix(0, observed).UTC()
		out = append(out, o)
	}
	return out, rows.Err()
}

func exportPortableMessages(ctx context.Context, db *sql.DB, workspaceID, peer, sessionKey string, since, until time.Time) ([]portableMessage, error) {
	query := `SELECT id, session_id, role, content, ts_unix, COALESCE(chat_id,''), COALESCE(meta_json,'') FROM turns WHERE 1=1`
	args := []any{}
	if sessionKey != "" {
		query += ` AND session_id = ?`
		args = append(args, sessionKey)
	}
	if peer != "" {
		query += ` AND chat_id = ?`
		args = append(args, peer)
	}
	if !since.IsZero() {
		query += ` AND ts_unix >= ?`
		args = append(args, since.Unix())
	}
	if !until.IsZero() {
		query += ` AND ts_unix <= ?`
		args = append(args, until.Unix())
	}
	query += ` ORDER BY id ASC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []portableMessage{}
	for rows.Next() {
		var m portableMessage
		var metaRaw string
		if err := rows.Scan(&m.ID, &m.SessionKey, &m.Role, &m.Content, &m.CreatedAt, &m.Peer, &metaRaw); err != nil {
			return nil, err
		}
		meta, _ := decodeLifecycleMeta(metaRaw)
		m.WorkspaceID = workspaceID
		m.Sequence = meta.Goncho.Sequence
		m.Metadata = meta.Goncho.Metadata
		out = append(out, m)
	}
	return out, rows.Err()
}

func exportPortableConclusions(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer, sessionKey string, since, until time.Time) ([]portableConclusion, error) {
	query := `SELECT id, workspace_id, profile_id, observer_peer_id, peer_id, COALESCE(session_key,''), content, kind, status, source, idempotency_key, evidence_json, scope, created_at, updated_at FROM goncho_conclusions WHERE workspace_id = ?`
	args := []any{workspaceID}
	query = appendPortableFilters(query, &args, profileID, peer, sessionKey, since, until, "created_at")
	query += ` ORDER BY id ASC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []portableConclusion{}
	for rows.Next() {
		var c portableConclusion
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.ProfileID, &c.Observer, &c.Peer, &c.SessionKey, &c.Content, &c.Kind, &c.Status, &c.Source, &c.Idempotency, &c.EvidenceJSON, &c.Scope, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if observer == "" || c.Observer == observer {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

func exportPortableMemorySlots(ctx context.Context, db *sql.DB, workspaceID, profileID, peer string) ([]MemorySlot, error) {
	query := `SELECT workspace_id, profile_id, peer_id, scope, name, kind, value, revision, deleted, created_at, updated_at FROM goncho_memory_slots WHERE workspace_id = ?`
	args := []any{workspaceID}
	if profileID != "" {
		query += ` AND profile_id = ?`
		args = append(args, profileID)
	}
	if peer != "" {
		query += ` AND peer_id = ?`
		args = append(args, peer)
	}
	query += ` ORDER BY peer_id, scope, name`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemorySlot{}
	for rows.Next() {
		slot, err := scanMemorySlot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	return out, rows.Err()
}

func appendPortableFilters(query string, args *[]any, profileID, peer, sessionKey string, since, until time.Time, timeColumn string) string {
	if profileID != "" {
		query += ` AND profile_id = ?`
		*args = append(*args, profileID)
	}
	if peer != "" {
		query += ` AND peer_id = ?`
		*args = append(*args, peer)
	}
	if sessionKey != "" {
		query += ` AND session_key = ?`
		*args = append(*args, sessionKey)
	}
	if !since.IsZero() {
		query += ` AND ` + timeColumn + ` >= ?`
		if timeColumn == "observed_at" {
			*args = append(*args, since.UnixNano())
		} else {
			*args = append(*args, since.Unix())
		}
	}
	if !until.IsZero() {
		query += ` AND ` + timeColumn + ` <= ?`
		if timeColumn == "observed_at" {
			*args = append(*args, until.UnixNano())
		} else {
			*args = append(*args, until.Unix())
		}
	}
	return query
}

func (s *Service) portableRecordExists(ctx context.Context, record PortableExportRecord) (bool, error) {
	var query string
	var arg any
	switch record.Type {
	case "observation":
		query = `SELECT COUNT(*) FROM goncho_observations WHERE id = ?`
		arg = strings.TrimPrefix(record.StableID, "observation:")
	case "message":
		query = `SELECT COUNT(*) FROM turns WHERE id = ?`
		arg = strings.TrimPrefix(record.StableID, "message:")
	case "conclusion":
		query = `SELECT COUNT(*) FROM goncho_conclusions WHERE id = ?`
		arg = strings.TrimPrefix(record.StableID, "conclusion:")
	case "review_item":
		query = `SELECT COUNT(*) FROM goncho_review_items WHERE id = ?`
		arg = strings.TrimPrefix(record.StableID, "review_item:")
	case "memory_slot":
		var slot MemorySlot
		_ = json.Unmarshal(record.Data, &slot)
		query = `SELECT COUNT(*) FROM goncho_memory_slots WHERE workspace_id=? AND profile_id=? AND peer_id=? AND scope=? AND name=?`
		var count int
		err := s.db.QueryRowContext(ctx, query, slot.WorkspaceID, slot.ProfileID, slot.Peer, slot.Scope, slot.Name).Scan(&count)
		return count > 0, err
	default:
		return false, nil
	}
	var count int
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&count)
	return count > 0, err
}

func (s *Service) importPortableRecord(ctx context.Context, record PortableExportRecord) error {
	switch record.Type {
	case "observation":
		var o portableObservation
		if err := json.Unmarshal(record.Data, &o); err != nil {
			return err
		}
		meta, _ := json.Marshal(o.Metadata)
		var success any = nil
		if o.Success != nil {
			if *o.Success {
				success = 1
			} else {
				success = 0
			}
		}
		_, err := s.db.ExecContext(ctx, `INSERT INTO goncho_observations(id, kind, workspace_id, profile_id, peer_id, session_key, context_id, input, output, success, metadata_json, input_truncated, output_truncated, input_original_bytes, output_original_bytes, redacted, redaction_count, checksum, observed_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, o.ID, string(o.Kind), o.WorkspaceID, o.ProfileID, o.PeerID, o.SessionKey, o.ContextID, o.Input, o.Output, success, string(meta), dbscan.BoolInt(o.InputTruncated), dbscan.BoolInt(o.OutputTruncated), o.InputOriginalBytes, o.OutputOriginalBytes, dbscan.BoolInt(o.Redacted), o.RedactionCount, o.Checksum, o.ObservedAt.UnixNano())
		return err
	case "message":
		var m portableMessage
		if err := json.Unmarshal(record.Data, &m); err != nil {
			return err
		}
		meta := lifecycleTurnMeta{Goncho: lifecycleMessageMeta{WorkspaceID: m.WorkspaceID, PeerID: m.Peer, Sequence: m.Sequence, Metadata: m.Metadata}}
		raw, _ := json.Marshal(meta)
		_, err := s.db.ExecContext(ctx, `INSERT INTO turns(id, session_id, role, content, ts_unix, chat_id, meta_json, memory_sync_status) VALUES(?,?,?,?,?,?,?,'ready')`, m.ID, m.SessionKey, m.Role, m.Content, m.CreatedAt, m.Peer, string(raw))
		return err
	case "conclusion":
		var c portableConclusion
		if err := json.Unmarshal(record.Data, &c); err != nil {
			return err
		}
		_, err := s.db.ExecContext(ctx, `INSERT INTO goncho_conclusions(id, workspace_id, profile_id, observer_peer_id, peer_id, session_key, content, kind, status, source, idempotency_key, evidence_json, created_at, updated_at, scope) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.WorkspaceID, c.ProfileID, c.Observer, c.Peer, sqlutil.NullIfBlank(c.SessionKey), c.Content, c.Kind, c.Status, c.Source, c.Idempotency, c.EvidenceJSON, c.CreatedAt, c.UpdatedAt, c.Scope)
		return err
	case "review_item":
		var item ReviewItem
		if err := json.Unmarshal(record.Data, &item); err != nil {
			return err
		}
		evidence, _ := json.Marshal(item.EvidenceIDs)
		var resolved any = nil
		if item.ResolvedAt != nil {
			resolved = item.ResolvedAt.UnixNano()
		}
		_, err := s.db.ExecContext(ctx, `INSERT INTO goncho_review_items(id, kind, status, workspace_id, peer_id, session_key, subject_id, related_id, reason, evidence_ids_json, created_at, resolution, resolved_by, resolution_reason, resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, string(item.Kind), string(item.Status), item.WorkspaceID, item.PeerID, item.SessionKey, item.SubjectID, item.RelatedID, item.Reason, string(evidence), item.CreatedAt.UnixNano(), string(item.Resolution), item.ResolvedBy, item.ResolutionReason, resolved)
		return err
	case "memory_slot":
		var slot MemorySlot
		if err := json.Unmarshal(record.Data, &slot); err != nil {
			return err
		}
		return upsertMemorySlotRow(ctx, s.db, slot)
	default:
		return nil
	}
}

func portableReviewStatus(records []PortableExportRecord, subject string) string {
	for _, r := range records {
		if r.Type != "review_item" {
			continue
		}
		var item ReviewItem
		_ = json.Unmarshal(r.Data, &item)
		if item.SubjectID == subject {
			return string(item.Status)
		}
	}
	return "none"
}
