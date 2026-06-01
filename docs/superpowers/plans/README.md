# Superpowers implementation plans

This folder holds date-stamped implementation plans grouped by the area they change.

## Shared planning contract

All plans in this tree should keep these contracts explicit instead of restating ad hoc process in new files:

- Use Goncho TDD discipline: name the failing public-interface test, implement the smallest behavior slice, then run the focused test before broad validation.
- Preserve public evidence paths, benchmark artifacts, migrations, generated files, and documented APIs unless the plan names the compatibility change.
- Prefer focused contracts, interfaces, value types, or helpers at the owning package boundary before adding cross-cutting utilities.
- Record verification commands and commit boundaries in the plan so agentic workers can resume safely.

## Topology

- `core/` — memory-kernel, lifecycle, retrieval, and governance plans.
- `docs-site/` — documentation-site implementation plans.
- `integrations/` — donor/provider integration plans that adapt external memory-system ideas to Goncho contracts.
- `locomo/` — LOCOMO benchmark and recall-improvement plans.
- `roadmap/` — larger cross-cutting roadmap and landscape plans, including the agentmemory feature-matrix roadmap.
