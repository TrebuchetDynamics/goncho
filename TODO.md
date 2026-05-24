# Goncho TODO

## Release state

- 2026-05-24: raw BEAM service artifacts now report conversion diagnostics for unscorable questions.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamHuggingFaceJSONLDatasetReportsUnscorableQuestions -count=1` proves `beam_e2e_results.json.metadata.diagnostics.conversion` names conversation/question counts and warns when a raw HuggingFace-style BEAM question lacks stable `relevant_ids` or `context_contains` for pure-recall scoring.
  - Result: Goncho now distinguishes recall failures from dataset-conversion evidence gaps when running raw BEAM samples, preserving no-answer-hint discipline while making real-BEAM readiness auditable.

- 2026-05-24: `goncho-bench --beam-convert-in` can now feed BEAM service artifacts directly.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamHuggingFaceJSONLDatasetWritesServiceArtifactsDirectly -count=1` proves raw HuggingFace-style BEAM JSONL can be converted in memory, ingested through `Service.Conclude`, and emitted as Mnemosyne-compatible per-question results, summary, and paired outcomes without requiring a manual intermediate converted JSONL file.
  - Result: Goncho now has a one-command raw BEAM sample path from exported dataset record to source-backed local recall artifacts, preserving stable IDs, conversation IDs, scales, ability labels, graph provenance, and no-answer-hint discipline.

- 2026-05-24: `goncho-bench --beam-service-results-out` now emits Mnemosyne-compatible per-question BEAM result artifacts.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamJSONLDatasetWritesMnemosyneCompatibleResultsFile -count=1` proves a converted JSONL case writes `beam_e2e_results.json`-style metadata, grouped conversation results, original question text, pure-recall score, and recall-provenance voice summaries with graph evidence.
  - Result: Goncho now has the third Mnemosyne artifact alongside summary and paired outcomes, making source-backed per-question BEAM diagnostics possible without LLM answerers, judges, answer hints, or external services.

- 2026-05-24: `goncho-bench --beam-convert-in/--beam-convert-out` now converts HuggingFace BEAM JSONL exports into Goncho's service-oracle format.
  - Evidence target: `go test ./cmd/goncho-bench -run TestConvertBeamHuggingFaceJSONLWritesStableIDDataset -count=1` proves nested BEAM chat plus Python-literal `probing_questions` convert into stable memory IDs, conversation-scoped question rows, mapped relevant message indices, required evidence kinds, and ABS expected-no-answer rows without importing ideal answers as retrieval hints.
  - Result: Goncho now has the missing raw-dataset conversion step before `--beam-jsonl`, preserving stable-ID and no-answer-hint discipline while moving closer to real BEAM sample runs.

- 2026-05-24: `goncho-bench --beam-jsonl` now runs external BEAM-style JSONL conversions through the service oracle.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamJSONLDatasetWritesMnemosyneCompatibleArtifacts -count=1` proves a converted JSONL file with stable memory IDs, conversation IDs, scale, ability, relevant IDs, and required graph evidence is ingested through `Service.Conclude`, recalled through the service oracle, and exported as Mnemosyne-compatible summary and paired-outcome artifacts.
  - Result: Goncho now has a dataset Adapter seam for real BEAM conversions instead of only built-in deterministic fixtures, while preserving no-answer-hint, no-LLM-judge, stable-ID recall discipline.

- 2026-05-24: service-backed BEAM oracle now covers ABS and SUM fixtures plus expected-no-answer scoring.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestRunBeamServiceRecallOracleWrites(AbilityReport|MnemosyneCompatibleArtifacts)' -count=1` proves the CLI summary and paired outcomes include all ten BEAM abilities (ABS, CR, EO, IE, IF, KU, MR, PF, SUM, TR) with perfect deterministic service-backed recall, context, token-budget, and provenance gates.
  - Result: Goncho can now produce local Mnemosyne-compatible BEAM-style artifacts for every BEAM ability dimension while still avoiding answer hints, LLM judges, external datasets, or final BEAM superiority claims.

- 2026-05-24: `goncho-bench --beam-service-summary-out` and `--beam-service-paired-out` now emit Mnemosyne-compatible BEAM artifacts.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamServiceRecallOracleWritesMnemosyneCompatibleArtifacts -count=1` proves the deterministic service oracle writes `beam_e2e_summary.json`-style ability scores and append-only `paired_outcomes.jsonl` rows with config IDs, scale, qid, ability, score, and correctness.
  - Result: Goncho's delivered MEMORIA fixture oracle can now be compared with Mnemosyne-style BEAM result tooling while still avoiding answer hints, LLM judges, external datasets, or final BEAM superiority claims.

- 2026-05-24: `goncho-bench --beam-service-out` now emits a deterministic BEAM-style MEMORIA recall report.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunBeamServiceRecallOracleWritesAbilityReport -count=1` proves the CLI runs public `Service.Conclude` fixtures for IE, MR, TR, PF, IF, EO, CR, and KU, writes a JSON `RecallBenchmarkReport`, and requires perfect recall/context/provenance rates for each ability.
  - Result: Goncho can generate a local comparison artifact for delivered MEMORIA abilities without answer hints, LLM judges, external datasets, or claiming final BEAM superiority.

- 2026-05-24: BEAM-style recall benchmark oracle now runs public Service.Conclude fixtures end-to-end.
  - Evidence target: `go test . -run TestEvaluateServiceRecallBenchmarkRunsBeamStyleCasesEndToEnd -count=1` proves tiny IE/MR fixtures ingest conclusions through `Service.Conclude`, run recall, map benchmark refs to concrete conclusion IDs, and feed ability/provenance aggregation with fact and graph evidence.
  - Result: Goncho can now track delivered MEMORIA behavior through a deterministic local BEAM-style service oracle without answer hints, LLM judges, external datasets, or claiming final BEAM superiority.

- 2026-05-24: BEAM-style recall benchmark reporting now exposes ability and provenance hit rates.
  - Evidence target: `go test . -run TestRecallBenchmarkReportsBeamAbilityBreakdownAndProvenance -count=1` proves `EvaluateRecallBenchmark` can report IE/MR-style ability slices, per-ability recall@5/@10, and required fact/graph provenance hit rates from existing `RecallTrace` evidence.
  - Result: Goncho now has a deterministic local oracle for tracking MEMORIA/BEAM ability progress without answer hints, LLM judges, retrieval tuning, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support relation-to-sequence, relation-to-decision, and relation-to-negation recall through durable annotations.
  - Evidence target: `go test . -run 'TestRecallExpands(Sequence|Decision|Negation)ThroughDurableKGRelation' -count=1` proves `Service.Conclude` derives durable relation facts plus sequence/decision/negation annotations, then recall selects the target memory for BEAM-style event-order and contradiction-resolution questions with graph provenance citing both annotation rows.
  - Result: annotation-backed graph recall now covers multi-hop EO/CR paths without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support relation-to-preference and relation-to-instruction recall through durable annotations.
  - Evidence target: `go test . -run 'TestRecallExpands(Preference|Instruction)ThroughDurableKGRelation' -count=1` proves `Service.Conclude` derives durable relation facts plus preference/instruction annotations, then recall selects the target memory for BEAM-style `storage used by Billing API` preference/rule questions with graph provenance citing both annotation rows.
  - Result: annotation-backed graph recall now covers multi-hop PF/IF paths without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support relation-to-location recall through durable annotations.
  - Evidence target: `go test . -run TestRecallExpandsLocationThroughDurableKGRelation -count=1` proves `Service.Conclude` derives durable `Billing API uses VectorDB` and `VectorDB is located at us-east-1` facts, then recall selects the location memory for `Where is the storage used by Billing API?` with graph provenance citing both annotation rows.
  - Result: annotation-backed graph recall now covers a BEAM-style dependency/location path without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support relation-to-metric recall through durable annotations.
  - Evidence target: `go test . -run TestRecallExpandsMetricThroughDurableKGRelation -count=1` proves `Service.Conclude` derives durable `Billing API uses VectorDB` and `VectorDB latency is 250ms` facts, then recall selects the metric memory for `How fast is the storage used by Billing API?` with graph provenance citing both annotation rows.
  - Result: annotation-backed graph recall now covers a BEAM-style dependency/metric path without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support owner-to-timeline recall through durable annotations.
  - Evidence target: `go test . -run TestRecallExpandsTimelineThroughOwnerRelation -count=1` proves `Service.Conclude` derives durable `Mira owns Orion` and `Orion occurs on 2026-06-01` facts, then recall selects the timeline memory for `When is the deadline for Mira's owned project?` with graph provenance citing both annotation rows.
  - Result: annotation-backed graph recall now covers a BEAM-style ownership/timeline path without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now support two-hop version recall through durable annotations.
  - Evidence target: `go test . -run TestRecallExpandsVersionThroughMultiHopDurableKGRelation -count=1` proves `Service.Conclude` derives durable `Billing API uses LedgerDB`, `LedgerDB runs on PostgreSQL`, and `PostgreSQL version is 14.2` facts, then recall selects the version memory for `What version is used by Billing API storage?` with graph provenance citing all three annotation rows.
  - Result: annotation-backed graph recall now covers a BEAM-style multi-hop dependency/version path without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA KG relation facts now expand Goncho recall through annotation-backed graph provenance.
  - Evidence target: `go test . -run TestRecallExpandsOwnerThroughDurableKGRelation -count=1` proves `Service.Conclude` derives a durable `Billing API uses LedgerDB` KG fact, joins it to the durable owner fact `Mira owns LedgerDB`, and recall selects the owner memory for `Who is responsible for storage used by Billing API?` with `graph` provenance citing the source and target annotation rows.
  - Result: the MEMORIA annotation lane now supports a deterministic knowledge-graph recall path without changing public Search JSON, adding LLM extraction, using answer hints, or claiming final BEAM superiority.

