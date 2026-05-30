package paired

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const beamPairedResultsDefaultConfigID = "beam-results"

type beamPairedResultsFile struct {
	Metadata struct {
		ConfigID     string `json:"config_id"`
		RunStartedAt string `json:"run_started_at"`
		Date         string `json:"date"`
	} `json:"metadata"`
	Results []struct {
		ConversationID string `json:"conversation_id"`
		Scale          string `json:"scale"`
		Results        []struct {
			QID      string  `json:"qid"`
			Ability  string  `json:"ability"`
			Question string  `json:"question"`
			Score    float64 `json:"score"`
		} `json:"results"`
	} `json:"results"`
}

func AppendPairedOutcomesFromResults(cfg Config) error {
	inputPath := strings.TrimSpace(cfg.ResultsIn)
	outPath := strings.TrimSpace(cfg.ResultsOut)
	if inputPath == "" {
		return fmt.Errorf("goncho-bench: --beam-paired-results-in is required")
	}
	if outPath == "" {
		return fmt.Errorf("goncho-bench: --beam-paired-results-out is required for --beam-paired-results-in")
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("goncho-bench: read BEAM paired results: %w", err)
	}
	var results beamPairedResultsFile
	if err := json.Unmarshal(raw, &results); err != nil {
		return fmt.Errorf("goncho-bench: decode BEAM paired results: %w", err)
	}
	rows, err := beamPairedOutcomesFromResults(results, cfg.ResultsConfigID, inputPath, checksumBytesSHA256(raw))
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("goncho-bench: BEAM paired results contained no question results")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM paired results output dir: %w", err)
	}
	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("goncho-bench: open BEAM paired results output: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("goncho-bench: write BEAM paired result row: %w", err)
		}
	}
	return nil
}

func beamPairedOutcomesFromResults(results beamPairedResultsFile, overrideConfigID, sourcePath, sourceSHA256 string) ([]servicePairedOutcome, error) {
	configID := strings.TrimSpace(overrideConfigID)
	if configID == "" {
		configID = strings.TrimSpace(results.Metadata.ConfigID)
	}
	if configID == "" {
		configID = beamPairedResultsDefaultConfigID
	}
	runStartedAt := strings.TrimSpace(results.Metadata.RunStartedAt)
	if runStartedAt == "" {
		runStartedAt = strings.TrimSpace(results.Metadata.Date)
	}
	if runStartedAt == "" {
		runStartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	out := []servicePairedOutcome{}
	for conversationIndex, conv := range results.Results {
		conversationID := strings.TrimSpace(conv.ConversationID)
		scale := strings.TrimSpace(conv.Scale)
		for resultIndex, result := range conv.Results {
			qid := strings.TrimSpace(result.QID)
			if qid == "" {
				return nil, fmt.Errorf("goncho-bench: BEAM paired result conversation %d result %d missing qid", conversationIndex+1, resultIndex+1)
			}
			score := roundMetric(result.Score)
			out = append(out, servicePairedOutcome{
				ConfigID:       configID,
				RunStartedAt:   runStartedAt,
				Scale:          scale,
				ConversationID: conversationID,
				QID:            qid,
				Ability:        strings.ToUpper(strings.TrimSpace(result.Ability)),
				Question:       strings.TrimSpace(result.Question),
				SourcePath:     strings.TrimSpace(sourcePath),
				SourceSHA256:   strings.TrimSpace(sourceSHA256),
				Score:          score,
				Correct:        score >= 0.5,
			})
		}
	}
	return out, nil
}
