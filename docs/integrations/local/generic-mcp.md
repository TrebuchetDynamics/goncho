# Generic MCP Integration

Status: supported-local

Shared contract: [Local-First Preview Connector Contract](../contracts/local-first-preview.md).

Connector status contract: Generic MCP hosts can connect to Goncho through local `goncho-server` transports. Use HTTP `POST /mcp` or `goncho-server stdio` depending on the host.

This path is local-first and preview-friendly: run `goncho-server health`, inspect `goncho-server security`, then configure the host to call the local loopback server. Non-loopback serving requires an explicit auth token guard.
