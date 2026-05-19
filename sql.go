package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type lifecycleSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type conclusionRow struct {
	WorkspaceID    string
	ObserverPeerID string
	PeerID         string
	SessionKey     string
	Content        string
	Kind           string
	Status         string
	Source         string
	IdempotencyKey string
	EvidenceJSON   string
	Scope          string
}

type sessionSummaryRow struct {
	WorkspaceID string
	SessionKey  string
	SummaryType string
	Content     string
	MessageID   int64
	CreatedAt   int64
	TokenCount  int
}

type lifecycleMessageMeta struct {
	WorkspaceID string         `json:"workspace_id"`
	PeerID      string         `json:"peer_id"`
	Sequence    int            `json:"seq_in_session"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type lifecycleTurnMeta struct {
	Goncho lifecycleMessageMeta `json:"goncho"`
}

func createLifecycleMessages(ctx context.Context, db lifecycleSQL, workspaceID, sessionKey string, maxMessageSize int, inputs []CreateMessage) ([]MessageRecord, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	sessionKey = strings.TrimSpace(sessionKey)
	if workspaceID == "" {
		return nil, fmt.Errorf("goncho: workspace_id is required")
	}
	if sessionKey == "" {
		return nil, fmt.Errorf("goncho: session_key is required")
	}
	if len(inputs) == 0 {
		return []MessageRecord{}, nil
	}

	lastSeq, err := lastLifecycleMessageSequence(ctx, db, workspaceID, sessionKey)
	if err != nil {
		return nil, err
	}
	out := make([]MessageRecord, 0, len(inputs))
	for i, input := range inputs {
		peer := strings.TrimSpace(input.Peer)
		if peer == "" {
			return nil, fmt.Errorf("goncho: peer_id is required")
		}
		content := strings.TrimSpace(input.Content)
		if content == "" {
			return nil, fmt.Errorf("goncho: message content is required")
		}
		if maxMessageSize > 0 && len(content) > maxMessageSize {
			return nil, fmt.Errorf("goncho: message content exceeds max size %d", maxMessageSize)
		}
		role := strings.TrimSpace(input.Role)
		if role == "" {
			role = "user"
		}
		createdAt := input.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		sequence := lastSeq + i + 1
		meta := lifecycleTurnMeta{
			Goncho: lifecycleMessageMeta{
				WorkspaceID: workspaceID,
				PeerID:      peer,
				Sequence:    sequence,
				Metadata:    copyMetadata(input.Metadata),
			},
		}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return nil, fmt.Errorf("goncho: marshal message metadata: %w", err)
		}
		res, err := db.ExecContext(ctx, `
			INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, memory_sync_status)
			VALUES(?, ?, ?, ?, ?, ?, 'ready')
		`, sessionKey, role, content, createdAt.Unix(), peer, string(metaJSON))
		if err != nil {
			return nil, fmt.Errorf("goncho: insert lifecycle message: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("goncho: lifecycle message id: %w", err)
		}
		out = append(out, MessageRecord{
			ID:          id,
			WorkspaceID: workspaceID,
			SessionKey:  sessionKey,
			Peer:        peer,
			Role:        role,
			Content:     content,
			Sequence:    sequence,
			CreatedAt:   createdAt.Unix(),
			Metadata:    copyMetadata(input.Metadata),
		})
	}
	return out, nil
}

func lastLifecycleMessageSequence(ctx context.Context, db lifecycleSQL, workspaceID, sessionKey string) (int, error) {
	var seq int
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(
			CASE
				WHEN json_valid(COALESCE(meta_json, ''))
				THEN json_extract(meta_json, '$.goncho.seq_in_session')
				ELSE NULL
			END AS INTEGER
		)), 0)
		FROM turns
		WHERE session_id = ?
		  AND memory_sync_status = 'ready'
		  AND CASE
			WHEN json_valid(COALESCE(meta_json, ''))
			THEN json_extract(meta_json, '$.goncho.workspace_id')
			ELSE ''
		  END = ?
	`, sessionKey, workspaceID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("goncho: read last message sequence: %w", err)
	}
	return seq, nil
}

