package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/memory"
	"github.com/TrebuchetDynamics/goncho/service"
)

type config struct {
	DatasetPath                       string
	OutPath                           string
	FailurePath                       string
	DatabasePath                      string
	Limit                             int
	Runs                              int
	System                            string
	DatasetRevision                   string
	DatasetSHA256                     string
	FailOnLeakage                     bool
	ClassifyReportPath                string
	ClassifyFailurePath               string
	ClassifyJSONLOut                  string
	ClassifyMarkdownOut               string
	LocomoMemoriesPath                string
	LocomoQuestionsPath               string
	LocomoMarkdownOut                 string
	LocomoName                        string
	LocomoCompareReport               string
	LocomoCompareJSONL                string
	LocomoCompareMD                   string
	LocomoCompareA                    string
	LocomoCompareB                    string
	LocomoBackendComparisonJSON       string
	LocomoBackendComparisonMD         string
	LocomoBackendComparisonFailures   string
	LocomoAgentMemoryResults          string
	LocomoMem0Results                 string
	BeamConvertIn                     string
	BeamConvertOut                    string
	BeamConvertScale                  string
	BeamJSONLPath                     string
	BeamServiceOut                    string
	BeamServiceResultsOut             string
	BeamServiceSummaryOut             string
	BeamServicePairedOut              string
	BeamServiceFailuresOut            string
	BeamServiceJudgeRequestsOut       string
	BeamServiceJudgmentsIn            string
	BeamServiceAllowPartialJudgments  bool
	BeamServiceConfigID               string
	BeamPairedResultsIn               string
	BeamPairedResultsOut              string
	BeamPairedResultsConfigID         string
	BeamPairedComparePath             string
	BeamPairedBaselineConfigID        string
	BeamPairedCandidateConfigID       string
	BeamPairedCompareJSONOut          string
	BeamPairedCompareMarkdownOut      string
	BeamPairedCompareBootstrapSamples int
	BeamPairedCompareEffectSizeFloor  float64
	BeamConversionDiagnostics         *beamConversionDiagnostics
	BeamServiceLeakageChecks          *beamServiceLeakageChecks
	BeamServiceJudgments              *beamServiceJudgmentSet
}

type dataset struct {
	Name      string
	Memories  []MemoryRecord
	Questions []QuestionRecord
}

type jsonlRecord struct {
	Type        string   `json:"type"`
	Dataset     string   `json:"dataset,omitempty"`
	ID          string   `json:"id,omitempty"`
	Peer        string   `json:"peer,omitempty"`
	SessionKey  string   `json:"session_key,omitempty"`
	Content     string   `json:"content,omitempty"`
	Query       string   `json:"query,omitempty"`
	RelevantIDs []string `json:"relevant_ids,omitempty"`
}

type MemoryRecord struct {
	ID         string `json:"id"`
	Peer       string `json:"peer"`
	SessionKey string `json:"session_key,omitempty"`
	Content    string `json:"content"`
}

type QuestionRecord struct {
	ID          string   `json:"id"`
	Peer        string   `json:"peer"`
	SessionKey  string   `json:"session_key,omitempty"`
	Query       string   `json:"query"`
	RelevantIDs []string `json:"relevant_ids"`
}

type BenchmarkReport struct {
	System          string                    `json:"system"`
	Dataset         string                    `json:"dataset"`
	DatasetRevision string                    `json:"dataset_revision,omitempty"`
	DatasetSHA256   string                    `json:"dataset_sha256,omitempty"`
	GoVersion       string                    `json:"go_version"`
	GOOS            string                    `json:"goos"`
	GOARCH          string                    `json:"goarch"`
	CPUCount        int                       `json:"cpu_count"`
	MemoryCount     int                       `json:"memory_count"`
	QuestionCount   int                       `json:"question_count"`
	Runs            int                       `json:"runs"`
	Leakage         LeakageReport             `json:"leakage"`
	RecallAt5       float64                   `json:"recall_at_5"`
	RecallAt10      float64                   `json:"recall_at_10"`
	RecallAnyAt5    float64                   `json:"recall_any_at_5"`
	RecallAnyAt10   float64                   `json:"recall_any_at_10"`
	MRR             float64                   `json:"mrr"`
	Questions       []BenchmarkQuestionReport `json:"questions"`
}