- 2026-05-23: Agentmemory/MEMORIA citation provenance now follows durable annotation rows into recall.
  - Evidence target: `go test . -run TestRecallCandidatesIncludeDurableFactAnnotationProvenance -count=1` proves `RecallCandidate.Provenance` fact evidence cites the real `goncho_memory_annotations.id`, carries `memory_source`, `memory_id`, extractor `source`, and confidence metadata, and still contributes to recall `fact_score`.
  - Result: durable annotation search ranking now has traceable recall citations, borrowing agentmemory's citation-provenance pattern without changing public Search JSON, adding LLM extraction, or claiming BEAM superiority.

- 2026-05-23: Mnemosyne MEMORIA negation/decision extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run 'TestServiceConclude(NegationAnnotationRanksDurableDenial|DecisionAnnotationRanksDurableDecision)' -count=1` proves `Service.Conclude` derives conservative negation and decision facts from prose (`I never approved auto-deleting audit logs`, `I decided to keep PostgreSQL for audit logs`) and `Service.Search` ranks those durable facts above question-shaped lexical echoes.
  - Result: the append-only fact annotation lane now covers MEMORIA-style contradiction-resolution and decision-recall signals without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA durable annotations now feed RecallCandidate provenance.
  - Evidence target: `go test . -run TestRecallCandidatesIncludeDurableFactAnnotationProvenance -count=1` proves recall candidate generation hydrates stored `goncho_memory_annotations` facts into `RecallCandidate.Provenance`, assigns `fact_score=1`, and selects the durable annotated fact over a lexical echo.
  - Result: the append-only fact annotation lane now supports both public search ranking and recall-trace provenance without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA sequence extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run TestServiceConcludeSequenceAnnotationRanksDurableSequence -count=1` proves `Service.Conclude` derives a conservative event-order fact from prose (`first freeze writes, then run migration, finally enable readers`) and `Service.Search` ranks that durable sequence above a question-shaped lexical echo.
  - Result: the append-only fact annotation lane now covers MEMORIA-style event ordering without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA metric/version extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run 'TestServiceConclude(MetricAnnotationRanksDurableMetric|VersionAnnotationRanksDurableVersion)' -count=1` proves `Service.Conclude` derives conservative numeric metric and semantic-version facts from prose (`dashboard API latency is 250ms`, `PostgreSQL version is 14.2`) and `Service.Search` ranks those durable facts above question-shaped lexical echoes.
  - Result: the append-only fact annotation lane now covers MEMORIA-style IE/KU measurement facts without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA timeline extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run TestServiceConcludeTimelineAnnotationRanksDurableDate -count=1` proves `Service.Conclude` derives a conservative timeline fact from prose (`Release Orion deadline is 2026-06-01`) and `Service.Search` ranks that durable date above a question-shaped lexical echo.
  - Result: the append-only fact annotation lane now covers MEMORIA-style timelines without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA instruction extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run TestServiceConcludeInstructionAnnotationRanksDurableInstruction -count=1` proves `Service.Conclude` derives a conservative instruction fact from prose (`Mira's instruction is never delete logs`) and `Service.Search` ranks that durable instruction above a question-shaped lexical echo.
  - Result: the append-only fact annotation lane now covers MEMORIA-style instructions without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA location extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run TestServiceConcludeLocationAnnotationRanksDurableLocation -count=1` proves `Service.Conclude` derives a conservative location fact from prose (`the escalation runbook location is Notion page RB-17`) and `Service.Search` ranks that durable location above a question-shaped lexical echo.
  - Result: the append-only fact annotation lane now covers owner, preference, and location MEMORIA categories without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA preference extraction now feeds Goncho search through durable annotations.
  - Evidence target: `go test . -run TestServiceConcludePreferenceAnnotationRanksDurablePreference -count=1` proves `Service.Conclude` derives a conservative preference fact from prose (`Mira's indentation preference is tabs`) and `Service.Search` ranks that durable preference above a question-shaped lexical echo.
  - Result: the append-only fact annotation lane now covers a second MEMORIA category (`preferences`) without changing the public Search JSON shape, adding LLM extraction, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA-style append-only fact annotations now feed Goncho search.
  - Evidence target: `go test . -run 'Test(RunMigrationsCreatesMemoryAnnotationTableIdempotently|ServiceConcludeFactAnnotationsRankInverseOwnerFact)' -count=1` proves migrations create the annotation lane idempotently and `Service.Search` ranks an inverse owner fact from a durable annotation above a question-shaped lexical echo.
  - Result: `Service.Conclude` now preserves raw conclusion text while deriving conservative `fact` annotations, then `findConclusions` hydrates those annotations as hidden ranking evidence without changing the public Search JSON shape, using LLM extraction, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA fact-intent search ranking has its first public Goncho slice.
  - Evidence target: `go test . -run 'Test(SearchFactIntentScoresOwnerAnswerButNotLexicalEcho|ServiceSearchFactIntentRanksAnswerOverLexicalEcho)' -count=1` proves a conservative owner-fact intent scorer recognizes an answer-shaped fact, rejects a question-shaped lexical echo, and makes `Service.Search` rank the answer fact first.
  - Result: public search now uses a MEMORIA-style structured fact signal without changing the Search API, adding LLM extraction, persisting triples, using answer hints, or regenerating benchmark artifacts.

- 2026-05-23: Mnemosyne MEMORIA fact-voice scoring has its first Goncho recall slice.
  - Evidence target: `go test . -run TestRecallFactVoiceRanksStructuredEvidence -count=1` proves structured `fact` evidence contributes to recall scoring, outranks a question-shaped lexical decoy, and surfaces `fact_score`/`fact=` diagnostics in the recall trace/report.
  - Result: Goncho can ingest a Mnemosyne-style structured fact voice through its existing evidence seam without adding LLM extraction, answer hints, benchmark gold IDs, or frozen-artifact changes.

- 2026-05-22: LOCOMO answer-ready closeout now has a guarded roadmap handoff.
  - Evidence target: `go test . -run TestBenchmarkRoadmapSurfacesLocomoAnswerReadyCloseout -count=1` proves internal and public benchmark roadmaps summarize the delivered graph, query-decomposition, temporal/speaker-routing, failure-bucket, backend-comparison, and docs-guard chain.
  - Result: the loop can stop cleanly with a source-backed answer for how to improve Goncho next: keep frozen LOCOMO metrics as the guardrail, choose an approved retrieval slice, and generate a new date-stamped full LOCOMO run only when the change is ready to compare.

- 2026-05-22: Public LOCOMO docs now guard backend-comparison failure-bucket summaries.
  - Evidence target: `go test . -run TestBenchmarkDocsDocumentBackendComparisonFailureBucketSummaries -count=1` proves README, retrieval reference docs, and external adapter notes state that backend-comparison reports expose stable-ID `failure_buckets` and a markdown `Failure buckets` table beside rank-based `failure_categories`.
  - Result: backend authors and benchmark readers can interpret failure-bucket summaries as reporting-only diagnostics without changing scoring or regenerating frozen LOCOMO artifacts.

- 2026-05-22: LOCOMO backend-comparison reports now summarize failure buckets.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLocomoBackendComparisonSummarizesFailureBuckets -count=1` proves comparable backend reports emit `failure_buckets` JSON and a markdown `Failure buckets` table with stable-ID-only `missing_companion_memory` counts.
  - Result: operators can compare failure-driven buckets beside rank-based failure categories without regenerating frozen LOCOMO full-run artifacts or changing scoring semantics.

- 2026-05-22: LOCOMO backend-comparison failure-audit validation now names wrong-branch buckets.
  - Evidence target: `go test ./cmd/goncho-bench -run TestWriteLocomoBackendComparisonFailuresRejectsOutOfConversationRetrievedID -count=1` proves out-of-conversation retrieved stable IDs are still rejected before JSONL emission, and the validation error names `failure_bucket "wrong_branch_retrieval"`.
  - Result: failure-audit operators get bucket-aligned diagnostics without admitting invalid cross-conversation rows into comparable backend audits.

- 2026-05-22: LOCOMO docs now guard wrong-branch external backend rejections.
  - Evidence target: `go test . -run TestBenchmarkDocsDocumentWrongBranchBackendRejections -count=1` proves README, public retrieval docs, and adapter notes state that an out-of-conversation stable `memory_id` is rejected before scoring, labeled `failure_bucket "wrong_branch_retrieval"`, and not rescued by content matching or answer text.
  - Result: backend authors can see that wrong-branch diagnostics are scoring-blocking stable-ID validation, not a new content-only rescue path.

- 2026-05-22: LOCOMO external backend wrong-branch rejections now name the failure bucket.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLocomoBackendComparisonRejectsExternalOutOfConversationMemoryID -count=1` proves external comparable rows that return another conversation's stable `memory_id` are still rejected, and the error names `failure_bucket "wrong_branch_retrieval"`.
  - Result: backend authors get the same failure-driven vocabulary without weakening stable-ID comparability or admitting wrong-branch rows into scoring.

- 2026-05-22: LOCOMO backend-comparison failure audits now emit stable-ID failure buckets.
  - Evidence target: `go test ./cmd/goncho-bench -run TestWriteLocomoBackendComparisonFailuresEmitsFailureBucket -count=1` proves comparable backend failure rows can label missing companion memories as `failure_bucket` without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.
  - Result: external-backend comparisons can share the same failure-driven evaluation vocabulary as Goncho's native LOCOMO failure audit.

