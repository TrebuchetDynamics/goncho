package goncho

import (
	"database/sql"
	"fmt"

	"github.com/TrebuchetDynamics/goncho/internal/observationlog"
	"github.com/TrebuchetDynamics/goncho/internal/reviewlog"
	"github.com/TrebuchetDynamics/goncho/internal/skillproposals"
	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
)

const GonchoSQLiteSchemaVersion = "goncho-sqlite-v1"

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
	ddl := append(append(append(append(append(append(append(append(append(append(append(observationlog.DDL, reviewlog.DDL...), skillproposals.DDL...), memoryAnnotationDDL...), memorySlotDDL...), actionGraphDDL...), actionLeaseDDL...), actionSignalReceiptDDL...), teamFeedDDL...), imageMemoryDDL...), retentionAuditDDL...), evalFeedbackDDL...)
	for _, stmt := range ddl {
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
	if sqlutil.IsSQLiteDuplicateColumnError(err) {
		return nil
	}
	return err
}
