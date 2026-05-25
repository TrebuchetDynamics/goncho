# goncho

`goncho` is the top-level operator CLI for productized Goncho workflows.

## Schema fingerprint

```bash
go run ./cmd/goncho schema-fingerprint --json
```

The schema fingerprint is non-mutating drift metadata for adapters: SQLite schema version, public tool names/count, host hook event names, and a SHA-256 fingerprint over that contract payload. Connector scripts can compare it before writing host config.

## Upgrade check

```bash
go run ./cmd/goncho upgrade-check --json --current v0.2.0 --latest v0.2.1
```

The upgrade check is non-mutating. It compares the current version with a trusted latest version supplied by release automation or an operator. Without `--latest`, it reports `unknown` and points the operator to GitHub releases/pkg.go.dev rather than guessing from the network.

## Doctor

```bash
go run ./cmd/goncho doctor --json --db ~/.local/share/goncho/goncho.db
```

The doctor command prints non-mutating local diagnostics for the configured DB path, migration presence, preferences file readability, and public tool registration. Missing DBs are reported with copy-paste suggestions; doctor does not create or migrate files.

## Version metadata

```bash
go run ./cmd/goncho version --json
```

The command prints non-mutating JSON with Goncho module version, git commit when available from Go build metadata, SQLite schema version, and public tool count. Hosts and connector scripts can use it for compatibility checks before writing config.

## Local operator preferences

```bash
go run ./cmd/goncho preferences --config ~/.config/goncho/preferences.json \
  --set db_path=~/.local/share/goncho/goncho.db \
  --set workspace_id=default \
  --set profile_id=operator \
  --set redaction_policy=strict \
  --set connector_permission=plan_only \
  --set bind_addr=127.0.0.1:8765
```

Without `--set`, `preferences` reads the JSON file or prints safe local defaults. With `--set`, it writes only the named preferences to the requested config path.

## Filesystem watcher connector plan

```bash
go run ./cmd/goncho connect filesystem-watcher --plan \
  --watch-root . \
  --include '**/*.md' \
  --include '**/*.go' \
  --exclude '.git/**' \
  --exclude 'node_modules/**'
```

The command prints a non-mutating watcher plan. Include globs are required. A host watcher should call `PreviewFilesystemWatcherImport` first, then `ImportFilesystemWatcherChanges` only after include/exclude rules are reviewed.

## Gormes connector plan

```bash
go run ./cmd/goncho connect gormes --plan \
  --profiles-dir .gormes/profiles \
  --profile mineru \
  --workspace gormes \
  --observer gormes
```

The command prints a JSON plan and does not mutate files. It derives the profile-local Goncho database path, markdown memory mirror path, public tool names, and supported host hook events.

Use `goncho remove gormes --plan` for the reversible disconnect plan. `--dry-run` remains accepted as a compatibility alias for older scripts.

## Codex connector plan

```bash
go run ./cmd/goncho connect codex --plan \
  --config ~/.codex/config.toml \
  --addr 127.0.0.1:8765
```

The command prints the TOML MCP server patch Codex would need, the generated hook events Goncho can map, and the local `goncho-server serve` command. It does not create `~/.codex` or write `config.toml`. Use `goncho remove codex --plan` to print the matching removal instructions.

## Pi connector plan

```bash
go run ./cmd/goncho connect pi --plan \
  --config ~/.pi/agent/settings.json \
  --extension ~/.pi/agent/extensions/goncho \
  --addr 127.0.0.1:8765
```

The command prints the Pi settings JSON patch, planned TypeScript extension file paths, generated hook events Goncho can map, and the local `goncho-server serve` URL. It follows Pi's documented extension locations and does not create `~/.pi`, copy extension files, or write `settings.json`. Use `goncho remove pi --plan` to print the matching removal instructions.

`--apply` is intentionally rejected until generated connector plans have golden-file tests and host-level smoke coverage.