func listLifecycleMessages(ctx context.Context, db lifecycleSQL, workspaceID, sessionKey string) ([]MessageRecord, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, role, content, ts_unix, COALESCE(chat_id, ''), COALESCE(meta_json, '')
		FROM turns
		WHERE session_id = ?
		  AND memory_sync_status = 'ready'
		  AND CASE
			WHEN json_valid(COALESCE(meta_json, ''))
			THEN json_extract(meta_json, '$.goncho.workspace_id')
			ELSE ''
		  END = ?
		ORDER BY CAST(json_extract(meta_json, '$.goncho.seq_in_session') AS INTEGER) ASC, id ASC
	`, sessionKey, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list lifecycle messages: %w", err)
	}
	defer rows.Close()

	var out []MessageRecord
	for rows.Next() {
		var msg MessageRecord
		var metaRaw string
		if err := rows.Scan(&msg.ID, &msg.SessionKey, &msg.Role, &msg.Content, &msg.CreatedAt, &msg.Peer, &metaRaw); err != nil {
			return nil, fmt.Errorf("goncho: scan lifecycle message: %w", err)
		}
		msg.WorkspaceID = workspaceID
		meta, err := decodeLifecycleMeta(metaRaw)
		if err != nil {
			return nil, err
		}
		msg.Sequence = meta.Goncho.Sequence
		if meta.Goncho.PeerID != "" {
			msg.Peer = meta.Goncho.PeerID
		}
		msg.Metadata = copyMetadata(meta.Goncho.Metadata)
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate lifecycle messages: %w", err)
	}
	return out, nil
}

func deleteLifecycleSession(ctx context.Context, db lifecycleSQL, workspaceID, sessionKey string) (SessionDeletionResult, error) {
	messagesDeleted, err := execDeleteCount(ctx, db, `
		DELETE FROM turns
		WHERE session_id = ?
		  AND CASE
			WHEN json_valid(COALESCE(meta_json, ''))
			THEN json_extract(meta_json, '$.goncho.workspace_id')
			ELSE ''
		  END = ?
	`, sessionKey, workspaceID)
	if err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: delete session messages: %w", err)
	}
	conclusionsDeleted, err := execDeleteCount(ctx, db, `
		DELETE FROM goncho_conclusions
		WHERE workspace_id = ? AND session_key = ?
	`, workspaceID, sessionKey)
	if err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: delete session conclusions: %w", err)
	}
	summariesDeleted, err := execDeleteCount(ctx, db, `
		DELETE FROM goncho_session_summaries
		WHERE workspace_id = ? AND session_key = ?
	`, workspaceID, sessionKey)
	if err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: delete session summaries: %w", err)
	}
	return SessionDeletionResult{
		WorkspaceID:        workspaceID,
		SessionKey:         sessionKey,
		MessagesDeleted:    messagesDeleted,
		ConclusionsDeleted: conclusionsDeleted,
		SummariesDeleted:   summariesDeleted,
	}, nil
}

func deleteLifecycleWorkspace(ctx context.Context, db lifecycleSQL, workspaceID string) (WorkspaceDeletionResult, error) {
	messagesDeleted, err := execDeleteCount(ctx, db, `
		DELETE FROM turns
		WHERE CASE
			WHEN json_valid(COALESCE(meta_json, ''))
			THEN json_extract(meta_json, '$.goncho.workspace_id')
			ELSE ''
		END = ?
	`, workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: delete workspace messages: %w", err)
	}
	peerCardsDeleted, err := execDeleteCount(ctx, db, `DELETE FROM goncho_peer_cards WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: delete workspace peer cards: %w", err)
	}
	conclusionsDeleted, err := execDeleteCount(ctx, db, `DELETE FROM goncho_conclusions WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: delete workspace conclusions: %w", err)
	}
	summariesDeleted, err := execDeleteCount(ctx, db, `DELETE FROM goncho_session_summaries WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: delete workspace summaries: %w", err)
	}
	dreamsDeleted, err := execDeleteCount(ctx, db, `DELETE FROM goncho_dreams WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: delete workspace dreams: %w", err)
	}
	return WorkspaceDeletionResult{
		WorkspaceID:        workspaceID,
		MessagesDeleted:    messagesDeleted,
		PeerCardsDeleted:   peerCardsDeleted,
		ConclusionsDeleted: conclusionsDeleted,
		SummariesDeleted:   summariesDeleted,
		DreamsDeleted:      dreamsDeleted,
	}, nil
}

func decodeLifecycleMeta(raw string) (lifecycleTurnMeta, error) {
	var meta lifecycleTurnMeta
	if strings.TrimSpace(raw) == "" {
		return meta, nil
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return meta, fmt.Errorf("goncho: decode lifecycle metadata: %w", err)
	}
	return meta, nil
}

func execDeleteCount(ctx context.Context, db lifecycleSQL, query string, args ...any) (int64, error) {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func copyMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func upsertPeerCard(ctx context.Context, db *sql.DB, workspaceID, observer, peer string, card []string) error {
	raw, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("goncho: marshal peer card: %w", err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO goncho_peer_cards(workspace_id, observer_peer_id, peer_id, card_json, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, observer_peer_id, peer_id)
		DO UPDATE SET card_json = excluded.card_json, updated_at = excluded.updated_at
	`, workspaceID, observer, peer, string(raw), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("goncho: upsert peer card: %w", err)
	}
	return nil
}