- 2026-05-22: LOCOMO failure-driven evaluation has its first implementation slice.
  - Evidence target: `go test ./cmd/goncho-bench -run 'Test(LocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets|WriteLocomoFailureAuditEmitsFailureBucket)' -count=1` proves wrong branch retrieval and missing companion memories can be classified from stable-ID LOCOMO rows, and partial multi-gold audit rows emit `failure_bucket` without regenerating LOCOMO full-run artifacts.
  - Result: future retrieval tuning can start from named failure-audit buckets instead of tuning aggregate recall alone.

- 2026-05-22: LOCOMO failure-driven evaluation now has an implementation plan.
  - Evidence target: `go test . -run TestBenchmarkPlanDocumentsLocomoFailureDrivenEvaluation -count=1` proves `docs/superpowers/plans/2026-05-22-locomo-failure-driven-evaluation.md` names wrong branch retrieval, missing companion memories, failure-audit buckets, stable-ID constraints, and no-answer-hint benchmark discipline.
  - Result: the next LOCOMO evaluation loop can start from a concrete plan before classifying new failure buckets or tuning retrieval behavior.

- 2026-05-22: LOCOMO speaker who-said-what routing has its first implementation slice.
  - Evidence target: `go test . -run TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch -count=1` proves explicit speaker provenance can steer selection to the right who-said-what branch even when the query also names another person.
  - Result: future LOCOMO speaker-routing work can separate the speaker from the object of speech without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.

- 2026-05-22: LOCOMO temporal current-truth routing has its first implementation slice.
  - Evidence target: `go test . -run TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence -count=1` proves current facts can outrank superseded evidence for now/current/latest-style questions while preserving the superseded candidate in `trace.Candidates`.
  - Result: future LOCOMO temporal work can distinguish current truth from past truth with a visible `superseded_evidence_observed` warning and without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.

- 2026-05-22: LOCOMO temporal and speaker routing recall now has an implementation plan.
  - Evidence target: `go test . -run TestBenchmarkPlanDocumentsLocomoTemporalSpeakerRoutingRecall -count=1` proves `docs/superpowers/plans/2026-05-22-locomo-temporal-speaker-routing-recall.md` names the TDD entrypoints, current-truth warning, who-said-what branch routing, superseded-evidence preservation, and stable-ID/no-answer-hint constraints.
  - Result: the next retrieval-code loop can start from a concrete temporal/speaker routing plan without changing frozen LOCOMO artifacts or scoring by answer text.

- 2026-05-22: LOCOMO query-decomposition recall has its first implementation slice.
  - Evidence target: `go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1` proves decomposed subqueries can retrieve each required stable-ID fact for a multi-part question before scoring.
  - Result: multi-hop recall work can cover more required facts without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.

- 2026-05-22: LOCOMO query-decomposition recall now has an implementation plan.
  - Evidence target: `go test . -run TestBenchmarkPlanDocumentsLocomoQueryDecompositionRecall -count=1` proves `docs/superpowers/plans/2026-05-22-locomo-query-decomposition-recall.md` names the TDD entrypoint, subquery split, stable-ID merge/deduplication, and stable-ID/no-answer-hint constraints.
  - Result: the next retrieval-code loop can start from a concrete query-decomposition plan without changing frozen LOCOMO artifacts or scoring by answer text.

- 2026-05-22: LOCOMO graph-assisted recall now has a coverage-aware selection slice.
  - Evidence target: `go test . -run TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion -count=1` proves a relation-path graph companion can beat a near-duplicate lexical hit when the selected set is small.
  - Result: multi-hop recall work can preserve complementary stable-ID memories in the selected context before any LOCOMO full-run artifact is regenerated.

- 2026-05-22: Graph-assisted LOCOMO multi-hop recall has its first implementation slice.
  - Evidence target: `go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1` proves graph-expanded recall retrieves a stable-ID companion memory with relation path provenance.
  - Result: LOCOMO improvement work can move from recommendations to a measured graph-assisted recall slice without changing frozen benchmark artifacts or scoring by answer text.

- 2026-05-22: Graph-assisted LOCOMO multi-hop recall now has an implementation plan.
  - Evidence target: `go test . -run TestBenchmarkPlanDocumentsLocomoGraphAssistedMultiHopRecall -count=1` proves `docs/superpowers/plans/2026-05-22-locomo-graph-assisted-multihop-recall.md` names the TDD entrypoint, stable-ID boundary, graph provenance, coverage-aware selection, and required validation commands.
  - Result: future retrieval-code work can start from a concrete plan without changing production retrieval behavior, frozen LOCOMO artifacts, or stable-ID scoring semantics in this slice.

- 2026-05-22: Benchmark roadmap now names the LOCOMO implementation gate.
  - Evidence target: `go test . -run TestBenchmarkRoadmapNamesLocomoImplementationGate -count=1` proves internal and public benchmark roadmap docs state recommendations are not approval to change retrieval behavior, require an approved plan before production retrieval changes, and require a focused failing recall test before graph/ranking code.
  - Result: future LOCOMO implementation loops have a test-protected approval/TDD boundary while preserving frozen artifacts and stable-ID scoring.

- 2026-05-22: Public benchmark docs now recommend LOCOMO improvement levers.
  - Evidence target: `go test . -run TestBenchmarkDocsRecommendLocomoImprovementLevers -count=1` proves README and Retrieval Benchmarks docs tie next improvements to weak multi-hop and strict-recall metrics, while preserving stable-ID, retrieval-only scoring constraints.
  - Result: readers can answer how to improve Goncho LOCOMO next without adding answer hints, LLM judges, answer-text scoring, benchmark-specific gold-ID hacks, or extra runtime tools.

- 2026-05-22: Benchmark roadmap now names LOCOMO improvement priorities.
  - Evidence target: `go test . -run TestBenchmarkRoadmapNamesLocomoImprovementPriorities -count=1` proves internal and public benchmark roadmap docs name multi-hop graph expansion, query decomposition, coverage-aware ranking, temporal and speaker routing, failure-audit buckets, and explicit multi-hop recall/strict-recall targets.
  - Result: future LOCOMO optimization loops have a test-protected roadmap before changing retrieval behavior or frozen benchmark artifacts.

- 2026-05-22: Public benchmark docs now surface Goncho strict LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoGonchoStrictCategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name Goncho strict_recall@5 and strict_recall@10 for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can distinguish any-hit category recall from all-gold-ID strict recall for Goncho without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface LOCOMO category question counts.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoCategoryQuestionCounts -count=1` proves README and Retrieval Benchmarks docs name 446 adversarial, 92 multi-hop, 841 open-domain, 282 single-hop, and 321 temporal retrieval questions from the frozen full run.
  - Result: LOCOMO readers can interpret category metrics with denominators without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface recency baseline LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoRecencyCategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name recency recall_any@5, recall_any@10, and MRR for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can compare real retrieval backends against the recency lower-bound category baseline without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface random baseline LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoRandomCategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name random recall_any@5, recall_any@10, and MRR for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can compare real retrieval backends against the random lower-bound category baseline without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface SQLite FTS5 LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoSQLiteFTS5CategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name SQLite FTS5 recall_any@5, recall_any@10, and MRR for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can compare Goncho and BM25 against the local SQLite FTS5 lexical baseline without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface BM25 LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoBM25CategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name BM25 recall_any@5, recall_any@10, and MRR for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can compare Goncho's category-level full-run performance against BM25 without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface Goncho LOCOMO category metrics.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoGonchoCategoryMetrics -count=1` proves README and Retrieval Benchmarks docs name Goncho recall_any@5, recall_any@10, and MRR for adversarial, multi-hop, open-domain, single-hop, and temporal retrieval categories.
  - Result: LOCOMO readers can inspect Goncho's category-level full-run performance without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface LOCOMO category metric groups.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoCategoryMetricGroups -count=1` proves README and Retrieval Benchmarks docs name the adversarial, multi-hop, open-domain, single-hop, and temporal retrieval groups reported by the frozen full run.
  - Result: LOCOMO readers can see which retrieval categories are summarized without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface LOCOMO leakage-check counts.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoLeakageCheckCounts -count=1` proves README and Retrieval Benchmarks docs name answer-text, gold-ID, and question-text leakage counts and explain why answer-text presence is reported separately from `answer_hint` indexing/scoring.
  - Result: LOCOMO readers can distinguish literal answer spans in gold memories from benchmark leakage or answer-hint scoring.

- 2026-05-22: Public benchmark docs now surface LOCOMO converted dataset evidence.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoConvertedDatasetEvidence -count=1` proves README and Retrieval Benchmarks docs name `data/locomo/memories.jsonl`, `data/locomo/questions.jsonl`, 1,982 questions, and 5,882 memories for the frozen full run.
  - Result: LOCOMO readers can connect the frozen result JSON to the converted dataset files and full-run scale without opening the generated full report first.

