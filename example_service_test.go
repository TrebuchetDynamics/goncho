package goncho_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

func ExampleNewService() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goncho-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = store.Close(ctx) }()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "example-agent",
		ObserverPeerID: "assistant",
	}, nil)

	if err := svc.SetProfile(ctx, "user:juan", []string{
		"Prefers SQLite-backed local memory.",
	}); err != nil {
		panic(err)
	}

	profile, err := svc.Profile(ctx, "user:juan")
	if err != nil {
		panic(err)
	}
	fmt.Println(profile.Card[0])

	// Output:
	// Prefers SQLite-backed local memory.
}

func ExampleService_Context() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goncho-context-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = store.Close(ctx) }()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "example-agent",
		ObserverPeerID: "assistant",
	}, nil)

	if err := svc.SetProfile(ctx, "user:juan", []string{
		"Prefers verification before action.",
	}); err != nil {
		panic(err)
	}
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "user:juan",
		Conclusion: "Use SQLite local memory for agent handoffs.",
	}); err != nil {
		panic(err)
	}

	orientation, err := svc.Context(ctx, goncho.ContextParams{
		Peer:      "user:juan",
		Query:     "handoffs",
		MaxTokens: 2000,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(orientation.Representation)

	// Output:
	// Representation for user:juan:
	//
	// Profile facts:
	// - Prefers verification before action.
	//
	// Current conclusions:
	// - Use SQLite local memory for agent handoffs.
}

func ExampleService_Search() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goncho-search-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = store.Close(ctx) }()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "example-agent",
		ObserverPeerID: "assistant",
	}, nil)

	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "user:juan",
		Conclusion: "Keep deployment notes evidence-first and searchable.",
	}); err != nil {
		panic(err)
	}

	results, err := svc.Search(ctx, goncho.SearchParams{
		Peer:  "user:juan",
		Query: "deployment evidence",
		Limit: 1,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: %s\n", results.Results[0].Source, results.Results[0].Content)

	// Output:
	// conclusion: Keep deployment notes evidence-first and searchable.
}

func ExampleService_Recall() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goncho-recall-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = store.Close(ctx) }()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "example-agent",
		ObserverPeerID: "assistant",
	}, nil)

	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "user:juan",
		Conclusion: "Recall traces preserve scoring provenance for deployment notes.",
	}); err != nil {
		panic(err)
	}

	trace, err := svc.Recall(ctx, goncho.RecallQuery{
		Peer:  "user:juan",
		Query: "scoring provenance",
		Limit: 1,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s selected=%d provenance=%t\n", trace.PipelineVersion, len(trace.Selected), len(trace.Selected[0].Candidate.Provenance) > 0)

	// Output:
	// goncho-recall-v1 selected=1 provenance=true
}
