package goncho

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrObservationConflict      = errors.New("goncho: observation conflict")
	ErrObservationNotFound      = errors.New("goncho: observation not found")
	ErrObservationSchemaMissing = errors.New("goncho: observation schema missing")
	ErrObservationInvalid       = errors.New("goncho: invalid observation")
)

type ObservationKind string

const (
	ObservationKindSessionStart      ObservationKind = "session_start"
	ObservationKindUserPrompt        ObservationKind = "user_prompt"
	ObservationKindToolCall          ObservationKind = "tool_call"
	ObservationKindToolResult        ObservationKind = "tool_result"
	ObservationKindToolError         ObservationKind = "tool_error"
	ObservationKindAssistantResponse ObservationKind = "assistant_response"
	ObservationKindCompact           ObservationKind = "compact"
	ObservationKindSessionEnd        ObservationKind = "session_end"
	ObservationKindCustom            ObservationKind = "custom"
)

type ObservationParams struct {
	ID          string            `json:"id,omitempty"`
	Kind        ObservationKind   `json:"kind"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	PeerID      string            `json:"peer_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	ContextID   string            `json:"context_id,omitempty"`
	Input       string            `json:"input"`
	Output      string            `json:"output"`
	Success     *bool             `json:"success,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ObservedAt  time.Time         `json:"observed_at,omitempty"`
	Reason      string            `json:"reason,omitempty"`
}

type Observation struct {
	ID                  string            `json:"id"`
	Kind                ObservationKind   `json:"kind"`
	WorkspaceID         string            `json:"workspace_id,omitempty"`
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

type ObservationResult struct {
	Observation Observation `json:"observation"`
	AuditID     string      `json:"audit_id"`
	Replayed    bool        `json:"replayed,omitempty"`
}

type ObservationQuery struct {
	WorkspaceID string            `json:"workspace_id,omitempty"`
	PeerID      string            `json:"peer_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	ContextID   string            `json:"context_id,omitempty"`
	Kinds       []ObservationKind `json:"kinds,omitempty"`
	Success     *bool             `json:"success,omitempty"`
	Since       time.Time         `json:"since,omitempty"`
	Until       time.Time         `json:"until,omitempty"`
	Limit       int               `json:"limit,omitempty"`
}

type ObservationList struct {
	Observations []Observation `json:"observations"`
	Count        int           `json:"count"`
}

const (
	observationIDMaxBytes       = 256
	observationScopeIDMaxBytes  = 512
	observationMetadataKeyMax   = 128
	observationMetadataValueMax = 4 * 1024
	observationMetadataJSONMax  = 16 * 1024
	observationInputMax         = 16 * 1024
	observationOutputMax        = 64 * 1024
	observationDefaultLimit     = 50
	observationMaxLimit         = 500
)

var validObservationKinds = map[ObservationKind]struct{}{
	ObservationKindSessionStart:      {},
	ObservationKindUserPrompt:        {},
	ObservationKindToolCall:          {},
	ObservationKindToolResult:        {},
	ObservationKindToolError:         {},
	ObservationKindAssistantResponse: {},
	ObservationKindCompact:           {},
	ObservationKindSessionEnd:        {},
	ObservationKindCustom:            {},
}

type normalizedObservation struct {
	obs               Observation
	metadataJSON      string
	auditMetadata     map[string]string
	auditMetadataJSON string
	reason            string
}

