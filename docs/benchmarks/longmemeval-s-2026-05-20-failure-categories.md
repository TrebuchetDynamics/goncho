# LongMemEval-S Failure Category Report

This report classifies remaining hard cases before any ranking optimization. It is diagnostic only; it does not change scoring or retrieval.

- system: `goncho`
- dataset: `longmemeval-s-cleaned`
- MRR: `0.9135`
- recall_any@10: `0.9800`
- hard cases classified: `45`

## Counts by bucket

| Bucket | Count |
| --- | ---: |
| misses in top 10 | 10 |
| rank-2 cases | 22 |
| rank-3 cases | 13 |

## Counts by category

| Category | Count |
| --- | ---: |
| `benchmark_gold_ambiguity` | 4 |
| `direct_answer_mismatch` | 6 |
| `lexical_miss` | 10 |
| `numeric_entity_exactness` | 2 |
| `temporal_ambiguity` | 23 |

## Examples

- `10d9b85a` rank `2` category `temporal_ambiguity`: How many days did I spend attending workshops, lectures, and conferences in April?
  - reason: query asks for time, order, recency, duration, or before/after comparison
  - relevant: `answer_e0585cb5_2`, `answer_e0585cb5_1`
  - retrieved top 5: `c51583cd_3`, `answer_e0585cb5_1`, `answer_e0585cb5_2`, `84889496_1`, `8dd2fca2`
- `195a1a1b` rank `2` category `direct_answer_mismatch`: Can you suggest some activities that I can do in the evening?
  - reason: relevant evidence is retrieved but not at the top rank
  - relevant: `answer_6dc4305e`
  - retrieved top 5: `9f12daa1_1`, `answer_6dc4305e`, `ultrachat_396124`, `87fff4b4_1`, `ultrachat_511593`
- `1d4e3b97` rank `2` category `direct_answer_mismatch`: I noticed my bike seems to be performing even better during my Sunday group rides. Could there be a reason for this?
  - reason: relevant evidence is retrieved but not at the top rank
  - relevant: `answer_e6b6353d`
  - retrieved top 5: `ecb24dd8_2`, `answer_e6b6353d`, `809cabca_1`, `c51583cd_3`, `1b320137_1`
- `2ebe6c92` rank `2` category `benchmark_gold_ambiguity`: Which book did I finish a week ago?
  - reason: multiple gold IDs or abstention variants suggest ambiguous strict attribution
  - relevant: `answer_c9d35c00_1`, `answer_c9d35c00_2`
  - retrieved top 5: `16756728_1`, `answer_c9d35c00_2`, `9d4312f6_3`, `ca3a4e4f_1`, `fb303dd2_2`
- `57f827a0` rank `2` category `temporal_ambiguity`: I was thinking about rearranging the furniture in my bedroom this weekend. Any tips?
  - reason: query asks for time, order, recency, duration, or before/after comparison
  - relevant: `answer_1bde8d3b`
  - retrieved top 5: `e85e0eaf_1`, `answer_1bde8d3b`, `d0f42e3f`, `5150a4e9_2`, `1fc5074c_2`
- `61f8c8f8` rank `2` category `temporal_ambiguity`: How much faster did I finish the 5K run compared to my previous year's time?
  - reason: query asks for time, order, recency, duration, or before/after comparison
  - relevant: `answer_872e8da2_2`, `answer_872e8da2_1`
  - retrieved top 5: `a3107e2a_1`, `answer_872e8da2_1`, `answer_872e8da2_2`, `7b88c38b_2`, `6e110a53_1`
- `8a2466db` rank `2` category `numeric_entity_exactness`: Can you recommend some resources where I can learn more about video editing?
  - reason: query asks for exact count, amount, name, entity, or object identification
  - relevant: `answer_edb03329`
  - retrieved top 5: `6a5b5a78`, `answer_edb03329`, `6dcf5fa0_1`, `d8b3e1c8_2`, `3392c0c7`
- `b86304ba` rank `2` category `numeric_entity_exactness`: How much is the painting of a sunset worth in terms of the amount I paid for it?
  - reason: query asks for exact count, amount, name, entity, or object identification
  - relevant: `answer_645b0329`
  - retrieved top 5: `sharegpt_xGoJZ6Z_0`, `answer_645b0329`, `ea8bb4f8_2`, `fe1e4351_1`, `ultrachat_328696`
- `gpt4_59149c78` rank `2` category `benchmark_gold_ambiguity`: I mentioned that I participated in an art-related event two weeks ago. Where was that event held at?
  - reason: multiple gold IDs or abstention variants suggest ambiguous strict attribution
  - relevant: `answer_d00ba6d1_1`, `answer_d00ba6d1_2`
  - retrieved top 5: `23754665`, `answer_d00ba6d1_2`, `sharegpt_GI6737T_2`, `answer_d00ba6d1_1`, `a8ac3d1d_1`
- `gpt4_a56e767c` rank `2` category `benchmark_gold_ambiguity`: How many movie festivals that I attended?
  - reason: multiple gold IDs or abstention variants suggest ambiguous strict attribution
  - relevant: `answer_cf9e3940_2`, `answer_cf9e3940_1`, `answer_cf9e3940_3`
  - retrieved top 5: `88c8df0e_3`, `answer_cf9e3940_3`, `answer_cf9e3940_2`, `answer_cf9e3940_1`, `d75245ea`
- `0edc2aef` rank `3` category `direct_answer_mismatch`: Can you suggest a hotel for my upcoming trip to Miami?
  - reason: relevant evidence is retrieved but not at the top rank
  - relevant: `answer_d586e9cd`
  - retrieved top 5: `f20e72e4_1`, `6a747f2e`, `answer_d586e9cd`, `53dc1394`, `08ca1f31_2`
- `06f04340` rank `0` category `lexical_miss`: What should I serve for dinner this weekend with my homegrown ingredients?
  - reason: no relevant ID appears in the retrieved top-k despite non-empty retrieval results
  - relevant: `answer_92d5f7cd`
  - retrieved top 5: `91223fd5_1`, `6e6fbb6b`, `0844dea6`, `8b156015_2`, `42924d15`
- `09d032c9` rank `0` category `lexical_miss`: I've been having trouble with the battery life on my phone lately. Any tips?
  - reason: no relevant ID appears in the retrieved top-k despite non-empty retrieval results
  - relevant: `answer_b10dce5e`
  - retrieved top 5: `3fc2244f`, `sharegpt_e9sAtcZ_63`, `af631aa3_2`, `26d9aaaf`, `e8bfacec_2`
- `2698e78f_abs` rank `0` category `lexical_miss`: How often do I see Dr. Johnson?
  - reason: no relevant ID appears in the retrieved top-k despite non-empty retrieval results
  - relevant: `answer_9282283d_abs_1`, `answer_9282283d_abs_2`
  - retrieved top 5: `sharegpt_IsRvBnc_11`, `ultrachat_182084`, `9e21d6ab_1`, `cdba3d9f_1`, `9316aae3_1`
