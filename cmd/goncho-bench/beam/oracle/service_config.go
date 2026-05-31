package oracle

import (
	"context"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

type ServiceConfig struct {
	DatasetPath                  string
	DatabasePath                 string
	FailOnLeakage                bool
	ConvertIn                    string
	ConvertOut                   string
	ConvertScale                 string
	JSONLPath                    string
	ServiceOut                   string
	ServiceResultsOut            string
	ServiceSummaryOut            string
	ServicePairedOut             string
	ServiceFailuresOut           string
	ServiceJudgeRequestsOut      string
	ServiceJudgmentsIn           string
	ServiceAllowPartialJudgments bool
	ServiceConfigID              string

	conversionDiagnostics *beamConversionDiagnostics
	leakageChecks         *beamServiceLeakageChecks
	judgments             *beamServiceJudgmentSet
}

func ArtifactRequested(cfg ServiceConfig) bool {
	return shared.HasNonEmptyTrimmed(cfg.ServiceOut) || shared.HasNonEmptyTrimmed(cfg.ServiceResultsOut) || shared.HasNonEmptyTrimmed(cfg.ServiceSummaryOut) || shared.HasNonEmptyTrimmed(cfg.ServicePairedOut) || shared.HasNonEmptyTrimmed(cfg.ServiceFailuresOut) || shared.HasNonEmptyTrimmed(cfg.ServiceJudgeRequestsOut)
}

func Run(ctx context.Context, cfg ServiceConfig) error {
	if shared.HasNonEmptyTrimmed(cfg.ConvertIn) {
		if ArtifactRequested(cfg) {
			return RunHuggingFaceServiceBenchmark(ctx, cfg)
		}
		return ConvertHuggingFaceJSONL(cfg.ConvertIn, cfg.ConvertOut, cfg.ConvertScale)
	}
	return RunServiceBenchmark(ctx, cfg)
}