- 2026-05-22: Public benchmark docs now surface LOCOMO source provenance.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoSourceProvenance -count=1` proves README and Retrieval Benchmarks docs name the LOCOMO source repository, pinned revision, source SHA256, and CC BY-NC license note.
  - Result: LOCOMO readers can trace the frozen result back to source provenance without opening the generated full report first.

- 2026-05-22: Public benchmark docs now name the frozen full LOCOMO baseline set.
  - Evidence target: `go test . -run TestBenchmarkDocsNameLocomoFullBaselineSet -count=1` proves README and Retrieval Benchmarks docs name random, recency, BM25, SQLite FTS5, and Goncho as the frozen full LOCOMO run's baseline set.
  - Result: LOCOMO readers can distinguish the full-run comparison set from smoke-only and external-backend adapter evidence.

- 2026-05-22: Public benchmark docs now state the LOCOMO benchmark scope explicitly.
  - Evidence target: `go test . -run TestBenchmarkDocsStateLocomoRetrievalOnlyScope -count=1` proves README and Retrieval Benchmarks docs say LOCOMO is retrieval-only, uses no answer generation, no LLM judge, ID-based scoring, and never indexes or scores `answer_hint` fields.
  - Result: LOCOMO readers can distinguish deterministic retrieval evidence from answer-generation or judge-based benchmark claims.

- 2026-05-22: Public benchmark docs now link the LOCOMO BM25-vs-Goncho candidate-generation failure comparison audit.
  - Evidence target: `go test . -run TestBenchmarkDocsLinkLocomoCandidateFailureComparisonAudit -count=1` proves README and Retrieval Benchmarks docs link `docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl` and name the BM25-win `missing_candidate` diagnosis.
  - Result: LOCOMO readers can trace the candidate-generation milestone back to the failure-comparison audit instead of trusting the summary alone.

- 2026-05-22: Public benchmark docs now label LOCOMO smoke failure audits as smoke-only evidence.
  - Evidence target: `go test . -run TestBenchmarkDocsLabelSmokeFailureAuditArtifacts -count=1` proves README and Retrieval Benchmarks docs distinguish `docs/benchmarks/failures/locomo-smoke-categories.jsonl` and `docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl` from historical full-run evidence.
  - Result: LOCOMO readers can use smoke failure audits for harness checks without mistaking them for frozen full-run audit artifacts.

- 2026-05-22: Public benchmark docs now link LOCOMO failure-audit artifacts beside result JSON evidence.
  - Evidence target: `go test . -run TestBenchmarkDocsLinkLocomoFailureAuditArtifacts -count=1` proves README and Retrieval Benchmarks docs link `docs/benchmarks/failures/locomo-2026-05-20-categories.jsonl` and `docs/benchmarks/failures/locomo-backend-comparison.jsonl`.
  - Result: LOCOMO readers can inspect retrieval-miss categories and not-comparable backend evidence without treating result JSON as the only audit trail.

- 2026-05-22: `make release-metadata-smoke` now runs the LOCOMO benchmark-result docs guards.
  - Evidence target: `go test . -run TestReleaseMetadataSmokeIncludesLocomoResultDocsGuards -count=1` proves the release metadata smoke regex includes the LOCOMO metric-surface, frozen-artifact, and reproduction-command guards.
  - Result: the narrow release gate keeps the public LOCOMO benchmark-result claims wired after future docs edits.

- 2026-05-22: Public benchmark docs now name exact LOCOMO reproduction commands for full, smoke, backend-smoke, and full-backend runs.
  - Evidence target: `go test . -run TestBenchmarkDocsNameLocomoReproductionCommands -count=1` proves README and Retrieval Benchmarks docs label `make bench-locomo`, `make bench-locomo-smoke`, `make bench-locomo-backends-smoke`, and `make bench-locomo-backends` with their reproduction roles.
  - Result: LOCOMO operators can pick the right command without confusing CI-safe smoke regeneration with manual full-result reproduction.

- 2026-05-22: Public benchmark docs now distinguish frozen LOCOMO full-run evidence from regenerated smoke/backend artifacts.
  - Evidence target: `go test . -run TestBenchmarkDocsDistinguishFrozenLocomoResultArtifacts -count=1` proves README and Retrieval Benchmarks docs say the frozen full-run JSON is not regenerated by smoke targets while regenerated smoke/backend artifacts are fresh harness checks with host-sensitive latency/RSS measurements.
  - Result: LOCOMO readers can cite the correct JSON artifact without treating smoke regeneration as a historical full-run update.

- 2026-05-22: Public benchmark docs now surface the LOCOMO result metric set beyond recall and MRR.
  - Evidence target: `go test . -run TestBenchmarkDocsSurfaceLocomoResultMetricSet -count=1` proves README and Retrieval Benchmarks docs mention NDCG@5, NDCG@10, latency distribution, RSS, database size, memory token estimate, Top-K, failure categories, leakage checks, and the frozen JSON evidence files.
  - Result: LOCOMO result readers see the full metric surface without rewriting frozen benchmark artifacts.

- 2026-05-22: Root package documentation now includes a trust-boundary guide for pkg.go.dev readers embedding Goncho in host agents.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesTrustBoundaryGuide|ReleaseMetadataSmokeIncludesPackageDocTrustBoundaryGuard)' -count=1` proves `go doc .` distinguishes Goncho orientation from host authority over authorization, live filesystem/API/deployment/credential state, money movement, destructive writes, external side effects, and live verification.
  - Result: readers landing directly on pkg.go.dev can adopt Goncho without mistaking memory retrieval for permission to skip host-side gates.

- 2026-05-22: README now includes a trust-boundary guide for pkg.go.dev readers embedding Goncho in host agents.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesTrustBoundaryGuide|ReleaseMetadataSmokeIncludesReadmeTrustBoundaryGuard)' -count=1` proves the README distinguishes Goncho orientation from host authority over authorization, live filesystem/API/deployment/credential state, money movement, destructive writes, external side effects, and live verification.
  - Result: readers can adopt Goncho without mistaking memory retrieval for permission to skip host-side gates.

- 2026-05-22: Root package documentation now includes a host integration checklist for pkg.go.dev readers embedding Goncho.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesHostIntegrationChecklist|ReleaseMetadataSmokeIncludesPackageDocHostIntegrationGuard)' -count=1` proves `go doc .` walks host integrators through SQLite opening, migrations before service construction, attribution config, explicit profile/peer/session scoping, context-before-tools, evidence-backed conclusions, and live verification.
  - Result: readers landing directly on pkg.go.dev can wire a safe host integration path without opening the README first.

- 2026-05-22: README now includes a host integration checklist for pkg.go.dev readers embedding Goncho.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesHostIntegrationChecklist|ReleaseMetadataSmokeIncludesReadmeHostIntegrationGuard)' -count=1` proves the README walks host integrators through SQLite opening, migrations, service construction, explicit profile/peer/session scoping, context-before-tools, evidence-backed conclusions, and live verification.
  - Result: readers can move from install/import guidance to a safe host wiring checklist without inferring operational boundaries from examples alone.

- 2026-05-22: Root package documentation now includes an import path guide for pkg.go.dev readers.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesImportPathGuide|ReleaseMetadataSmokeIncludesPackageDocImportPathGuard)' -count=1` proves `go doc .` distinguishes the root library package, `memory` SQLite opener, and `cmd/goncho-bench` command-only path while release metadata smoke keeps the guard wired.
  - Result: readers landing directly on pkg.go.dev can choose the correct import/install path without opening the README first.

- 2026-05-22: README now includes an import path guide for pkg.go.dev readers.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesImportPathGuide|ReleaseMetadataSmokeIncludesReadmeImportPathGuard)' -count=1` proves the README distinguishes the root library package, `memory` SQLite opener, and `cmd/goncho-bench` command path while release metadata smoke keeps the guard wired.
  - Result: readers can choose the correct Go import/install path without starting from the full pkg.go.dev symbol index.

- 2026-05-22: Root package documentation now explains versioning and adoption notes for pkg.go.dev readers.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesVersioningAndAdoptionNotes|ReleaseMetadataSmokeIncludesPackageDocVersioningGuard)' -count=1` proves `go doc .` includes pre-1.0 stability guidance, pinned v0.1.1 install guidance, `@latest` deployment-lock warning, Stable version interpretation, Imported by 0 interpretation, reverse-dependency context, and ecosystem smoke guidance while release metadata smoke keeps the guard wired.
  - Result: readers landing on pkg.go.dev can interpret go.dev stability and adoption metadata directly from the package overview.

- 2026-05-22: README now explains versioning and adoption notes for pkg.go.dev readers.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesVersioningAndAdoptionNotes|ReleaseMetadataSmokeIncludesReadmeVersioningGuard)' -count=1` proves the README includes pre-1.0 stability guidance, a pinned v0.1.1 dependency command, an `@latest` deployment-lock warning, Imported by 0 interpretation, and ecosystem smoke guidance while release metadata smoke keeps the guard wired.
  - Result: readers can interpret go.dev stability and adoption metadata without confusing popularity counters or `@latest` with deployment readiness.

- 2026-05-22: Root package documentation now maps go.dev package signals to local proof commands.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesGoDevPackageSignals|ReleaseMetadataSmokeIncludesPackageDocGoDevSignalGuard)' -count=1` proves `go doc .` includes public version, go.mod, MIT license, package-doc, external-import, and command-install smoke guidance while release metadata smoke keeps the guard wired.
  - Result: pkg.go.dev readers can connect the package page metadata to reproducible local checks without leaving the API overview.

- 2026-05-22: README now maps go.dev package signals to local proof commands.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesGoDevSignalMap|ReleaseMetadataSmokeIncludesReadmeGoDevSignalGuard)' -count=1` proves the README includes go.dev version, published date, go.mod, license, package-doc, external-import, install-path, and Imported by guidance while release metadata smoke keeps the guard wired.
  - Result: readers can connect the pkg.go.dev metadata panel to reproducible local checks instead of trusting badges alone.

- 2026-05-22: README now gives pkg.go.dev readers a minimal embedded host skeleton.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesMinimalEmbeddedSkeleton|ReleaseMetadataSmokeIncludesReadmeMinimalSkeletonGuard)' -count=1` proves the README includes a copy-paste local SQLite service skeleton and release metadata smoke keeps the guard wired.
  - Result: readers can start a Go module around Goncho without confusing the library package with the benchmark CLI.

