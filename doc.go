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
//	go get github.com/TrebuchetDynamics/goncho@latest
//
// The root module is a library package, not a root go install target. To
// install the reproducible retrieval benchmark CLI, use:
//
//	go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest
//
// Import path guide:
//
//   - github.com/TrebuchetDynamics/goncho is the root library package for
//     RunMigrations, NewService, service params, and public tool constructors.
//   - github.com/TrebuchetDynamics/goncho/memory is the SQLite opener for
//     memory.OpenSqlite when an embedded host wants a local file-backed store.
//   - github.com/TrebuchetDynamics/goncho/cmd/goncho-bench is command-only;
//     do not import cmd/goncho-bench into an agent host.
//
// When in doubt, stay on public service and tool APIs before reaching for
// lower-level storage or benchmark internals.
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
//   - Service.Context assembles an orientation pack for the next action.
//   - Service.Profile stores and reads durable profile facts.
//
// For host integrations, prefer the public tool constructors such as
// NewGonchoContextTool, NewGonchoSearchTool, NewGonchoRememberTool,
// NewReviewTool, and NewGonchoHandoffTool so callers stay on the public
// boundary instead of database internals.
//
// On pkg.go.dev, use the rendered pkg.go.dev examples as the shortest checked
// path through the API: ExampleNewService shows setup, ExampleService_Context
// shows orientation-pack assembly, and ExampleService_Search shows scoped
// retrieval against stored conclusions.
//
// go.dev package signals to check before adopting: the public module is
// currently v0.1.1, has a valid go.mod, a redistributable MIT license, and
// package documentation. Use make package-doc-smoke for this overview and its
// examples, make public-module-smoke for external imports, and make
// install-smoke for the cmd/goncho-bench command path.
//
// Versioning and adoption notes: Goncho is pre-1.0, so read the go.dev Stable
// version signal as not yet v1-stable. For reproducible builds, pin with
// go get github.com/TrebuchetDynamics/goncho@v0.1.1 or a reviewed commit;
// @latest is a discovery shortcut, not a deployment lock. pkg.go.dev currently
// shows Imported by 0, but that reverse-dependency count is adoption context,
// not a correctness gate. Before upgrading a pinned host, run make
// ecosystem-smoke from a checkout.
//
// Goncho is pre-1.0. Pin the module version or commit you deploy against, keep
// live verification in the host, and treat retrieved memory as orientation
// until current evidence confirms it.
package goncho
