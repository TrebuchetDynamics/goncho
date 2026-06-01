# LOCOMO recall validation contract

Shared benchmark policy: [LOCOMO benchmark contract](../shared/benchmark-contract.md).

All LOCOMO recall plans validate retrieval identity with the stable inserted `memory_id`. Use no answer hints, no LLM judges, no answer-text scoring. Preserve frozen LOCOMO artifacts unless a plan explicitly schedules a new date-stamped full run.

Every recall slice must name and pass a focused public behavior test named by the plan before broad validation. After the focused proof passes, run the repository gate:

```sh
go test ./... -count=1
```
