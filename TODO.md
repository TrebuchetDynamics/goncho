# Goncho TODO

## Release state

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
