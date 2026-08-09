# Generic isolated structural measurement

`buildopt profile measure` closes the manual gap between a structural
opportunity and `buildopt profile evaluate`. It measures the exact installed
BuildOpt candidate against optimized native Gradle without sharing source,
Gradle home, native build cache or execution order between the two arms.

The repository supplies the semantic contract that Gradle cannot infer:

- the original and alternative entrypoints in the Build Impact manifest;
- required output paths;
- the exact base-to-target changed path set;
- a global-change path set that must restore the full graph; and
- the common Gradle option vector.

The command requires a clean tracked target revision and a distinct immutable
ancestor revision. It checks that the changes file exactly matches `git diff`
before running any build. Local clones are warmed independently at the base
revision. Eight target-revision pairs alternate native-first and
candidate-first order. Every run starts from its arm's independently restored
native-cache seed, no pair permits more than five seconds between arms, and the
installed launcher overhead is included.

The emitted evidence binds both the declared BuildOpt revision and the SHA-256
computed from the exact installed executable used for the candidate arm.

```bash
buildopt profile measure \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json \
  --changes-file buildopt-changes.txt \
  --fallback-changes-file buildopt-fallback-changes.txt \
  --base-revision "$BASE_REVISION" \
  --buildopt-revision "$BUILDOPT_REVISION" \
  --evidence-output buildopt-structural-evidence.json
```

The default control enables the native Gradle build cache, parallel execution,
the daemon and plain console output while disabling Configuration Cache and
build scans. Repeat `--gradle-option` to replace that vector for a repository's
real optimized-native policy. Both arms always receive the same vector.

Evidence is `QUALIFIED` only with eight positive pairs, at least 500 ms and 2%
mean savings, a positive paired-bootstrap lower bound, byte-identical stable
outputs, no product failure and a successful output-identical full-graph
fallback. A valid but non-positive experiment writes `INCONCLUSIVE` evidence.
Build failure, output mismatch, source drift or unsafe paths write no evidence.

The output is reviewable input to the existing decision command:

```bash
buildopt profile evaluate \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json \
  --evidence buildopt-structural-evidence.json \
  --profile-output buildopt-qualified-profile.json
```

Neither command activates the resulting profile. This is POC workflow
compression, not autonomous tuning or production promotion. Validate the
contract with `./dev/check-poc-generic-measurement`.
