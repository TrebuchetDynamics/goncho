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
// From there, call Service methods such as Conclude, Search, Context, Chat,
// and Profile, or expose the public tools for context, search, remember,
// review, and handoff workflows.
//
// Goncho is pre-1.0. Pin the module version or commit you deploy against, keep
// live verification in the host, and treat retrieved memory as orientation
// until current evidence confirms it.
package goncho
