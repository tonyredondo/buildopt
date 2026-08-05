# Public-repository performance replication

This contract preregisters `POC-REALWORLD-002` before any public-repository
timing is observed. It compares optimized native Gradle with the installed
BuildOpt entry point on the exact Spotless, Mockito, and SpotBugs revisions
admitted by `POC-REALWORLD-G01`.

Each repository has two separate cells:

- `NO_CHANGE` executes the same full graph in both arms and must remain within
  the two-percent native-parity guardrail;
- `LEAF_SOURCE_CHANGE` appends the same non-semantic source comment to both
  workspaces. Native Gradle executes the preregistered full entrypoints while
  BuildOpt executes the declared Build Impact alternative. The affected class
  outputs must be byte-identical and the declared unrelated task must be absent
  from every candidate sample.

The source-change alternative is an owner-operated POC experiment, not a
production promotion. It does not bypass the existing 30-day/3,000-decision
promotion contract or claim that the public repository adopted BuildOpt.

## Measurement boundary

Every cell uses private workspaces, Gradle homes, BuildOpt state, and persistent
Gradle daemons for its two arms. An online preflight resolves only public source
and dependencies; containers are disconnected before the offline warm-up and
all eight measured pairs. Pair order alternates in two four-pair batches with
opposite starting arms. Static onboarding, cloning, packaging, dependency
resolution, and warm-up are outside measured time.

The optimized native arm uses the Wrapper, build cache, parallelism, daemon,
and Configuration Cache only where the repository passed the unmeasured
compatibility audit. The candidate uses the installed `buildopt gradle`
command, its default native L1, and no previously rejected Runtime Tuning
profile.

An accelerator cell requires at least 500 ms and 2% mean savings, a positive
paired-bootstrap lower bound, identical non-empty outputs, and zero product
failures. A parity cell may regress by at most 2%. Thresholds cannot move after
measurement and percentages are never added across cells.

The public claim broadens only if both cells qualify in all three repositories.
Any other outcome retains the bounded synthetic claim and records the exact
failed or unstable cells without authorizing product tuning by itself.

Before the first Mockito timing, the preregistration was amended once to hash
`mockito-subclass`'s real resource output instead of a nonexistent classes
directory. The failed preflight produced no Mockito sample; the complete
matrix is restarted from zero. Tasks, mutations, thresholds, pair ordering,
and decision rules are unchanged.

Validate the frozen contract or checked evidence with:

```bash
./dev/check-poc-real-world-performance --spec-only
./dev/check-poc-real-world-performance
```
