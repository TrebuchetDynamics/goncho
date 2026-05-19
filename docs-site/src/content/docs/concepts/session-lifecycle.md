---
title: Session Lifecycle
description: How sessions create evidence and consolidation boundaries.
---

Sessions are where memory gets formed.

```text
session start -> messages and tools -> compaction pressure -> session end -> consolidation
```

## Lifecycle Boundaries

- Session start needs orientation.
- User messages and tool results create evidence.
- Tool failures can become negative memory.
- Compaction is a pressure point where working memory can be lost.
- Session end is a consolidation boundary.

Current Goncho exposes session-aware APIs such as `CreateMessages`, `OnSessionEnd`, summaries, and `Context`.

:::note[Architecture direction]
Hooks such as `SessionStart`, `PostToolUse`, `PreCompact`, and `Stop` are cognitive transition boundaries. They are concept language here, not stable public hook APIs in this package.
:::
