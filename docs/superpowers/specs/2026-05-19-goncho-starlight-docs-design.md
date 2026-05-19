# Goncho Starlight Documentation Site

Date: 2026-05-19

## Background

Goncho needs public documentation that explains both the current Go library and the architecture philosophy emerging around it. The current README positions Goncho as a Honcho-compatible, local-first memory system for Go agents, but the stronger product direction is trust-preserving context architecture: preserve evidence, derive scoped beliefs, and return compact orientation instead of dumping memory into prompts.

The docs should make that direction legible without overstating the current implementation. Goncho is pre-release. The site must distinguish shipped behavior from architecture direction.

## Goals

- Add an Astro Starlight documentation site for Goncho.
- Keep the docs site isolated from the Go module root.
- Present Goncho as trust-preserving context for Go agents, not a generic memory store or vector database.
- Give developers a short path to install and try the current Go library.
- Explain the architecture vocabulary: evidence, claims, beliefs, scope, orientation, review, repair, and negative memory.
- Document current public surfaces at a conceptual level and link exact Go signatures to `pkg.go.dev`.
- Keep v1 buildable locally without committing to hosting, CI, generated API docs, or a custom visual system.

## Non-Goals

- Do not rewrite Goncho's Go implementation.
- Do not add generated Go API reference docs.
- Do not publish database schema documentation.
- Do not expose the raw `docs/opensource-memory-systems/` research corpus in public nav.
- Do not add hosting-specific deployment config, custom domains, CI workflows, or a root Node workspace.
- Do not add screenshots, custom logo work, or a bespoke Starlight theme.
- Do not claim shipped support for first-class claims, quantitative confidence, temporal validity fields, review workflows, or memory repair unless current exported code supports them.

## Site Location

Create an isolated Astro Starlight project under `docs-site/`:

```text
docs-site/
  README.md
  package.json
  package-lock.json
  astro.config.mjs
  src/
    content.config.ts
    content/
      docs/
        index.md
        start/
        concepts/
        reference/
        roadmap/
```

Keep all Node dependency state inside `docs-site/`. Do not add a root `package.json`, root lockfile, or workspace config.

## Starlight Setup

Use the current Starlight conventions:

- configure the Starlight integration in `astro.config.mjs`;
- configure the docs content collection in `src/content.config.ts`;
- store docs pages under `src/content/docs/`;
- configure sidebar groups explicitly instead of relying on implementation-module ordering.

Use mostly default Starlight behavior: sidebar, local search, code blocks, table of contents, dark/light support, and default responsive layout. Configure title and description only. Leave `site` unset until a canonical deployment URL exists.

## Identity

Use this short positioning line:

```text
Trust-preserving context for Go agents.
```

Use this longer positioning line where more detail is needed:

```text
Goncho is a local-first context architecture for Go agents that preserves evidence, derives scoped beliefs, and returns compact orientation instead of dumping memory into prompts.
```

Use `memory` as a bridge term for discoverability, but teach the sharper distinction that memory is state carried forward over time, not just retrieval.

## Navigation

The v1 sidebar should follow the reader's learning path:

```text
Start
  Quick Start
  Current Capabilities

Concepts
  Trust-Preserving Context
  Local-First Memory
  Evidence, Claims, and Beliefs
  Session Lifecycle
  Orientation Packs
  Negative Memory
  Design Boundaries
  Glossary

Reference
  Core API
  Local Markdown Memory
  Memory Tools
  Honcho Compatibility

Roadmap
  Architecture Direction
```

Do not add maintainer internals, cookbook pages, deployment pages, generated API reference, or database schema reference in v1.

## Homepage

The homepage should optimize for orientation in the first ten seconds:

- state that Goncho is a Go library for local-first agent memory;
- state the deeper thesis: trust-preserving context, not "RAG with SQLite";
- show two clear paths: start using Goncho, understand the architecture;
- include a small text architecture flow:

```text
raw evidence -> claims -> scoped beliefs -> orientation -> action -> revision
```

The homepage should be thesis-led, not a glossy marketing page. Use sober titles and sharp opening claims.

## Page Rules

Use plain Markdown files for v1. Use MDX only later if custom components become necessary.

Use Starlight callouts sparingly with a small recurring set:

- Current status
- Design constraint
- Failure mode
- Conceptual example
- Pre-release note

Use text diagrams where they clarify architecture. Do not add heavy visual assets or screenshots.

