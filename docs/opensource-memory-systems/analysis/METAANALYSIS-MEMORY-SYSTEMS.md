# Meta-Analysis: Open-Source Agent Memory Systems for Goncho

Generated: 2026-05-19

This document compares the locally cloned open-source memory systems in the parent research-corpus directory against the academic memory architecture brief, "The Architecture of Memory in Artificial Agents: Bridging Academic Frontiers and Production Systems."

The purpose is not to pick one project to copy. The purpose is to extract the taxonomy, flows, techniques, strengths, and gaps we need in order to design Goncho as trust-preserving context architecture.

## Evidence Base

Local repositories inspected:

| Repository | Local directory | Observed role |
| --- | --- | --- |
| agentmemory | `../agentmemory/` | Hook-native, local-first agent memory with SQLite, hybrid retrieval, graph retrieval, REST, MCP, and broad agent integrations. |
| paradigm-memory | `../paradigm-memory/` | Cognitive-map memory engine with tree activation, local SQLite, audit log, and desktop UI. |
| agent-memory-mcp | `../agent-memory-mcp/` | Go MCP server for typed engineering memory, RAG indexing, temporal validity, sedimentation, and knowledge stewardship. |
| mii-memory | `../mii-memory/` | Rust local-first CLI and MCP memory store with global/workspace/session scopes, tags, MiniLM embeddings, relevance, and expiration rules. |
| memory-mcp | `../memory-mcp/` | PostgreSQL/pgvector/ltree semantic memory server with taxonomy, conflict resolution, system primer, TTL, and admin tooling. |
| memory-engine | `../memory-engine/` | Timescale/PostgreSQL memory API and MCP proxy using pgvector, pg_textsearch, ltree, JSONB metadata, temporal ranges, and ACLs. |
| anamnesis | `../anamnesis/` | Strategic memory engine using 4D retrieval: semantic, temporal, relational, and strategic weighting. |
| nautilus-compass | `../nautilus-compass/` | Black-box local memory with raw embeddings, drift detection, feedback anchors, and hook integration. |
| MARM-Systems | `../MARM-Systems/` | MCP memory layer with semantic search, session logs, notebooks, summaries, and dashboard. |
| nebu-ctx | `../nebu-ctx/` | Broader context runtime with Rust client, .NET host, hooks, project/session/brain stores, search, graph context, and dashboard. |
| shokunin | `../shokunin/` | Coding ecosystem with Chroma-backed memory, hybrid recall, freshness decay, claim verification, and skills. |

Inspection method:

- Read each project's root README and, where present in the target document, compared its architecture notes, tool surface, storage model, and lifecycle claims against the current Goncho design direction.
- Treated README claims as project self-description, not verified benchmark truth, unless the claim was directly represented by local source files or docs in the checkout.
- Favored reusable architectural patterns over implementation details tied to one language, hosted service, or agent runtime.
- Considered Goncho's product goal to be local-first coding-agent context reliability, not generic chatbot memory.

Coverage caveat:

This is a local-source meta-analysis, not a live external market survey. It deliberately studies the open-source projects cloned in the parent research-corpus directory and extracts design pressure from their concrete interfaces, schemas, flows, and failure modes.

Academic concepts used as the lens:

- Memory-augmented neural networks and DNC-style read/write separation.
- RAG and iterative retrieval.
- Working, episodic, semantic, and procedural memory.
- Cognitive architectures, spreading activation, and graph traversal.
- GraphRAG and multi-hop reasoning.
- Continual learning, catastrophic forgetting, EWC, and experience replay.
- Streaming attention, KV-cache compression, and attention sinks.
- Stale memory, belief revision, temporal knowledge graphs, and conflict resolution.
- Privacy-preserving memory, local-first design, and secure edge/cloud boundaries.
- Meta-learning, decay, reflection, and memory utility scoring.

Market context added in this revision:

- Enterprise retrieval is shifting from classic RAG pipelines toward context architecture.
- Agents generate far more retrieval and state requests than human users.
- The winning layer is not just vector search; it combines live data access, semantic tools, memory, caching, governance, freshness, and cost control.
- Redis Iris is one recent market signal: it frames context retrieval, agent memory, semantic caching, and real-time data integration as one agent runtime layer rather than separate RAG pieces.

## Thesis

Goncho should not be a memory store. It should be a trust-preserving context architecture.

The core architecture is:

```text
raw evidence
  -> claims
  -> scoped temporal beliefs
  -> retrieved orientation
  -> agent action
  -> consolidation / revision / forgetting
```

This is the main abstraction break:

```text
memory != retrieval
memory = stateful epistemic infrastructure
```

Or more concretely:

```text
events -> interpretation -> cognition -> action -> new evidence
```

Goncho's memory layer should maintain what the agent believes, why it believes it, when that belief was valid, where it applies, how confident the system is, and what evidence could revise it.

## Executive Synthesis

The strongest production memory and context systems are converging on the same architecture:

1. Capture everything important at the edges through hooks, tools, and watchers.
2. Normalize observations into typed, scoped, timestamped memory records.
3. Store both raw evidence and distilled claims.
4. Retrieve with hybrid methods: lexical, vector, graph, temporal, and trust-aware scoring.
5. Inject a small context pack, not a memory dump.
6. Maintain lifecycle state: fresh, canonical, superseded, stale, archived, review-required.
7. Make writes auditable, reversible, and visible.
8. Keep private operation local by default.

The weakest systems treat memory as "top-k vector search over old text." That solves recall only at small scale. It does not solve stale facts, multi-hop reasoning, conflict adjudication, prompt-injection persistence, user trust, or token budget pressure.

Goncho should treat memory as a claims and evidence system, not as a vector database. Vectors are one retrieval index. They are not the source of truth.

RAG is not disappearing. It is being demoted from architecture to technique. Retrieval remains important, but the architectural unit is now the context layer:

```text
agent -> context layer -> live data, memory, cache, tools, governance
```

That changes the primary question from:

```text
What chunks are semantically similar?
```

to:

```text
What does this agent need to know right now?
How fresh must it be?
Who is allowed to access it?
What does it cost to retrieve?
How much should the agent trust it?
```

## Unique Lessons By Project

| Project | Unique lesson | Goncho design implication |
| --- | --- | --- |
| agentmemory | End-to-end hook capture is the difference between a memory product and a note store. | Build hook-native capture early, but expose fewer public tools than agentmemory. |
| paradigm-memory | Cognitive maps reduce context noise better than flat result lists. | Add tree/map orientation as a retrieval gate before broad graph expansion. |
| agent-memory-mcp | Engineering memory needs stewardship: stale scans, conflict groups, canonical promotion, and review queues. | Treat maintenance as a product surface, not background magic. |
| mii-memory | Simple scopes, tags, alerts, and expiration rules make local memory understandable. | Start with explicit global/workspace/project/session scopes and lightweight prospective memory. |
| memory-mcp | A generated System Primer can orient a session if it is budgeted, cited, and regenerated from canonical state. | Generate boot packs from memory; never let hand-written flat files become the memory source of truth. |
| memory-engine | Content plus tree path plus metadata plus temporal range is a compact production-grade record shape. | Keep the source record small and provider-neutral; put FTS/vector indexes beside it. |
| anamnesis | Strategic value and reasoning fields matter because not all remembered facts deserve equal attention. | Store why a memory matters and separate authority from confidence. |
| nautilus-compass | Raw local embeddings and negative drift anchors catch behavioral recurrence without LLM extraction. | Preserve an extraction-free evidence lane and make negative memory first-class. |
| MARM-Systems | Human-facing notebooks, sessions, summaries, and dashboards increase trust and adoption. | Provide a simple inspection UI before advanced graph UX. |
| nebu-ctx | Context runtime design works best with a tiny public MCP surface and rich internal routing. | Keep Goncho's agent-facing contract small: context, search, remember, review, handoff. |
| shokunin | Old codebase memories are claims from frozen time, not facts. | Verify file/function/API claims against live repo state before using them. |

