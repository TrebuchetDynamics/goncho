// Package goncho provides high-trust local memory for Go-native AI agent
// runtimes.
//
// Goncho is an embedded memory kernel, not a hosted memory service. It stores
// local evidence, derives scoped recall, assembles context, records review
// signals, and helps callers verify remembered claims before an agent acts on
// them. The core operating rule is evidence before belief and verification
// before action: memory can orient an agent, but current evidence decides
// whether an action is safe.
//
// Use Goncho when an agent host needs durable local state, auditable recall,
// scoped peer/session memory, review queues, stale-claim warnings, and
// deterministic benchmark evidence without a Python service, Docker sidecar,
// hosted vector database, or always-online memory API.
//
// Install the library with:
//
//	go get github.com/TrebuchetDynamics/goncho/service@latest
//
// The service package is a library package, not a root go install target. To
// install the reproducible retrieval benchmark CLI, use:
//
//	go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest
//
// Import path guide:
//
//   - github.com/TrebuchetDynamics/goncho/service is the service library package for
//     RunMigrations, NewService, service params, and public tool constructors.
//   - github.com/TrebuchetDynamics/goncho/memory is the SQLite opener for
//     memory.OpenSqlite when an embedded host wants a local file-backed store.
//   - github.com/TrebuchetDynamics/goncho/cmd/goncho-bench is command-only;
//     do not import cmd/goncho-bench into an agent host.
//
// When in doubt, stay on public service and tool APIs before reaching for
// lower-level storage or benchmark internals.
//
// Trust boundary for host agents:
//
// Goncho can orient the agent by storing evidence, ranking scoped memory,
// assembling context packs, and warning when remembered claims may be stale.
// The host remains authoritative for decisions that require current state or
// external authority.
//
//   - Authorization and policy decisions still belong to the host runtime,
//     gateway, or operator.
//   - Live filesystem, API, deployment, and credential state must be checked
//     at action time.
//   - Money movement, destructive writes, and external side effects require
//     explicit host-side gates.
//   - Treat retrieved memory as evidence to check, not as permission to skip
//     live verification.
//
// Quick start:
//
//	store, err := memory.OpenSqlite("goncho.db", 0, nil)
//	if err != nil {
//		return err
//	}
//	defer store.Close(ctx)
//
//	if err := goncho.RunMigrations(store.DB()); err != nil {
//		return err
//	}
//
//	svc := goncho.NewService(store.DB(), goncho.Config{
//		WorkspaceID:    "local-agent",
//		ObserverPeerID: "agent",
//	}, nil)
//
// Host integration checklist:
//
//   - Open local SQLite with memory.OpenSqlite and close the store during host shutdown.
//   - RunMigrations before NewService on every boot so the database matches the service schema.
//   - Set WorkspaceID and ObserverPeerID so memory, reviews, and audits are attributable.
//   - Pass ProfileID, Peer, and SessionKey explicitly when the host has profile or session routing.
//   - Call Service.Context before tool execution to build orientation, then let the host verify live state.
//   - Store evidence-backed conclusions after observations, user-visible decisions, or verified tool results.
//   - Verify live state before acting: paths, APIs, credentials, deployments, and services still need current proof.
//
// Primary API path:
//
//   - Service.Conclude records evidence-backed conclusions.
//   - Service.Search retrieves scoped memory candidates.
//   - Service.Recall returns a scored RecallTrace with provenance, warnings,
//     and selection/rejection reasoning before projection.
//   - Service.Context assembles an orientation pack for the next action.
//   - Service.Profile stores and reads durable profile facts.
//   - Service.ExtractMemoryProposals inspects a bounded session window and
//     returns add/update/supersede/delete/noop proposals with message evidence;
//     review-required proposals are queued without writing active memory.
//   - Service.ProviderHealthDiagnostics reports optional extraction, embedding,
//     reranking, and summarization provider state. Semantic provider failures
//     degrade recall with warnings while lexical/local evidence remains usable.
//   - Service.PreviewRetention and Service.ApplyRetention provide non-destructive
//     retention planning and audited archive/tombstone application; archived
//     conclusions keep stable IDs but are excluded from active recall.
//   - Service.ExportPortableJSONL, Service.PreviewPortableImport,
//     Service.ImportPortableJSONL, and Service.ExportPortableMarkdown provide
//     portable local mirrors with checksummed manifests and preview-first import.
//   - Service.RecordEvalFailures and Service.RecordRecallFeedback turn benchmark
//     misses and runtime labels into reviewable improvement evidence without
//     promoting claims; EvaluateRegressionGate enforces deterministic tolerances.
//   - Service.AcquireActionLease, Service.RenewActionLease,
//     Service.ExpireActionLeases, and Service.ListActionLeaseAudit provide
//     local server-mode coordination primitives with TTLs, owner checks, and
//     audit-visible allow/deny/expire evidence.
//   - Service.RecordActionSignalReceipt, Service.ListActionSignalReceipts, and
//     Service.ListActionSignalReceiptAudit add read receipts to action signals
//     with observable workspace/profile authorization decisions.
//   - Service.TeamFeed and Service.ListTeamFeedAudit expose a read-only,
//     paginated team feed over authorized action signals with observable ACL
//     allow/deny evidence.
//   - Service.PreviewFilesystemWatcherImport and
//     Service.ImportFilesystemWatcherChanges let local watcher connectors import
//     changed project docs/code as scoped observations only after explicit
//     include/exclude rules select the files.
//   - ServerModeSecurityRequirements exposes the requirements-only threat model
//     and RBAC vocabulary future shared/team mode must satisfy without enabling
//     network sharing or weakening local-first SQLite mode.
//
// For host integrations, prefer the public tool constructors such as
// NewGonchoContextTool, NewGonchoSearchTool, NewGonchoRecallTool,
// NewGonchoRememberTool, NewReviewTool, and NewGonchoHandoffTool so callers stay on the public
// boundary instead of database internals.
//
// On pkg.go.dev, use the rendered pkg.go.dev examples as the shortest checked
// path through the API: ExampleNewService shows setup, ExampleService_Context
// shows orientation-pack assembly, ExampleService_Search shows scoped retrieval
// against stored conclusions, and ExampleService_Recall shows auditable recall
// traces.
//
// go.dev package signals to check before adopting: the public module is
// currently v0.2.0, has a valid go.mod, a redistributable MIT license, and
// package documentation. Use make package-doc-smoke for this overview and its
// examples, make public-module-smoke for external imports, and make
// install-smoke for the cmd/goncho-bench command path.
//
// Versioning and adoption notes: Goncho is pre-1.0, so read the go.dev Stable
// version signal as not yet v1-stable. For reproducible builds, pin with
// go get github.com/TrebuchetDynamics/goncho/service@v0.2.0 or a reviewed commit;
// @latest is a discovery shortcut, not a deployment lock. pkg.go.dev currently
// shows Imported by 0, but that reverse-dependency count is adoption context,
// not a correctness gate. Before upgrading a pinned host, run make
// ecosystem-smoke from a checkout.
//
// Goncho is pre-1.0. Pin the module version or commit you deploy against, keep
// live verification in the host, and treat retrieved memory as orientation
// until current evidence confirms it.
package goncho
