# OpenTelemetry fallback-equivalence correction

This directory preserves the fresh OpenTelemetry-only terminal evidence for
`POC-GENERIC-PROFILE-MATRIX-003`. It corrects only the untimed fallback proof
that failed in v3. The source, 53-entrypoint workflow, required outputs, 12
workers, eight alternating timed pairs, thresholds, and measured scheduling
remain unchanged.

Both measured daemons are stopped before fallback. The full graph then runs
with `--no-daemon` while retaining `--parallel --max-workers=12`. This avoids
overlapping three Gradle heaps without changing the scheduling mode that
produced the stable measured outputs.

| Graph | Native mean | BuildOpt mean | Saved | Interval | Result |
| ---: | ---: | ---: | ---: | ---: | --- |
| 1,024 -> 34 projects | 83.934 s | 71.825 s | **12.110 s / 14.43%** | +9.819..+14.267 s | **Qualified**, 8/8 positive |

All eight pairs used four observations per order, had a zero-millisecond
process-only inter-arm gap, produced the same 125 required files with SHA-256
`53269cb1...9c0d`, and reported zero product-attributable failures. The
full-graph fallback completed successfully and reproduced the required output.
The independent evaluator returned `EXACT_EVIDENCE_QUALIFIED`; review remains
mandatory and activation remains non-automatic.

Validate this bundle without network access:

```bash
./dev/check-generic-profile-matrix \
  benchmarks/results/poc-generic-profile-matrix-v4 \
  specs/poc-generic-profile-matrix-v4.json \
  opentelemetry-java-instrumentation
```

Final five-repository reporting combines this corrected OpenTelemetry row with
the immutable v3 Spring, Kafka, Micronaut, and Groovy rows. It must label the
revision split and must never average repository percentages or add mechanism
percentages.
