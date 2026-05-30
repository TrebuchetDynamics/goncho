package oracle

import (
	"context"
	"strings"
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
	return trimNonEmpty(cfg.ServiceOut) || trimNonEmpty(cfg.ServiceResultsOut) || trimNonEmpty(cfg.ServiceSummaryOut) || trimNonEmpty(cfg.ServicePairedOut) || trimNonEmpty(cfg.ServiceFailuresOut) || trimNonEmpty(cfg.ServiceJudgeRequestsOut)
}

func trimNonEmpty(value string) bool { return stringsTrimSpace(value) != "" }

var stringsTrimSpace = strings.TrimSpace

func Run(ctx context.Context, cfg ServiceConfig) error {
	if stringsTrimSpace(cfg.ConvertIn) != "" {
		if ArtifactRequested(cfg) {
			return RunHuggingFaceServiceBenchmark(ctx, cfg)
		}
		return ConvertHuggingFaceJSONL(cfg.ConvertIn, cfg.ConvertOut, cfg.ConvertScale)
	}
	return RunServiceBenchmark(ctx, cfg)
}