func Observe(ctx context.Context, db *sql.DB, p ObservationParams) (ObservationResult, error) {
	if err := ctx.Err(); err != nil {
		return ObservationResult{}, err
	}
	if db == nil {
		return ObservationResult{}, fmt.Errorf("%w: nil db", ErrObservationInvalid)
	}
	norm, err := normalizeObservationParams(p)
	if err != nil {
		return ObservationResult{}, err
	}
	if norm.obs.ID == "" {
		id, err := newObservationID("obs")
		if err != nil {
			return ObservationResult{}, err
		}
		norm.obs.ID = id
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ObservationResult{}, wrapObservationSQLError("begin observe", err)
	}
	defer tx.Rollback()

	existing, found, err := getObservationByID(ctx, tx, norm.obs.ID)
	if err != nil {
		return ObservationResult{}, err
	}
	if found {
		if existing.Checksum != norm.obs.Checksum {
			return ObservationResult{}, fmt.Errorf("%w: %s", ErrObservationConflict, norm.obs.ID)
		}
		auditID, err := firstObserveAuditID(ctx, tx, existing.ID)
		if err != nil {
			return ObservationResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ObservationResult{}, wrapObservationSQLError("commit observe replay", err)
		}
		return ObservationResult{Observation: existing, AuditID: auditID, Replayed: true}, nil
	}

	if err := insertObservation(ctx, tx, norm); err != nil {
		return ObservationResult{}, err
	}
	auditID, err := newObservationID("audit")
	if err != nil {
		return ObservationResult{}, err
	}
	createdAt := time.Now().UTC()
	event := AuditEvent{
		ID:          auditID,
		Action:      AuditActionObserve,
		TargetType:  AuditTargetObservation,
		TargetID:    norm.obs.ID,
		WorkspaceID: norm.obs.WorkspaceID,
		PeerID:      norm.obs.PeerID,
		SessionKey:  norm.obs.SessionKey,
		Reason:      norm.reason,
		Metadata:    norm.auditMetadata,
		CreatedAt:   createdAt,
	}
	if err := insertAuditEvent(ctx, tx, event, norm.auditMetadataJSON); err != nil {
		return ObservationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ObservationResult{}, wrapObservationSQLError("commit observe", err)
	}
	return ObservationResult{Observation: norm.obs, AuditID: auditID}, nil
}

func ListObservations(ctx context.Context, db *sql.DB, q ObservationQuery) (ObservationList, error) {
	if err := ctx.Err(); err != nil {
		return ObservationList{}, err
	}
	if db == nil {
		return ObservationList{}, fmt.Errorf("%w: nil db", ErrObservationInvalid)
	}
	for _, kind := range q.Kinds {
		if !isValidObservationKind(kind) {
			return ObservationList{}, fmt.Errorf("%w: unsupported observation kind %q", ErrObservationInvalid, kind)
		}
	}
	limit := normalizeObservationLimit(q.Limit)
	args := []any{}
	var where []string
	appendExactFilter := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		where = append(where, column+" = ?")
		args = append(args, value)
	}
	appendExactFilter("workspace_id", q.WorkspaceID)
	appendExactFilter("peer_id", q.PeerID)
	appendExactFilter("session_key", q.SessionKey)
	appendExactFilter("context_id", q.ContextID)
	if len(q.Kinds) > 0 {
		placeholders := make([]string, len(q.Kinds))
		for i, kind := range q.Kinds {
			placeholders[i] = "?"
			args = append(args, string(kind))
		}
		where = append(where, "kind IN ("+strings.Join(placeholders, ",")+")")
	}
	if q.Success != nil {
		where = append(where, "success = ?")
		args = append(args, boolInt(*q.Success))
	}
	if !q.Since.IsZero() {
		where = append(where, "observed_at >= ?")
		args = append(args, q.Since.UTC().UnixNano())
	}
	if !q.Until.IsZero() {
		where = append(where, "observed_at <= ?")
		args = append(args, q.Until.UTC().UnixNano())
	}

	query := `SELECT id, kind, workspace_id, peer_id, session_key, context_id, input, output, success, metadata_json, input_truncated, output_truncated, input_original_bytes, output_original_bytes, redacted, redaction_count, checksum, observed_at FROM goncho_observations`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY observed_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return ObservationList{}, wrapObservationSQLError("list observations", err)
	}
	defer rows.Close()

	out := ObservationList{Observations: []Observation{}}
	for rows.Next() {
		obs, err := scanObservation(rows)
		if err != nil {
			return ObservationList{}, err
		}
		out.Observations = append(out.Observations, obs)
	}
	if err := rows.Err(); err != nil {
		return ObservationList{}, wrapObservationSQLError("iterate observations", err)
	}
	out.Count = len(out.Observations)
	return out, nil
}

