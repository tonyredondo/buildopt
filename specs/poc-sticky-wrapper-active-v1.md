# Sticky wrapper active execution POC v1

This contract closes `SWL-011` for the owner-operated sticky-wrapper learning
POC. It proves the smallest safe path from a qualified signed decision to a
runtime candidate, while making native Gradle the authority for correctness
and regression detection. It is a mechanism proof, not production rollout
authority.

## Activation gate

The active runner consumes a signed `STICKY_DECISION` only when the decision
is unexpired, not revoked, cryptographically valid, `ACTIVE_RUNTIME_PROFILE`,
`QUARANTINE_VALIDATED`, and exactly equal to the current action and binding.
The checked-in SWL-010 report is deliberately rejected: it has four exact
pairs but zero positive pairs and a mean saving of `-555.008 ms`.

There is no repository name, task name, file extension or hand-authored
profile rule in the runner. The profile supplies two direct commands and a
relative required-output set; commands are never passed through a shell.

## Execution and suspension

Every invocation revalidates the decision before starting a candidate. A
qualified candidate runs first. On the configured counterfactual cadence, the
same required outputs are produced by the authoritative native command in a
separate directory. Candidate and native output trees must hash identically.

The runner suspends the action and retains native execution when any of these
conditions occurs:

- candidate failure or cancellation;
- native counterfactual failure;
- missing or different required outputs; or
- candidate duration exceeds native duration plus the configured tolerance.

After suspension, later invocations remain native until a new decision
generation is supplied. Bypass, expiry, revocation, binding drift and an
unsupported decision all retain native without executing the candidate.

## Evidence

Run the focused checker:

```bash
./dev/check-sticky-wrapper-active
```

The checker builds [`sticky-active-benchmark`](../cmd/sticky-active-benchmark),
executes eight direct synthetic scenarios and validates the Draft 2020-12
schema. The versioned result is
[`benchmarks/results/sticky-wrapper-active-v1.json`](../benchmarks/results/sticky-wrapper-active-v1.json).

The current result shows one active execution saving roughly 24.6 ms with
exact outputs, three suspensions (regression, output mismatch and candidate
failure), and four native retentions (bypass, binding drift, expiry and
revocation). These timings are synthetic control-flow evidence; they are not
a claim about a customer repository. The only repository-level trial used for
qualification remains the negative SWL-010 result above.

## POC boundary

The runner is an executable control boundary for the POC. It does not claim
that a profile is profitable merely because a decision is signed, does not
replace the repository Gradle Wrapper, and does not add a production service,
soak campaign or design-partner requirement. SWL-012 will address durable
native optimization proposals; SWL-013 will expose the lifecycle and
economics through the customer-facing status surface.
