# Filesystem Watcher Connector

Status: supported-plan

The filesystem watcher connector is local-first and preview-first. It turns changed project docs/code into scoped Goncho observations through public service APIs; it does not mutate source files, install daemons, or broaden recall scope automatically.

## Plan command

```bash
go run ./cmd/goncho connect filesystem-watcher --plan \
  --watch-root . \
  --include '**/*.md' \
  --include '**/*.go' \
  --exclude '.git/**' \
  --exclude 'node_modules/**'
```

The plan reports watch roots plus explicit include/exclude globs. Include globs are required so a watcher cannot silently ingest an entire workspace. If the watcher runs out-of-process, keep `goncho-server` on loopback and route imports through a local host adapter.

## Import flow

1. A host-specific watcher detects a file change.
2. The host calls `Service.PreviewFilesystemWatcherImport` with changed paths, watch root, include globs, exclude globs, peer ID, and session key.
3. The operator/host inspects importable and skipped counts.
4. The host calls `Service.ImportFilesystemWatcherChanges` only after rules are approved.
5. Goncho writes one `custom` observation per changed local text file with metadata: `connector=filesystem_watcher`, `path`, `change_kind`, `checksum`, size, truncation, and source.

## Defaults and safety

Recommended excludes: `.git/**`, `node_modules/**`, `dist/**`, `build/**`, `coverage/**`, `*.log`, and `*.lock`.

Large files are preview-truncated. Binary files are skipped unless a host explicitly opts into binary previews. Re-importing unchanged content replays deterministic observations instead of duplicating evidence.
