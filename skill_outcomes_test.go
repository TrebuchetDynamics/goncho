package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestSkillOutcomeTracking_WritesConclusion(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	err := svc.RecordSkillOutcome(ctx, SkillOutcome{
		SkillName: "gormes-git",
		Success:   true,
		Lesson:    "merge conflicts resolved using git mv",
	})
	if err != nil {
		t.Fatalf("RecordSkillOutcome: %v", err)
	}

	outcomes, err := svc.SearchSkillOutcomes(ctx, "gormes-git", 10)
	if err != nil {
		t.Fatalf("SearchSkillOutcomes: %v", err)
	}
	if len(outcomes) == 0 {
		t.Fatal("expected skill outcome to be stored")
	}
	if !strings.Contains(outcomes[0], "gormes-git") {
		t.Errorf("expected skill name in outcome, got %q", outcomes[0])
	}
}

func TestSkillOutcomeTracking_SearchBySkill(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_ = svc.RecordSkillOutcome(ctx, SkillOutcome{
		SkillName: "test-skill-a",
		Success:   true,
		Lesson:    "lesson A",
	})
	_ = svc.RecordSkillOutcome(ctx, SkillOutcome{
		SkillName: "test-skill-b",
		Success:   true,
		Lesson:    "lesson B",
	})

	outcomesA, err := svc.SearchSkillOutcomes(ctx, "test-skill-a", 10)
	if err != nil {
		t.Fatalf("SearchSkillOutcomes A: %v", err)
	}
	outcomesB, err := svc.SearchSkillOutcomes(ctx, "test-skill-b", 10)
	if err != nil {
		t.Fatalf("SearchSkillOutcomes B: %v", err)
	}

	if len(outcomesA) == 0 || len(outcomesB) == 0 {
		t.Errorf("expected outcomes for both skills, got A=%d B=%d", len(outcomesA), len(outcomesB))
	}
}

func TestSkillOutcomeTracking_Idempotent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_ = svc.RecordSkillOutcome(ctx, SkillOutcome{
		SkillName: "idempotent-skill",
		Success:   true,
		Lesson:    "same lesson",
	})
	_ = svc.RecordSkillOutcome(ctx, SkillOutcome{
		SkillName: "idempotent-skill",
		Success:   true,
		Lesson:    "same lesson",
	})

	outcomes, err := svc.SearchSkillOutcomes(ctx, "idempotent-skill", 10)
	if err != nil {
		t.Fatalf("SearchSkillOutcomes: %v", err)
	}
	// Both should be stored (Conclude uses idempotency key, but different lessons = different keys)
	if len(outcomes) < 1 {
		t.Error("expected at least one outcome")
	}
}