- 2026-05-22: Root package documentation now maps pkg.go.dev readers to the primary API path.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesPrimaryAPIPath|ReleaseMetadataSmokeIncludesPackageDocAPIPathGuard)' -count=1` proves `go doc .` names `Service.Conclude`, `Service.Search`, `Service.Context`, the public tool constructors, and the database-internals boundary while release metadata smoke keeps the guard wired.
  - Result: readers can orient on the service/tool entry points before scanning the large API index.

- 2026-05-22: Root package documentation now tells pkg.go.dev readers how to install the library versus the benchmark command.
  - Evidence target: `go test . -run 'Test(PackageDocSurfacesInstallAndCommandBoundary|ReleaseMetadataSmokeIncludesPackageDocInstallGuard)' -count=1` proves `go doc .` includes the `go get` path, root-library boundary, and `goncho-bench@latest` command path while release metadata smoke keeps the guard wired.
  - Result: pkg.go.dev readers can distinguish the importable library from the installable benchmark CLI without learning that boundary from a failed `go install` attempt.

- 2026-05-22: README now gives pkg.go.dev readers an API map from evaluation goals to public entry points.
  - Evidence target: `go test . -run 'Test(ReadmeSurfacesPkgGoDevAPIMap|ReleaseMetadataSmokeIncludesReadmeAPIMapGuard)' -count=1` proves the README includes the map and release metadata smoke keeps it guarded.
  - Result: readers can find `memory.OpenSqlite`, `goncho.RunMigrations`, `goncho.NewService`, `svc.Conclude`, `svc.Search`, `svc.Context`, public tools, and `goncho-bench@latest` without scanning the full package index.

- 2026-05-22: Root package overview now points pkg.go.dev readers to compiled examples for setup, orientation packs, and scoped retrieval.
  - Evidence target: `go test . -run 'Test(PackageDocPointsPkgGoDevReadersToCompiledExamples|ReleaseMetadataSmokeIncludesPackageDocExamplesGuard)' -count=1` proves `go doc .` includes the example path and release metadata smoke keeps it guarded.
  - Result: pkg.go.dev readers can jump from the overview to checked examples instead of inferring the first API path from the large index.

- 2026-05-22: Root package examples now give pkg.go.dev readers a compiled `Service.Search` scoped-retrieval path.
  - Evidence target: `go test . -run 'Test(PackageDocsIncludeCompiledSearchExample|ReleaseMetadataSmokeIncludesSearchExampleGuard)' -count=1` plus `go test . -run ExampleService_Search -count=1` proves the example is present, release-smoke guarded, and executable.
  - Result: pkg.go.dev can render a minimal search call that retrieves a stored conclusion by query and prints its evidence source.

- 2026-05-22: Root package examples now give pkg.go.dev readers a compiled `Service.Context` orientation-pack path.
  - Evidence target: `go test . -run 'Test(PackageDocsIncludeCompiledContextExample|ReleaseMetadataSmokeIncludesContextExampleGuard)' -count=1` plus `go test . -run ExampleService_Context -count=1` proves the example is present, release-smoke guarded, and executable.
  - Result: pkg.go.dev can render the first useful orientation call after SQLite setup, profile facts, and a stored conclusion.

- 2026-05-22: Root package examples now give pkg.go.dev readers a compiled `NewService` setup path.
  - Evidence target: `go test . -run 'Test(PackageDocsIncludeCompiledNewServiceExample|ReleaseMetadataSmokeIncludesPackageExampleGuard)' -count=1` plus `go test . -run ExampleNewService -count=1` proves the example is present, release-smoke guarded, and executable.
  - Result: pkg.go.dev can render an example that opens SQLite, runs migrations, creates `goncho.NewService`, writes a profile fact, and reads it back.

- 2026-05-22: README now gives pkg.go.dev readers a concise `At a Glance` evaluation path.
  - Evidence target: `go test . -run TestReadmeSurfacesPkgGoDevEvaluationPath -count=1` proves the README includes pkg.go.dev evaluation, first-call, and next-reading markers.
  - Result: README readers can quickly identify install command, use cases, non-goals, first useful service call, and trust boundary.

- 2026-05-22: Root package documentation now gives pkg.go.dev readers a stronger landing page.
  - Evidence target: `go test . -run TestPackageDocSurfacesPkgGoDevLandingContent -count=1` proves `go doc .` includes use-case, quick-start, verification-before-action, and `goncho.NewService` markers.
  - Result: pkg.go.dev/go.dev package readers see a clearer first screen before the API index.

- 2026-05-22: Checked-in LOCOMO smoke benchmark artifacts now include `goncho-no-rank` retrieval and backend-comparison baselines.
  - Evidence target: `make bench-locomo-smoke` plus `AGENTMEMORY_SOURCE_DIR=/home/xel/git/sages-openclaw/workspace-mineru/goncho/docs/opensource-memory-systems/agentmemory make bench-locomo-backends-smoke` regenerates the tracked smoke JSON, markdown, and failure-audit artifacts with `goncho-no-rank`; `python3 -m json.tool` validates the regenerated JSON artifacts.
  - Result: CI-safe smoke evidence now matches the current benchmark harness and documents the no-ranking baseline in checked-in artifacts.

- 2026-05-22: LOCOMO backend-comparison reports now include a `goncho-no-rank` no-ranking baseline alongside current Goncho.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves backend-comparison JSON and markdown include `goncho-no-rank` as a comparable local no-ranking baseline.
  - Result: backend-comparison artifacts now satisfy the roadmap baseline requirement for Goncho without current ranking while preserving centralized stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval reports now include a `goncho-no-rank` no-ranking baseline alongside current Goncho.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestRunLocomoSmokeProducesReport|TestRetrieveLocomoReturnsNoIDsForNonPositiveLimits' -count=1` proves the smoke JSON includes `goncho-no-rank`, the markdown baseline note names Goncho no-rank, and non-positive limits remain safe for the new baseline.
  - Result: retrieval benchmark artifacts now satisfy the roadmap baseline requirement for Goncho without current ranking while preserving deterministic stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval and backend-comparison markdown now include one-command reproduction lines.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomoMarkdownIncludesReproductionCommand|TestRunLocomoBackendComparisonWritesJSONAndMarkdown' -count=1` proves markdown summaries emit copy-pasteable `go run ./cmd/goncho-bench` commands with the same fixture, output, failure-audit, markdown, and limit paths.
  - Result: benchmark markdown now satisfies the one-command reproduction scientific-control requirement without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval markdown now includes converted-artifact checksums when metadata is available.
  - Evidence target: `go test ./cmd/goncho-bench -run TestWriteLocomoMarkdownIncludesConvertedChecksums -count=1` proves markdown summaries include converted memories and questions SHA256 values from source metadata.
  - Result: retrieval markdown now surfaces the converted-artifact checksum scientific control already present in JSON metadata without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison markdown now includes dataset provenance when metadata is available.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves markdown summaries include source URL, source revision, source checksum, converted fixture checksums, and license note.
  - Result: backend-comparison markdown now surfaces the scientific controls already present in JSON metadata without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison reports now include per-backend category metrics.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves JSON artifacts emit `category_metrics` and markdown summaries include per-backend category metric tables.
  - Result: backend-comparison artifacts now mirror LOCOMO retrieval category-metric reporting without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison markdown now includes per-backend failure-category counts.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves markdown summaries include `## Failure categories` and backend/category count rows.
  - Result: backend-comparison markdown now surfaces the failure taxonomy already emitted in JSON without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison reports now include per-backend latency distribution stats.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves JSON artifacts emit `latency_ms` and markdown summaries include latency distribution columns.
  - Result: backend-comparison artifacts now mirror LOCOMO retrieval latency min/p50/p95/max reporting without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison markdown now includes per-backend insert latency and RSS metrics.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves markdown summaries include `Insert latency ms` and `RSS bytes` columns alongside existing search latency.
  - Result: backend-comparison markdown now surfaces the resource metrics already emitted in JSON without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison reports now record per-backend NDCG@5 and NDCG@10 metrics.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoBackendComparisonUsesStableMemoryIDs|TestRunLocomoBackendComparisonWritesJSONAndMarkdown' -count=1` proves backend-comparison entries aggregate ID-based NDCG metrics and markdown summaries include `NDCG@5`/`NDCG@10` columns.
  - Result: backend-comparison artifacts now satisfy the roadmap's NDCG reporting requirement without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO backend-comparison reports now record LOCOMO leakage checks.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBackendComparisonWritesJSONAndMarkdown -count=1` proves JSON artifacts emit `leakage_checks` and markdown summaries include `## Leakage checks`.
  - Result: backend-comparison artifacts now satisfy the roadmap's same-leakage-checks requirement without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval and backend-comparison reports now record converted fixture database byte sizes.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestRunLocomoBenchmarkHonorsConfiguredLimit|TestRunLocomoSmokeProducesReport|TestRunLocomoBackendComparisonWritesJSONAndMarkdown' -count=1` proves JSON artifacts emit `database_size_bytes` and markdown summaries print `Database size bytes`.
  - Result: benchmark artifacts now satisfy the roadmap's database-size reporting requirement without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval reports now record per-system NDCG@5 and NDCG@10 metrics.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoScoringStrictAnyAndMRR|TestLocomoCategoryMetricAggregation|TestRunLocomoSmokeProducesReport' -count=1` proves ID-based NDCG scoring, system/category aggregation, JSON fields, and markdown columns.
  - Result: retrieval artifacts now satisfy the roadmap's NDCG reporting requirement without changing retrieval or stable-ID scoring semantics.

