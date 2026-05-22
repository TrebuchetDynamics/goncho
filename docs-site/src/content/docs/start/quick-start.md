---
title: Quick Start
description: Use Goncho as a Go module and wire the current service API.
---

Goncho is a Go library for local-first agent memory and context assembly.

Use the module in an embedded Go runtime:

```sh
go get github.com/TrebuchetDynamics/goncho
```

From a checkout, verify the public module, local package docs, external import path, and benchmark CLI together:

```sh
make ecosystem-smoke
```

For a narrower external-import-only check, run `make public-module-smoke`.

From a checkout, verify the benchmark CLI only when you need reproducible local retrieval reports:

```sh
make install-smoke
```

The root module is a library package; `goncho-bench` is the installable command in `./cmd/goncho-bench`. Public `@latest` currently resolves to v0.1.0, so use the checkout-local command until the next v0.1.x tag contains the benchmark CLI.

:::note[Pre-1.0 note]
Goncho is pre-1.0. The setup flow is intentionally small, and operators should pin the module version or commit they deploy against.
:::

## Minimal Service Shape

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

func main() {
	ctx := context.Background()

	store, err := memory.OpenSqlite("memory.db", 0, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			log.Printf("close memory store: %v", err)
		}
	}()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		log.Fatal(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "my-agent",
		ObserverPeerID: "assistant",
		RecentMessages: 4,
	}, nil)

	if err := svc.SetProfile(ctx, "telegram:12345", []string{
		"Works in finance",
		"Prefers SQLite over Postgres",
	}); err != nil {
		log.Fatal(err)
	}

	profile, err := svc.Profile(ctx, "telegram:12345")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("profile facts: %v\n", profile.Card)

	orientation, err := svc.Context(ctx, goncho.ContextParams{
		Peer:      "telegram:12345",
		Query:     "database preferences",
		MaxTokens: 8000,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(orientation.Representation)
}
```

## What This Demonstrates

- `Config.WorkspaceID` keeps one agent runtime from collapsing into another runtime's memory.
- `Config.ObserverPeerID` names the observing agent perspective.
- `memory.OpenSqlite` initializes the service tables used by peer cards, search, context, summaries, and memory tools.
- `RunMigrations` initializes the observation and audit tables.
- `SetProfile` writes stable peer-card facts.
- `Profile` reads the peer card back.
- `Context` returns an orientation product for prompt construction.

The example ends with `Context` because the core payoff is orientation, not just storage.