type BenchmarkQuestionReport struct {
	ID           string   `json:"id"`
	Query        string   `json:"query"`
	RelevantIDs  []string `json:"relevant_ids"`
	RetrievedIDs []string `json:"retrieved_ids"`
	Rank         int      `json:"rank"`
	RecallAt5    float64  `json:"recall_at_5"`
	RecallAt10   float64  `json:"recall_at_10"`
	MRR          float64  `json:"mrr"`
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.DatasetPath, "dataset", "", "LongMemEval-style JSONL dataset path")
	flag.StringVar(&cfg.OutPath, "out", "", "JSON report output path")
	flag.StringVar(&cfg.FailurePath, "failures", "", "JSONL failure audit output path")
	flag.StringVar(&cfg.DatabasePath, "db", "", "SQLite database path; defaults to a temp file")
	flag.StringVar(&cfg.System, "system", "goncho", "retrieval system: goncho, goncho-no-rank, random, bm25, sqlite-fts5; LOCOMO also supports goncho-recall, goncho-recall-rank, and goncho-recall-annotated")
	flag.StringVar(&cfg.DatasetRevision, "dataset-revision", "", "dataset source revision for report metadata")
	flag.StringVar(&cfg.DatasetSHA256, "dataset-sha256", "", "dataset source sha256 for report metadata")
	flag.BoolVar(&cfg.FailOnLeakage, "fail-on-leakage", false, "exit non-zero if leakage checks find query/gold-id leakage or BEAM rubric-label leakage")
	flag.StringVar(&cfg.ClassifyReportPath, "classify-report", "", "existing benchmark JSON report to classify instead of running retrieval")
	flag.StringVar(&cfg.ClassifyFailurePath, "classify-failures", "", "optional existing failure JSONL to validate as a top-10 miss audit reference; classification still uses the full report")
	flag.StringVar(&cfg.ClassifyJSONLOut, "classify-jsonl-out", "", "JSONL output path for one failure-category row per hard case")
	flag.StringVar(&cfg.ClassifyMarkdownOut, "classify-md-out", "", "Markdown output path for failure-category summary")
	flag.StringVar(&cfg.LocomoMemoriesPath, "locomo-memories", "", "LOCOMO-style memories JSONL path for retrieval-first benchmark")
	flag.StringVar(&cfg.LocomoQuestionsPath, "locomo-questions", "", "LOCOMO-style questions JSONL path for retrieval-first benchmark")
	flag.StringVar(&cfg.LocomoMarkdownOut, "locomo-md-out", "", "Markdown output path for LOCOMO benchmark report")
	flag.StringVar(&cfg.LocomoName, "locomo-name", "", "LOCOMO benchmark display name; defaults to LOCOMO smoke")
	flag.StringVar(&cfg.LocomoCompareReport, "locomo-compare-report", "", "existing LOCOMO JSON report for paired system comparison")
	flag.StringVar(&cfg.LocomoCompareJSONL, "locomo-compare-jsonl-out", "", "JSONL output path for LOCOMO paired system comparison")
	flag.StringVar(&cfg.LocomoCompareMD, "locomo-compare-md-out", "", "Markdown output path for LOCOMO paired system comparison")
	flag.StringVar(&cfg.LocomoCompareA, "locomo-compare-a", "bm25", "A system for LOCOMO paired comparison")
	flag.StringVar(&cfg.LocomoCompareB, "locomo-compare-b", "goncho", "B system for LOCOMO paired comparison")
	flag.StringVar(&cfg.LocomoBackendComparisonJSON, "locomo-backend-comparison-json-out", "", "JSON output path for LOCOMO external-backend adapter comparison")
	flag.StringVar(&cfg.LocomoBackendComparisonMD, "locomo-backend-comparison-md-out", "", "Markdown output path for LOCOMO external-backend adapter comparison")
	flag.StringVar(&cfg.LocomoBackendComparisonFailures, "locomo-backend-comparison-failures-out", "", "JSONL failure output path for LOCOMO external-backend adapter comparison")
	flag.StringVar(&cfg.LocomoAgentMemoryResults, "locomo-agentmemory-results", "", "optional JSONL results from scripts/bench_agentmemory_locomo.py")
	flag.StringVar(&cfg.LocomoMem0Results, "locomo-mem0-results", "", "optional JSONL results from scripts/bench_mem0_locomo.py")
	flag.StringVar(&cfg.BeamConvertIn, "beam-convert-in", "", "HuggingFace BEAM JSONL export to convert into Goncho's BEAM service JSONL format or run directly with BEAM service artifact flags")
	flag.StringVar(&cfg.BeamConvertOut, "beam-convert-out", "", "output path for convert-only --beam-convert-in; use - for stdout")
	flag.StringVar(&cfg.BeamConvertScale, "beam-convert-scale", "", "fallback BEAM scale for converted records when the source record omits scale")
	flag.StringVar(&cfg.BeamJSONLPath, "beam-jsonl", "", "BEAM-style JSONL dataset path for service-backed recall evaluation")
	flag.StringVar(&cfg.BeamServiceOut, "beam-service-out", "", "JSON output path for Goncho's deterministic service-backed BEAM-style recall oracle")
	flag.StringVar(&cfg.BeamServiceResultsOut, "beam-service-results-out", "", "Mnemosyne-compatible beam_e2e_results.json output path for the service-backed BEAM-style oracle")
	flag.StringVar(&cfg.BeamServiceSummaryOut, "beam-service-summary-out", "", "Mnemosyne-compatible beam_e2e_summary.json output path for the service-backed BEAM-style oracle")
	flag.StringVar(&cfg.BeamServicePairedOut, "beam-service-paired-out", "", "Mnemosyne-compatible paired_outcomes.jsonl append path for the service-backed BEAM-style oracle")
	flag.StringVar(&cfg.BeamServiceFailuresOut, "beam-service-failures-out", "", "JSONL failure audit output path for the service-backed BEAM-style oracle")
	flag.StringVar(&cfg.BeamServiceJudgeRequestsOut, "beam-service-judge-requests-out", "", "JSONL answer/judge request export path for official BEAM-compatible evaluation; answer prompts exclude ideal answers and rubrics")
	flag.StringVar(&cfg.BeamServiceJudgmentsIn, "beam-service-judgments-in", "", "official BEAM answer/judge results to merge into service-backed BEAM artifacts; accepts flat JSONL rows or nested Mnemosyne-compatible beam_e2e_results.json")
	flag.BoolVar(&cfg.BeamServiceAllowPartialJudgments, "beam-service-allow-partial-judgments", false, "allow --beam-service-judgments-in to leave missing/unmatched judgment rows as diagnostics instead of failing")
	flag.StringVar(&cfg.BeamServiceConfigID, "beam-service-config-id", "", "config_id written to service-backed BEAM paired outcomes and summary metadata")
	flag.StringVar(&cfg.BeamPairedResultsIn, "beam-paired-results-in", "", "nested Mnemosyne-compatible beam_e2e_results.json to append as paired_outcomes.jsonl rows")
	flag.StringVar(&cfg.BeamPairedResultsOut, "beam-paired-results-out", "", "paired_outcomes.jsonl append path for --beam-paired-results-in")
	flag.StringVar(&cfg.BeamPairedResultsConfigID, "beam-paired-results-config-id", "", "config_id for --beam-paired-results-in rows; defaults to results metadata config_id")
	flag.StringVar(&cfg.BeamPairedComparePath, "beam-paired-compare", "", "Mnemosyne-compatible paired_outcomes.jsonl path to compare two BEAM config_id arms")
	flag.StringVar(&cfg.BeamPairedBaselineConfigID, "beam-paired-baseline-config-id", "", "baseline config_id for --beam-paired-compare")
	flag.StringVar(&cfg.BeamPairedCandidateConfigID, "beam-paired-candidate-config-id", "", "candidate config_id for --beam-paired-compare")
	flag.StringVar(&cfg.BeamPairedCompareJSONOut, "beam-paired-json-out", "", "JSON report output path for --beam-paired-compare; defaults to stdout when no outputs are set")
	flag.StringVar(&cfg.BeamPairedCompareMarkdownOut, "beam-paired-md-out", "", "Markdown report output path for --beam-paired-compare")
	flag.IntVar(&cfg.BeamPairedCompareBootstrapSamples, "beam-paired-bootstrap-samples", 5000, "deterministic bootstrap samples for --beam-paired-compare 95% CI")
	flag.Float64Var(&cfg.BeamPairedCompareEffectSizeFloor, "beam-paired-effect-size-floor", 0.02, "minimum absolute score delta for a BEAM paired-comparison superiority verdict")
	flag.IntVar(&cfg.Limit, "limit", 10, "retrieval limit per question")
	flag.IntVar(&cfg.Runs, "runs", 1, "number of benchmark runs to aggregate")
	flag.Parse()
	if err := run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	if strings.TrimSpace(cfg.BeamPairedComparePath) != "" {
		return runBeamPairedComparison(cfg)
	}
	if strings.TrimSpace(cfg.BeamPairedResultsIn) != "" {
		return appendBeamPairedOutcomesFromResults(cfg)
	}
	if strings.TrimSpace(cfg.BeamConvertIn) != "" {
		if beamServiceArtifactRequested(cfg) {
			return runBeamHuggingFaceServiceBenchmark(ctx, cfg)
		}
		return convertBeamHuggingFaceJSONL(cfg.BeamConvertIn, cfg.BeamConvertOut, cfg.BeamConvertScale)
	}
	if strings.TrimSpace(cfg.BeamJSONLPath) != "" || beamServiceArtifactRequested(cfg) {
		return runBeamServiceBenchmark(ctx, cfg)
	}
	if strings.TrimSpace(cfg.LocomoCompareReport) != "" {
		return generateLocomoComparison(cfg.LocomoCompareReport, cfg.LocomoCompareJSONL, cfg.LocomoCompareMD, cfg.LocomoCompareA, cfg.LocomoCompareB)
	}
	if strings.TrimSpace(cfg.LocomoBackendComparisonJSON) != "" || strings.TrimSpace(cfg.LocomoBackendComparisonMD) != "" {
		return runLocomoBackendComparison(ctx, cfg)
	}
	if strings.TrimSpace(cfg.ClassifyReportPath) != "" {
		return generateFailureCategoryReports(cfg.ClassifyReportPath, cfg.ClassifyFailurePath, cfg.ClassifyJSONLOut, cfg.ClassifyMarkdownOut)
	}
	if strings.TrimSpace(cfg.LocomoMemoriesPath) != "" || strings.TrimSpace(cfg.LocomoQuestionsPath) != "" {
		return runLocomoBenchmark(ctx, cfg)
	}
	if strings.TrimSpace(cfg.DatasetPath) == "" {
		return errors.New("goncho-bench: --dataset is required")
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 10
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 1
	}
	data, err := loadDataset(cfg.DatasetPath)
	if err != nil {
		return err
	}
	reports := make([]BenchmarkReport, 0, cfg.Runs)
	for i := 0; i < cfg.Runs; i++ {
		runCfg := cfg
		if strings.TrimSpace(runCfg.DatabasePath) == "" || cfg.Runs > 1 {
			dir, err := os.MkdirTemp("", "goncho-bench-*")
			if err != nil {
				return fmt.Errorf("goncho-bench: create temp db dir: %w", err)
			}
			runCfg.DatabasePath = filepath.Join(dir, "bench.db")
		}
		report, err := evaluateOnce(ctx, data, runCfg)
		if err != nil {
			return err
		}
		reports = append(reports, report)
	}
	report := aggregateReports(reports)
	report.DatasetRevision = cfg.DatasetRevision
	report.DatasetSHA256 = cfg.DatasetSHA256
	report.Leakage = checkLeakage(data)
	if cfg.FailOnLeakage && (report.Leakage.QueryInMemory > 0 || report.Leakage.GoldIDInMemory > 0) {
		return fmt.Errorf("goncho-bench: leakage check failed: query_in_memory=%d gold_id_in_memory=%d", report.Leakage.QueryInMemory, report.Leakage.GoldIDInMemory)
	}
	if err := writeFailureAudit(cfg.FailurePath, report); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho-bench: encode report: %w", err)
	}
	raw = append(raw, '\n')
	if strings.TrimSpace(cfg.OutPath) == "" {
		_, err = os.Stdout.Write(raw)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.OutPath), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create report dir: %w", err)
	}
	if err := os.WriteFile(cfg.OutPath, raw, 0o644); err != nil {
		return fmt.Errorf("goncho-bench: write report: %w", err)
	}
	return nil
}

