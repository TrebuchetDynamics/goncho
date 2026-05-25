# goncho-server

`goncho-server` is the first standalone local runtime surface for Goncho.

## Onboarding

```bash
go run ./cmd/goncho-server onboarding -config ./goncho-server.json -db ./goncho.db -addr 127.0.0.1:8765
```

The onboarding command prints first-run guidance without mutating files: DB path, config path, loopback bind address, MCP URL, copy-paste commands, and host-hook guidance.

## Init

```bash
go run ./cmd/goncho-server init -config ./goncho-server.json -db ./goncho.db
```

The init command creates the SQLite DB path, runs migrations, and writes a local JSON config. It refuses to overwrite an existing config file.

## Health

```bash
go run ./cmd/goncho-server health -db ./goncho.db
```

The command opens the SQLite store, runs Goncho migrations, and prints JSON with:

- overall status;
- module version;
- database path/status;
- migration status;
- public tool availability;
- optional provider diagnostics for extraction, embedding, reranking, and summarization adapters;
- DB/image/vector disk usage and optional over-budget flags when `-max-*-bytes`, `-image-dir`, or `-vector-dir` are supplied.

## Doctor

```bash
go run ./cmd/goncho-server doctor -db ./goncho.db -addr 127.0.0.1:8765
```

The doctor command prints JSON checks for the DB path, migrations, write permissions, port availability, disk usage, public tool registration, and optional-provider degradation. Failed checks include copy-paste suggestions; doctor does not apply connector, provider, retention, or host configuration fixes.

## Demo

```bash
go run ./cmd/goncho-server demo -db ./goncho.db
```

The demo command seeds one tiny project-memory fact, then proves both recall and context can retrieve it. It exits non-zero if either proof fails.

## Security requirements

```bash
go run ./cmd/goncho-server security
```

The security command prints Goncho's requirements-only server-mode threat model summary: auth, profiles, workspaces, audit, backup, retention, admin operations, roles, PostgreSQL adapter conformance, and snapshot-manifest backup/restore requirements. It is non-mutating and does not enable shared/team authorization yet.

## Serve

```bash
go run ./cmd/goncho-server serve -db ./goncho.db
```

By default the server binds to `127.0.0.1:8765` only. Use `-addr` to choose another explicit local address. Non-loopback binds fail closed unless `-auth-token` is supplied; token enforcement for shared/team mode remains future work, so prefer loopback until server-mode ACLs are implemented.

Endpoints:

- `GET /health` — JSON health report.
- `POST /mcp` — JSON-RPC MCP-compatible transport with `initialize`, `ping`, `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list`, and `prompts/get` for Goncho public tools/resources/prompts. `goncho-server stdio` exposes the same request handling over newline-delimited stdio JSON-RPC.
- `/v3/...` — existing local Goncho HTTP adapter routes from `github.com/TrebuchetDynamics/goncho/http`, including read-only `GET /v3/workspaces/{workspace}/viewer` snapshots and `GET /v3/workspaces/{workspace}/viewer/sessions/{session}/timeline` timelines for local viewer clients.
