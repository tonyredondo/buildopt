# Automatic central profile reuse in `buildopt optimize`

## Purpose

This POC connects the already proven central state transport to the one-command
optimization path. After a repository has been connected once,
`buildopt optimize <Gradle workflow>` automatically looks up verified remote
portfolio/evidence state before Gradle and publishes newly learned local state
afterward. A separate `buildopt sync` command is no longer required in the
normal optimize flow.

The machine-readable contract is
[`poc-central-optimize-integration-v1.json`](./poc-central-optimize-integration-v1.json).
The executable proof is
[`dev/check-central-optimize-integration`](../dev/check-central-optimize-integration).

## Customer flow

Connect the checkout once:

```bash
buildopt connect https://buildopt.example.com \
  --token-file ./buildopt.token \
  --ca-file ./buildopt-ca.pem
```

Then keep using the POC command:

```bash
buildopt optimize build
```

An unconnected checkout behaves exactly as before. A connected checkout runs a
bounded pre-sync, tries exact local replay first, then considers a remote
profile. At completion it persists local state and value reports, attempts a
post-sync, and only then publishes the invocation result. Credentials stay in
the private connection directory and are never passed to Gradle.

## Cross-commit selection boundary

A remote profile is not trusted merely because the server returned it. Before
Gradle starts, BuildOpt requires all of the following:

1. canonical snapshots whose portfolio references the exact evidence manifest;
2. exact repository, Wrapper, BuildOpt executable, workflow and Gradle options;
3. an evidence commit that is an ancestor of the current commit;
4. no intervening Gradle/build-logic change;
5. current changed paths owned by the verified graph;
6. the same structural change family and required outputs;
7. exact profile preconditions and recomputed qualification evidence; and
8. a newly computed Build Impact plan that still selects the qualified tasks.

This allows reuse across ordinary source commits without pretending that an
old profile is universally valid. A build script, Wrapper, tool, workflow,
unknown path, family, output or evidence change retains optimized native
Gradle before the target process starts.

## Offline and publication behavior

Service failure is fail-open for the build. Only a previously stored snapshot
whose canonical bytes, digests, scope, kind, generation and evidence reference
all verify may be considered offline. Otherwise BuildOpt runs native and makes
no remote-selection claim.

Post-build publication is also non-authoritative. An unavailable server leaves
the local result complete and reports the publication failure; it cannot alter
the Gradle exit code. Immutable-object, manifest and compare-and-swap semantics
remain owned by the central state-sync contract.

## Evidence and interpretation

The selection test uses the retained public Kafka `shadowJar` qualification:
eight pairs, 30,106.5 ms mean saving and a 66.87% reduction at its evidence
commit. A source-only descendant is re-owned and replanned successfully; an
additional `build.gradle.kts` commit is rejected before Gradle. The transport
test proves automatic pre/post synchronization and verified offline lookup.

Those Kafka timings are existing qualification evidence, not a new central
wall-time result. Remote evidence contributes observed performance, but
calibration economics remain unavailable because the portfolio does not yet
republish the original calibration cost. The next two-machine block must
measure the complete installed central path under the same native Gradle cache
opportunity.

## POC boundary

`productionAuthorized` remains false. This block adds no autonomous production
promotion, production HA/RBAC/KMS/backups, soak requirement, design-partner
dependency, repository-name rule or Test Optimization behavior.

## Verification

```bash
./dev/check-central-optimize-integration
```

The checker validates the machine contract, runs the real Kafka evidence
revalidation and structural-drift case, and exercises automatic publication
plus verified offline lookup through the real TLS handler under the Go race
detector.
