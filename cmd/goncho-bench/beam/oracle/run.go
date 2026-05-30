package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func RunServiceBenchmark(ctx context.Context, cfg ServiceConfig) error {
	cases := goncho.DefaultRecallBenchmarkServiceCases()
	if datasetPath := strings.TrimSpace(cfg.JSONLPath); datasetPath != "" {
		loaded, err := loadBeamServiceJSONLCases(datasetPath)
		if err != nil {
			return err
		}
		cases = loaded
	}
	return runServiceBenchmarkCases(ctx, cfg, cases)
}

func RunHuggingFaceServiceBenchmark(ctx context.Context, cfg ServiceConfig) error {
	if strings.TrimSpace(cfg.JSONLPath) != "" {
		return fmt.Errorf("goncho-bench: --beam-convert-in direct service run cannot be combined with --beam-jsonl")
	}
	records, diagnostics, err := loadBeamHuggingFaceRecordsWithDiagnostics(cfg.ConvertIn, cfg.ConvertScale)
	if err != nil {
		return err
	}
	cfg.conversionDiagnostics = &diagnostics
	cases, err := beamServiceCasesFromJSONLRecords(records)
	if err != nil {
		return err
	}
	return runServiceBenchmarkCases(ctx, cfg, cases)
}

func runServiceBenchmarkCases(ctx context.Context, cfg ServiceConfig, cases []goncho.RecallBenchmarkServiceCase) error {
	runStartedAt := time.Now().UTC()
	leakageChecks := checkBeamServiceLeakage(cases)
	cfg.leakageChecks = &leakageChecks
	if cfg.FailOnLeakage && beamServiceHasBlockingLeakage(leakageChecks) {
		return fmt.Errorf("goncho-bench: BEAM leakage check failed: question_text_in_memory=%d relevant_id_in_memory=%d rubric_text_in_memory=%d", leakageChecks.QuestionTextInMemory, leakageChecks.RelevantIDInMemory, leakageChecks.RubricTextInMemory)
	}
	if path := strings.TrimSpace(cfg.ServiceJudgmentsIn); path != "" {
		judgments, err := loadBeamServiceJudgments(path)
		if err != nil {
			return err
		}
		cfg.judgments = judgments
	}
	databasePath := strings.TrimSpace(cfg.DatabasePath)
	if databasePath == "" {
		dir, err := os.MkdirTemp("", "goncho-beam-service-*")
		if err != nil {
			return fmt.Errorf("goncho-bench: create BEAM service temp db dir: %w", err)
		}
		databasePath = filepath.Join(dir, "beam-service.db")
	}
	store, err := memory.OpenSqlite(databasePath, 0, nil)
	if err != nil {
		return fmt.Errorf("goncho-bench: open BEAM service sqlite: %w", err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		return fmt.Errorf("goncho-bench: run BEAM service migrations: %w", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "goncho-beam-service", ObserverPeerID: "goncho-bench", RecentMessages: 0}, nil)
	report, err := goncho.EvaluateServiceRecallBenchmark(ctx, svc, cases)
	if err != nil {
		return fmt.Errorf("goncho-bench: evaluate BEAM service oracle: %w", err)
	}
	if cfg.judgments != nil && !cfg.ServiceAllowPartialJudgments {
		if err := requireCompleteBeamServiceJudgments(*cfg.judgments, report); err != nil {
			return err
		}
	}
	if outPath := strings.TrimSpace(cfg.ServiceOut); outPath != "" {
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("goncho-bench: encode BEAM service report: %w", err)
		}
		raw = append(raw, '\n')
		if outPath == "-" {
			if _, err := os.Stdout.Write(raw); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return fmt.Errorf("goncho-bench: create BEAM service report dir: %w", err)
			}
			if err := os.WriteFile(outPath, raw, 0o644); err != nil {
				return fmt.Errorf("goncho-bench: write BEAM service report: %w", err)
			}
		}
	}
	return writeBeamServiceComparisonArtifacts(report, cfg, runStartedAt)
}
