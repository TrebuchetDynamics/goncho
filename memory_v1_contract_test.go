package goncho

import (
	"os"
	"path/filepath"
	"testing"


)

func TestMemoryV1ToolContractDocumentsDurableToolSemantics(t *testing.T) {
	contract := MemoryV1ToolContract()

	if contract.ContractVersion != "1" {
		t.Fatalf("ContractVersion = %q, want 1", contract.ContractVersion)
	}
	if contract.PurgePolicy != "explicit_operator_only" {
		t.Fatalf("PurgePolicy = %q, want explicit_operator_only", contract.PurgePolicy)
	}
	for _, want := range []string{"store_memory", "retrieve_memory", "update_memory", "summarize_memories", "forget_memory"} {
		spec, ok := contract.Tools[want]
		if !ok {
			t.Fatalf("tool contract missing %s: %#v", want, contract.Tools)
		}
		if spec.Name != want || spec.ResultContractVersion != "1" {
			t.Fatalf("tool spec for %s = %+v", want, spec)
		}
	}
	if got := contract.Tools["forget_memory"].DeleteSemantics; got != "soft_tombstone" {
		t.Fatalf("forget_memory DeleteSemantics = %q, want soft_tombstone", got)
	}
	if !contract.Tools["store_memory"].RequiresProvenance || !contract.Tools["update_memory"].CreatesRevision {
		t.Fatalf("store/update specs = %+v / %+v, want provenance + revisions", contract.Tools["store_memory"], contract.Tools["update_memory"])
	}
}

func TestMemoryV1ContractFixtureMatchesGonchoToolContract(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "memory", "testdata", "goncho_v1", "tool_transcript.jsonl"))
	if err != nil {
		t.Fatalf("read tool transcript fixture: %v", err)
	}
	entries, err := DecodeMemoryV1ToolTranscript(body)
	if err != nil {
		t.Fatalf("DecodeMemoryV1ToolTranscript: %v", err)
	}
	contract := MemoryV1ToolContract()
	if len(entries) != len(contract.Tools) {
		t.Fatalf("fixture entries = %d, contract tools = %d", len(entries), len(contract.Tools))
	}
	for _, entry := range entries {
		spec, ok := contract.Tools[entry.Tool]
		if !ok {
			t.Fatalf("fixture references non-contract tool %q", entry.Tool)
		}
		if got := entry.Result["contract_version"]; got != contract.ContractVersion {
			t.Fatalf("%s result contract_version = %#v, want %q", entry.Tool, got, contract.ContractVersion)
		}
		if spec.RequiresStableID && entry.Result["id"] == "" && entry.Tool == "store_memory" {
			t.Fatalf("%s result missing stable id: %+v", entry.Tool, entry.Result)
		}
	}
}

func TestMemoryV1AgentIsolationPolicyMatchesMemoryRecallHelper(t *testing.T) {
	doc, err := memory.ParseGonchoMemoryV1Markdown(mustReadFixture(t, filepath.Join("..", "memory", "testdata", "goncho_v1", "memory.md")))
	if err != nil {
		t.Fatalf("ParseGonchoMemoryV1Markdown: %v", err)
	}
	items := map[string]memory.GonchoMemoryV1Item{}
	for _, item := range doc.Items {
		items[item.MemoryID] = item
	}
	contract := MemoryV1ToolContract()
	if !contract.PrivateAgentMemoryDefault || !contract.SelfImprovementPerAgentDefault {
		t.Fatalf("contract isolation defaults = %+v, want private per-agent defaults", contract)
	}

	allowed, reason := memory.CanRecallGonchoMemoryV1(memory.GonchoMemoryV1RecallRequest{
		AgentID:     "agent-b",
		WorkspaceID: "workspace-private",
	}, items["mem_agent_a_project"])
	if allowed || reason != "private_agent_boundary" {
		t.Fatalf("cross-agent private recall allowed=%t reason=%q", allowed, reason)
	}
}

func mustReadFixture(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return body
}
