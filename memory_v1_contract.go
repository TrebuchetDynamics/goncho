package goncho

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"


)

type MemoryV1ToolContractInfo struct {
	ContractVersion                string                      `json:"contract_version"`
	PrivateAgentMemoryDefault      bool                        `json:"private_agent_memory_default"`
	SelfImprovementPerAgentDefault bool                        `json:"self_improvement_per_agent_default"`
	PurgePolicy                    string                      `json:"purge_policy"`
	Tools                          map[string]MemoryV1ToolSpec `json:"tools"`
}

type MemoryV1ToolSpec struct {
	Name                  string `json:"name"`
	Mutating              bool   `json:"mutating"`
	Idempotent            bool   `json:"idempotent"`
	RequiresStableID      bool   `json:"requires_stable_id"`
	RequiresProvenance    bool   `json:"requires_provenance"`
	CreatesRevision       bool   `json:"creates_revision"`
	DeleteSemantics       string `json:"delete_semantics,omitempty"`
	ResultContractVersion string `json:"result_contract_version"`
}

type MemoryV1ToolTranscriptEntry struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Result    map[string]any `json:"result"`
}

func MemoryV1ToolContract() MemoryV1ToolContractInfo {
	return MemoryV1ToolContractInfo{
		ContractVersion:                memory.GonchoMemoryV1ContractVersion,
		PrivateAgentMemoryDefault:      true,
		SelfImprovementPerAgentDefault: true,
		PurgePolicy:                    "explicit_operator_only",
		Tools: map[string]MemoryV1ToolSpec{
			"store_memory": {
				Name:                  "store_memory",
				Mutating:              true,
				Idempotent:            false,
				RequiresStableID:      true,
				RequiresProvenance:    true,
				CreatesRevision:       false,
				ResultContractVersion: memory.GonchoMemoryV1ContractVersion,
			},
			"retrieve_memory": {
				Name:                  "retrieve_memory",
				Mutating:              false,
				Idempotent:            true,
				RequiresStableID:      false,
				RequiresProvenance:    false,
				CreatesRevision:       false,
				ResultContractVersion: memory.GonchoMemoryV1ContractVersion,
			},
			"update_memory": {
				Name:                  "update_memory",
				Mutating:              true,
				Idempotent:            false,
				RequiresStableID:      true,
				RequiresProvenance:    true,
				CreatesRevision:       true,
				ResultContractVersion: memory.GonchoMemoryV1ContractVersion,
			},
			"summarize_memories": {
				Name:                  "summarize_memories",
				Mutating:              true,
				Idempotent:            false,
				RequiresStableID:      false,
				RequiresProvenance:    true,
				CreatesRevision:       false,
				ResultContractVersion: memory.GonchoMemoryV1ContractVersion,
			},
			"forget_memory": {
				Name:                  "forget_memory",
				Mutating:              true,
				Idempotent:            true,
				RequiresStableID:      true,
				RequiresProvenance:    true,
				CreatesRevision:       true,
				DeleteSemantics:       "soft_tombstone",
				ResultContractVersion: memory.GonchoMemoryV1ContractVersion,
			},
		},
	}
}

func DecodeMemoryV1ToolTranscript(body []byte) ([]MemoryV1ToolTranscriptEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	var out []MemoryV1ToolTranscriptEntry
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry MemoryV1ToolTranscriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("goncho: parse memory v1 tool transcript line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(entry.Tool) == "" {
			return nil, fmt.Errorf("goncho: memory v1 tool transcript line %d missing tool", lineNo)
		}
		out = append(out, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho: scan memory v1 tool transcript: %w", err)
	}
	return out, nil
}
