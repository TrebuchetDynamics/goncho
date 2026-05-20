# LOCOMO Dataset Provenance

Goncho uses the official LoCoMo repository for the full LOCOMO retrieval benchmark.

## Source

- Repository: `snap-research/locomo`
- URL: `https://github.com/snap-research/locomo`
- Pinned revision: `3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376`
- Source file: `data/locomo10.json`
- Source SHA256: `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`
- License file: `LICENSE.txt`
- License SHA256: `41003d4a74749c0220e33dd415042164b5a1093ed401f36277234f772d22d3d0`
- License note: Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0)

## Conversion

Command:

```sh
python3 ./scripts/prepare_locomo.py \
  --raw-dir ./data/locomo/raw \
  --out-dir ./data/locomo
```

Outputs:

- `data/locomo/memories.jsonl`
- `data/locomo/questions.jsonl`
- `data/locomo/metadata.json`

Converted artifact checksums from the current pinned conversion:

- memories SHA256: `bd24ddbebb21e3dfeb9108c4f869048afc8d0425003424b37630bde1b35b48ff`
- questions SHA256: `904c90f99963b9744117d4bfabd5f7570044c94d014c8b05a42ff444af27e5cd`

Counts:

- memories: `5,882`
- questions: `1,982`

The official dataset contains two QA evidence references that do not map to dialogue turns in `locomo10.json`. The converter records them in `data/locomo/metadata.json` and keeps only resolvable gold memory IDs:

- `locomo-conv-42-D10-19`
- `locomo-conv-47-D4-36`

## Evaluation rule

This benchmark is retrieval-first only:

- no LLM judge,
- no answer-generation scoring,
- deterministic ID-based memory retrieval metrics,
- `answer_hint` is parsed for leakage reporting only and is not indexed or scored.
