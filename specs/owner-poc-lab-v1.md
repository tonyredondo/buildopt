# Owner-operated POC lab v1

This contract closes `POC-O2` with one synthetic, source-preserving command
that can run without an owner repository or external infrastructure. It
composes existing production paths instead of implementing a second cache or
Edge simulator.

Run the lab and stream its machine-readable result to standard output:

```bash
./dev/run-owner-poc-lab
```

Or publish the result atomically outside the source tree:

```bash
./dev/run-owner-poc-lab --output /tmp/buildopt-poc-lab-result.json
```

The four fail-fast steps prove a real synthetic Gradle deliverable, repeat the
tamper-evident Shared fault slice three times under the race detector, resolve
an online/offline two-Edge collision through Shared, and exercise the complete
Edge process/reload/status/package lifecycle. Each successful result binds the
steps to the current Git revision and records whether the source tree was clean
when execution started.

Run the strict contract and result validator with:

```bash
./dev/check-owner-poc-lab
```

Base CI runs that checker on a clean, read-only checkout. This bounded lab uses
only project-owned synthetic fixtures. It does not execute the deferred
eight-hour soak, use external design partners, or authorize production
promotion.
