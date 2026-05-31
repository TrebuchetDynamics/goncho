package leakagecheck

import (
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestCheckClassifiesBlockingAndReportedLeakage(t *testing.T) {
	checks := Check([]goncho.RecallBenchmarkServiceCase{{
		ID:           "q1",
		Query:        "Where is the launch key hidden?",
		IdealAnswer:  "inside the blue locker",
		RelevantRefs: []string{"mem-42"},
		Rubric:       []string{"mention the blue locker"},
		Memories: []goncho.RecallBenchmarkServiceMemory{{
			Ref:        "m1",
			Conclusion: "mem-42 says: Where is the launch key hidden? mention the blue locker",
		}},
	}})

	if checks.QuestionTextInMemory != 1 || checks.RelevantIDInMemory != 1 || checks.RubricTextInMemory != 1 {
		t.Fatalf("blocking counts = question:%d relevant:%d rubric:%d, want 1/1/1", checks.QuestionTextInMemory, checks.RelevantIDInMemory, checks.RubricTextInMemory)
	}
	if !HasBlocking(checks) {
		t.Fatalf("HasBlocking() = false, want true")
	}
	if got, want := len(checks.Examples), 3; got != want {
		t.Fatalf("examples len = %d, want %d", got, want)
	}
}

func TestContainsTextRequiresSubstantiveNeedle(t *testing.T) {
	if ContainsText("alpha beta", "alpha") {
		t.Fatalf("ContainsText short needle = true, want false")
	}
	if !ContainsText("Alpha Beta Gamma", "beta gam") {
		t.Fatalf("ContainsText case-insensitive substantive needle = false, want true")
	}
}