The strongest shared signal is that memory quality is determined less by the database and more by the control loop around it: capture, classify, retrieve, verify, inject, observe outcome, and steward lifecycle.

## Unified Taxonomy

### Memory Forms

| Form | Meaning | Production representation |
| --- | --- | --- |
| Working memory | Current active context and scratch state. | Prompt buffer, active session state, current task state, short-lived observations. |
| Surface memory | Fresh unprocessed observations. | Raw events, tool outputs, user prompts, session logs, recent notes. |
| Episodic memory | Time-bound records of what happened. | Session entries, observations, actions, summaries, transcripts, command history. |
| Semantic memory | Durable facts and concepts detached from one moment. | Canonical facts, project knowledge, entity descriptions, terminology, architecture facts. |
| Procedural memory | Reusable skills and workflows. | Routines, runbooks, scripts, tool sequences, known workflows, coding patterns. |
| Relational memory | Connections between things. | Graph nodes, graph edges, triples, dependencies, supersession chains, sequence edges. |
| Prospective memory | Things to surface later. | Alerts, reminders, handoffs, verification queues, pending actions. |
| Negative memory | Things to avoid or treat as risky. | Dead ends, drift anchors, failed attempts, rejected approaches, anti-patterns. |
| Belief memory | Probabilistic or confidence-scored claims. | Confidence, source trust, evidence count, valid intervals, review state. |

### Context Architecture Layers

| Layer | Role | Goncho implication |
| --- | --- | --- |
| Evidence | Immutable observations from user prompts, tools, sessions, files, and systems. | Preserve before interpretation. |
| Claims | Interpreted statements derived from evidence. | Store with proof, source, confidence, and time. |
| Beliefs | Current state of one or more claims under scope, time, trust, and conflict rules. | Maintain as revisable state, not static text. |
| Live data | Data that must be fetched fresh from tools, databases, code, APIs, or files. | Do not rely on memory when current truth is required. |
| Cache | Cheap reusable answers, embeddings, summaries, and computed context. | Reduce latency and cost without becoming source of truth. |
| Orientation | Compact working-memory projection for the current task. | Inject briefing packs, not dumps. |
| Governance | Access control, audit, provenance, privacy, and review. | Decide whether memory can be used, not just whether it is relevant. |

### Memory Lifecycle States

Goncho should separate memory type from memory lifecycle. A memory can be semantic and stale, procedural and canonical, episodic and archived, or negative and active.

Recommended lifecycle states:

| State | Meaning |
| --- | --- |
| `surface` | Newly captured, minimally processed. |
| `active` | Available for normal retrieval. |
| `draft` | Candidate distilled memory awaiting confidence or review. |
| `canonical` | High-confidence memory eligible for boot packs and strong scoring. |
| `superseded` | No longer current, but preserved historically. |
| `outdated` | Probably stale or contradicted; retrieval should warn or suppress. |
| `archived` | Retained for history, excluded from normal context injection. |
| `quarantined` | Potentially malicious, secret-bearing, or prompt-injection-contaminated. |
| `review_required` | Needs human or agent steward adjudication before promotion. |

### Core Axes For Comparing Systems

| Axis | Design question |
| --- | --- |
| Capture | How does information enter memory? |
| Representation | What is stored: raw text, typed claims, graph edges, trees, summaries, routines? |
| Scope | Is memory global, workspace, project, session, user, team, or tenant-bound? |
| Retrieval | How are memories selected under a token budget? |
| Context shaping | How are retrieved memories compressed and injected? |
| Lifecycle | How does memory decay, consolidate, verify, expire, or promote? |
| Conflict handling | How are contradictions and stale claims resolved? |
| Trust | How are source, confidence, recency, verification, and authority modeled? |
| Privacy | Is the system local-first? Does it redact secrets? Can it isolate users/projects? |
| Observability | Can users inspect, audit, edit, and understand why a memory was used? |
| Integration | Does it expose MCP, CLI, hooks, REST, HTTP, dashboard, or editor plugins? |

## Technique Taxonomy

### Retrieval Techniques

| Technique | Useful for | Failure mode |
| --- | --- | --- |
| Vector semantic search | Fuzzy recall, paraphrase matching, long text similarity. | Retrieves stale or semantically nearby but wrong memories. |
| BM25/FTS keyword search | Exact names, files, symbols, commands, identifiers. | Misses paraphrases and implicit concepts. |
| Reciprocal Rank Fusion | Combining lexical and semantic rankings. | Can still fuse bad candidates if candidate generation is noisy. |
| Graph traversal | Multi-hop reasoning and relationship-aware recall. | Requires good extraction and can add latency or wrong edges. |
| Tree/taxonomy gating | Reducing search space before retrieval. | Bad categorization hides relevant memories. |
| Spreading activation | Cognitive-map style branch selection. | Needs tuned activation decay and branch thresholds. |
| Temporal filtering | "What was true then?" and stale memory suppression. | Requires valid intervals and disciplined writes. |
| Trust-aware scoring | Prefer verified, authoritative, canonical memories. | Can entrench wrong canonical facts without review. |
| Reranking | Better final ordering under small context budgets. | Adds cost and latency. |

### Ingestion Techniques

| Technique | Useful for | Failure mode |
| --- | --- | --- |
| Hook capture | Silent collection of actual agent behavior. | Can collect too much noise or secrets. |
| Explicit memory tools | User or agent-controlled writes. | Agents may over-write or under-write. |
| Document indexing | Project docs, runbooks, ADRs, RFCs, changelogs. | Chunking can destroy context or provenance. |
| LLM extraction | Entities, relationships, decisions, summaries. | Extraction hallucinations become persistent errors. |
| Raw black-box storage | Privacy, low cost, no extraction errors. | Weaker symbolic reasoning and conflict resolution. |
| Deduplication | Prevents memory bloat. | Over-aggressive dedup destroys nuance. |
| Redaction | Keeps secrets out of long-term memory. | Poor redaction loses useful context or misses secrets. |

### Lifecycle Techniques

| Technique | Useful for | Failure mode |
| --- | --- | --- |
| Decay curves | Let weak memories fade. | Important rare facts may disappear. |
| Sedimentation | Promote repeated or verified facts to stable layers. | Repetition can promote wrong facts. |
| Supersession chains | Preserve old facts while marking current truth. | Needs conflict detection at write time. |
| Temporal validity intervals | Avoid stale current-state recall. | Hard to infer valid_until automatically. |
| Review inbox | Human-visible stewardship. | Backlog can grow without triage UX. |
| Verification prompts | Re-check claims against files or users. | Adds friction and latency. |
| Audit logs | Accountability and reversibility. | Storage and UX overhead. |

## Flow Archetypes