func beamServiceArtifactRequested(cfg config) bool {
	return strings.TrimSpace(cfg.BeamServiceOut) != "" || strings.TrimSpace(cfg.BeamServiceResultsOut) != "" || strings.TrimSpace(cfg.BeamServiceSummaryOut) != "" || strings.TrimSpace(cfg.BeamServicePairedOut) != "" || strings.TrimSpace(cfg.BeamServiceFailuresOut) != "" || strings.TrimSpace(cfg.BeamServiceJudgeRequestsOut) != ""
}

func runBeamServiceBenchmark(ctx context.Context, cfg config) error {
	cases := goncho.DefaultRecallBenchmarkServiceCases()
	if datasetPath := strings.TrimSpace(cfg.BeamJSONLPath); datasetPath != "" {
		loaded, err := loadBeamServiceJSONLCases(datasetPath)
		if err != nil {
			return err
		}
		cases = loaded
	}
	return runBeamServiceBenchmarkCases(ctx, cfg, cases)
}

func runBeamHuggingFaceServiceBenchmark(ctx context.Context, cfg config) error {
	if strings.TrimSpace(cfg.BeamJSONLPath) != "" {
		return fmt.Errorf("goncho-bench: --beam-convert-in direct service run cannot be combined with --beam-jsonl")
	}
	records, diagnostics, err := loadBeamHuggingFaceRecordsWithDiagnostics(cfg.BeamConvertIn, cfg.BeamConvertScale)
	if err != nil {
		return err
	}
	cfg.BeamConversionDiagnostics = &diagnostics
	cases, err := beamServiceCasesFromJSONLRecords(records)
	if err != nil {
		return err
	}
	return runBeamServiceBenchmarkCases(ctx, cfg, cases)
}

