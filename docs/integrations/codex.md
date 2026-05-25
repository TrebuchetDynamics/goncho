# Codex Connector

Status: supported-plan

Codex support is local-first MCP configuration planning. Use `goncho connect codex --plan --config ~/.codex/config.toml --addr 127.0.0.1:8765` to preview the TOML patch.

Start `goncho-server serve` on loopback before connecting Codex. The command does not write Codex config; review the preview and keep `--apply` disabled until host smoke coverage is explicit.
