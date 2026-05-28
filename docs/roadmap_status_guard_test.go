package docs_test

import (
	"os"
	"strings"
	"testing"
)

// TestTodoCurrentStateDoesNotContradictDeliveredChecklist ensures the TODO
// "still lacks" section does not list items that are already delivered.
//
// If a capability exists in the module (even if only through a specific
// entry point like cmd/goncho or cmd/goncho-server), it should not appear
// in the "still lacks" block — doing so makes future implementation agents
// plan from stale evidence.
//
// Follows the same principle as agentmemory's consistency.test.ts:
// doc claims are checked against actual code state.
func TestTodoCurrentStateDoesNotContradictDeliveredChecklist(t *testing.T) {
	raw, err := os.ReadFile("../TODO.md")
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	todo := string(raw)

	// Scan for the "still lacks" marker. Everything between it and the next
	// ##-level section heading is the current "still lacks" block.
	// ###-level sub-sections (e.g., "Delivered but may need UX polish")
	// are excluded — they document polish gaps for already-delivered items.
	lacksMarker := "What Goncho still lacks versus agentmemory:"
	lacksIdx := strings.Index(todo, lacksMarker)
	if lacksIdx < 0 {
		t.Fatal("TODO.md missing section: What Goncho still lacks versus agentmemory")
	}

	lacksBlock := todo[lacksIdx+len(lacksMarker):]
	if idx := strings.Index(lacksBlock, "\n## "); idx >= 0 {
		lacksBlock = lacksBlock[:idx]
	}
	if idx := strings.Index(lacksBlock, "\n### "); idx >= 0 {
		lacksBlock = lacksBlock[:idx]
	}
	lacksBlock = strings.ToLower(lacksBlock)

	// Each of these capabilities IS delivered in the module. If any appears
	// in the "still lacks" block, the TODO is stale and must be updated.
	//
	// "Delivered" means the capability exists somewhere in the module — in
	// service/, cmd/goncho, cmd/goncho-server, http/, etc. It does not mean
	// every possible UX polish is done.
	//
	// Naming matches the exact phrasing in the TODO.md bullet items so the
	// test error message is immediately readable.
	delivered := []struct {
		marker string // Lowercase substring to match in the lacks block
		proof  string // Where it exists (for the error message)
	}{
		// cmd/goncho has doctor, version, upgrade-check as top-level commands.
		// The TODO claims "top-level goncho doctor, version --json, and
		// upgrade-check polish beyond goncho-server doctor" is lacking.
		// doctor:      cmd/goncho/main.go runDoctor — handles --db, --config, --json
		// version:     cmd/goncho/main.go runVersion — handles --json
		// upgrade-check: cmd/goncho/main.go runUpgradeCheck — handles --current, --latest, --json
		{marker: "top-level goncho doctor", proof: "cmd/goncho/main.go — `doctor --json --db <path>`"},
		{marker: "version --json", proof: "cmd/goncho/main.go — `version --json`"},
		{marker: "upgrade-check", proof: "cmd/goncho/main.go — `upgrade-check --json --current v0.3.0 --latest v0.3.1`"},

		// service.MemoryFacade with stable caller IDs, metadata filters,
		// and history/provenance over memory slots + observations exists.
		// The TODO claims "Mem0-simple API facade" is lacking.
		{marker: "mem0-simple api facade", proof: "service/memory_facade.go — MemoryFacade with Add/Search/Update/Delete/History"},

		// cmd/goncho preferences/connect/remove provide onboarding/remove/
		// preference UX. The TODO claims these are lacking.
		{marker: "onboarding/remove/preference ux", proof: "cmd/goncho/main.go — preferences/connect/remove commands"},

		// Portable JSONL + Markdown export/import exists. The TODO claims
		// "portable export formats" is lacking.
		{marker: "portable export formats", proof: "service/portable_export.go + service/portable_import.go — JSONL/Markdown export/import"},

		// Optional-provider resilience diagnostics with circuit-breaker
		// state, timeouts, payload guards exist. The TODO claims "Provider
		// resilience" is lacking.
		{marker: "provider resilience", proof: "service/provider_resilience.go — circuit-breaker state, fallback diagnostics"},

		// Retention/disk budget preview and archive apply path exists. The
		// TODO claims "disk-budget retention" is lacking.
		{marker: "disk-budget retention", proof: "service/retention.go — disk budget preview + archive apply path"},

		// Eval registry, self-correction candidates, regression gate helpers
		// exist. The TODO claims "eval feedback loops" is lacking.
		{marker: "eval feedback loops", proof: "service/eval_registry.go — eval candidates, feedback labels, regression gates"},
	}

	var contradictions []string
	for _, d := range delivered {
		if strings.Contains(lacksBlock, d.marker) {
			contradictions = append(contradictions,
				"TODO still-lacks claims "+d.marker+" is missing, but it is delivered in "+d.proof)
		}
	}

	if len(contradictions) > 0 {
		t.Errorf("TODO.md contains %d contradictions between 'still lacks' and delivered code:\n\n%s\n\n"+
			"Remove these from the 'still lacks' block or mark them as polish gaps in a sub-bullet.",
			len(contradictions),
			strings.Join(contradictions, "\n"))
	}
}