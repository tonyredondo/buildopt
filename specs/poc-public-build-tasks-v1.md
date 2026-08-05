# Public build-task value experiments

This contract corrects the decision drawn from the public-workflow diagnostic
without rewriting its raw observations. Test execution dominated Mockito and
SpotBugs, but that does not place test-source compilation, test fixtures,
resources, code generation, or packaging outside Build Optimization. The
product owns that build preparation; Test Optimization continues to own which
tests run and how they are ordered, sharded, or retried.

Mockito's exact workflow took 629.165 seconds. Its `build` invocation took
593.290 seconds and `:mockito-core:compileTestJava` alone occupied 242.690
seconds, or 40.91% of the invocation wall time. The task compiles 402 test
sources and already belongs to the Tier 1 core-source-set `JavaCompile`
allowlist. The prior diagnostic disabled Build Cache and forced every task, so
these figures identify a cost centre but do not prove that BuildOpt improves
it.

SpotBugs remains a negative decision for build preparation in this exact
workflow. Its requested `Test` task occupied 242.120 of 271.920 seconds, while
`:spotbugs-tests:compileTestJava` occupied only 1.119 seconds. The task is in
BuildOpt's ownership, but the profile does not support a material generic
hypothesis under the unchanged threshold.

## Preregistered experiments

The Spotless experiment preserves both commands from its real sanity workflow,
including `testClasses`. The candidate may replace only the second command
with the preregistered affected-project alternative. It must produce identical
main and test classes and must still run the full `spotlessCheck` command.
The pinned 2025 source is normalized identically in both arms to the 2026
copyright header required by Spotless at measurement time, and its mutable
`origin/main` ratchet name is anchored to the pinned revision. The original
source-archive digest and normalized source digest are both recorded; this
preflight correction happens before warm-up and never enters a timed pair.

The Mockito experiment isolates build preparation with
`:mockito-core:testClasses`. Both arms execute the same task graph after an
unmeasured warm-up and clean-output reset. The control uses Gradle's optimized
native local cache; the candidate uses BuildOpt's private L1 plus the Tier 1
policy through the explicit `BUILDOPT_SAFE_CACHE=1` POC opt-in. Parity is not
enough: the candidate must clear the same 500-ms, 2%, and positive-lower-bound
gate. If it does not, Safe Cache stops for this workload class. Only a
qualifying mechanism may proceed to the exact three-command workflow, where
every requested test must remain unchanged.

No Gradle `Test` task is requested by either mechanism experiment. This is not
a way to select or omit tests; it measures only production of the classes and
resources that tests consume. No whole-workflow saving may be claimed from an
isolated mechanism result.

Validate the frozen preregistration with:

```bash
./dev/check-poc-public-build-tasks
```
