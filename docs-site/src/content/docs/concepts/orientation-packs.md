---
title: Orientation Packs
description: Why Goncho returns context products instead of memory dumps.
---

An agent does not need every remembered fact. It needs the facts that orient the next action.

An orientation pack is the compact product Goncho assembles through `Context`.

An orientation product should contain:

- relevant peer facts;
- useful conclusions;
- session summary;
- recent turns when they matter;
- search hits tied to the current query;
- unavailable or degraded capabilities when relevant.

Current `Context` results already move in this direction with peer cards, conclusions, summaries, search hits, recent messages, and unavailable evidence.

:::caution[Failure mode]
Context stuffing makes stale or irrelevant facts compete with the current task. Orientation should be selective.
:::