func (s *Service) Observe(ctx context.Context, p ObservationParams) (ObservationResult, error) {
	if s == nil {
		return ObservationResult{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	if strings.TrimSpace(p.WorkspaceID) == "*" {
		return ObservationResult{}, fmt.Errorf("%w: wildcard workspace is not valid for observe", ErrObservationInvalid)
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		p.WorkspaceID = s.workspaceID
	}
	return Observe(ctx, s.db, p)
}

func (s *Service) ListObservations(ctx context.Context, q ObservationQuery) (ObservationList, error) {
	if s == nil {
		return ObservationList{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return ListObservations(ctx, s.db, q)
}

func normalizeObservationParams(p ObservationParams) (normalizedObservation, error) {
	kind := ObservationKind(strings.TrimSpace(string(p.Kind)))
	if !isValidObservationKind(kind) {
		return normalizedObservation{}, fmt.Errorf("%w: unsupported observation kind %q", ErrObservationInvalid, p.Kind)
	}
	metadata, redactionCount, redactionKinds, err := normalizeObservationMetadata(kind, p.Metadata)
	if err != nil {
		return normalizedObservation{}, err
	}

	redactedInput, inputRedactions, inputKinds := redactObservationString(strings.ToValidUTF8(p.Input, "\uFFFD"))
	redactedOutput, outputRedactions, outputKinds := redactObservationString(strings.ToValidUTF8(p.Output, "\uFFFD"))
	redactionCount += inputRedactions + outputRedactions
	redactionKinds = append(redactionKinds, inputKinds...)
	redactionKinds = append(redactionKinds, outputKinds...)

	inputOriginalBytes := len([]byte(redactedInput))
	outputOriginalBytes := len([]byte(redactedOutput))
	input, inputTruncated := truncateUTF8Bytes(redactedInput, observationInputMax)
	output, outputTruncated := truncateUTF8Bytes(redactedOutput, observationOutputMax)

	workspaceID, err := normalizeObservationIDPart("workspace_id", p.WorkspaceID, observationScopeIDMaxBytes, false)
	if err != nil {
		return normalizedObservation{}, err
	}
	peerID, err := normalizeObservationIDPart("peer_id", p.PeerID, observationScopeIDMaxBytes, false)
	if err != nil {
		return normalizedObservation{}, err
	}
	sessionKey, err := normalizeObservationIDPart("session_key", p.SessionKey, observationScopeIDMaxBytes, false)
	if err != nil {
		return normalizedObservation{}, err
	}
	contextID, err := normalizeObservationIDPart("context_id", p.ContextID, observationScopeIDMaxBytes, false)
	if err != nil {
		return normalizedObservation{}, err
	}
	id, err := normalizeObservationIDPart("id", p.ID, observationIDMaxBytes, false)
	if err != nil {
		return normalizedObservation{}, err
	}

	metadataJSON, err := marshalObservationMetadata(metadata)
	if err != nil {
		return normalizedObservation{}, err
	}
	if len([]byte(metadataJSON)) > observationMetadataJSONMax {
		return normalizedObservation{}, fmt.Errorf("%w: metadata_json exceeds %d bytes", ErrObservationInvalid, observationMetadataJSONMax)
	}
	observedAt := p.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	obs := Observation{
		ID:                  id,
		Kind:                kind,
		WorkspaceID:         workspaceID,
		PeerID:              peerID,
		SessionKey:          sessionKey,
		ContextID:           contextID,
		Input:               input,
		Output:              output,
		Success:             copyBoolPtr(p.Success),
		Metadata:            metadata,
		InputTruncated:      inputTruncated,
		OutputTruncated:     outputTruncated,
		InputOriginalBytes:  inputOriginalBytes,
		OutputOriginalBytes: outputOriginalBytes,
		Redacted:            redactionCount > 0,
		RedactionCount:      redactionCount,
		ObservedAt:          observedAt,
	}
	obs.Checksum = observationChecksum(obs)

	reason := strings.TrimSpace(p.Reason)
	if reason == "" {
		reason = string(AuditActionObserve)
	}
	reason, _, _ = redactObservationString(strings.ToValidUTF8(reason, "\uFFFD"))
	reason, _ = truncateUTF8Bytes(reason, observationMetadataValueMax)
	auditMetadata := observationAuditMetadata(obs, uniqueSortedStrings(redactionKinds))
	auditMetadataJSON, err := marshalObservationMetadata(auditMetadata)
	if err != nil {
		return normalizedObservation{}, err
	}
	return normalizedObservation{
		obs:               obs,
		metadataJSON:      metadataJSON,
		auditMetadata:     auditMetadata,
		auditMetadataJSON: auditMetadataJSON,
		reason:            reason,
	}, nil
}

func normalizeObservationMetadata(kind ObservationKind, in map[string]string) (map[string]string, int, []string, error) {
	out := map[string]string{}
	redactions := 0
	var redactionKinds []string
	for rawKey, rawValue := range in {
		key := strings.TrimSpace(rawKey)
		if err := validateObservationTextID("metadata key", key, observationMetadataKeyMax); err != nil {
			return nil, 0, nil, err
		}
		if _, exists := out[key]; exists {
			return nil, 0, nil, fmt.Errorf("%w: duplicate metadata key %q after trim", ErrObservationInvalid, key)
		}
		value, count, kinds := redactObservationString(strings.ToValidUTF8(rawValue, "\uFFFD"))
		if len([]byte(value)) > observationMetadataValueMax {
			return nil, 0, nil, fmt.Errorf("%w: metadata value %q exceeds %d bytes", ErrObservationInvalid, key, observationMetadataValueMax)
		}
		out[key] = value
		redactions += count
		redactionKinds = append(redactionKinds, kinds...)
	}
	if kind == ObservationKindCustom && strings.TrimSpace(out["custom_kind"]) == "" {
		return nil, 0, nil, fmt.Errorf("%w: custom observation requires metadata custom_kind", ErrObservationInvalid)
	}
	return out, redactions, redactionKinds, nil
}

func normalizeObservationIDPart(name, value string, maxBytes int, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && !required {
		return "", nil
	}
	if err := validateObservationTextID(name, value, maxBytes); err != nil {
		return "", err
	}
	return value, nil
}

func validateObservationTextID(name, value string, maxBytes int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is required", ErrObservationInvalid, name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8", ErrObservationInvalid, name)
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: %s contains NUL", ErrObservationInvalid, name)
	}
	if len([]byte(value)) > maxBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrObservationInvalid, name, maxBytes)
	}
	return nil
}

