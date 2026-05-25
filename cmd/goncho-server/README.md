# goncho-server

`goncho-server` is the first standalone local runtime surface for Goncho.

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
- public tool availability.

## Doctor

```bash
go run ./cmd/goncho-server doctor -db ./goncho.db -addr 127.0.0.1:8765
```

The doctor command prints JSON checks for the DB path, migrations, write permissions, port availability, and public tool registration.

## Demo

```bash
go run ./cmd/goncho-server demo -db ./goncho.db
```

The demo command seeds one tiny project-memory fact, then proves both recall and context can retrieve it. It exits non-zero if either proof fails.

## Serve

```bash
go run ./cmd/goncho-server serve -db ./goncho.db
```

By default the server binds to `127.0.0.1:8765` only. Use `-addr` to choose another explicit local address.

Endpoints:

- `GET /health` — JSON health report.
- `POST /mcp` — minimal JSON-RPC MCP-compatible transport with `initialize`, `ping`, `tools/list`, and `tools/call` for Goncho public tools.
- `/v3/...` — existing local Goncho HTTP adapter routes from `github.com/TrebuchetDynamics/goncho/http`.
