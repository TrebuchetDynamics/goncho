package contexttokens

// EffectiveContextLimit returns the context tool token budget, preserving the
// legacy precedence where the deprecated Tokens field wins over MaxTokens.
func EffectiveContextLimit(tokens, maxTokens int) int {
	if tokens > 0 {
		return tokens
	}
	return maxTokens
}

// EffectiveSearchLimit returns the search token budget embedded in a context
// request, preserving the legacy precedence where MaxTokens wins over Tokens.
func EffectiveSearchLimit(tokens, maxTokens int) int {
	if maxTokens > 0 {
		return maxTokens
	}
	return tokens
}

// SplitSummaryMessageBudget reserves 40% of a positive context token budget for
// summaries and leaves the remainder for recent messages.
func SplitSummaryMessageBudget(tokenLimit int) (summaryBudget, messageBudget int) {
	if tokenLimit <= 0 {
		return 0, 0
	}
	summaryBudget = int(float64(tokenLimit) * 0.4)
	messageBudget = tokenLimit - summaryBudget
	return summaryBudget, messageBudget
}