### 1. Passive Store/Search

Flow:

```text
agent/user writes memory -> database indexes it -> query retrieves top matches -> context injection
```

Examples: early MCP memory servers, minimal CLI memory stores, notebook-style systems.

Strength: simple and easy to reason about.

Weakness: does not solve stale memories, context budget, multi-hop reasoning, or automatic capture.

### 2. Hook-Driven Observe/Compress/Inject

Flow:

```text
UserPromptSubmit / PreToolUse / PostToolUse / Stop / PreCompact
  -> raw event capture
  -> session compression
  -> embedding + keyword index
  -> SessionStart recall
  -> compact context pack injection
```

Examples: agentmemory, nautilus-compass, nebu-ctx, shokunin.

Strength: memory follows actual agent work.

Weakness: hooks must be carefully filtered, redacted, and budgeted.

### 3. Typed Stewardship Loop

Flow:

```text
store typed memory -> score trust/confidence -> detect duplicates/conflicts/staleness
  -> review inbox -> promote, supersede, archive, or verify
```

Examples: agent-memory-mcp, memory-mcp, shokunin.

Strength: handles engineering knowledge as claims with lifecycle.

Weakness: requires review UX and source-aware rules.

### 4. Cognitive Map Gate

Flow:

```text
query -> activate tree nodes -> open high-activation branches
  -> retrieve local memories from selected branches
  -> inject minimal orientation
```

Example: paradigm-memory.

Strength: reduces context noise and gives the agent navigational orientation.

Weakness: the tree must stay well maintained.

### 5. Taxonomy Primer Loop

Flow:

```text
ingest -> categorize into ltree taxonomy -> synthesize primer
  -> SessionStart loads primer + relevant canonical memories
  -> retrieval feedback updates ranking
```

Examples: memory-mcp, memory-engine.

Strength: strong bootstrapping and browsing.

Weakness: bad taxonomy or over-large primer can become another flat-file problem.

### 6. Black-Box Drift Guard

Flow:

```text
embed raw text -> compare prompt to positive/negative anchors
  -> detect likely drift or repeated failure
  -> alert before agent repeats mistake
```

Example: nautilus-compass.

Strength: cheap, local, privacy-preserving, and catches behavioral recurrence.

Weakness: does not produce explicit knowledge structure by itself.

### 7. Context Runtime Layer

Flow:

```text
client/editor/hooks -> local shell/read/search/context tools
  -> server-backed project/session/brain/code-index stores
  -> context routing, compression, graph, analytics, dashboard
```

Example: nebu-ctx.

Strength: memory becomes one part of a broader context operating system.

Weakness: broad surface area increases complexity.

## Repository Comparison Matrix

| System | Archetype | Storage | Capture/write flow | Representation | Retrieval | Lifecycle/conflict | Best contribution to Goncho | Main risk |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| agentmemory | Hook-native local memory backend | SQLite/local StateKV, MiniLM embeddings | Agent hooks, MCP, REST, watcher integrations | Observations, sessions, summaries, semantic/procedural memories, graph nodes/edges, audit, signals, routines | BM25 + vector + graph traversal + temporal graph | Decay, tiered consolidation, supersession/versioning, audit | Broadest end-to-end local-first agent integration model | Large surface and many features can become hard to govern |
| paradigm-memory | Cognitive map | SQLite, optional vectors, Tauri UI | MCP writer proposes, substrate writes | Tree nodes, memory items, activation, confidence, importance, freshness | Node activation, branch selection, FTS/vector local retrieval | Audit log, logical delete, supersedes, consolidator proposals | Context as navigation map instead of memory dump | Tree quality and branch routing are critical |
| agent-memory-mcp | Typed engineering memory | SQLite, local embeddings via providers | MCP tools, document watcher, RAG indexing | Typed memories, docs, temporal fields, sediment layers | Semantic + text + trust/layer/freshness scoring | Duplicate/conflict/stale detection, review inbox, valid intervals | Strong stewardship model for engineering teams | More process and metadata than small users may tolerate |
| mii-memory | Scoped local CLI/MCP store | SQLite, embedded MiniLM ONNX | CLI and MCP memory_set, alerts | Global/workspace/session scopes, tags, alerts, relevance | Semantic ranking, tag filters, text match, positive/negative relevance | Expiration conditions, session lineage, relevance adjustment | Clean scope model and expiration primitives | Minimal graph/conflict semantics |
| memory-mcp | Self-organizing semantic memory server | PostgreSQL, pgvector, ltree, BM25 | MCP HTTP ingestion queue and admin tools | Memories, category paths, edges, system primer, tiers | Hybrid vector/BM25/RRF, taxonomy filters, feedback rerank | Conflict audit, supersession, TTL, primer regeneration | Best taxonomy + primer + admin discipline | Requires Postgres/OpenAI-style infrastructure |
| memory-engine | Production memory API | PostgreSQL, pgvector, pg_textsearch, ltree, JSONB, tstzrange | MCP proxy to hosted/self-hosted API, imports/exports | Content + tree path + metadata + temporal ranges | Semantic, fulltext, hybrid RRF, metadata/tree/time filters | Update/delete/tree ops, ACLs, engines | Clean production data model and team isolation | API/server dependency; less agent-specific lifecycle |
| anamnesis | Strategic memory engine | PostgreSQL + pgvector | Retain, recall, reflect, decay_check, reweight | Memory banks, strategic weight, facts, entities, relations, authority | 4D RRF: semantic, temporal, relational, strategic | Decay conditions, supersession, reweighting | Weighting memory by strategic value, not just similarity | Heavier conceptual model and fewer hook-native flows |
| nautilus-compass | Black-box drift detector | Local embeddings and memory daemon | Prompt/tool/stop hooks; raw text ingest | Raw observations, profiles, anchors, feedback labels | BGE-m3 semantic recall, drift detection against anchors | Feedback loop adapts anchors | Anti-drift and privacy-preserving raw memory | Less explicit state adjudication or graph reasoning |
| MARM-Systems | Notebook/session memory MCP | SQLite WAL, sentence-transformers | MCP tools, session logs, notebook entries, summaries | Session logs, notebooks, summaries, semantic store | Semantic recall and contextual logs | Manual notebook and summaries | Simple human-friendly session/notebook layer | Weak formal conflict and lifecycle model |
| nebu-ctx | Context runtime | PostgreSQL-backed stores, Rust client, .NET host | Hooks, editor/MCP/shell, project/session/brain/code-index stores | Context ledger, graph index, semantic cache, episodic/procedural/prospective memory | Context search, graph context, adaptive routing | Promotion, compression safety, lifecycle modules | Memory as part of full context operations platform | Scope may be too broad for a first Goncho kernel |
| shokunin | Coding memory and skills ecosystem | ChromaDB SQLite-backed | MCP tools plus coding workflow memory | Context entries, claims, sessions, skills, freshness metadata | Vector + BM25 + temporal + RRF | Freshness decay, claim verification, consolidation | "Memory as claims from frozen time" and path verification | Coupled to a larger skill ecosystem |

## Per-System Analysis

### agentmemory

Agentmemory is the closest to a complete local-first agent memory platform. It combines MCP, REST, hooks, SQLite/local storage, embeddings, BM25, graph retrieval, temporal graph extraction, and broad integrations across coding agents.

Flow:

