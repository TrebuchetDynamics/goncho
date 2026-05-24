# BEAM Paired Outcome Comparison

Deterministic paired comparison over Mnemosyne-compatible `paired_outcomes.jsonl` rows. Scores are joined by scale, conversation, and qid; unpaired rows are dropped.

- Source: `./artifacts/beam-smoke/paired_outcomes.jsonl`
- Baseline config: `mnemosyne-smoke`
- Candidate config: `goncho-smoke`
- JSON report: `./docs/benchmarks/results/beam-smoke-paired-comparison.json`
- Paired questions: `1`
- Dropped unpaired rows: `0`
- Effect-size floor: `0.0200`
- Verdict: `candidate_superior` (`candidate_ci_above_effect_floor`)
- Bootstrap: `200` samples, seed `42`, score-delta 95% CI [`+0.5000`, `+0.5000`]

## Score summary

| Ability | Paired | Baseline avg | Candidate avg | Δ score | Candidate wins | Baseline wins | Ties |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| OVERALL | 1 | 0.5000 | 1.0000 | +0.5000 | 1 | 0 | 0 |
| MR | 1 | 0.5000 | 1.0000 | +0.5000 | 1 | 0 | 0 |

## Interpretation

Use this report as the BEAM arm-comparison oracle: a positive Δ means the candidate config scored higher on the same paired questions. Treat CIs crossing zero as inconclusive and inspect per-ability rows before claiming superiority.
