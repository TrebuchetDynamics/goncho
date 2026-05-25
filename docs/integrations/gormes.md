# Gormes Connector

Status: supported-plan

Gormes is the canonical first-party integration path for Goncho. It is local-first and preview-first: use `goncho connect gormes --plan` before any host configuration change.

Run `goncho-server serve -db <planned-profile-db> -addr 127.0.0.1:8765` after reviewing the plan. The plan derives the profile-local SQLite DB, markdown memory mirror, public tool names, and host hook events.

Do not enable hook forwarding until the profile owner has reviewed the generated plan and local redaction policy.