```text
agent hooks / MCP / REST / watcher
  -> observations, sessions, actions, summaries
  -> embeddings + BM25 + graph nodes/edges + audit records
  -> recall through hybrid search and graph expansion
  -> context injection at session start or active recall
```

Taxonomy and representation:

- Sessions, observations, memories, summaries, embeddings, BM25 index.
- Graph nodes and graph edges, including temporal graph history.
- Semantic and procedural memory namespaces.
- Audit, actions, leases, routines, signals, checkpoints, mesh coordination.
- Retention scores, access logs, slots, sketches, facets, sentinels, crystals, lessons.

Techniques:

- BM25 plus vector retrieval.
- Graph traversal from entities and chunks.
- Temporal graph extraction with valid time fields.
- Version/supersession logic.
- Ebbinghaus-style decay and tiered consolidation.
- Hook-based capture and integration-specific behavior.
- Privacy filtering and local operation.

What Goncho should steal:

- Treat hooks as first-class capture surfaces.
- Store both evidence events and distilled memories.
- Use hybrid retrieval as the default, not as an add-on.
- Include graph retrieval early, even if extraction starts conservative.
- Build around local-first operation and zero required external database.

Risks:

- Very wide namespace and tool surface can become difficult to explain.
- More features mean more chances for silent incorrect writes.
- Goncho should preserve the architecture direction but expose a smaller public API.

### paradigm-memory

Paradigm-memory's strongest idea is that context should be an orientation map, not a pile of retrieved snippets. It models memory as a cognitive map: a tree of nodes with activation, confidence, importance, freshness, and retrieval policy.

Flow:

```text
query
  -> activate relevant map nodes
  -> select branches by activation thresholds
  -> retrieve memories locally inside selected branches
  -> inject a compact navigation-oriented context pack
```

Taxonomy and representation:

- Memory nodes with parent/child tree structure.
- Node fields include label, summary, one-liner, type, status, importance, activation, confidence, freshness, retrieval policy, keywords, links, sources, and stats.
- Memory items belong under nodes and carry content, tags, source, importance, confidence, expiration, status, deletion state, and supersession.

Techniques:

- Cognitive-map branch gating.
- Activation propagation across a tree.
- FTS5 and optional vector search.
- Audit log for every mutation.
- Writer proposes, substrate writes.
- Consolidator proposes duplicate/stale/overloaded/orphan fixes.

What Goncho should steal:

- Add an orientation layer above raw retrieval.
- Open only high-activation branches and keep weakly related branches latent.
- Treat automatic consolidation as proposals before mutation.
- Maintain a forensic audit log for trust.

Risks:

- Requires good taxonomy hygiene.
- A bad map can hide relevant memories.
- Needs strong UI or CLI affordances for reorganizing the map.

### agent-memory-mcp

Agent-memory-mcp is one of the best examples of memory as engineering knowledge stewardship. It is typed, local, temporal, and review-oriented.

Flow:

```text
MCP store/index tools or document watcher
  -> typed memory or RAG document chunk
  -> embedding + text indexing
  -> trust, sediment layer, temporal metadata
  -> recall with scoring and filters
  -> stewardship jobs detect duplicates, conflicts, stale records, and promotion candidates
```

Taxonomy and representation:

- Memory types: episodic, semantic, procedural, working.
- Sediment layers: surface, episodic, semantic, character.
- Temporal fields: valid_from, valid_until, observed_at, superseded_by, replaces.
- Metadata includes source, lifecycle, owner, review state, confidence, and tags.

Techniques:

- Semantic and text retrieval with model-mismatch fallback.
- Trust scoring based on source/type/lifecycle/layer/review.
- Sedimentation promotions and demotions.
- Temporal recall as-of a specific date.
- Knowledge timeline.
- RAG indexing with source-aware boosts for runbooks, postmortems, ADRs, RFCs, CI, Helm, Terraform, Kubernetes, and dead ends.
- Review inbox, drift scan, verification candidates, canonical promotion.

What Goncho should steal:

- Separate type from sediment layer.
- Use temporal validity as a first-class retrieval filter.
- Score memory by trust, not just similarity.
- Build a review inbox for questionable memory mutations.
- Treat dead ends as retrievable knowledge.

Risks:

- Metadata model can become burdensome.
- Review workflow must be ergonomic or it will be ignored.

### mii-memory

Mii-memory has the cleanest small-system mental model. Its global/workspace/session scopes and tag filters are practical and easy to adopt.

Flow:

```text
CLI or MCP memory_set
  -> scoped record with required tags
  -> content + tag embedding
  -> retrieval by scope, tags, text, and semantic similarity
  -> expiration checks on access
```

Taxonomy and representation:

- Scopes: global, workspace, session.
- Session lineage: parent/child visibility.
- Tags are required and behave like lightweight directories.
- Alerts are one-shot session reminders.
- Relevance includes positive and negative scores.

Techniques:

- Embedded MiniLM ONNX model.
- Semantic ranking over content plus tags.
- Positive and negative tag filters.
- Relevance reinforcement and negative scoring for competing older memories.
- Expiration conditions: time, usage, file existence, file pristine state, period.

What Goncho should steal:

- Global/workspace/session scoping should be part of the storage model from day one.
- Expiration should support more than time.
- Alerts and handoffs should be modeled as prospective memory, not ordinary notes.
- Local embedded embeddings reduce setup friction.

Risks:

- No deep graph or temporal conflict model.
- Tags alone are too weak for complex project memory.

### memory-mcp

Memory-mcp is the strongest example of a self-organizing memory server with taxonomy, primer generation, conflict audit, and admin controls. It is heavier than a local tool but offers useful production discipline.

Flow:

```text
memorize_context
  -> staging/ingestion queue
  -> chunk, embed, categorize into ltree taxonomy
  -> deduplicate and evaluate conflicts
  -> store canonical/historical/ephemeral memory
  -> regenerate system primer when needed
  -> initialize_context loads primer and pending verification/handoffs
```

Taxonomy and representation:

- PostgreSQL memories table with content, vector embedding, category_path, supersedes_id, archived_at, tier, metadata, lexical search, timestamps, and verify_after.
- Memory edges for supersedes, relates_to, depends_on, sequence_next.
- Taxonomy roots such as profile, projects, organizations, concepts, and reference.system.primer.
- Ingestion staging and conflict audit events.
- Retrieval feedback and context_store TTL.

Techniques:

- pgvector semantic search.
- BM25/lexical search and RRF.
- ltree taxonomy.
- System Primer for boot.
- Conflict evaluation with structured resolution schema.
- Supersession chains and decision timelines.
- TTL daemon and verification prompts.
- Admin MCP tools for pruning, export, recategorization, metadata repair, diagnostics, and staging inspection.

What Goncho should steal:

- System Primer as a generated boot artifact, not a hand-written flat file.
- Conflict audit events.
- Admin tooling as a separate surface from normal agent tools.
- Tiers: canonical, historical, ephemeral.
- Retrieval feedback.

Risks:

- PostgreSQL and external model dependencies raise the setup floor.
- Auto-generated primers can become context bloat if not hard-budgeted.

### memory-engine

Memory-engine is a clean production data model for memory as content plus tree path plus metadata plus temporal range. It is less agent-opinionated but strong on infrastructure.

Flow:

