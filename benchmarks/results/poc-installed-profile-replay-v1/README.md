# Public installed qualified-profile replay

This bundle proves that the six reviewed structural profiles can be consumed
through the public `v0.3.1` Linux AMD64 package and the user-facing
`buildopt poc` command. It is an adoption and correctness replay, not a new
timing experiment.

## Result

| Workflow and change | Public decision | Drift decision | Same-replay output | Historical qualification retained |
| --- | --- | --- | --- | ---: |
| Groovy `jar`, leaf source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 52,911.75 ms / 73.82% |
| Groovy `jar`, shared source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 51,589 ms / 65.73% |
| Kafka Checkstyle, metadata source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 23,703.125 ms / 28.44% |
| Kafka Checkstyle, client-utils source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 28,356.125 ms / 31.68% |
| Kafka `shadowJar`, clients source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 29,638.125 ms / 66.40% |
| Kafka `shadowJar`, generator source | `POC_CANDIDATE` | `FULL_GRAPH` | equal | 29,121.625 ms / 79.15% |

All six exact profiles selected their reviewed entrypoints. Adding harmless
whitespace to each digest-bound manifest then produced
`FULL_GRAPH / PROFILE_PRECONDITION_FAILED`. The selective candidate and its
same-run native fallback produced the same owner-reviewed semantic output in
all six cells. The terminal timing values above are preserved claims from the
prior qualification; they were not remeasured or averaged here.

Four Kafka outputs also match their historical cross-capture digest. The two
Groovy outputs do not because their JAR embeds a date-dependent `BuildDate`
property that the reviewed contract does not normalize. A direct native build
on the same frozen checkout produced the same current digest as BuildOpt, so
this is a limitation of the cross-date diagnostic rather than a candidate
correctness failure.

## Audit trail

Each cell contains the public candidate plan, drift fallback plan, normalized
logs, semantic-output records and a derived result. `summary.json` binds the
public tag, release revision, archive, installer and executable hashes.

Two pre-result corrections remain visible in the preregistered contract:

- `v0.3.0` could not consume the reviewed fourth output-equivalence binding;
  no Gradle result was accepted, and `v0.3.1` corrected the generic launcher.
- the cross-date terminal digest was demoted to diagnostic before any complete
  cell result after same-checkout native Gradle isolated Groovy's `BuildDate`.

Validate the checked evidence and its negative mutations with:

```bash
./dev/check-installed-profile-replay-result
./dev/test-installed-profile-replay-result
```

The checker reconstructs `summary.json` byte for byte from the raw cell files,
verifies every log hash and rejects summary or log tampering. Private capture
paths are normalized. No repository-name product rule, automatic activation,
production authority, new timing claim, soak, design-partner dependency or
Test Optimization scope is introduced.
