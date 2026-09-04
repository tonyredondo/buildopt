# WCNCP-009A attempt 1

This immutable attempt completed the three permitted unmeasured dependency
prefetches and then stopped at the stability gate before every controlled
diagnostic. The seven primary samples produced a 5.179528996 maximum/minimum
ratio against the frozen 1.15 maximum. The result is therefore
`INCOMPLETE_PERFORMANCE_ENVIRONMENT` with zero public-source mutations,
controlled diagnostic starts, candidates, paired value samples, speedup claims,
or product failures.

The independent checker reconstructs the ratio from
[`preflight-rows.jsonl`](./preflight-rows.jsonl), compares those raw rows with
[`environment.json`](./environment.json), verifies the embedded environment in
[`result.json`](./result.json), and rejects any claimed family output. Prefetch
console logs were setup diagnostics, not outcome evidence, and are not retained.

Run:

```bash
./dev/check-wcncp-controlled-materiality \
  "$PWD/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009a/attempt-1"
```