- 2026-05-22: LOCOMO retrieval reports now record per-system latency distribution stats.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoLatencyMetricAggregation|TestRunLocomoSmokeProducesReport' -count=1` proves JSON system rows emit `latency_ms` min/p50/p95/max and markdown summaries include latency distribution columns.
  - Result: retrieval artifacts now satisfy the roadmap's latency min/p50/p95/max reporting requirement without changing scoring semantics.

- 2026-05-22: LOCOMO retrieval reports now record per-system failure-category counts.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoSmokeProducesReport -count=1` proves JSON system rows emit `failure_categories` and markdown summaries include a failure-category section.
  - Result: retrieval artifacts now satisfy the roadmap's failure-category reporting requirement without changing scoring semantics.

- 2026-05-22: LOCOMO retrieval and backend-comparison reports now record deterministic memory token estimates.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestRunLocomo(BenchmarkHonorsConfiguredLimit|BackendComparisonWritesJSONAndMarkdown|SmokeProducesReport)' -count=1` proves JSON artifacts emit `memory_token_estimate` and markdown summaries print the value.
  - Result: benchmark artifacts now satisfy the roadmap's memory-count plus total-token-estimate reporting requirement without changing scoring semantics.

- 2026-05-22: LOCOMO retrieval reports now record per-system search latency and RSS metrics.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoSmokeProducesReport -count=1` proves JSON system rows emit `search_latency_ms` and `rss_bytes`, and markdown summaries include resource metric columns.
  - Result: retrieval artifacts now satisfy the benchmark roadmap's latency/RSS evidence requirement without changing scoring semantics.

- 2026-05-22: LOCOMO retrieval and backend-comparison reports now record the effective top-K scoring window.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestRunLocomo(BenchmarkHonorsConfiguredLimit|BackendComparisonHonorsConfiguredLimitForExternalRows|SmokeProducesReport|BackendComparisonWritesJSONAndMarkdown)' -count=1` proves JSON artifacts emit `top_k` and markdown summaries print `Top-K`.
  - Result: benchmark artifacts are self-describing when operators run LOCOMO with non-default retrieval limits.

- 2026-05-22: LOCOMO failure-audit notes now report the actual retrieved top-K window.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoFailureJSONL(Generation|NotesUseRetrievedWindow)|TestWriteLocomoBackendComparisonFailuresRejectsUnknownRetrievedID' -count=1` proves failure JSONL says `top 1` for a top-1 miss instead of hard-coding `top 10`.
  - Result: benchmark miss notes remain accurate when LOCOMO reports run with non-default retrieval limits.

- 2026-05-22: `goncho_review` list output now echoes the effective status filter.
  - Evidence target: `go test . -run 'TestReviewTool(ListOutputIncludesEffectiveStatus|TreatsBlankStatusAsOpenDefault|ListFiltersByWorkspaceID|FiltersReviewChainsBySubjectAndRelatedID|ListsAndResolvesReviewItems)' -count=1` proves blank status requests default to `open` and the list response reports that effective status.
  - Result: operators can audit which review queue status was listed without inferring silent defaults from item rows.

- 2026-05-22: `goncho_review` list requests now support workspace ID filters.
  - Evidence target: `go test . -run 'TestReviewTool(ListFiltersByWorkspaceID|ResolveOutputIncludesWorkspaceID|ResolveOutputIncludesCreatedAt|ResolveOutputIncludesReason|ResolveOutputIncludesEvidenceIDs|ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves list responses honor `workspace_id` filters instead of falling back to the service default workspace.
  - Result: operators can inspect a workspace-specific review queue through the public review tool.

- 2026-05-22: `goncho_review` resolve output now includes workspace IDs.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesWorkspaceID|ResolveOutputIncludesCreatedAt|ResolveOutputIncludesReason|ResolveOutputIncludesEvidenceIDs|ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `workspace_id`.
  - Result: operators can audit which workspace owned the closed review item without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes original creation timestamps.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesCreatedAt|ResolveOutputIncludesReason|ResolveOutputIncludesEvidenceIDs|ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `created_at` timestamp.
  - Result: operators can audit when the review item was opened without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes original review reasons.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesReason|ResolveOutputIncludesEvidenceIDs|ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `reason`.
  - Result: operators can audit why the review item existed without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes evidence IDs.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesEvidenceIDs|ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `evidence_ids`.
  - Result: operators can audit which proof identifiers were reviewed without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes peer/session scope.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesScope|ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `peer_id` and `session_key`.
  - Result: operators can audit which scoped review queue item was closed without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes review kind.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesKind|ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the adjudicated review item's `kind`.
  - Result: operators can audit whether a closed review item was conflict or stale without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes review-chain identifiers.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesReviewChain|ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses echo the `subject_id` and `related_id` for the adjudicated review item.
  - Result: operators can audit which memory/review chain was adjudicated without issuing a second list call.

- 2026-05-22: `goncho_review` resolve output now includes `resolved_at` audit timestamps.
  - Evidence target: `go test . -run 'TestReviewTool(ResolveOutputIncludesResolvedAt|ListsAndResolvesReviewItems)' -count=1` proves resolve responses include a parseable `resolved_at` timestamp matching the persisted resolved review item.
  - Result: operators can audit when a review item was adjudicated without issuing a second list call.

- 2026-05-22: single-item `review_required` context warnings now use singular wording.
  - Evidence target: `go test . -run 'TestContextReportsReviewWarning(MarksOmittedEvidenceIDs|MarksOmittedDetails)' -count=1` proves one open review item says `1 open review item requires adjudication` while multi-item warnings keep plural wording.
  - Result: lifecycle review warnings read cleanly for both single-item and multi-item review queues.

- 2026-05-22: bounded `review_required` context warnings now report omitted evidence-ID counts.
  - Evidence target: `go test . -run 'TestContextReportsReviewWarning(MarksOmittedEvidenceIDs|MarksOmittedDetails)' -count=1` proves context unavailable evidence says `evidence_omitted=N` when more unique evidence IDs exist than the bounded preview shows.
  - Result: lifecycle review warnings stay compact without hiding that additional proof identifiers exist for open review work.

- 2026-05-22: unscoped `review_required` context warnings now report omitted session-key counts.
  - Evidence target: `go test . -run 'TestContextReportsReviewWarning(MarksOmittedSessionKeys|IncludesSessionKeysWhenUnscoped)' -count=1` proves peer-level review warnings include `session_keys_omitted=N` when more distinct affected sessions exist than the bounded preview shows.
  - Result: lifecycle review warnings stay compact without hiding that additional sessions have open review work.

- 2026-05-22: unscoped `review_required` context warnings now preview affected session keys.
  - Evidence target: `go test . -run 'TestContextReports(OpenReviewItemsAsUnavailableEvidence|ReviewWarningIncludesSessionKeysWhenUnscoped)' -count=1` proves peer-level context warnings include bounded `session_keys=...` detail while session-scoped warnings keep `session_key=<session>`.
  - Result: lifecycle review warnings are easier to triage when a peer has open review work spread across multiple sessions.

- 2026-05-22: session-scoped `review_required` context warnings now name their session key.
  - Evidence target: `go test . -run TestContextReportsOpenReviewItemsAsUnavailableEvidence -count=1` proves context unavailable evidence includes `session_key=<session>` while keeping same-session counts, review item IDs, chains, and evidence IDs.
  - Result: lifecycle review warnings are easier to audit because the compact warning states the scope used to filter open review items.

- 2026-05-22: bounded `review_required` context warnings now report omitted detail counts.
  - Evidence target: `go test . -run 'TestContextReports(OpenReviewItemsAsUnavailableEvidence|ReviewWarningMarksOmittedDetails)' -count=1` proves context unavailable evidence says `item_details_omitted=N` when more open review items exist than the bounded item/chains/evidence preview shows.
  - Result: lifecycle review warnings stay compact without hiding that additional open review items need adjudication.

- 2026-05-22: `review_required` context warnings are scoped to the requested session.
  - Evidence target: `go test . -run TestContextReportsOpenReviewItemsAsUnavailableEvidence -count=1` proves context unavailable evidence excludes open review items from another session for the same peer while keeping same-session review counts, chains, item IDs, and evidence IDs.
  - Result: lifecycle review warnings no longer let unrelated same-peer sessions steer the current session context.

- 2026-05-22: `review_required` context warnings now include review item IDs.
  - Evidence target: `go test . -run TestContextReportsOpenReviewItemsAsUnavailableEvidence -count=1` proves context unavailable evidence surfaces bounded review item IDs alongside counts, subject chains, and evidence IDs.
  - Result: lifecycle review warnings are directly actionable because operators can resolve the listed review items without first running a separate list call.

- 2026-05-22: `review_required` context warnings now include review evidence IDs.
  - Evidence target: `go test . -run TestContextReportsOpenReviewItemsAsUnavailableEvidence -count=1` proves context unavailable evidence surfaces bounded `evidence_ids` alongside review counts and subject chains.
  - Result: lifecycle review warnings are easier to audit from context output without silently dropping proof identifiers.

- 2026-05-22: LOCOMO failure audits now reject out-of-conversation gold stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomo(FailureAuditRejectsOutOfConversationGoldMemoryID|BackendComparisonFailuresRejectsOutOfConversationGoldMemoryID)' -count=1` proves both Goncho and backend-comparison failure JSONL fail closed when a report row carries a `gold_memory_id` from a different `conversation_id` than the question.
  - Result: failure reports preserve conversation-scoped evidence IDs for expected and retrieved memory rows.

