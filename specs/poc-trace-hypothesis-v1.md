# Trace-gated hypothesis decision

This contract decides whether retained installed-path traces justify one new
BuildOpt optimization experiment. It does not collect timing, search parameter
values, activate mechanisms, or turn diagnostic trace durations into a
performance claim.

## Authorization rule

A hypothesis is authorized only when the same named phase:

- is product-addressable and causally recoverable;
- exposes at least 500 ms of non-overlapping critical-path work;
- satisfies that threshold in at least two independent repository or workload
  families;
- preserves every required output with zero product-attributable failures; and
- is not a previously rejected mechanism being reopened without materially
  new trace evidence.

At most one hypothesis may be emitted. When no phase satisfies every rule, the
only valid result is `NO_ACTIONABLE_HYPOTHESIS` and no product implementation or
accepted timing follows.

## Retained evidence

The analysis consumes two immutable, SHA-256-bound inputs:

1. the installed native-only phase trace for the realistic synthetic Kotlin and
   Groovy workload families; and
2. the installed Spring verification trace with candidate-specific BuildOpt
   phase timing.

Task durations that overlap under Gradle parallel execution are not summed into
critical-path savings. Existing Build Impact task-interval savings remain
evidence for that already-qualified mechanism, not a new hypothesis.

## Result

No phase qualifies:

- BuildOpt-specific setup reaches at most 1.238233 ms;
- launcher and Gradle-client startup reaches at most 364.875 ms and has no
  retained causal attribution;
- Gradle finalization reaches at most 97 ms;
- launcher and Gradle-client teardown reaches at most 87 ms and has no retained
  causal attribution; and
- configuration reaches 682 ms only in Spring, is not causally attributed to
  BuildOpt, and does not clear 500 ms in a second workload family.

The terminal decision is `NO_ACTIONABLE_HYPOTHESIS`. Runtime Tuning, Hot State,
Standard Copy, Safe Cache and Standard Jar remain closed absent materially new
trace evidence. The next decision is the terminal POC portfolio synthesis.

Validate and reproduce the checked result with:

```bash
./dev/check-poc-trace-hypothesis-v1
```

This is POC evidence only. It changes no production authority, requires no
soak or design partner, and excludes Test Optimization.
