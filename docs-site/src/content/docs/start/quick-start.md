---
title: Quick Start
description: Install Goncho and wire the current Go service API.
---

Goncho is a Go library for local-first agent memory and context assembly.

```sh
go get github.com/TrebuchetDynamics/goncho
```

:::caution[Pre-release setup note]
The current README mentions `goncho.RunMigrations`, but the package does not currently export that symbol. This page shows the current service integration shape and calls out the missing public schema initialization helper instead of pretending the example is fully copy-paste runnable.
:::

## Minimal Service Shape

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/goncho"
)

func main() {
	db, err := sql.Open("sqlite3", "memory.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Pre-release note: this package does not currently expose a public
	// schema migration helper. The database must be initialized by the host
	// runtime until Goncho exposes that helper.

	svc := goncho.NewService(db, goncho.Config{
		WorkspaceID:    "my-agent",
		ObserverPeerID: "assistant",
		RecentMessages: 4,
	}, nil)

	ctx := context.Background()

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
- `SetProfile` writes stable peer-card facts.
- `Profile` reads the peer card back.
- `Context` returns an orientation product for prompt construction.

Because Goncho is pre-release, pin the module version or commit you deploy against.