func isValidObservationKind(kind ObservationKind) bool {
	_, ok := validObservationKinds[kind]
	return ok
}

func insertObservation(ctx context.Context, tx *sql.Tx, norm normalizedObservation) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO goncho_observations(
			id, kind, workspace_id, peer_id, session_key, context_id, input, output, success,
			metadata_json, input_truncated, output_truncated, input_original_bytes,
			output_original_bytes, redacted, redaction_count, checksum, observed_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		norm.obs.ID,
		string(norm.obs.Kind),
		norm.obs.WorkspaceID,
		norm.obs.PeerID,
		norm.obs.SessionKey,
		norm.obs.ContextID,
		norm.obs.Input,
		norm.obs.Output,
		sqlBoolPtr(norm.obs.Success),
		norm.metadataJSON,
		boolInt(norm.obs.InputTruncated),
		boolInt(norm.obs.OutputTruncated),
		norm.obs.InputOriginalBytes,
		norm.obs.OutputOriginalBytes,
		boolInt(norm.obs.Redacted),
		norm.obs.RedactionCount,
		norm.obs.Checksum,
		norm.obs.ObservedAt.UTC().UnixNano(),
	)
	if err != nil {
		return wrapObservationSQLError("insert observation", err)
	}
	return nil
}

func getObservationByID(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (Observation, bool, error) {
	row := q.QueryRowContext(ctx, `SELECT id, kind, workspace_id, peer_id, session_key, context_id, input, output, success, metadata_json, input_truncated, output_truncated, input_original_bytes, output_original_bytes, redacted, redaction_count, checksum, observed_at FROM goncho_observations WHERE id = ?`, id)
	obs, err := scanObservation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Observation{}, false, nil
	}
	if err != nil {
		return Observation{}, false, err
	}
	return obs, true, nil
}

