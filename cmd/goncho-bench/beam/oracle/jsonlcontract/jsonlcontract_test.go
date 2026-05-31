package jsonlcontract

import "testing"

func TestNormalizeConversationIDDefaultsBlank(t *testing.T) {
	if got := NormalizeConversationID("  "); got != "goncho-service-memoria-fixtures" {
		t.Fatalf("NormalizeConversationID blank = %q", got)
	}
	if got := NormalizeConversationID(" conv "); got != "conv" {
		t.Fatalf("NormalizeConversationID explicit = %q", got)
	}
}

func TestScoringConfigWeightsGraphRequiredEvidence(t *testing.T) {
	cfg := ScoringConfig(Question{Ability: "Graph", RequiredEvidenceKinds: []string{" graph "}})
	if cfg.Version != "beam-jsonl-graph-v1" {
		t.Fatalf("version = %q", cfg.Version)
	}
	if cfg.Weights["graph"] != 0.80 || cfg.Weights["fact"] != 0.10 {
		t.Fatalf("graph weights = %#v", cfg.Weights)
	}
}

func TestScoringConfigWeightsFactDefault(t *testing.T) {
	cfg := ScoringConfig(Question{Ability: "Fact"})
	if cfg.Weights["fact"] != 0.75 || cfg.Weights["graph"] != 0.05 {
		t.Fatalf("default weights = %#v", cfg.Weights)
	}
}
