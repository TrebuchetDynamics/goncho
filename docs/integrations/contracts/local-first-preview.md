# Local-First Preview Connector Contract

All Goncho integration connectors default to local-first operation and preview-before-apply behavior.

Connector docs that reference this contract inherit these requirements:

- Prefer loopback `goncho-server` endpoints for local hosts.
- Preview generated configuration, import counts, sample records, and redaction effects before writes.
- Avoid host configuration mutation, repository writes, daemon installation, webhook registration, or token storage unless an operator explicitly applies a reviewed plan.
- Keep non-loopback serving behind explicit authentication controls.
- Preserve scoped observations with enough source metadata for audit, deduplication, and rollback.
