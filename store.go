package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/schema"
)

var (
	entityNameRe = regexp.MustCompile(`\b([A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)*)\b`)
	toolNameRe   = regexp.MustCompile(`\b([a-z]+(?:DB|CLI|API|SDK|HTTP|RPC|TLS|SSH|SQL|NoSQL|Postgres|SQLite|MySQL|Redis|Mongo|Docker|K8s|Kubernetes|Git|npm|yarn|pnpm|Go|Rust|Python|Java|TypeScript|JavaScript))\b`)
	prefRe       = regexp.MustCompile(`(?:prefer|like|love|hate|dislike|use|avoid|choose|want|need)\s+(?:to\s+)?(?:the\s+)?(\S+(?:\s+\S+)?)`)
)

func StoreMemory(ctx context.Context, db *sql.DB, p StoreParams) (StoreResult, error) {
	if err := validateStoreParams(p); err != nil {
		return StoreResult{}, err
	}

	now := time.Now().UTC()
	id := fmt.Sprintf("mem_%d", now.UnixNano())
	cs := checksum(p.Content)

	importance := p.Importance
	if importance == 0 {
		importance = 0.5
	}
	scope := p.Scope
	if scope == "" {
		scope = ScopePrivate
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return StoreResult{}, fmt.Errorf("goncho: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO memories (id, kind, content, peer_id, workspace_id, scope, context_id,
		                      importance, valid_from, valid_until, supersedes_id,
		                      created_at, updated_at, checksum)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?)
	`, id, p.Kind, p.Content, p.PeerID, p.WorkspaceID, scope, p.ContextID,
		importance, now.Unix(), now.Unix(), now.Unix(), cs)
	if err != nil {
		return StoreResult{}, fmt.Errorf("goncho: insert memory: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO memory_fts(content, memory_id) VALUES (?, ?)`, p.Content, id)
	if err != nil {
		return StoreResult{}, fmt.Errorf("goncho: insert fts: %w", err)
	}

	rels := extractRelations(p.Content)
	for _, r := range rels {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO memory_relations (source_id, target_entity, relation_type, confidence, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, id, r.TargetEntity, r.RelationType, r.Confidence, now.Unix())
		if err != nil {
			return StoreResult{}, fmt.Errorf("goncho: insert relation: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return StoreResult{}, fmt.Errorf("goncho: commit: %w", err)
	}

	return StoreResult{Memory: Memory{
		ID: id, Kind: p.Kind, Content: p.Content, PeerID: p.PeerID,
		WorkspaceID: p.WorkspaceID, Scope: scope, ContextID: p.ContextID,
		Importance: importance, ValidFrom: now, CreatedAt: now, UpdatedAt: now, Checksum: cs,
	}}, nil
}

func UpdateMemory(ctx context.Context, db *sql.DB, p UpdateParams) (UpdateResult, error) {
	if p.ID == "" {
		return UpdateResult{}, errors.New("goncho: id is required")
	}
	if strings.TrimSpace(p.Content) == "" {
		return UpdateResult{}, errors.New("goncho: content is required")
	}

	now := time.Now().UTC()
	newID := fmt.Sprintf("mem_%d", now.UnixNano())
	cs := checksum(p.Content)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("goncho: begin tx: %w", err)
	}
	defer tx.Rollback()

	var old Memory
	var validFrom, createdAt, updatedAt int64
	err = tx.QueryRowContext(ctx, `
		SELECT id, kind, content, peer_id, workspace_id, scope, context_id,
		       importance, valid_from, created_at, updated_at, checksum
		FROM memories WHERE id = ? AND valid_until IS NULL
	`, p.ID).Scan(&old.ID, &old.Kind, &old.Content, &old.PeerID, &old.WorkspaceID,
		&old.Scope, &old.ContextID, &old.Importance, &validFrom,
		&createdAt, &updatedAt, &old.Checksum)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateResult{}, fmt.Errorf("goncho: memory %s not found or already superseded", p.ID)
		}
		return UpdateResult{}, fmt.Errorf("goncho: query old memory: %w", err)
	}
	old.ValidFrom = time.Unix(validFrom, 0).UTC()
	old.CreatedAt = time.Unix(createdAt, 0).UTC()
	old.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	_, err = tx.ExecContext(ctx, `UPDATE memories SET valid_until = ?, updated_at = ? WHERE id = ?`,
		now.Unix(), now.Unix(), p.ID)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("goncho: supersede old: %w", err)
	}

	importance := p.Importance
	if importance == 0 {
		importance = old.Importance
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO memories (id, kind, content, peer_id, workspace_id, scope, context_id,
		                      importance, valid_from, valid_until, supersedes_id,
		                      created_at, updated_at, checksum)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?)
	`, newID, old.Kind, p.Content, old.PeerID, old.WorkspaceID, old.Scope, old.ContextID,
		importance, now.Unix(), p.ID, now.Unix(), now.Unix(), cs)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("goncho: insert new: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO memory_fts(content, memory_id) VALUES (?, ?)`, p.Content, newID)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("goncho: insert fts: %w", err)
	}

	rels := extractRelations(p.Content)
	for _, r := range rels {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO memory_relations (source_id, target_entity, relation_type, confidence, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, newID, r.TargetEntity, r.RelationType, r.Confidence, now.Unix())
		if err != nil {
			return UpdateResult{}, fmt.Errorf("goncho: insert relation: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return UpdateResult{}, fmt.Errorf("goncho: commit: %w", err)
	}

	old.ValidUntil = now
	newMem := Memory{
		ID: newID, Kind: old.Kind, Content: p.Content, PeerID: old.PeerID,
		WorkspaceID: old.WorkspaceID, Scope: old.Scope, ContextID: old.ContextID,
		Importance: importance, ValidFrom: now, SupersedesID: p.ID,
		CreatedAt: now, UpdatedAt: now, Checksum: cs,
	}

	return UpdateResult{Memory: newMem, Supersede: old}, nil
}

func ForgetMemory(ctx context.Context, db *sql.DB, id string, p ForgetParams) error {
	if id == "" {
		return errors.New("goncho: id is required")
	}

	now := time.Now().UTC()
	res, err := db.ExecContext(ctx, `
		UPDATE memories SET valid_until = ?, updated_at = ?
		WHERE id = ? AND valid_until IS NULL
	`, now.Unix(), now.Unix(), id)
	if err != nil {
		return fmt.Errorf("goncho: forget: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("goncho: forget rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("goncho: memory %s not found or already forgotten", id)
	}
	return nil
}

func validateStoreParams(p StoreParams) error {
	if strings.TrimSpace(p.Content) == "" {
		return errors.New("goncho: content is required")
	}
	if p.Kind == "" {
		return errors.New("goncho: kind is required")
	}
	return nil
}

func extractRelations(content string) []Relation {
	var rels []Relation
	for _, m := range prefRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			rels = append(rels, Relation{
				TargetEntity: strings.TrimSpace(m[1]),
				RelationType: "prefers",
				Confidence:   0.7,
			})
		}
	}
	for _, m := range entityNameRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			rels = append(rels, Relation{
				TargetEntity: m[1],
				RelationType: "mentions",
				Confidence:   0.4,
			})
		}
	}
	for _, m := range toolNameRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			rels = append(rels, Relation{
				TargetEntity: m[1],
				RelationType: "uses_tool",
				Confidence:   0.6,
			})
		}
	}
	return rels
}

func init() {
	_ = schema.RunMigrations
}
