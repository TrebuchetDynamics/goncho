package goncho

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const gonchoProofMatrixVersion = "goncho-full-local-proof-v1"

type gonchoProofMatrixReportInput struct {
	SQLiteRestartVerified      bool
	APIContractsVerified       []string
	ScopeIsolationVerified     bool
	TombstoneExclusionVerified bool
	Traces                     []RecallTrace
	Benchmark                  RecallBenchmarkReport
	KernelE2EFixture           string
	NegativeControlsRejected   []string
}

type gonchoProofMatrixReport struct {
	Service                    string                      `json:"service"`
	ProofVersion               string                      `json:"proof_version"`
	SQLiteRestartVerified      bool                        `json:"sqlite_restart_verified"`
	APIContractsVerified       []string                    `json:"api_contracts_verified"`
	ScopeIsolationVerified     bool                        `json:"scope_isolation_verified"`
	TombstoneExclusionVerified bool                        `json:"tombstone_exclusion_verified"`
	TraceProjectionInvariant   string                      `json:"trace_projection_invariant"`
	StableTraceIDs             []string                    `json:"stable_trace_ids"`
	WarningCodesSeen           []string                    `json:"warning_codes_seen"`
	NegativeControlsRejected   []string                    `json:"negative_controls_rejected"`
	BenchmarkSummary           gonchoProofBenchmarkSummary `json:"benchmark_summary"`
	KernelE2EFixture           string                      `json:"kernel_e2e_fixture"`
	SelectedMemoryIDs          []string                    `json:"selected_memory_ids"`
	RejectedMemoryIDs          []string                    `json:"rejected_memory_ids"`
	StableJSONSHA256           string                      `json:"stable_json_sha256"`
}

type gonchoProofBenchmarkSummary struct {
	CorpusVersion       string  `json:"corpus_version"`
	CaseCount           int     `json:"case_count"`
	RecallAt5           float64 `json:"recall_at_5"`
	RecallAt10          float64 `json:"recall_at_10"`
	ContextHitRate      float64 `json:"context_hit_rate"`
	TokenBudgetPassRate float64 `json:"token_budget_pass_rate"`
	WarningCount        int     `json:"warning_count"`
}

func buildGonchoProofMatrixReport(input gonchoProofMatrixReportInput) (gonchoProofMatrixReport, error) {
	if len(input.Traces) == 0 {
		return gonchoProofMatrixReport{}, fmt.Errorf("goncho proof matrix: at least one recall trace is required")
	}
	warningCodes := map[string]struct{}{}
	selectedIDs := []string{}
	rejectedIDs := []string{}
	stableTraceIDs := make([]string, 0, len(input.Traces))
	traceHasher := sha256.New()
	projectionInvariant := ""
	for _, trace := range input.Traces {
		if strings.TrimSpace(trace.TraceID) == "" {
			return gonchoProofMatrixReport{}, fmt.Errorf("goncho proof matrix: recall trace missing trace_id")
		}
		stableTraceIDs = append(stableTraceIDs, trace.TraceID)
		stableJSON, err := trace.StableJSON()
		if err != nil {
			return gonchoProofMatrixReport{}, fmt.Errorf("goncho proof matrix: stable trace JSON: %w", err)
		}
		_, _ = traceHasher.Write(stableJSON)
		for _, warning := range trace.Warnings {
			code := strings.TrimSpace(warning.Code)
			if code == "" {
				return gonchoProofMatrixReport{}, fmt.Errorf("goncho proof matrix: degraded recall warning missing code")
			}
			warningCodes[code] = struct{}{}
		}
		for _, item := range trace.Selected {
			if id := strings.TrimSpace(item.Candidate.MemoryID); id != "" {
				selectedIDs = append(selectedIDs, id)
			}
		}
		for _, item := range trace.Rejected {
			if id := strings.TrimSpace(item.Candidate.MemoryID); id != "" {
				rejectedIDs = append(rejectedIDs, id)
			}
		}
		diagnostics := BuildRecallDiagnostics(trace)
		if projectionInvariant == "" {
			projectionInvariant = diagnostics.ProjectionInvariant
		} else if projectionInvariant != diagnostics.ProjectionInvariant {
			return gonchoProofMatrixReport{}, fmt.Errorf("goncho proof matrix: inconsistent projection invariant %q vs %q", projectionInvariant, diagnostics.ProjectionInvariant)
		}
	}

	report := gonchoProofMatrixReport{
		Service:                    "goncho",
		ProofVersion:               gonchoProofMatrixVersion,
		SQLiteRestartVerified:      input.SQLiteRestartVerified,
		APIContractsVerified:       sortedGonchoProofStrings(input.APIContractsVerified),
		ScopeIsolationVerified:     input.ScopeIsolationVerified,
		TombstoneExclusionVerified: input.TombstoneExclusionVerified,
		TraceProjectionInvariant:   projectionInvariant,
		StableTraceIDs:             cloneStrings(stableTraceIDs),
		WarningCodesSeen:           sortedGonchoProofStringSet(warningCodes),
		NegativeControlsRejected:   sortedGonchoProofStrings(input.NegativeControlsRejected),
		BenchmarkSummary: gonchoProofBenchmarkSummary{
			CorpusVersion:       input.Benchmark.CorpusVersion,
			CaseCount:           input.Benchmark.CaseCount,
			RecallAt5:           input.Benchmark.RecallAt5,
			RecallAt10:          input.Benchmark.RecallAt10,
			ContextHitRate:      input.Benchmark.ContextHitRate,
			TokenBudgetPassRate: input.Benchmark.TokenBudgetPassRate,
			WarningCount:        input.Benchmark.WarningCount,
		},
		KernelE2EFixture:  input.KernelE2EFixture,
		SelectedMemoryIDs: selectedIDs,
		RejectedMemoryIDs: rejectedIDs,
		StableJSONSHA256:  hex.EncodeToString(traceHasher.Sum(nil)),
	}
	if report.StableTraceIDs == nil {
		report.StableTraceIDs = []string{}
	}
	if report.WarningCodesSeen == nil {
		report.WarningCodesSeen = []string{}
	}
	if report.SelectedMemoryIDs == nil {
		report.SelectedMemoryIDs = []string{}
	}
	if report.RejectedMemoryIDs == nil {
		report.RejectedMemoryIDs = []string{}
	}
	return report, nil
}

func sortedGonchoProofStrings(values []string) []string {
	return textutil.UniqueTrimmed(values, true)
}

func sortedGonchoProofStringSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