func runBeamServiceBenchmarkCases(ctx context.Context, cfg config, cases []goncho.RecallBenchmarkServiceCase) error {
	runStartedAt := time.Now().UTC()
	leakageChecks := checkBeamServiceLeakage(cases)
	cfg.BeamServiceLeakageChecks = &leakageChecks
	if cfg.FailOnLeakage && beamServiceHasBlockingLeakage(leakageChecks) {
		return fmt.Errorf("goncho-bench: BEAM leakage check failed: question_text_in_memory=%d relevant_id_in_memory=%d rubric_text_in_memory=%d", leakageChecks.QuestionTextInMemory, leakageChecks.RelevantIDInMemory, leakageChecks.RubricTextInMemory)
	}
	if path := strings.TrimSpace(cfg.BeamServiceJudgmentsIn); path != "" {
		judgments, err := loadBeamServiceJudgments(path)
		if err != nil {
			return err
		}
		cfg.BeamServiceJudgments = judgments
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
	if cfg.BeamServiceJudgments != nil && !cfg.BeamServiceAllowPartialJudgments {
		if err := requireCompleteBeamServiceJudgments(*cfg.BeamServiceJudgments, report); err != nil {
			return err
		}
	}
	if outPath := strings.TrimSpace(cfg.BeamServiceOut); outPath != "" {
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

func evaluateOnce(ctx context.Context, data dataset, cfg config) (BenchmarkReport, error) {
	system := strings.TrimSpace(cfg.System)
	if system == "" {
		system = "goncho"
	}
	var svc *goncho.Service
	contentIDs := map[string][]string{}
	var closeStore func() error
	if system == "goncho" {
		store, err := memory.OpenSqlite(cfg.DatabasePath, 0, nil)
		if err != nil {
			return BenchmarkReport{}, fmt.Errorf("goncho-bench: open sqlite: %w", err)
		}
		closeStore = func() error { return store.Close(ctx) }
		defer closeStore()
		if err := goncho.RunMigrations(store.DB()); err != nil {
			return BenchmarkReport{}, fmt.Errorf("goncho-bench: run migrations: %w", err)
		}
		svc = goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "goncho-bench", ObserverPeerID: "goncho-bench", RecentMessages: 0}, nil)
		for _, mem := range data.Memories {
			if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: mem.Peer, SessionKey: mem.SessionKey, Conclusion: mem.Content, Scope: "benchmark"}); err != nil {
				return BenchmarkReport{}, fmt.Errorf("goncho-bench: store memory %s: %w", mem.ID, err)
			}
			key := contentIDKey(mem.Peer, mem.Content)
			contentIDs[key] = append(contentIDs[key], mem.ID)
		}
	}
	report := BenchmarkReport{System: system, Dataset: data.Name, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUCount: runtime.NumCPU(), MemoryCount: len(data.Memories), QuestionCount: len(data.Questions), Runs: 1, Questions: []BenchmarkQuestionReport{}}
	for _, q := range data.Questions {
		retrievedIDs, err := retrieveForSystem(ctx, svc, data, q, contentIDs, system, cfg.Limit)
		if err != nil {
			return BenchmarkReport{}, fmt.Errorf("goncho-bench: search question %s: %w", q.ID, err)
		}
		qr := BenchmarkQuestionReport{ID: q.ID, Query: q.Query, RelevantIDs: q.RelevantIDs, RetrievedIDs: retrievedIDs, Rank: firstRelevantRank(retrievedIDs, q.RelevantIDs)}
		qr.RecallAt5 = recallAtKForIDs(retrievedIDs, q.RelevantIDs, 5)
		qr.RecallAt10 = recallAtKForIDs(retrievedIDs, q.RelevantIDs, 10)
		if qr.Rank > 0 {
			qr.MRR = roundMetric(1 / float64(qr.Rank))
		}
		report.Questions = append(report.Questions, qr)
	}
	report.RecallAt5, report.RecallAt10, report.MRR = summarizeMetrics(report.Questions)
	report.RecallAnyAt5, report.RecallAnyAt10 = summarizeRecallAny(report.Questions)
	return report, nil
}

