# Local Shared Service Deployment

Status: conservative server-mode packaging for local teams and local agent hosts. This is not a hosted/cloud deployment guide.

## Target

The first supported deployment target is a **local shared service** on a trusted workstation or LAN-controlled development box. Goncho remains local-first: embedded SQLite is still the reference path, and Docker Compose is only packaging for operators who explicitly want a shared runtime.

Do not expose directly to the internet. Put any future remote deployment behind a reviewed reverse proxy, TLS, stronger auth, backups, and ACL tests.

## Run with Docker Compose

```bash
export GONCHO_SERVER_AUTH_TOKEN="change-this-local-token"
docker compose up -d --build
docker compose ps
docker compose logs -f goncho-server
```

The compose file publishes `127.0.0.1:8765:8765` so the host-side listener remains loopback-only. Inside the container, `goncho-server serve` binds to `0.0.0.0:8765` because container networking requires it, and an explicit auth token via `-auth-token` is supplied to satisfy Goncho's non-loopback bind guard.

Check health:

```bash
curl http://127.0.0.1:8765/health
```

Run the packaged smoke:

```bash
make docker-compose-smoke
```

The smoke starts compose, waits for `/health`, runs `goncho-server demo` inside the container to write/read memory against `/data/goncho.db`, then shuts down with `docker compose down -v`.

## Backup, export, and restore

Prefer portable export before moving or replacing a shared service:

```bash
# From a source checkout or container with the same DB mounted.
go run ./cmd/goncho-server health -db ./goncho.db
# Use service.ExportPortableJSONL or host-specific export tooling to write a checksummed snapshot manifest.
```

Backup/restore requirements:

- use snapshot manifest checksums;
- preserve provenance, lifecycle state, review state, retention/archive state, workspace ID, profile ID, peer ID, and session keys;
- preview imports before apply;
- fail closed on stable-ID conflicts;
- record the operator/admin actor for restore and retention changes.

Until a dedicated restore CLI exists, treat JSONL/Markdown export APIs and snapshot manifests as the safe interchange format and keep the raw SQLite file as an operator backup artifact.

## Shutdown

```bash
docker compose down
```

Use `docker compose down -v` only when intentionally deleting the local shared-service volume after export/backup.