func getPeerCard(ctx context.Context, db *sql.DB, workspaceID, observer, peer string) ([]string, error) {
	var raw string
	err := db.QueryRowContext(ctx, `
		SELECT card_json
		FROM goncho_peer_cards
		WHERE workspace_id = ? AND observer_peer_id = ? AND peer_id = ?
	`, workspaceID, observer, peer).Scan(&raw)
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("goncho: get peer card: %w", err)
	}
	var card []string
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		return nil, fmt.Errorf("goncho: decode peer card: %w", err)
	}
	return card, nil
}

func upsertSessionSummary(ctx context.Context, db *sql.DB, row sessionSummaryRow) error {
	createdAt := row.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO goncho_session_summaries(
			workspace_id, session_key, summary_type, content, message_id, created_at, token_count
		)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, session_key, summary_type)
		DO UPDATE SET
			content = excluded.content,
			message_id = excluded.message_id,
			created_at = excluded.created_at,
			token_count = excluded.token_count
	`,
		row.WorkspaceID,
		row.SessionKey,
		row.SummaryType,
		row.Content,
		row.MessageID,
		createdAt,
		row.TokenCount,
	)
	if err != nil {
		return fmt.Errorf("goncho: upsert session summary: %w", err)
	}
	return nil
}

func getSessionSummary(ctx context.Context, db *sql.DB, workspaceID, sessionKey, summaryType string) (*SessionSummary, error) {
	var summary SessionSummary
	err := db.QueryRowContext(ctx, `
		SELECT content, message_id, summary_type, created_at, token_count
		FROM goncho_session_summaries
		WHERE workspace_id = ? AND session_key = ? AND summary_type = ?
	`, workspaceID, sessionKey, summaryType).Scan(
		&summary.Content,
		&summary.MessageID,
		&summary.SummaryType,
		&summary.CreatedAt,
		&summary.TokenCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("goncho: get session summary: %w", err)
	}
	return &summary, nil
}

func getSessionSummaries(ctx context.Context, db *sql.DB, workspaceID, sessionKey string) (*SessionSummary, *SessionSummary, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT content, message_id, summary_type, created_at, token_count
		FROM goncho_session_summaries
		WHERE workspace_id = ? AND session_key = ?
	`, workspaceID, sessionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("goncho: get session summaries: %w", err)
	}
	defer rows.Close()

	var shortSummary *SessionSummary
	var longSummary *SessionSummary
	for rows.Next() {
		var summary SessionSummary
		if err := rows.Scan(
			&summary.Content,
			&summary.MessageID,
			&summary.SummaryType,
			&summary.CreatedAt,
			&summary.TokenCount,
		); err != nil {
			return nil, nil, fmt.Errorf("goncho: scan session summary: %w", err)
		}
		switch summary.SummaryType {
		case "short":
			item := summary
			shortSummary = &item
		case "long":
			item := summary
			longSummary = &item
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("goncho: iterate session summaries: %w", err)
	}
	return shortSummary, longSummary, nil
}

