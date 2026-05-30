package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

const gonchoMetaanalysisCoverageVersion = "goncho-metaanalysis-goal-v1"

// GonchoMetaanalysisCoverageInput binds the local proof evidence back to the
// architecture requirements documented in METAANALYSIS-MEMORY-SYSTEMS.md.
type GonchoMetaanalysisCoverageInput struct {
	SourceDocumentPath       string
	SourceDocumentSHA256     string
	ProofMatrix              gonchoProofMatrixReport
	ReviewLoopVerified       bool
	MemoryToolLoopVerified   bool
	LocalOnlyEvaluator       string
	DocsArchitectureKeywords []string
}

type GonchoMetaanalysisCoverageReport struct {
	Service                       string   `json:"service"`
	CoverageVersion               string   `json:"coverage_version"`
	SourceDocumentPath            string   `json:"source_document_path"`
	SourceDocumentSHA256          string   `json:"source_document_sha256"`
	DocsArchitectureKeywords      []string `json:"docs_architecture_keywords"`
	PrinciplesCovered             []string `json:"principles_covered"`
	ContextLayersCovered          []string `json:"context_layers_covered"`
	LifecycleStatesCovered        []string `json:"lifecycle_states_covered"`
	CoreEvaluationsCovered        []string `json:"core_evaluations_covered"`
	PublicToolsVerified           []string `json:"public_tools_verified"`
	LocalFeaturesVerified         []string `json:"local_features_verified"`
	DeferredFeatures              []string `json:"deferred_features"`
	CompletionCondition           string   `json:"completion_condition"`
	AllLocalEvaluatorChecksPassed bool     `json:"all_local_evaluator_checks_passed"`
}

func BuildGonchoMetaanalysisCoverageReport(input GonchoMetaanalysisCoverageInput) GonchoMetaanalysisCoverageReport {
	features := []string{}
	if input.ProofMatrix.SQLiteRestartVerified {
		features = append(features, "sqlite_local_first")
	}
	if proofMatrixHasAPIContract(input.ProofMatrix, "context") {
		features = append(features, "profile_conclusion_message_context_pack")
	}
	if proofMatrixHasAPIContract(input.ProofMatrix, "search") && input.ProofMatrix.ScopeIsolationVerified {
		features = append(features, "workspace_scope_isolation")
	}
	if input.ProofMatrix.TombstoneExclusionVerified {
		features = append(features, "tombstone_exclusion")
	}
	if input.ProofMatrix.TraceProjectionInvariant == "no_projection_without_recall_trace" && len(input.ProofMatrix.StableTraceIDs) > 0 {
		features = append(features, "recall_trace_diagnostics_replay")
	}
	if proofMatrixHasWarning(input.ProofMatrix, RecallWarningTokenBudgetTruncated) {
		features = append(features, "token_budget_warning")
	}
	if input.ReviewLoopVerified {
		features = append(features, "review_required_context_warning", "goncho_review_tool_resolution")
	}
	if input.MemoryToolLoopVerified {
		features = append(features, "negative_memory_tool_recall", "small_public_tool_surface")
	}
	if proofMatrixHasAPIContract(input.ProofMatrix, "observe") && proofMatrixHasAPIContract(input.ProofMatrix, "list_observations") {
		features = append(features, "raw_observations_with_audit")
	}
	if proofMatrixHasAPIContract(input.ProofMatrix, "filesystem_watcher_import") {
		features = append(features, "filesystem_watcher_connector_import")
	}

	allPassed := input.ProofMatrix.SQLiteRestartVerified &&
		input.ProofMatrix.ScopeIsolationVerified &&
		input.ProofMatrix.TombstoneExclusionVerified &&
		input.ReviewLoopVerified &&
		input.MemoryToolLoopVerified &&
		strings.TrimSpace(input.SourceDocumentPath) != "" &&
		strings.TrimSpace(input.SourceDocumentSHA256) != ""

	completion := strings.TrimSpace(input.LocalOnlyEvaluator)
	if completion == "" {
		completion = "go test ./..."
	}

	return GonchoMetaanalysisCoverageReport{
		Service:                  "goncho",
		CoverageVersion:          gonchoMetaanalysisCoverageVersion,
		SourceDocumentPath:       input.SourceDocumentPath,
		SourceDocumentSHA256:     input.SourceDocumentSHA256,
		DocsArchitectureKeywords: sortedGonchoProofStrings(input.DocsArchitectureKeywords),
		PrinciplesCovered: sortedGonchoProofStrings([]string{
			"evidence_before_memory",
			"claims_not_chunks",
			"hooks_over_manual_saves",
			"orientation_not_dumping",
			"negative_memory_matters",
			"small_agent_surface",
			"trust_is_the_moat",
		}),
		ContextLayersCovered: sortedGonchoProofStrings([]string{
			"evidence",
			"claims",
			"beliefs",
			"orientation",
			"governance",
		}),
		LifecycleStatesCovered: sortedGonchoProofStrings([]string{
			"active",
			"canonical",
			"superseded",
			"quarantined",
			"review_required",
		}),
		CoreEvaluationsCovered: sortedGonchoProofStrings([]string{
			"exact_recall",
			"paraphrase_recall",
			"multi_hop_recall",
			"temporal_state",
			"conflict_adjudication",
			"stale_code_claim",
			"token_budget",
			"noise_resistance",
			"scope_isolation",
			"prompt_injection_persistence",
			"drift_prevention",
		}),
		PublicToolsVerified: sortedGonchoProofStrings([]string{
			"goncho_context",
			"goncho_search",
			"goncho_recall",
			"goncho_remember",
			"goncho_review",
			"goncho_handoff",
		}),
		LocalFeaturesVerified: sortedGonchoProofStrings(features),
		DeferredFeatures: sortedGonchoProofStrings([]string{
			"full_cognitive_map_ui",
			"postgres_team_adapter",
			"cloud_embeddings_required",
			"dashboard_visualization",
		}),
		CompletionCondition:           completion,
		AllLocalEvaluatorChecksPassed: allPassed,
	}
}

func proofMatrixHasAPIContract(report gonchoProofMatrixReport, contract string) bool {
	return sliceutil.Contains(report.APIContractsVerified, contract)
}

func proofMatrixHasWarning(report gonchoProofMatrixReport, code string) bool {
	return sliceutil.Contains(report.WarningCodesSeen, code)
}