```text
MCP proxy or API write
  -> memory_create with content, tree path, metadata, temporal fields
  -> async embeddings and immediate fulltext
  -> search by semantic, fulltext, or hybrid
  -> filter by tree, metadata, temporal ranges, and grep
```

Taxonomy and representation:

- Memory is content plus tree plus metadata plus temporal data.
- Tree paths are typically 2 to 4 levels.
- Metadata convention includes type, status, source, confidence.
- Temporal point/range fields support contains, overlaps, and within.
- Engines isolate databases, users, roles, grants, and API keys.

Techniques:

- pgvector semantic index.
- pg_textsearch BM25.
- ltree tree paths.
- JSONB metadata with GIN indexes.
- tstzrange temporal queries.
- Row-level security and role-based access.
- Import/export and subtree operations.

What Goncho should steal:

- Content/tree/meta/time is a strong minimal record shape.
- Temporal range queries should be query-native, not post-filters.
- Team mode needs engines, ACLs, and role isolation.
- Keep MCP client stateless when a server mode exists.

Risks:

- Hosted/API model is less local-first than Goncho likely needs.
- Lifecycle and agent hooks must be added around the infrastructure.

### anamnesis

Anamnesis focuses on strategic value: why a memory matters, who authorized it, what it depends on, and when it should decay. Its 4D retrieval model is a strong counterweight to naive similarity scoring.

Flow:

```text
retain
  -> validate bank/write agent
  -> embed content
  -> extract facts/entities if configured
  -> link graph triples
  -> calculate strategic weight
  -> recall through semantic, fulltext, temporal, relational, and strategic fusion
```

Taxonomy and representation:

- Memory banks with mission, directives, weight factors, and decay policy.
- Content types: fact, decision, observation, instruction, event.
- Source, reasoning, authority, confidence, decay_condition, tags, supersedes, depends_on.
- Extracted facts and entity graph triples.
- Weight, status, decayed_at, superseded_by, access count.

Techniques:

- Four retrieval dimensions: semantic, temporal, relational, strategic.
- RRF fusion.
- Authority caps: explicit, system, inferred.
- Reweighting based on access, connectivity, and confidence.
- Reflection and decay checks.

What Goncho should steal:

- Add a strategic utility score to memory.
- Record why a memory was stored, not just what it says.
- Distinguish source authority from confidence.
- Weight project directives and durable decisions higher than incidental chatter.

Risks:

- Strategic weighting can feel opaque unless explanations are exposed.
- Requires discipline around bank definitions and write authority.

### nautilus-compass

Nautilus-compass makes a deliberate tradeoff: avoid white-box extraction at index time and use raw local embeddings plus drift detection. It is valuable because not every useful memory operation needs symbolic structure.

Flow:

```text
UserPromptSubmit / PostToolUse / Stop hooks
  -> raw text ingest
  -> local BGE-m3 embeddings
  -> recall and profile aggregation
  -> drift_check against positive/negative anchors
  -> feedback labels adapt anchors
```

Taxonomy and representation:

- Raw observations and session summaries.
- Project-scoped and optional same-user cross-project recall.
- Positive and negative drift anchors.
- Feedback labels for alert quality.
- Profiles aggregated from raw memory.

Techniques:

- Local BGE-m3 embeddings.
- Drift detection through cosine similarity to anchors.
- Proof-of-recall token.
- Feedback loop for false positive and true positive alerts.
- Privacy-preserving raw embedding store.
- Cross-project recall without white-box entity ID dependence.

What Goncho should steal:

- Implement anti-memory and drift anchors.
- Detect when the agent is about to repeat a known failure pattern.
- Preserve an extraction-free raw evidence lane for privacy and robustness.
- Use explicit feedback labels to improve memory alerts.

Risks:

- Raw embeddings alone do not solve stale claims.
- Graph and temporal reasoning need a complementary structured lane.

### MARM-Systems

MARM is a pragmatic MCP memory layer with session logs, notebooks, semantic search, summaries, and a dashboard. It is useful as a human-facing workflow model, even though its lifecycle semantics are lighter.

Flow:

```text
start session / log entries / add notebook
  -> SQLite WAL storage
  -> sentence-transformer embeddings
  -> smart recall and context bridge
  -> dashboard inspection
```

Taxonomy and representation:

- Session logs.
- Notebook entries.
- Summaries.
- Semantic memory records.
- Context bridge artifacts.

Techniques:

- SQLite WAL and connection pooling.
- Lazy local embedding model loading.
- MCP tools for session and notebook workflows.
- Dashboard for browsing memory.
- Response size limits and rate limits.

What Goncho should steal:

- Keep a human-friendly notebook layer separate from automatic observations.
- Provide a dashboard early enough to build user trust.
- Session logs are a useful episodic substrate.

Risks:

- Notebook memory can devolve into manual flat files.
- Lacks strong conflict, temporal, and trust semantics.

### nebu-ctx

Nebu-ctx is less a memory backend and more a context runtime. It treats memory as one component in a system that also manages shell, code index, graph context, adaptive compression, session state, and dashboard visibility.

Flow:

```text
editor/client/hooks
  -> Rust thin client
  -> .NET host with project/session/brain/code-index stores
  -> context routing, graph search, semantic cache, shell/read tools
  -> dashboard and analytics
```

Taxonomy and representation:

- Project, knowledge, session, brain, code-index, and checkout binding stores.
- Context ledger.
- Graph index.
- Semantic cache.
- Episodic, procedural, and prospective memory modules.
- Hook-derived session snapshots and knowledge promotion.

Techniques:

- Public MCP surface reduced to a small tool set.
- Context gateway pattern.
- Hook family: post-tool, pre-compact, pre-tool bash/read, session start, stop, user prompt submit.
- Adaptive routing and compression safety.
- Graph-driven context and local deterministic evals.

What Goncho should steal:

- Keep the agent-facing tool surface small, even if internals are rich.
- Build context compression and memory retrieval together.
- Treat code index, shell history, and memory as related context sources.
- Use deterministic eval gates for context quality.

Risks:

- Very broad scope can delay a crisp memory kernel.
- Goncho should start with the memory kernel, then grow into a context runtime.

### shokunin

Shokunin's most transferable idea is memory as claims from a frozen point in time. It explicitly verifies file/path claims before relying on them, which addresses a major practical stale-memory failure in coding agents.

Flow:

```text
store_context / save_message / session summary
  -> Chroma-backed vector memory with metadata
  -> BM25 and temporal candidate generation
  -> RRF fusion and freshness decay
  -> claim verification before acting on old file/function/API claims
```

Taxonomy and representation:

- Entry types: decision, file, command, preference, checkpoint, session_end, general.
- Claim types: claim_file, claim_function, claim_flag, claim_api.
- Sessions and summaries.
- Skills and coding workflow context.

Techniques:

- ChromaDB with all-MiniLM-L6-v2.
- BM25 with tuned parameters.
- Reciprocal Rank Fusion.
- Freshness decay with 30-day half-life style scoring.
- Claim verification with verified_at.
- Consolidation.

What Goncho should steal:

- Treat memories about files, APIs, and code as stale until verified.
- Add explicit claim categories for codebase facts.
- Blend vector, lexical, and temporal retrieval by default.
- Use freshness as a continuous scoring factor, not only a filter.

Risks:

- Chroma as a dependency may be heavier than a pure SQLite first version.
- Memory design is coupled to the broader Shokunin skill ecosystem.

## Cross-System Patterns

