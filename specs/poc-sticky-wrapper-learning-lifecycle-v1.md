# Sticky wrapper learning lifecycle v1

This POC contract proves that the already implemented observation, paired
trial, signed decision, active execution and economic-ledger components can be
composed without giving cache data execution authority or weakening native
Gradle fallback.

## Customer path

`internal/launcher/sticky_learning.go` is the single composition root. An
ordinary committed wrapper invocation selects only a verified local decision;
it does not wait for the network. Native, candidate and counterfactual commands
continue through the launcher process supervisor. `BUILDOPT_STICKY_LEARNING`
and credentials are removed before Gradle starts.

Trusted learning additionally requires committed `mode = "auto"`, a non-zero
trial budget no greater than five percent, `BUILDOPT_STICKY_LEARNING=1`, and a
credential carrying `CACHE_READ`, `STATE_READ` and `STATE_WRITE`. Missing or
invalid authority retains native Gradle.

## Lifecycle and value

The fixture exercises:

1. `UNSEEN → OBSERVE → SHADOW → TRIAL → QUALIFIED → ACTIVE`;
2. exact paired outputs and conservative value qualification;
3. a signed active decision and periodic native counterfactual;
4. regression suspension followed by retirement; and
5. ten negative cases that all retain native execution.

Value is computed by `stickyvalue.Evaluate`. Pair effects are signed native
wall time minus candidate wall time. Its deterministic paired bootstrap,
nearest-rank p95 and cost ledger use checked signed 64-bit arithmetic. Every
cost category must be present—even when its observed value is zero—or the
action cannot qualify.

This is POC evidence, not production rollout authority. Durable proposals stay
review-only, and the fixture does not mutate a customer branch.

## Verification

```bash
./dev/check-sticky-wrapper-learning-lifecycle
```

The committed result is
`benchmarks/results/sticky-wrapper-learning-lifecycle-v1.json`.
