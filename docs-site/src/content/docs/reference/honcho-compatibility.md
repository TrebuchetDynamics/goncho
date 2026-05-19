---
title: Honcho Compatibility
description: Goncho's compatibility surface for Honcho-style integrations.
---

Honcho compatibility is a migration bridge, not Goncho's identity.

Goncho preserves familiar external names where useful:

```text
honcho_profile
honcho_search
honcho_context
honcho_chat
honcho_reasoning
honcho_conclude
```

The goal is to let existing agent integrations move toward local-first memory without rewriting every tool call first.

## Where Compatibility Ends

Goncho is not hosted Honcho. It is a Go library that runs locally, stores memory in local persistence, and is evolving toward trust-preserving context architecture.

Use compatibility to migrate. Use the Goncho concepts to design new integrations.