### What Works Repeatedly

1. Local-first storage wins developer trust.
2. MCP is now the default interop layer, but hooks are what make memory automatic.
3. Hybrid retrieval beats pure vector search.
4. Scope is mandatory: global, workspace, project, session, user, team.
5. Temporal metadata is not optional for long-lived agents.
6. Memory must preserve evidence and support audit.
7. Session boot should provide orientation, not a dump.
8. Lifecycle jobs are necessary because memory quality decays.
9. Human review is still needed for promotion, deletion, and conflict resolution.
10. Dashboards and CLI inspection matter because invisible memory is hard to trust.

### What Fails Repeatedly

1. Pure flat files eventually become token bloat.
2. Pure vector top-k retrieval eventually returns plausible noise.
3. Automatic summarization destroys rare but critical details.
4. Silent mutation of long-term memory creates hidden bugs.
5. Deleting obsolete facts destroys historical reasoning.
6. Too many MCP tools increase tool-selection cost and agent confusion.
7. Cloud-only memory breaks privacy expectations for developer workflows.
8. Storing every event equally makes retrieval worse over time.
9. Boot primers become another context-stuffing problem without hard budgets.

## Goncho Design Decisions From The Survey

These are the concrete product decisions implied by the local project study.

| Decision | Adopted direction | Why |
| --- | --- | --- |
| Source of truth | Event-sourced evidence plus derived claims. | Prevents hallucinated summaries from becoming untraceable truth. |
| First storage backend | SQLite local kernel. | Matches developer trust and low setup friction from agentmemory, mii-memory, paradigm-memory, and shokunin. |
| Server mode | Optional PostgreSQL adapter later. | memory-mcp, memory-engine, anamnesis, and nebu-ctx show why team mode benefits from Postgres features, but it should not block local use. |
| Public MCP surface | Five stable tools: context, search, remember, review, handoff. | nebu-ctx shows small surfaces are easier for agents; memory-mcp shows admin tools should be separate. |
| Retrieval default | Hybrid lexical + vector + recent/session + graph + temporal + trust scoring. | Every mature system moves beyond vector top-k. |
| Boot behavior | Generated orientation pack with citations and hard token budget. | Avoids both amnesia and prompt stuffing. |
| Memory lifecycle | Active stewardship queue with promotion, supersession, staleness, quarantine, and review. | Long-lived memory decays without maintenance. |
| Code facts | Verify before acting. | Shokunin's claim verification directly addresses stale coding-agent failures. |
| Negative memory | First-class type and drift anchor source. | Nautilus-compass shows repeated mistakes are one of memory's highest-value targets. |
| UI | Inspection-first dashboard before complex visualization. | Users must see why memory was stored, retrieved, trusted, or suppressed. |

Practical first-kernel cut:

```text
SQLite + FTS5 + local embeddings
  + events/memories/links/reviews schema
  + scoped context pack generation
  + explicit remember/search/handoff/review tools
  + basic claim verification and citation output
```

Defer until the kernel proves reliable:

```text
full cognitive-map UI
team ACLs
PostgreSQL adapter
automated graph extraction at scale
complex strategic reweighting
cross-agent mesh coordination
```

## Recommended Goncho Architecture

### Design Principles

Goncho should optimize for these principles:

1. Local-first by default, server/team mode optional.
2. MCP-first, hook-native.
3. Small agent-facing tool surface, rich internal pipeline.
4. Every memory is either evidence, claim, routine, alert, or relationship.
5. Memory records must be scoped.
6. Memory records must be time-aware.
7. Retrieval must be hybrid and budgeted.
8. Context injection must produce cited packs, not raw dumps.
9. Lifecycle changes must be auditable and reversible.
10. Stale, conflicting, and low-confidence memories should be visible, not silently ignored.
11. Negative memory and dead ends are first-class.
12. Secrets and prompt-injection-like content must be quarantined before promotion.
13. Live truth should be pulled from governed tools when memory is insufficient or stale.
14. Every surfaced memory should be able to answer: why this, why now, why trust it?

The seven operating principles are:

| Principle | Meaning |
| --- | --- |
| Evidence before memory | Preserve raw events and tool outputs first; derive memories second. |
| Claims, not chunks | Store what is believed, with proof, confidence, scope, and time. |
| Hooks over manual saves | Capture memory at cognitive transition boundaries such as SessionStart, PostToolUse, PreCompact, and Stop. |
| Orientation, not dumping | Boot agents with current goals, trusted facts, warnings, dead ends, and unresolved conflicts. |
| Negative memory matters | Failed paths and rejected approaches are part of intelligence. |
| Small agent surface | Expose stable primitives: context, search, remember, review, handoff. |
| Trust is the moat | Prove why memory surfaced and whether it may be stale. |

### Proposed Storage Kernel

Start with SQLite for solo/local use. Add PostgreSQL as a team/server adapter later.

Core tables:

| Table | Purpose |
| --- | --- |
| `events` | Raw hook/tool/session evidence. Append-only. |
| `memories` | Current memory records and claims. |
| `memory_versions` | Supersession history and previous forms. |
| `memory_links` | Supersedes, depends_on, relates_to, caused_by, contradicts, sequence_next. |
| `graph_nodes` | Entities, files, concepts, people, services, tasks. |
| `graph_edges` | Extracted relationships with validity intervals and evidence refs. |
| `documents` | Indexed docs, ADRs, runbooks, code docs, external imports. |
| `sessions` | Session metadata, lineage, summaries, active goals. |
| `scopes` | Global, user, workspace, project, repo, session, team. |
| `retrieval_feedback` | Whether a retrieved memory helped, harmed, or was ignored. |
| `reviews` | Conflict/stale/duplicate/promotion review queue. |
| `alerts` | Prospective memory and one-shot reminders. |
| `audit_log` | Who/what changed memory and why. |

Recommended indexes:

- SQLite FTS5 for content, title, tags, file paths, symbols.
- Local vector index using a SQLite-compatible vector extension or embedded index.
- B-tree indexes on scope, type, status, lifecycle, valid_from, valid_until, observed_at, source, confidence.
- Graph indexes on node id, edge relation, source, target, and valid intervals.

### Proposed Memory Record Shape

```text
id
scope_id
type: working | episodic | semantic | procedural | relational | prospective | negative
layer: surface | episodic | semantic | character
status: active | draft | canonical | superseded | outdated | archived | quarantined | review_required
title
content
summary
source: user | agent | hook | document | tool | import | system
source_ref
owner
tags
tree_path
entities
confidence
authority
importance
strategic_weight
freshness_score
access_count
valid_from
valid_until
observed_at
verified_at
expires_at
supersedes_id
evidence_event_ids
created_at
updated_at
```

This combines:

- mii-memory's scopes.
- agent-memory-mcp's type/layer split.
- memory-engine's content/tree/meta/time model.
- memory-mcp's tier and primer discipline.
- anamnesis's authority and strategic weighting.
- shokunin's verification fields.
- agentmemory's graph and hook-native evidence model.

### Capture Plane

Goncho should capture from:

