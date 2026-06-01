# Additional Agent Host Connector Plans

Status: supported-plan

Shared contract: [Local-First Preview Connector Contract](../contracts/local-first-preview.md).

Connector status contract: these hosts are local-first MCP configuration planning surfaces. Goncho prints reviewable patches and does not mutate host config. Run `goncho-server serve` on loopback before connecting any host, then use `goncho connect <host> --plan --addr 127.0.0.1:8765` to preview the patch.

Supported plan-only hosts from the current agentmemory second pass:

- `copilot-cli` — GitHub Copilot CLI `mcpServers` config.
- `qwen` — Qwen Code standard `mcpServers` config.
- `antigravity` — Antigravity user MCP config.
- `kiro` — Kiro user MCP config.
- `warp` — Warp standard `mcpServers` config.
- `cline` — Cline standard `mcpServers` config.
- `continue` — Continue `config.yaml` array-form `mcpServers`; existing YAML remains manual-review only.
- `zed` — Zed `context_servers` config, not `mcpServers`.
- `droid` — Droid/Factory MCP config.

`--apply` remains disabled until each generated plan has host-level smoke coverage. Use `goncho remove <host> --plan` to preview the inverse patch without deleting local Goncho data.
