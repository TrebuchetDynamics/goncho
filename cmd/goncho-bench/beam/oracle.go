package beam

import (
	"context"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle"
)

type ServiceConfig = oracle.ServiceConfig

func ArtifactRequested(cfg ServiceConfig) bool {
	return oracle.ArtifactRequested(cfg)
}

func Run(ctx context.Context, cfg ServiceConfig) error {
	return oracle.Run(ctx, cfg)
}

func RunServiceBenchmark(ctx context.Context, cfg ServiceConfig) error {
	return oracle.RunServiceBenchmark(ctx, cfg)
}

func RunHuggingFaceServiceBenchmark(ctx context.Context, cfg ServiceConfig) error {
	return oracle.RunHuggingFaceServiceBenchmark(ctx, cfg)
}

func ConvertHuggingFaceJSONL(inputPath, outputPath, fallbackScale string) error {
	return oracle.ConvertHuggingFaceJSONL(inputPath, outputPath, fallbackScale)
}