Use mostly impersonal technical voice:

- "Goncho treats memory as scoped belief over evidence."
- "Vector search can help retrieval, but it does not define memory."
- "Context products should orient the agent."

Avoid manifesto-heavy language and avoid overusing "we believe."

## Current vs Direction

Every architecture page must avoid collapsing implemented behavior and future direction.

Current shipped capability may include:

- SQLite-backed persistence;
- peer cards;
- search;
- context assembly;
- session summaries;
- local markdown memory;
- Honcho/MCP compatibility where verified from current code.

Architecture direction may include:

- first-class claims and evidence lineage;
- qualitative confidence and staleness;
- scoped temporal beliefs;
- negative-memory review;
- structured review and repair workflows;
- handoff products;
- richer consolidation hooks.

Do not use fake exported field names such as `valid_from` or `valid_until`. Do not use numeric confidence examples such as `0.82` unless a stable scoring contract exists.

## Examples

Current API examples must compile against the current Go code or be clearly marked conceptual.

`Quick Start` should include one complete minimal Go program:

- import Goncho from `github.com/TrebuchetDynamics/goncho`;
- import the SQLite driver already used by the repo;
- open a SQLite database;
- run migrations;
- construct `goncho.Service` with real current `Config` fields;
- call `SetProfile`;
- call `Profile`;
- call `Context` as the orientation payoff.

Use compact but real error handling. Do not normalize ignored errors in trust-oriented examples.

Add sample output only if it can be verified from the current code. If output cannot be verified cleanly, describe the result conceptually.

Because Goncho is pre-release, include one dependency-policy sentence: users should pin the module version or commit they deploy against.

## Reference Scope

Reference pages should explain the stable human model of the API and link to `pkg.go.dev` for exact signatures.

`Core API` explains how `Profile`, `SetProfile`, `Search`, `Context`, and session lifecycle methods map to Goncho's mental model.

`Local Markdown Memory` documents the human-editable memory surface as a shipped trust and repair primitive, after verifying current behavior.

`Memory Tools` documents generic memory tools if their exported names and schemas are verified.

`Honcho Compatibility` documents Honcho-compatible tool/API concepts as a migration bridge, not as Goncho's identity.

Do not duplicate generated Go API docs or expose SQLite table contracts.

## Tone About Alternatives

Be blunt about abstractions and restrained about named competitors.

Acceptable claims:

- Vector search is useful, but it is not memory.
- Context stuffing is not orientation.
- Summaries without evidence lineage become unverifiable.
- Globally true forever memories corrupt long-running agents.

Avoid turning the docs into a competitor attack page. Keep named comparisons minimal in v1.

Do not say Goncho replaces RAG. Say Goncho changes the memory abstraction and can coexist with retrieval pipelines.

## Operational Boundaries

Document practical local-first and security boundaries without compliance claims:

- memory is stored locally by default;
- SQLite and markdown files are application data and should be protected, backed up, and migrated according to the host application's policy;
- optional adapters may send data outside the process depending on adapter behavior;
- users should avoid storing secrets unless their host application has an explicit policy;
- MCP/tool integrations should respect host permissions and peer/workspace isolation.

Do not add a formal threat model in v1.

## Implementation Verification

Verify the docs site with:

```sh
cd docs-site
npm run build
```

Verify the Quick Start example against current Go symbols before publishing the page.

Do not claim `go test ./...` passes as part of this docs task. The current repo fails setup because `proof_matrix_test.go` imports `github.com/TrebuchetDynamics/gormes-agent/internal/transcript`, which is not available from this module.

## README Boundary

The Starlight site becomes the canonical long-form docs. The root `README.md` remains the concise project entrypoint.

During v1 implementation, touch `README.md` only to fix confusing or broken docs references, especially the missing migration guide reference. Avoid a broad README rewrite until the docs site exists.

## Acceptance Criteria

- `docs-site/` contains an isolated Astro Starlight project.
- The Starlight site builds locally with `npm run build`.
- The sidebar follows the approved `Start`, `Concepts`, `Reference`, and `Roadmap` structure.
- v1 pages distinguish current implementation from architecture direction.
- Quick Start uses current Goncho symbols and the repo's SQLite driver.
- No deployment, CI, generated API docs, schema docs, screenshots, or custom theme work is added in v1.
- Any README change is minimal and directly tied to the new docs site.