func aggregateReports(reports []BenchmarkReport) BenchmarkReport {
	if len(reports) == 0 {
		return BenchmarkReport{}
	}
	out := reports[len(reports)-1]
	out.Runs = len(reports)
	var r5, r10, any5, any10, mrr float64
	for _, report := range reports {
		r5 += report.RecallAt5
		r10 += report.RecallAt10
		any5 += report.RecallAnyAt5
		any10 += report.RecallAnyAt10
		mrr += report.MRR
	}
	out.RecallAt5 = roundMetric(r5 / float64(len(reports)))
	out.RecallAt10 = roundMetric(r10 / float64(len(reports)))
	out.RecallAnyAt5 = roundMetric(any5 / float64(len(reports)))
	out.RecallAnyAt10 = roundMetric(any10 / float64(len(reports)))
	out.MRR = roundMetric(mrr / float64(len(reports)))
	return out
}

func loadDataset(path string) (dataset, error) {
	file, err := os.Open(path)
	if err != nil {
		return dataset{}, fmt.Errorf("goncho-bench: open dataset: %w", err)
	}
	defer file.Close()
	data := dataset{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var rec jsonlRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return dataset{}, fmt.Errorf("goncho-bench: decode dataset line %d: %w", lineNo, err)
		}
		switch strings.ToLower(strings.TrimSpace(rec.Type)) {
		case "meta":
			if strings.TrimSpace(rec.Dataset) != "" {
				data.Name = rec.Dataset
			}
		case "memory":
			if rec.ID == "" || rec.Content == "" {
				return dataset{}, fmt.Errorf("goncho-bench: memory line %d requires id and content", lineNo)
			}
			if rec.Peer == "" {
				rec.Peer = "benchmark-peer"
			}
			data.Memories = append(data.Memories, MemoryRecord{ID: rec.ID, Peer: rec.Peer, SessionKey: rec.SessionKey, Content: rec.Content})
		case "question":
			if rec.ID == "" || rec.Query == "" {
				return dataset{}, fmt.Errorf("goncho-bench: question line %d requires id and query", lineNo)
			}
			if len(rec.RelevantIDs) == 0 {
				return dataset{}, fmt.Errorf("goncho-bench: question line %d requires relevant_ids", lineNo)
			}
			if rec.Peer == "" {
				rec.Peer = "benchmark-peer"
			}
			data.Questions = append(data.Questions, QuestionRecord{ID: rec.ID, Peer: rec.Peer, SessionKey: rec.SessionKey, Query: rec.Query, RelevantIDs: rec.RelevantIDs})
		default:
			return dataset{}, fmt.Errorf("goncho-bench: line %d has unsupported type %q", lineNo, rec.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return dataset{}, fmt.Errorf("goncho-bench: scan dataset: %w", err)
	}
	if len(data.Memories) == 0 || len(data.Questions) == 0 {
		return dataset{}, errors.New("goncho-bench: dataset requires at least one memory and one question")
	}
	return data, nil
}

func firstRelevantRank(retrievedIDs, relevantIDs []string) int {
	relevant := set(relevantIDs)
	for i, id := range retrievedIDs {
		if _, ok := relevant[id]; ok {
			return i + 1
		}
	}
	return 0
}

func recallAtKForIDs(retrievedIDs, relevantIDs []string, k int) float64 {
	if len(relevantIDs) == 0 || k <= 0 {
		return 0
	}
	relevant := set(relevantIDs)
	limit := k
	if len(retrievedIDs) < limit {
		limit = len(retrievedIDs)
	}
	found := map[string]struct{}{}
	for _, id := range retrievedIDs[:limit] {
		if _, ok := relevant[id]; ok {
			found[id] = struct{}{}
		}
	}
	return roundMetric(float64(len(found)) / float64(len(relevant)))
}

func summarizeMetrics(questions []BenchmarkQuestionReport) (float64, float64, float64) {
	if len(questions) == 0 {
		return 0, 0, 0
	}
	var r5, r10, mrr float64
	for _, q := range questions {
		r5 += q.RecallAt5
		r10 += q.RecallAt10
		mrr += q.MRR
	}
	return roundMetric(r5 / float64(len(questions))), roundMetric(r10 / float64(len(questions))), roundMetric(mrr / float64(len(questions)))
}

func summarizeRecallAny(questions []BenchmarkQuestionReport) (float64, float64) {
	if len(questions) == 0 {
		return 0, 0
	}
	var any5, any10 float64
	for _, q := range questions {
		if q.Rank > 0 && q.Rank <= 5 {
			any5++
		}
		if q.Rank > 0 && q.Rank <= 10 {
			any10++
		}
	}
	return roundMetric(any5 / float64(len(questions))), roundMetric(any10 / float64(len(questions)))
}

func set(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func roundMetric(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}