| Surface | What to capture |
| --- | --- |
| `SessionStart` | Active project, current goal, previous handoff, primer request. |
| `UserPromptSubmit` | User intent, durable preferences, possible drift trigger. |
| `PreToolUse` | High-risk commands, file reads, shell intent, expected target. |
| `PostToolUse` | Actual file changes, command results, errors, decisions implied by action. |
| `PreCompact` | Pending details at risk of being lost. |
| `Stop` | Session summary, unresolved tasks, decisions, commands, failures, handoff. |
| Explicit MCP tools | User-approved durable memories and manual corrections. |
| Repo/doc watcher | ADRs, README changes, runbooks, changelogs, architecture docs. |

Capture should produce raw evidence first. Distilled memory can be generated after filtering and review rules.

### Ingestion Pipeline

Recommended pipeline:

```text
raw event
  -> normalize schema
  -> assign scope
  -> redact secrets and credentials
  -> classify memory candidate
  -> chunk if needed
  -> lexical index
  -> embedding index
  -> conservative entity extraction
  -> duplicate/conflict check
  -> write evidence and candidate memory
  -> enqueue review or promotion decision
```

Write-time checks:

- Secret detection.
- Prompt-injection pattern detection.
- Duplicate detection.
- Conflict detection.
- Source trust assignment.
- Scope leakage check.
- Temporal validity inference.
- Token budget estimate.

### Retrieval Pipeline

Recommended query flow:

```text
query + current task state
  -> classify intent
  -> determine scopes
  -> retrieve lexical candidates
  -> retrieve vector candidates
  -> retrieve recent/session candidates
  -> retrieve canonical/project primer candidates
  -> graph expand selected entities
  -> apply temporal filters
  -> score by RRF + trust + freshness + importance + layer + feedback
  -> verify risky code claims
  -> produce context pack with citations and warnings
```

Candidate score should include:

```text
score =
  retrieval_score
  + lexical_exactness
  + graph_relevance
  + temporal_relevance
  + trust_score
  + freshness_score
  + strategic_weight
  + layer_boost
  + access_feedback
  - staleness_penalty
  - conflict_penalty
  - scope_distance_penalty
```

Retrieval should return a structured context pack:

```text
orientation
current canonical facts
relevant recent episodes
related procedures
known dead ends
stale/conflicting warnings
source citations
verification requirements
```

### Boot Flow

Session start should not load "all memory." It should load a compact orientation pack:

```text
SessionStart
  -> identify workspace/project/session lineage
  -> load project primer under hard token budget
  -> load latest handoff and unresolved tasks
  -> load top canonical facts for current project
  -> load recent high-signal episodes
  -> load known pitfalls and dead ends
  -> surface review/verification warnings
```

The boot pack should be generated from memory, not hand-maintained as a flat file. It should have a max token budget and source citations.

### Lifecycle and Stewardship

Goncho needs background stewardship jobs:

| Job | Purpose |
| --- | --- |
| Duplicate scan | Merge or link equivalent memories without deleting evidence. |
| Conflict scan | Detect contradictions and create review items. |
| Staleness scan | Mark records needing verification or decay. |
| Canonical promotion | Promote high-confidence, repeated, verified facts. |
| Sedimentation | Move useful memory from surface to episodic to semantic/character layers. |
| Decay | Lower priority for unused, low-confidence, or obsolete memories. |
| Dead-end consolidation | Preserve failed attempts and repeated mistakes as negative memory. |
| Primer regeneration | Update compact boot artifacts after meaningful change. |
| Graph repair | Find orphan nodes, weak edges, and overloaded concepts. |

Mutation policy:

- Never overwrite a memory without a version record.
- Prefer supersession over deletion.
- Preserve historical truth with valid intervals.
- Require review for low-confidence LLM-extracted contradictions.
- Quarantine suspected prompt injection or secrets.

### Agent-Facing MCP Surface

Keep the public surface small. Internals can be rich.

Recommended first MCP tools:

| Tool | Purpose |
| --- | --- |
| `goncho_context` | Read the current orientation/context pack. |
| `goncho_search` | Search memory with scope, type, time, and status filters. |
| `goncho_remember` | Store an explicit memory or correction. |
| `goncho_review` | Inspect and resolve review-required items. |
| `goncho_handoff` | Save or load session handoff/prospective memory. |

Optional admin tools should be separate:

- export/import.
- prune/archive.
- recategorize.
- rebuild indexes.
- diagnostics.
- audit trace.

### UI and Observability

Goncho should expose:

- Memory timeline.
- Project cognitive map.
- Review inbox.
- Supersession chain viewer.
- Search debugger showing why each memory was retrieved.
- Source/evidence view for every claim.
- Drift/dead-end alerts.
- Primer preview with token budget.
- Scope and privacy inspector.

Without observability, users will not trust persistent agent memory.

## Evaluation Plan

Goncho should evaluate memory quality with deterministic local tests before any benchmark claims.

### Core Evaluations

| Eval | Measures |
| --- | --- |
| Exact recall | Can it retrieve names, files, commands, decisions, and IDs? |
| Paraphrase recall | Can it retrieve semantically equivalent memories? |
| Multi-hop recall | Can it connect facts across graph edges? |
| Temporal state | Can it answer "what is true now" vs "what was true then"? |
| Conflict adjudication | Does newer contradictory evidence supersede older claims correctly? |
| Stale code claim | Does it verify files/functions before acting on old memory? |
| Token budget | Does boot/retrieval stay below context limits? |
| Noise resistance | Does irrelevant old memory stay out of context? |
| Scope isolation | Does project A memory avoid leaking into project B? |
| Prompt-injection persistence | Does malicious text fail to promote into trusted memory? |
| Drift prevention | Does negative memory prevent repeated failed behavior? |

### Suggested Test Fixtures

1. Alice owns auth; auth owns DB permissions; later Bob replaces Alice.
2. File `src/auth.ts` existed, then moved to `src/security/auth.ts`.
3. User first prefers Mocha, later switches project to Vitest.
4. Agent tried three failed Docker fixes; fourth succeeded.
5. A prompt-injection string appears in a document import.
6. A runbook says one command; a later incident postmortem supersedes it.
7. Same concept appears in README, ADR, and session log with slightly different wording.

## Goncho Roadmap

### Phase 1: Local Memory Kernel

Build:

- SQLite storage.
- FTS5 lexical index.
- Local embeddings.
- MCP tools: context, search, remember, handoff, review.
- Hook capture for session start, prompt submit, post-tool, pre-compact, stop.
- Scopes: global, workspace, project, session.
- Types: episodic, semantic, procedural, prospective, negative.
- Basic context pack with citations.

Goal:

Get reliable local recall without cloud dependencies.

### Phase 2: Lifecycle and Trust

Build:

- Temporal fields and valid intervals.
- Supersession chains.
- Claim verification for files/functions/APIs.
- Trust/confidence scoring.
- Review inbox.
- Decay and freshness scoring.
- Primer generation under a token budget.

Goal:

Prevent stale and conflicting memory from silently corrupting agent behavior.

### Phase 3: Graph and Cognitive Map

Build:

- Conservative entity extraction.
- Graph nodes and edges.
- Graph traversal for multi-hop recall.
- Tree/taxonomy paths.
- Activation-based branch gating.
- Graph/map dashboard.

Goal:

Move from similarity recall to relationship-aware context.

### Phase 4: Drift and Negative Memory

Build:

- Dead-end memory type.
- Positive/negative anchors.
- Drift detector.
- Feedback labels.
- Alerts for repeated failure patterns.

Goal:

Make memory prevent repeated mistakes, not just recall facts.

