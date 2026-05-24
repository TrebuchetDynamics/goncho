package goncho

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/observationlog"
	"github.com/TrebuchetDynamics/goncho/internal/reviewlog"
)

func RunMigrations(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("%w: nil db", ErrObservationInvalid)
	}
	for _, stmt := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("goncho: run migration pragma %q: %w", stmt, err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("goncho: begin migrations: %w", err)
	}
	defer tx.Rollback()
	for _, stmt := range append(append(observationlog.DDL, reviewlog.DDL...), gonchoSkillLearningProposalDDL...) {
		if err := applyGonchoMigrationStmt(tx, stmt); err != nil {
			return fmt.Errorf("goncho: apply migration: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("goncho: commit observation migrations: %w", err)
	}
	return nil
}

type gonchoMigrationExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func applyGonchoMigrationStmt(exec gonchoMigrationExecer, stmt string) error {
	_, err := exec.Exec(stmt)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return nil
	}
	return err
}