- 2026-05-22: LOCOMO failure audits now reject unknown gold stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomo(FailureAuditRejectsUnknownGoldMemoryID|BackendComparisonFailuresRejectsUnknownGoldMemoryID)' -count=1` proves both Goncho and backend-comparison failure JSONL fail closed when a report row carries a `gold_memory_id` absent from the loaded LOCOMO fixture.
  - Result: failure reports preserve known evidence IDs for both expected and retrieved memory rows.

- 2026-05-22: LOCOMO failure audits now reject question conversation mismatches.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomo(FailureAuditRejectsQuestionConversationMismatch|BackendComparisonFailuresRejectsQuestionConversationMismatch)' -count=1` proves both Goncho and backend-comparison failure JSONL fail closed when a report row's `question_id` exists but its `conversation_id` disagrees with the loaded LOCOMO fixture.
  - Result: failure reports preserve fixture-scoped question identity before evaluating retrieved stable IDs.

- 2026-05-22: LOCOMO failure audits now reject unknown question IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomo(FailureAuditRejectsUnknownQuestionID|BackendComparisonFailuresRejectsUnknownQuestionID)' -count=1` proves both Goncho and backend-comparison failure JSONL fail closed when a failure row references a `question_id` absent from the loaded LOCOMO fixture.
  - Result: failure reports preserve the same fixture-scoped stable question-ID invariant as centralized scoring.

- 2026-05-22: LOCOMO failure audits now reject out-of-conversation retrieved stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestWriteLocomo(FailureAuditRejectsOutOfConversationRetrievedID|BackendComparisonFailuresRejectsOutOfConversationRetrievedID)' -count=1` proves both Goncho and backend-comparison failure JSONL fail closed when a top-hit `memory_id` belongs to another `conversation_id`.
  - Result: failure reports preserve the same conversation-scoped stable-ID invariant as centralized scoring.

- 2026-05-22: LOCOMO backend-comparison failure audits now reject unknown retrieved stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run TestWriteLocomoBackendComparisonFailuresRejectsUnknownRetrievedID -count=1` proves backend-comparison failure JSONL fails closed when a top-hit `memory_id` is not present in the loaded LOCOMO fixture.
  - Result: comparison failure reports no longer hide backend/report stable-ID drift behind blank memory metadata rows.

- 2026-05-22: LOCOMO SQLite FTS retrieval now skips temporary database setup for tokenless queries.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRetrieveLocomoSQLiteFTSSkipsStoreForTokenlessQuery -count=1` proves stopword-only LOCOMO questions use the recency fallback without creating a temporary SQLite FTS store.
  - Result: report generation avoids wasted temp DB creation/population for questions with no indexable FTS tokens while preserving fallback ordering.

- 2026-05-22: LOCOMO failure audits now reject unknown retrieved stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run TestWriteLocomoFailureAuditRejectsUnknownRetrievedID -count=1` proves Goncho failure-audit output fails closed when a top-hit `memory_id` is not present in the loaded LOCOMO fixture.
  - Result: failure JSONL no longer hides retrieval/stable-ID drift behind blank memory metadata rows.

- 2026-05-22: LOCOMO leakage checks now reuse the conversation index.
  - Evidence target: `go test ./cmd/goncho-bench -run TestCheckLocomoLeakageUsesConversationIndex -count=1` proves leakage auditing reads the precomputed per-conversation LOCOMO memory index when available.
  - Result: LOCOMO report generation avoids rebuilding a duplicate conversation map for leakage checks while preserving conversation-scoped answer/gold/question leakage accounting.

- 2026-05-22: LOCOMO direct retrieval now rejects non-positive limits before backend work.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRetrieveLocomoReturnsNoIDsForNonPositiveLimits -count=1` proves direct calls with zero or negative limits return no IDs across random, recency, BM25, SQLite FTS5, and Goncho retrieval paths.
  - Result: internal LOCOMO retrieval now treats non-positive top-K windows as empty instead of panicking in slice helpers or letting SQLite FTS5 return all rows for `LIMIT -1`.

- 2026-05-22: LOCOMO fixture loading now rejects duplicate gold stable IDs before scoring.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLoadLocomoDatasetRejectsDuplicateGoldStableIDs -count=1` proves repeated `gold_memory_ids` within one question fail at fixture-load time instead of silently reaching centralized stable-ID scoring.
  - Result: LOCOMO reports and external-backend comparisons now fail closed when gold evidence IDs are not unique per question.

- 2026-05-22: LOCOMO Goncho adapters now cap duplicate-content stable-ID fan-out to top-K.
  - Evidence target: `go test ./cmd/goncho-bench -run 'Test(RunLocomoBenchmarkCapsGonchoStableIDFanoutToLimit|GonchoBackendScopedSearchCapsStableIDFanoutToTopK)' -count=1` proves duplicate content mapping to multiple stable IDs cannot expand a configured top-K window in the LOCOMO report path or backend-comparison Goncho adapter.
  - Result: reproducible LOCOMO scoring now treats content-to-ID collisions like external duplicate rows: the requested top-K result window is the scoring boundary.

- 2026-05-22: LOCOMO fixture loading now rejects invalid gold stable IDs before scoring.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLoadLocomoDatasetRejectsInvalidGoldStableIDs -count=1` proves unknown `gold_memory_ids` and gold IDs from a different `conversation_id` fail at fixture-load time.
  - Result: LOCOMO reports and external-backend comparisons now fail closed when gold evidence cannot be scored as known same-conversation memory IDs.

- 2026-05-22: LOCOMO fixture loading now rejects duplicate stable IDs before scoring.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLoadLocomoDatasetRejectsDuplicateStableIDs -count=1` proves duplicate `memory_id` and `question_id` values fail at fixture-load time instead of reaching centralized stable-ID scoring.
  - Result: LOCOMO reports and external-backend comparisons now fail closed when converted fixture IDs are not unique enough for deterministic evidence scoring.

- 2026-05-22: LOCOMO smoke/full retrieval reports now honor the configured top-K limit.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBenchmarkHonorsConfiguredLimit -count=1` proves `--limit 1` reaches the LOCOMO retrieval report path and caps each local system's reported retrieved IDs.
  - Result: reproducible LOCOMO retrieval reports now use the operator-requested top-K window instead of always evaluating every local system with top 10.

- 2026-05-22: LOCOMO backend comparison now honors the configured top-K limit in the full report path.
  - Evidence target: `go test ./cmd/goncho-bench -run 'Test(RunLocomoBackendComparisonHonorsConfiguredLimitForExternalRows|LocomoBackendComparisonDuplicateExternalRowsDoNotExpandTopK)' -count=1` proves `--limit 1` reaches external adapter scoring and duplicate external rows cannot expand the top-K window.
  - Result: reproducible backend comparison reports now use the operator-requested top-K window consistently across local and external backends.

- 2026-05-22: LOCOMO external adapter scoring now clamps top-K rows and rejects out-of-conversation stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoBackendComparison(LimitsExternalRowsToTopK|RejectsExternalOutOfConversationMemoryID)' -count=1` proves comparable external rows obey the requested top-K window and cannot return a stable `memory_id` from a different `conversation_id` than the question.
  - Result: the Go scorer now enforces the documented conversation-scoped backend comparison contract before stable-ID scoring, so external adapters cannot get comparable credit by over-returning rows or crossing conversation boundaries.

- 2026-05-22: benchmark docs now surface conversation-scoped backend comparison.
  - Evidence target: `go test . -run TestBenchmarkDocsMentionConversationScopedBackendComparison -count=1` proves README, Retrieval Benchmarks, operator runbook, and external adapter docs say LOCOMO backend comparison is conversation-scoped.
  - Result: public benchmark methodology now explains why duplicate or near-duplicate content in another conversation cannot win by content-only matching before stable-ID scoring.

- 2026-05-22: public release metadata smoke now checks documented latest metadata.
  - Evidence target: `go test . -run 'Test(PublicReleaseSmokeChecksDocumentedLatestMetadata|PublicDocsExplainDocumentedLatestPublicReleaseSmoke)' -count=1` proves `make public-release-smoke` checks the documented public `@latest` version and published date, and first-touch public docs explain that guard.
  - Result: ecosystem-readiness smoke now catches drift between official public module metadata and the documented v0.1.1 / May 22, 2026 milestone instead of accepting any `Version`/`Time` fields.

- 2026-05-22: first-touch public docs now surface the public docs site smoke.
  - Evidence target: `go test . -run 'Test(DocsSiteSmokeBuildsPublicDocs|PublicDocsMentionDocsSiteSmoke)' -count=1` proves `make docs-site-smoke` checks the local docs-site build with `npm run build`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the public docs site signal without claiming local smoke proves remote hosting or indexing.

- 2026-05-22: first-touch public docs now surface the package documentation smoke.
  - Evidence target: `go test . -run 'Test(PackageDocSmokeChecksLocalGoDoc|PublicDocsMentionPackageDocSmoke)' -count=1` proves `make package-doc-smoke` checks local package docs with `go doc .`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the package documentation signal without claiming that local smoke proves remote pkg.go.dev indexing.

- 2026-05-22: first-touch public docs now surface the local go.mod metadata smoke.
  - Evidence target: `go test . -run 'Test(LocalModuleSmokeChecksGoModMetadata|PublicDocsMentionLocalModuleSmoke)' -count=1` proves `make local-module-smoke` checks the local module path and Go version with `go list -m -json`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the valid Go module signal without conflating it with public `@latest` metadata or root CLI installability.