type observationScanner interface {
	Scan(...any) error
}

func scanObservation(scanner observationScanner) (Observation, error) {
	var obs Observation
	var kind string
	var success sql.NullInt64
	var metadataJSON string
	var inputTruncated, outputTruncated, redacted int
	var observedAt int64
	err := scanner.Scan(
		&obs.ID,
		&kind,
		&obs.WorkspaceID,
		&obs.PeerID,
		&obs.SessionKey,
		&obs.ContextID,
		&obs.Input,
		&obs.Output,
		&success,
		&metadataJSON,
		&inputTruncated,
		&outputTruncated,
		&obs.InputOriginalBytes,
		&obs.OutputOriginalBytes,
		&redacted,
		&obs.RedactionCount,
		&obs.Checksum,
		&observedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Observation{}, err
		}
		return Observation{}, wrapObservationSQLError("scan observation", err)
	}
	obs.Kind = ObservationKind(kind)
	if success.Valid {
		value := success.Int64 != 0
		obs.Success = &value
	}
	metadata, err := decodeObservationMetadata(metadataJSON)
	if err != nil {
		return Observation{}, fmt.Errorf("goncho: decode observation metadata for %s: %w", obs.ID, err)
	}
	obs.Metadata = metadata
	obs.InputTruncated = inputTruncated != 0
	obs.OutputTruncated = outputTruncated != 0
	obs.Redacted = redacted != 0
	obs.ObservedAt = time.Unix(0, observedAt).UTC()
	return obs, nil
}

func observationChecksum(obs Observation) string {
	payload := struct {
		Kind                ObservationKind   `json:"kind"`
		WorkspaceID         string            `json:"workspace_id"`
		PeerID              string            `json:"peer_id"`
		SessionKey          string            `json:"session_key"`
		ContextID           string            `json:"context_id"`
		Input               string            `json:"input"`
		Output              string            `json:"output"`
		Success             *bool             `json:"success,omitempty"`
		Metadata            map[string]string `json:"metadata"`
		InputTruncated      bool              `json:"input_truncated"`
		OutputTruncated     bool              `json:"output_truncated"`
		InputOriginalBytes  int               `json:"input_original_bytes"`
		OutputOriginalBytes int               `json:"output_original_bytes"`
		Redacted            bool              `json:"redacted"`
		RedactionCount      int               `json:"redaction_count"`
	}{
		Kind:                obs.Kind,
		WorkspaceID:         obs.WorkspaceID,
		PeerID:              obs.PeerID,
		SessionKey:          obs.SessionKey,
		ContextID:           obs.ContextID,
		Input:               obs.Input,
		Output:              obs.Output,
		Success:             copyBoolPtr(obs.Success),
		Metadata:            obs.Metadata,
		InputTruncated:      obs.InputTruncated,
		OutputTruncated:     obs.OutputTruncated,
		InputOriginalBytes:  obs.InputOriginalBytes,
		OutputOriginalBytes: obs.OutputOriginalBytes,
		Redacted:            obs.Redacted,
		RedactionCount:      obs.RedactionCount,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func newObservationID(prefix string) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("goncho: generate %s id: %w", prefix, err)
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(b[:])), nil
}

