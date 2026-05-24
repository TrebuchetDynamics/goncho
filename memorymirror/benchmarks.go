package memorymirror

type BenchmarkTargetSet struct {
	LongMemEval  LongMemEvalTarget  `json:"longmemeval_s"`
	TokenSavings TokenSavingsTarget `json:"token_savings"`
}

type LongMemEvalTarget struct {
	Dataset                 string  `json:"dataset"`
	QuestionCount           int     `json:"question_count"`
	EmbeddingModel          string  `json:"embedding_model"`
	ReferenceRecallAnyAt5   float64 `json:"reference_recall_any_at_5"`
	ReferenceRecallAnyAt10  float64 `json:"reference_recall_any_at_10"`
	ReferenceMRR            float64 `json:"reference_mrr"`
	SimilarRecallAnyAt10Gap float64 `json:"similar_recall_any_at_10_gap"`
}

type TokenSavingsTarget struct {
	PasteFullContextTokensPerYear int     `json:"paste_full_context_tokens_per_year"`
	SummarizedTokensPerYear       int     `json:"summarized_tokens_per_year"`
	TargetTokensPerYear           int     `json:"target_tokens_per_year"`
	TargetCostUSDPerYear          float64 `json:"target_cost_usd_per_year"`
	LocalEmbeddingCostUSDPerYear  float64 `json:"local_embedding_cost_usd_per_year"`
	EmbeddingModel                string  `json:"embedding_model"`
}

type LongMemEvalEvidence struct {
	System        string  `json:"system"`
	Dataset       string  `json:"dataset"`
	QuestionCount int     `json:"question_count"`
	RecallAnyAt5  float64 `json:"recall_any_at_5"`
	RecallAnyAt10 float64 `json:"recall_any_at_10"`
	MRR           float64 `json:"mrr"`
}

type LongMemEvalAssessment struct {
	MeetsSimilarGate   bool    `json:"meets_similar_gate"`
	RecallAnyAt5Delta  float64 `json:"recall_any_at_5_delta"`
	RecallAnyAt10Delta float64 `json:"recall_any_at_10_delta"`
	MRRDelta           float64 `json:"mrr_delta"`
	Reason             string  `json:"reason,omitempty"`
}

func BenchmarkTargets() BenchmarkTargetSet {
	return BenchmarkTargetSet{
		LongMemEval: LongMemEvalTarget{
			Dataset:                 "LongMemEval-S",
			QuestionCount:           500,
			EmbeddingModel:          "all-MiniLM-L6-v2",
			ReferenceRecallAnyAt5:   0.952,
			ReferenceRecallAnyAt10:  0.986,
			ReferenceMRR:            0.882,
			SimilarRecallAnyAt10Gap: 0.01,
		},
		TokenSavings: TokenSavingsTarget{
			PasteFullContextTokensPerYear: 19_500_000,
			SummarizedTokensPerYear:       650_000,
			TargetTokensPerYear:           170_000,
			TargetCostUSDPerYear:          10,
			LocalEmbeddingCostUSDPerYear:  0,
			EmbeddingModel:                "all-MiniLM-L6-v2",
		},
	}
}

func AssessLongMemEval(evidence LongMemEvalEvidence, target LongMemEvalTarget) LongMemEvalAssessment {
	assessment := LongMemEvalAssessment{
		RecallAnyAt5Delta:  evidence.RecallAnyAt5 - target.ReferenceRecallAnyAt5,
		RecallAnyAt10Delta: evidence.RecallAnyAt10 - target.ReferenceRecallAnyAt10,
		MRRDelta:           evidence.MRR - target.ReferenceMRR,
	}
	if target.QuestionCount > 0 && evidence.QuestionCount != target.QuestionCount {
		assessment.Reason = "question count mismatch"
		return assessment
	}
	if assessment.RecallAnyAt5Delta < 0 {
		assessment.Reason = "recall_any@5 below reference"
		return assessment
	}
	if assessment.MRRDelta < 0 {
		assessment.Reason = "MRR below reference"
		return assessment
	}
	if assessment.RecallAnyAt10Delta < -target.SimilarRecallAnyAt10Gap {
		assessment.Reason = "recall_any@10 outside similarity gap"
		return assessment
	}
	assessment.MeetsSimilarGate = true
	return assessment
}