- 2026-05-22: first-touch public docs now surface the external backend comparison smoke.
  - Evidence target: `go test . -run TestPublicDocsMentionBackendComparisonSmoke -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `make bench-locomo-backends-smoke`.
  - Result: ecosystem-readiness docs now expose the CI-safe local proof command for external adapter comparison without rerunning or changing frozen benchmark artifacts.

- 2026-05-22: first-touch public docs now surface the external adapter contract.
  - Evidence target: `go test . -run TestPublicDocsSurfaceExternalAdapterContract -count=1` proves README, docs home, current-capabilities, and quick-start docs mention the external adapter contract and current agentmemory PR #583 stable-ID status.
  - Result: ecosystem-readiness docs now expose adapter/upstream credibility at adoption time without overstating backend scores or root CLI installability.

- 2026-05-22: first-touch public docs now link benchmark methodology.
  - Evidence target: `go test . -run TestPublicDocsLinkRetrievalBenchmarksReference -count=1` proves README, docs home, current-capabilities, and quick-start docs link the Retrieval Benchmarks reference.
  - Result: public package adoption now exposes deterministic benchmark methodology and stable-ID backend comparison evidence without making benchmark claims in setup prose.

- 2026-05-22: public docs now warn against root-level `go install` overclaims.
  - Evidence target: `go test . -run TestPublicDocsWarnRootGoInstallIsUnsupported -count=1` proves README, docs home, current-capabilities, and quick-start docs say the root module is not a root `go install` target.
  - Result: first-touch docs preserve the `go get github.com/TrebuchetDynamics/goncho@latest` library path without implying an unavailable root CLI install.

- 2026-05-22: public docs now surface the v0.1.1 published date.
  - Evidence target: `go test . -run TestPublicDocsMentionPublishedReleaseDate -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `published May 22, 2026`.
  - Result: first-touch docs now show both public version and published-date signals from the official module metadata.

- 2026-05-22: public adoption docs now use version-qualified `go get`.
  - Evidence target: `go test . -run TestPublicDocsUseLatestQualifiedGoGet -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `go get github.com/TrebuchetDynamics/goncho@latest`.
  - Result: first-touch setup guidance matches the public `@latest` release signal while keeping the root module framed as a library package.

- 2026-05-22: public release metadata smoke added.
  - Evidence target: `make public-release-smoke` checks `go list -m -json github.com/TrebuchetDynamics/goncho@latest` for public version and published-time metadata.
  - Result: the pkg.go.dev-style `Version` and `Published` signal is locally checkable before broader ecosystem smoke and release decisions.

- 2026-05-22: docs home now frames the root module as a library package.
  - Evidence target: `go test . -run TestPublicDocsFrameRootModuleAsLibrary -count=1` proves README, docs home, current-capabilities, and quick-start docs say the root module is a library package.
  - Result: first-touch public docs preserve `go get` library semantics and avoid implying root-level CLI installability.

- 2026-05-22: docs home now names the current public `@latest` release as v0.1.1.
  - Evidence target: `go test . -run TestPublicDocsMentionLatestReleaseVersion -count=1` proves README, docs home, current-capabilities, and quick-start docs mention v0.1.1.
  - Result: first-touch public docs show the official tagged release signal and public benchmark CLI availability at `@latest`.

- 2026-05-22: README and docs home now expose the narrower public module smoke.
  - Evidence target: `go test . -run TestPublicAdoptionDocsMentionPublicModuleSmoke -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `make public-module-smoke`.
  - Result: public adoption docs separate the broad ecosystem smoke from the external-import-only proof for `github.com/TrebuchetDynamics/goncho@latest`.

- 2026-05-22: docs home now surfaces local ecosystem smoke.
  - Evidence target: `go test . -run TestPublicDocsMentionEcosystemSmoke -count=1` proves README, docs home, operator runbook, current-capabilities, and quick-start docs mention `make ecosystem-smoke`.
  - Result: public adoption docs expose the local proof command for module resolution, package docs, external importability, and checkout-local benchmark CLI readiness.

- 2026-05-22: docs home and quick-start docs now link the public Go reference.
  - Evidence target: `go test . -run TestPublicDocsLinkGoReference -count=1` proves README, docs home, current-capabilities, and quick-start docs link `https://pkg.go.dev/github.com/TrebuchetDynamics/goncho`.
  - Result: public adoption docs surface pkg.go.dev API reference at first use instead of hiding it in status pages.

- 2026-05-21: operator-facing release smoke docs now mention the release metadata guard.
  - Evidence target: `go test . -run TestReleaseSmokeDocsMentionMetadataGuard -count=1` proves README, quick-start, and runbook release-smoke guidance mention release metadata checks.
  - Result: public docs stay aligned with the local pre-tag gate instead of describing only ecosystem smoke plus Go/docs checks.

- 2026-05-22: release metadata now has an explicit smoke target.
  - Evidence target: `make release-metadata-smoke` runs tag/changelog consistency and release-smoke docs drift tests before broader release checks.
  - Result: operators can check changelog/tag consistency and release-smoke docs directly, and `make release-smoke` includes that guard before ecosystem validation.

- 2026-05-22: changelog release headings are now guarded against untagged version overclaims.
  - Evidence target: `go test . -run TestChangelogReleaseHeadingsHaveMatchingTags -count=1` proves each `## vX.Y.Z - ...` changelog release heading has a matching local git tag.
  - Result: public release notes can keep candidate notes without implying that untagged versions are already published.

- 2026-05-22: blank `goncho_review` list status values now default to open review items.
  - Evidence target: `go test . -run TestReviewToolTreatsBlankStatusAsOpenDefault -count=1` proves whitespace-only `status` behaves like omitted `status` and does not leak resolved items into the default review queue.
  - Result: review queue inspection is safer when host/tool callers pass blank form values instead of omitting optional fields.

- 2026-05-22: invalid `goncho_review` resolve resolution values now return enum-specific guidance.
  - Evidence target: `go test . -run TestReviewToolRejectsInvalidResolveResolution -count=1` proves an invalid `resolution` value is rejected without closing the open review item.
  - Result: lifecycle review queues are safer when host/tool callers bypass schema enum validation.

- 2026-05-22: same-timestamp review item ID collision fixed.
  - Evidence target: `go test . -run TestCreateReviewItemAllowsDistinctItemsWithSameCreatedAt -count=1` proves two distinct review items sharing one `CreatedAt` get distinct IDs and remain listable.
  - Result: review queues are safer when lifecycle scanners create multiple findings in the same timestamp bucket.

- 2026-05-22: `goncho_review` list filter validation added.
  - Evidence target: `go test . -run TestReviewToolRejectsInvalidListFilters -count=1` proves invalid `status` and `kind` list filters return operator-visible errors instead of empty review queues.
  - Result: review queue inspection is safer when host/tool callers bypass schema enum validation.

- 2026-05-22: local release smoke added.
  - Evidence target: `make release-smoke` runs `make ecosystem-smoke`, `go test ./...`, `go vet ./...`, `go test -race ./...`, and the docs-site build.
  - Result: next v0.1.x prep has one local pre-tag command without claiming CI or creating a tag.

- 2026-05-22: `goncho_review` review-chain filters added.
  - Evidence target: `go test . -run TestReviewToolFiltersReviewChainsBySubjectAndRelatedID -count=1` proves `subject_id` plus `related_id` narrows open review items to one matching chain edge.
  - Result: review/staleness/supersession items are easier to inspect without losing historical evidence.

- 2026-05-22: ecosystem smoke added for core public release-readiness signals.
  - Evidence target: `make ecosystem-smoke` runs public module resolution, local `go doc .`, external import smoke, and checkout-local benchmark CLI installation.
  - Result: the milestone now has one operator command for library importability plus local benchmark CLI readiness without overstating `cmd/goncho-bench@latest`.

- 2026-05-22: public module adoption smoke added for `github.com/TrebuchetDynamics/goncho@latest`.
  - Evidence target: `make public-module-smoke` creates a temporary external Go module, runs `go get github.com/TrebuchetDynamics/goncho@latest`, and compiles a minimal public API import.
  - Result: release readiness now separates library importability proof from the still-checkout-local benchmark CLI.

- 2026-05-22: public `@latest` now resolves to v0.1.1, so `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` is available.
  - Evidence target: `GOBIN=$(mktemp -d) go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` verifies the public benchmark CLI installs from the published tag.
  - Result: docs now point benchmark CLI users at the public `go install .../cmd/goncho-bench@latest` path while keeping checkout-local `make install-smoke` as a local verification path.

- 2026-05-22: generated primer/token-budget E2E coverage added for the public `goncho_context` tool.
  - Focused evidence: `go test . -run TestGonchoGoalPublicContextToolGeneratesPrimerWithinTokenBudgetE2E -count=1` passed.
  - Result: public context-tool coverage now proves generated orientation output preserves the newest in-budget turns and excludes older turns outside `max_tokens`.

- 2026-05-21 20:11 CST: stale `cmd/goncho-bench` expectation-drift blocker from 2026-05-20 is resolved on current `main`.
  - Focused evidence: `go test ./cmd/goncho-bench -run 'TestClassifyFailureCasesSelectsHardRanksAndCategories|TestWriteFailureCategoryReportsEmitsJSONLAndMarkdown' -count=1` passed.
  - Full Go evidence: `go test ./... -count=1` passed.
  - Result: benchmark classifier expectation drift no longer blocks Go verification.

- 2026-05-19: stale full-verification blocker resolved. Current release gate passes with:
  - `go test ./integration/gormes`
  - `go test ./...`
  - `cd docs-site && npm run build`

## Next roadmap items

- Continue lifecycle trust work: temporal validity, supersession chains, and confidence/freshness scoring.
- Expand graph/cognitive-map features behind deterministic tests.
- Add optional PostgreSQL/team adapter only after local SQLite API remains stable.
