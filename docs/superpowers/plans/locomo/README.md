# LOCOMO implementation plans

LOCOMO plans share a stricter benchmark contract:

- Prove each recall or evaluation change with a focused failing test before implementation.
- Score and document behavior through stable inserted `memory_id` evidence, not answer text.
- Do not use answer hints, LLM judges, or full frozen-artifact regeneration for small recall slices.
- Preserve frozen LOCOMO artifacts unless a plan explicitly schedules a new date-stamped full run.
- Update roadmap/docs only after behavior is proven by the focused command and broad Go validation.

These rules are the reusable policy for the LOCOMO plans in this folder; individual plans may add narrower guardrails for graph, query-decomposition, temporal, speaker-routing, or failure-bucket work.
