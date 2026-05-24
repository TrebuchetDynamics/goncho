package goncho

// DefaultRecallBenchmarkServiceCases returns the local BEAM-style recall oracle
// for the MEMORIA categories Goncho currently implements deterministically.
// The cases intentionally use public Service.Conclude ingestion and benchmark
// refs instead of storage IDs, answer hints, LLM judges, or external datasets.
func DefaultRecallBenchmarkServiceCases() []RecallBenchmarkServiceCase {
	return []RecallBenchmarkServiceCase{
		{
			ID:                    "beam-ie-owner-fact",
			Ability:               "IE",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-ie",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "owner", Conclusion: "Project note: Owner of LedgerDB is Mira."}, {Ref: "decoy", Conclusion: "Who owns LedgerDB? owns LedgerDB owns LedgerDB owns LedgerDB. This checklist repeats the retrieval words but names no owner."}},
			Query:                 "Who owns LedgerDB?",
			RelevantRefs:          []string{"owner"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-ie-v1"),
		},
		{
			ID:         "beam-mr-owner-through-storage",
			Ability:    "MR",
			Peer:       "team",
			SessionKey: "sess-beam-default-mr",
			Memories: []RecallBenchmarkServiceMemory{
				{Ref: "uses", Conclusion: "Project note: Billing API uses LedgerDB."},
				{Ref: "owner", Conclusion: "Project note: Owner of LedgerDB is Mira."},
				{Ref: "decoy", Conclusion: "Who is responsible for storage used by Billing API? responsible storage used Billing API responsible storage used Billing API. This checklist repeats the retrieval words but names no owner."},
			},
			Query:                 "Who is responsible for storage used by Billing API?",
			RelevantRefs:          []string{"owner"},
			RequiredEvidenceKinds: []string{"graph"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkGraphScoringConfig("beam-default-mr-v1"),
		},
		{
			ID:                    "beam-tr-deadline",
			Ability:               "TR",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-tr",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "deadline", Conclusion: "Project note: Release Orion deadline is 2026-06-01."}, {Ref: "decoy", Conclusion: "When is Release Orion? Release Orion when Release Orion when Release Orion when. This checklist repeats the retrieval words but does not state the date."}},
			Query:                 "When is Release Orion?",
			RelevantRefs:          []string{"deadline"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-tr-v1"),
		},
		{
			ID:                    "beam-pf-preference",
			Ability:               "PF",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-pf",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "preference", Conclusion: "Project note: Mira's indentation preference is tabs."}, {Ref: "decoy", Conclusion: "What indentation does Mira prefer? indentation prefer indentation prefer indentation prefer. This checklist repeats the retrieval words but does not answer it."}},
			Query:                 "What indentation does Mira prefer?",
			RelevantRefs:          []string{"preference"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-pf-v1"),
		},
		{
			ID:                    "beam-if-instruction",
			Ability:               "IF",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-if",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "instruction", Conclusion: "Project note: Mira's instruction is never delete logs."}, {Ref: "decoy", Conclusion: "What instruction did Mira give about logs? instruction logs instruction logs instruction logs instruction logs. This checklist repeats the retrieval words but does not state the rule."}},
			Query:                 "What instruction did Mira give about logs?",
			RelevantRefs:          []string{"instruction"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-if-v1"),
		},
		{
			ID:                    "beam-eo-release-sequence",
			Ability:               "EO",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-eo",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "sequence", Conclusion: "Project note: Release rollout sequence: first freeze writes, then run migration, finally enable readers."}, {Ref: "decoy", Conclusion: "Walk me through the release rollout sequence? release rollout sequence release rollout sequence release rollout sequence. This checklist repeats the retrieval words but does not state the order."}},
			Query:                 "Walk me through the release rollout sequence.",
			RelevantRefs:          []string{"sequence"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-eo-v1"),
		},
		{
			ID:                    "beam-cr-negation",
			Ability:               "CR",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-cr",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "denial", Conclusion: "Project note: I never approved auto-deleting audit logs."}, {Ref: "decoy", Conclusion: "Have I approved auto-deleting audit logs? approved auto-deleting audit logs approved auto-deleting audit logs. This checklist repeats the retrieval words but does not state the denial."}},
			Query:                 "Have I approved auto-deleting audit logs?",
			RelevantRefs:          []string{"denial"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-cr-v1"),
		},
		{
			ID:                    "beam-ku-version",
			Ability:               "KU",
			Peer:                  "team",
			SessionKey:            "sess-beam-default-ku",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "version", Conclusion: "Project note: PostgreSQL version is 14.2."}, {Ref: "decoy", Conclusion: "What PostgreSQL version? PostgreSQL version PostgreSQL version PostgreSQL version. This checklist repeats the retrieval words but does not state the version."}},
			Query:                 "What PostgreSQL version?",
			RelevantRefs:          []string{"version"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         recallBenchmarkFactScoringConfig("beam-default-ku-v1"),
		},
	}
}

func recallBenchmarkFactScoringConfig(version string) RecallScoringConfig {
	return RecallScoringConfig{
		Version:     version,
		Weights:     map[string]float64{"keyword": 0.10, "fact": 0.75, "graph": 0.05, "scope": 0.10},
		RRFK:        60,
		MMRLambda:   1,
		TokenBudget: 320,
	}
}

func recallBenchmarkGraphScoringConfig(version string) RecallScoringConfig {
	return RecallScoringConfig{
		Version:     version,
		Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
		RRFK:        60,
		MMRLambda:   1,
		TokenBudget: 320,
	}
}