### Phase 5: Team and Server Mode

Build:

- PostgreSQL adapter.
- HTTP server.
- Auth and workspace ACLs.
- Import/export.
- Admin tools.
- Multi-user audit.

Goal:

Support shared project memory without sacrificing local-first solo workflows.

## Best Ideas To Adopt By Source

| Source | Adopt |
| --- | --- |
| agentmemory | Hook-native capture, hybrid retrieval, temporal graph, broad integration, local-first stance. |
| paradigm-memory | Cognitive map, activation thresholds, audit-first mutation, proposal-based consolidation. |
| agent-memory-mcp | Typed memory, sediment layers, trust scoring, valid intervals, stewardship queue. |
| mii-memory | Global/workspace/session scopes, required tags, expiration primitives, one-shot alerts. |
| memory-mcp | System Primer, taxonomy, conflict audit, admin separation, retrieval feedback. |
| memory-engine | Content/tree/meta/time schema, temporal range queries, ACL/engine isolation. |
| anamnesis | Strategic weighting, authority caps, reasoning/source fields, 4D retrieval. |
| nautilus-compass | Drift detection, negative anchors, raw local embedding lane, feedback labels. |
| MARM-Systems | Human-friendly notebooks, session logs, dashboard simplicity. |
| nebu-ctx | Small public tool surface, context gateway, compression safety, context runtime thinking. |
| shokunin | Claim verification, freshness decay, RRF hybrid recall, codebase fact skepticism. |

## Risk Matrix From The Studied Systems

| Risk | Seen when | Mitigation for Goncho |
| --- | --- | --- |
| Feature sprawl | A memory system exposes dozens of agent tools or many loosely related concepts. | Keep public MCP tools small; hide rich internals behind context/search/review. |
| Taxonomy lock-in | Tree paths or cognitive maps become wrong but still gate recall. | Support audit, reparenting, orphan detection, and fallback hybrid search. |
| Silent false belief | LLM extraction writes confident but unsupported facts. | Preserve evidence refs, mark extraction confidence, and queue risky claims for review. |
| Primer bloat | Session boot loads an ever-growing summary. | Enforce token budgets, citations, freshness, and separate boot packs from full memory. |
| Stale code facts | Old memories mention moved files, removed flags, or changed APIs. | Run live verification before using codebase claims. |
| Privacy leakage | Hooks capture secrets or prompt-injection documents. | Redact before indexing, quarantine suspicious text, and separate raw evidence from trusted claims. |
| Review backlog | Stewardship produces more tasks than users can resolve. | Auto-apply only low-risk maintenance; batch and prioritize review-required items. |
| Hosted dependency creep | Memory requires cloud embeddings, hosted DBs, or online auth. | Make local embeddings and local SQLite the default; add remote providers as optional adapters. |
| Over-summarization | Session compression loses rare but critical details. | Keep raw evidence append-only and cite it from summaries. |
| Historical erasure | Updating memory deletes the reason old decisions made sense. | Prefer supersession with valid intervals over destructive overwrite. |

## Design Traps To Avoid

1. Do not make flat markdown files the source of truth.
2. Do not rely on vector top-k alone.
3. Do not inject uncited memories into context.
4. Do not allow silent destructive memory mutation.
5. Do not store secrets or prompt-injection text as trusted memory.
6. Do not expose dozens of tools to the agent initially.
7. Do not let the boot primer grow without a hard budget.
8. Do not collapse historical truth into current truth.
9. Do not treat all observations as equally important.
10. Do not require cloud services for the base developer workflow.
11. Do not treat RAG as the architecture; treat it as one retrieval technique inside context architecture.
12. Do not use memory when the correct answer requires live, governed data access.

## Concrete Goncho Positioning

Goncho should be:

```text
A local-first, hook-native context and belief runtime for coding agents.
It preserves raw evidence, derives claims, maintains scoped temporal beliefs,
retrieves task-specific orientation, and revises or forgets memory through
explicit lifecycle stewardship.
```

The core differentiator should be not "we remember more." It should be:

```text
Goncho helps agents know what they know, why they know it, when it may be stale,
who is allowed to use it, what not to repeat, and when to pull fresh context instead.
```

The product category should be:

```text
trust-preserving context architecture
```

Not:

```text
memory store
RAG layer
vector database
prompt stuffing system
```

## Proposed Default Goncho Flow

```text
1. Session starts.
2. Goncho identifies workspace, project, repo, branch, active task, and session lineage.
3. Goncho loads a small orientation pack:
   - project primer
   - current handoff
   - relevant canonical facts
   - recent high-signal episodes
   - known dead ends
   - verification warnings
4. User asks for work.
5. Goncho checks prompt against memory, graph, and drift anchors.
6. Agent works with hooks capturing evidence.
7. Important tool results and decisions become memory candidates.
8. Stop/pre-compact creates session summary and unresolved task handoff.
9. Background steward promotes, decays, supersedes, or queues review.
10. Next session starts from the updated orientation pack.
```

## Minimum Viable Goncho Schema

For the first implementation, avoid over-modeling. This schema is enough to start:

```sql
memories(
  id text primary key,
  scope text not null,
  type text not null,
  layer text not null default 'surface',
  status text not null default 'active',
  title text,
  content text not null,
  summary text,
  tags text,
  tree_path text,
  source text not null,
  source_ref text,
  confidence real default 0.5,
  importance real default 0.5,
  strategic_weight real default 0.0,
  valid_from text,
  valid_until text,
  observed_at text,
  verified_at text,
  supersedes_id text,
  created_at text not null,
  updated_at text not null
);

events(
  id text primary key,
  scope text not null,
  session_id text,
  event_type text not null,
  content text not null,
  metadata text,
  created_at text not null
);

memory_links(
  id text primary key,
  source_memory_id text not null,
  target_memory_id text not null,
  relation text not null,
  confidence real default 0.5,
  valid_from text,
  valid_until text,
  evidence_event_id text,
  created_at text not null
);

reviews(
  id text primary key,
  memory_id text,
  review_type text not null,
  reason text not null,
  status text not null default 'open',
  created_at text not null,
  resolved_at text
);
```

Add FTS and vector side tables around this instead of baking one vector provider into the source of truth.

## Final Recommendation

The best Goncho is a hybrid of four patterns:

1. agentmemory's hook-native local backend.
2. paradigm-memory's cognitive map and audit discipline.
3. agent-memory-mcp plus memory-mcp's lifecycle stewardship.
4. shokunin plus nautilus-compass's skepticism: verify claims and remember failures.

That combination directly addresses the academic frontier problems:

- Catastrophic forgetting becomes scoped persistence plus lifecycle promotion.
- Retrieval-induced noise becomes hybrid, budgeted, trust-aware context packing.
- Stale knowledge becomes temporal validity, supersession, and verification.
- Multi-hop reasoning becomes graph traversal and cognitive-map activation.
- Token pressure becomes primer/context-pack generation rather than memory dumping.
- Privacy risk becomes local-first storage, redaction, quarantine, and explicit audit.

Goncho should not try to be the biggest memory system. It should be the context architecture that keeps agents coherent under real engineering work.

The final thesis is:

```text
Goncho should not be a memory store.
Goncho should be a trust-preserving context architecture:
an event-sourced, temporally scoped, provenance-aware belief system
that turns raw experience and live data into auditable orientation for action.
```
