package results

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/resultscontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

type Config struct {
	ResultsIn       string
	ResultsOut      string
	ResultsConfigID string
}

func AppendOutcomesFromResults(cfg Config) error {
	inputPath := strings.TrimSpace(cfg.ResultsIn)
	outPath := strings.TrimSpace(cfg.ResultsOut)
	if inputPath == "" {
		return fmt.Errorf("goncho-bench: --beam-paired-results-in is required")
	}
	if outPath == "" {
		return fmt.Errorf("goncho-bench: --beam-paired-results-out is required for --beam-paired-results-in")
	}
	raw, sourceSHA256, err := shared.ReadFileWithChecksum(inputPath, "goncho-bench: read BEAM paired results")
	if err != nil {
		return err
	}
	var results resultscontract.File
	if err := json.Unmarshal(raw, &results); err != nil {
		return fmt.Errorf("goncho-bench: decode BEAM paired results: %w", err)
	}
	rows, err := resultscontract.OutcomesFromResults(results, cfg.ResultsConfigID, inputPath, sourceSHA256)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("goncho-bench: BEAM paired results contained no question results")
	}
	return shared.AppendJSONLFileWithParents(outPath, "goncho-bench: create BEAM paired results output dir", "goncho-bench: open BEAM paired results output", "goncho-bench: write BEAM paired result row", rows)
}