func marshalObservationMetadata(metadata map[string]string) (string, error) {
	if metadata == nil {
		metadata = map[string]string{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("%w: marshal metadata: %v", ErrObservationInvalid, err)
	}
	return string(raw), nil
}

func decodeObservationMetadata(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]string{}
	}
	return out, nil
}

func observationAuditMetadata(obs Observation, redactionKinds []string) map[string]string {
	return map[string]string{
		"redacted":              strconv.FormatBool(obs.Redacted),
		"redaction_count":       strconv.Itoa(obs.RedactionCount),
		"redaction_kinds":       strings.Join(redactionKinds, ","),
		"input_truncated":       strconv.FormatBool(obs.InputTruncated),
		"output_truncated":      strconv.FormatBool(obs.OutputTruncated),
		"input_original_bytes":  strconv.Itoa(obs.InputOriginalBytes),
		"output_original_bytes": strconv.Itoa(obs.OutputOriginalBytes),
	}
}

func serviceObservationWorkspace(defaultWorkspace, requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "*" {
		return ""
	}
	if requested == "" {
		return defaultWorkspace
	}
	return requested
}

func truncateUTF8Bytes(value string, limit int) (string, bool) {
	if len([]byte(value)) <= limit {
		return value, false
	}
	raw := []byte(value)
	raw = raw[:limit]
	for len(raw) > 0 && !utf8.Valid(raw) {
		raw = raw[:len(raw)-1]
	}
	return string(raw), true
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func sqlBoolPtr(value *bool) any {
	if value == nil {
		return nil
	}
	return boolInt(*value)
}

func copyBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeObservationLimit(limit int) int {
	if limit <= 0 {
		return observationDefaultLimit
	}
	if limit > observationMaxLimit {
		return observationMaxLimit
	}
	return limit
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func wrapObservationSQLError(op string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return fmt.Errorf("%w: %s: %v", ErrObservationSchemaMissing, op, err)
	}
	return fmt.Errorf("goncho: %s: %w", op, err)
}

type redactionRule struct {
	kind string
	re   *regexp.Regexp
}

var observationRedactionRules = []redactionRule{
	{kind: "private", re: regexp.MustCompile(`(?is)<private>.*?</private>`)},
	{kind: "pem_private_key", re: regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{kind: "authorization", re: regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+[^\s\r\n]+`)},
	{kind: "json_secret", re: regexp.MustCompile(`(?i)"([^"]*(?:secret|token|password|api_key|private_key|authorization)[^"]*)"\s*:\s*"[^"]*"`)},
	{kind: "env_secret", re: regexp.MustCompile(`(?im)\b[A-Z0-9_]*(?:SECRET|TOKEN|PASSWORD|API_KEY|PRIVATE_KEY)[A-Z0-9_]*\s*=\s*[^\s\r\n]+`)},
	{kind: "api_key", re: regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]+|ghp_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+)\b`)},
}

func redactObservationString(value string) (string, int, []string) {
	total := 0
	var kinds []string
	for _, rule := range observationRedactionRules {
		count := 0
		value = rule.re.ReplaceAllStringFunc(value, func(match string) string {
			count++
			if rule.kind == "json_secret" {
				parts := strings.SplitN(match, ":", 2)
				if len(parts) == 2 {
					return parts[0] + `:"[REDACTED:json_secret]"`
				}
			}
			return "[REDACTED:" + rule.kind + "]"
		})
		if count > 0 {
			total += count
			for i := 0; i < count; i++ {
				kinds = append(kinds, rule.kind)
			}
		}
	}
	return value, total, kinds
}
