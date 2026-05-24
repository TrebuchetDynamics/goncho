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

Use this page with the [Core API](/reference/core-api/), [Memory Tools](/reference/memory-tools/), and [Gormes Agent Integration](/integrations/gormes-agent/) references when planning a migration path.

For new local-first tool surfaces, prefer the Goncho-native public tools:

```text
goncho_context
goncho_search
goncho_recall
goncho_remember
goncho_review
goncho_handoff
```

## Where Compatibility Ends

Goncho is not hosted Honcho. It is a Go library that runs locally, stores memory in local persistence, and is evolving toward trust-preserving context architecture.

Use compatibility to migrate. Use the Goncho concepts to design new integrations.