func upsertConclusion(ctx context.Context, db *sql.DB, row conclusionRow) (int64, string, error) {
	now := time.Now().Unix()
	scope := row.Scope
	if scope == "" {
		scope = "workspace"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO goncho_conclusions(
			workspace_id, observer_peer_id, peer_id, session_key, content,
			kind, status, source, idempotency_key, evidence_json, created_at, updated_at, scope
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, observer_peer_id, peer_id, idempotency_key)
		DO UPDATE SET updated_at = excluded.updated_at
	`,
		row.WorkspaceID,
		row.ObserverPeerID,
		row.PeerID,
		nullIfBlank(row.SessionKey),
		row.Content,
		row.Kind,
		row.Status,
		row.Source,
		row.IdempotencyKey,
		row.EvidenceJSON,
		now,
		now,
		scope,
	)
	if err != nil {
		return 0, "", fmt.Errorf("goncho: upsert conclusion: %w", err)
	}

	var id int64
	var status string
	err = db.QueryRowContext(ctx, `
		SELECT id, status
		FROM goncho_conclusions
		WHERE workspace_id = ? AND observer_peer_id = ? AND peer_id = ? AND idempotency_key = ?
	`, row.WorkspaceID, row.ObserverPeerID, row.PeerID, row.IdempotencyKey).Scan(&id, &status)
	if err != nil {
		return 0, "", fmt.Errorf("goncho: lookup conclusion after upsert: %w", err)
	}
	return id, status, nil
}

func deleteConclusion(ctx context.Context, db *sql.DB, workspaceID, observer, peer string, id int64) (bool, error) {
	res, err := db.ExecContext(ctx, `
		DELETE FROM goncho_conclusions
		WHERE id = ? AND workspace_id = ? AND observer_peer_id = ? AND peer_id = ?
	`, id, workspaceID, observer, peer)
	if err != nil {
		return false, fmt.Errorf("goncho: delete conclusion: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("goncho: delete conclusion rows affected: %w", err)
	}
	return affected > 0, nil
}

func findConclusions(ctx context.Context, db *sql.DB, workspaceID, observer, peer, query, sessionKey string, filter compiledSearchFilter, limit int) ([]SearchHit, error) {
	base := `
		SELECT id, content, COALESCE(session_key, '')
		FROM goncho_conclusions
		WHERE observer_peer_id = ? AND peer_id = ?
		  AND (workspace_id = ? AND scope = 'workspace' OR scope = 'global')
	`
	args := []any{observer, peer, workspaceID}
	if trimmed := strings.TrimSpace(sessionKey); trimmed != "" {
		base += ` AND (session_key = ? OR session_key IS NULL)`
		args = append(args, trimmed)
	}
	if len(filter.SessionIDs) > 0 && !filterHasWildcard(filter.SessionIDs) {
		base += ` AND `
		var b strings.Builder
		appendInClause(&b, "session_key", filter.SessionIDs, &args)
		base += b.String()
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		base += ` AND content LIKE ?`
		args = append(args, "%"+trimmed+"%")
	}
	base += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: find conclusions: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var hit SearchHit
		hit.Source = "conclusion"
		if err := rows.Scan(&hit.ID, &hit.Content, &hit.SessionKey); err != nil {
			return nil, fmt.Errorf("goncho: scan conclusion: %w", err)
		}
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate conclusions: %w", err)
	}
	return hits, nil
}

func insertAssistantChatTurn(ctx context.Context, db *sql.DB, sessionID, peer, content, metaJSON string) error {
	sessionID = strings.TrimSpace(sessionID)
	peer = strings.TrimSpace(peer)
	if sessionID == "" || peer == "" || strings.TrimSpace(content) == "" {
		return nil
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, memory_sync_status)
		VALUES(?, 'assistant', ?, ?, ?, ?, 'ready')
	`, sessionID, content, time.Now().Unix(), peer, nullIfBlank(metaJSON))
	if err != nil {
		return fmt.Errorf("goncho: insert assistant chat turn: %w", err)
	}
	return nil
}

