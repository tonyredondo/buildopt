# Terminal POC portfolio decision

This contract converts the accepted installed-path evidence into one terminal
BuildOpt POC decision. It does not collect another timing sample, average
repository percentages, add mechanism effects, or turn a bounded result into a
general product claim.

## Decision policy

- `CONTINUE_GENERAL_POC` requires at least two independent repository families
  to qualify through the installed path with exact required outputs, zero
  product-attributable failures, and complete fallback.
- `SPECIALIZE_BOUNDED_KAFKA_PROFILE` applies when exactly one family qualifies.
  Only its exact reviewed profile remains available; every other family stays
  on optimized native Gradle.
- `STOP_OR_REFRAME_POC` applies when no installed family qualifies or when the
  user-facing path loses the measured value or safety contract.

Repository percentages are never averaged and mechanism effects are never
added. The decision was fixed before this synthesis.

## Result

The terminal decision is `SPECIALIZE_BOUNDED_KAFKA_PROFILE`:

- Kafka qualifies through the installed profile at **28,523.25 ms / 81.85%**
  mean saving, with 8/8 positive pairs, exact normalized output, native
  full-graph fallback, and byte-identical HTTP-503 local fallback.
- Spring retains optimized native Gradle. Its installed candidate averaged
  1,895 ms / 14.33% faster, but one of eight pairs regressed by 57 ms and the
  unchanged all-positive gate failed.
- OpenTelemetry retains optimized native Gradle. Preparation ended before any
  accepted observation, so no performance claim is inferred.
- Deterministic discovery may reproduce the exact reviewed Kafka profile, but
  it remains read-only, review-required, and never activates automatically.
- The trace gate found no new generic product hypothesis: the largest
  BuildOpt-specific setup delta was 1.238233 ms, far below the 500-ms rule.

The broad accelerator claim is withdrawn. The retained POC claim is only that
the fixed Kafka repository, change, output, normalized input, and modeled
network profile can benefit from Build Impact plus read-only Edge locality.

Validate and reproduce the checked decision with:

```bash
./dev/check-poc-portfolio-decision-v1
```

This closes the current evidence plan. It authorizes no production hardening,
soak, design-partner work, autonomous activation, new timing, or Test
Optimization.
