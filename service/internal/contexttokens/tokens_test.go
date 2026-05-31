package contexttokens

import "testing"

func TestEffectiveContextLimitKeepsLegacyTokensPrecedence(t *testing.T) {
	if got := EffectiveContextLimit(40, 100); got != 40 {
		t.Fatalf("EffectiveContextLimit = %d, want deprecated tokens precedence", got)
	}
	if got := EffectiveContextLimit(0, 100); got != 100 {
		t.Fatalf("EffectiveContextLimit fallback = %d, want max tokens", got)
	}
}

func TestEffectiveSearchLimitKeepsLegacyMaxTokensPrecedence(t *testing.T) {
	if got := EffectiveSearchLimit(40, 100); got != 100 {
		t.Fatalf("EffectiveSearchLimit = %d, want max tokens precedence", got)
	}
	if got := EffectiveSearchLimit(40, 0); got != 40 {
		t.Fatalf("EffectiveSearchLimit fallback = %d, want tokens", got)
	}
}

func TestSplitSummaryMessageBudget(t *testing.T) {
	summary, messages := SplitSummaryMessageBudget(101)
	if summary != 40 || messages != 61 {
		t.Fatalf("SplitSummaryMessageBudget = (%d, %d), want (40, 61)", summary, messages)
	}
	summary, messages = SplitSummaryMessageBudget(0)
	if summary != 0 || messages != 0 {
		t.Fatalf("SplitSummaryMessageBudget zero = (%d, %d), want (0, 0)", summary, messages)
	}
}
