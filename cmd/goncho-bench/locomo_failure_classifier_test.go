package main

import "testing"

func TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets(t *testing.T) {
	cases := []struct {
		name                  string
		question              locomoQuestionResult
		memoryConversationIDs map[string]string
		want                  string
	}{
		{
			name: "wrong branch retrieval",
			question: locomoQuestionResult{
				QuestionID:     "locomo-conv-7-q-003",
				ConversationID: "locomo-conv-7",
				GoldMemoryIDs:  []string{"locomo-conv-7-m-011"},
				RetrievedIDs:   []string{"locomo-conv-8-m-002", "locomo-conv-8-m-004"},
				Rank:           0,
				Category:       "missing_candidate",
			},
			memoryConversationIDs: map[string]string{
				"locomo-conv-7-m-011": "locomo-conv-7",
				"locomo-conv-8-m-002": "locomo-conv-8",
				"locomo-conv-8-m-004": "locomo-conv-8",
			},
			want: "wrong_branch_retrieval",
		},
		{
			name: "missing companion memories",
			question: locomoQuestionResult{
				QuestionID:       "locomo-conv-9-q-014",
				ConversationID:   "locomo-conv-9",
				GoldMemoryIDs:    []string{"locomo-conv-9-m-010", "locomo-conv-9-m-027"},
				RetrievedIDs:     []string{"locomo-conv-9-m-010", "locomo-conv-9-m-040"},
				Rank:             1,
				RecallAnyAt10:    1,
				StrictRecallAt10: 0,
				Category:         "multi_hop_retrieval",
			},
			memoryConversationIDs: map[string]string{
				"locomo-conv-9-m-010": "locomo-conv-9",
				"locomo-conv-9-m-027": "locomo-conv-9",
				"locomo-conv-9-m-040": "locomo-conv-9",
			},
			want: "missing_companion_memory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyLocomoFailureBucket(tc.question, tc.memoryConversationIDs); got != tc.want {
				t.Fatalf("classifyLocomoFailureBucket() = %q, want %q", got, tc.want)
			}
		})
	}
}