func findTurns(ctx context.Context, db *sql.DB, query, sessionKey string, filter compiledSearchFilter, limit int) ([]SearchHit, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return nil, nil
	}
	if !sessionKeyMatchesSources(sessionKey, filter.Sources) {
		return nil, nil
	}

	base := `
		SELECT content, COALESCE(chat_id, ''), COALESCE(session_id, '')
		FROM turns
		WHERE (chat_id = ? OR session_id = ?)
		  AND memory_sync_status = 'ready'
	`
	args := []any{sessionKey, sessionKey}
	if len(filter.SessionIDs) > 0 && !filterHasWildcard(filter.SessionIDs) {
		base += ` AND `
		var b strings.Builder
		appendInClause(&b, "session_id", filter.SessionIDs, &args)
		base += b.String()
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		base += ` AND content LIKE ?`
		args = append(args, "%"+trimmed+"%")
	}
	base += ` ORDER BY ts_unix DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: find turns: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var content, chatID, rowSessionID string
		if err := rows.Scan(&content, &chatID, &rowSessionID); err != nil {
			return nil, fmt.Errorf("goncho: scan turn: %w", err)
		}
		hits = append(hits, SearchHit{
			Source:       "turn",
			OriginSource: originSourceFromChatKey(firstNonBlank(chatID, sessionKey)),
			Content:      content,
			SessionKey:   firstNonBlank(rowSessionID, sessionKey),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate turns: %w", err)
	}
	return hits, nil
}

func originSourceFromChatKey(chatKey string) string {
	chatKey = strings.TrimSpace(chatKey)
	idx := strings.Index(chatKey, ":")
	if idx <= 0 {
		return ""
	}
	return chatKey[:idx]
}

func recentTurns(ctx context.Context, db *sql.DB, sessionKey string, limit int) ([]MessageSlice, error) {
	return recentTurnsAfter(ctx, db, sessionKey, 0, limit)
}

func recentTurnsAfter(ctx context.Context, db *sql.DB, sessionKey string, afterID int64, limit int) ([]MessageSlice, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT role, content
		FROM turns
		WHERE (chat_id = ? OR session_id = ?)
		  AND memory_sync_status = 'ready'
		  AND id > ?
		ORDER BY ts_unix DESC, id DESC
		LIMIT ?
	`, sessionKey, sessionKey, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("goncho: recent turns: %w", err)
	}
	defer rows.Close()

	var reverse []MessageSlice
	for rows.Next() {
		var msg MessageSlice
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, fmt.Errorf("goncho: scan recent turn: %w", err)
		}
		reverse = append(reverse, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate recent turns: %w", err)
	}

	out := make([]MessageSlice, 0, len(reverse))
	for i := len(reverse) - 1; i >= 0; i-- {
		out = append(out, reverse[i])
	}
	return out, nil
}

func recentTurnsByTokenBudget(ctx context.Context, db *sql.DB, sessionKey string, afterID int64, tokenBudget int) ([]MessageSlice, error) {
	if tokenBudget <= 0 {
		return []MessageSlice{}, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT role, content
		FROM turns
		WHERE (chat_id = ? OR session_id = ?)
		  AND memory_sync_status = 'ready'
		  AND id > ?
		ORDER BY ts_unix DESC, id DESC
	`, sessionKey, sessionKey, afterID)
	if err != nil {
		return nil, fmt.Errorf("goncho: recent turns by token budget: %w", err)
	}
	defer rows.Close()

	used := 0
	var reverse []MessageSlice
	for rows.Next() {
		var msg MessageSlice
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, fmt.Errorf("goncho: scan recent turn by token budget: %w", err)
		}
		cost := approxTokens(msg.Content)
		if used+cost > tokenBudget {
			break
		}
		reverse = append(reverse, msg)
		used += cost
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate recent turns by token budget: %w", err)
	}

	out := make([]MessageSlice, 0, len(reverse))
	for i := len(reverse) - 1; i >= 0; i-- {
		out = append(out, reverse[i])
	}
	return out, nil
}

func countReadySessionTurns(ctx context.Context, db *sql.DB, sessionKey string) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM turns
		WHERE (chat_id = ? OR session_id = ?)
		  AND memory_sync_status = 'ready'
	`, sessionKey, sessionKey).Scan(&count); err != nil {
		return 0, fmt.Errorf("goncho: count ready session turns: %w", err)
	}
	return count, nil
}

func readySessionTurnIDAtPosition(ctx context.Context, db *sql.DB, sessionKey string, position int) (int64, error) {
	if position <= 0 {
		return 0, nil
	}
	var id int64
	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM turns
		WHERE (chat_id = ? OR session_id = ?)
		  AND memory_sync_status = 'ready'
		ORDER BY ts_unix ASC, id ASC
		LIMIT 1 OFFSET ?
	`, sessionKey, sessionKey, position-1).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("goncho: find ready session turn position: %w", err)
	}
	return id, nil
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func appendInClause(b *strings.Builder, column string, values []string, args *[]any) {
	b.WriteString(column)
	b.WriteString(` IN (`)
	for i, value := range values {
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(`?`)
		*args = append(*args, value)
	}
	b.WriteString(`)`)
}

func sessionKeyMatchesSources(sessionKey string, sources []string) bool {
	if len(sources) == 0 || filterHasWildcard(sources) {
		return true
	}
	source, _, ok := strings.Cut(strings.TrimSpace(sessionKey), ":")
	if !ok {
		return false
	}
	return slices.Contains(sources, strings.ToLower(strings.TrimSpace(source)))
}
