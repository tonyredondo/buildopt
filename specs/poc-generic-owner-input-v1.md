# Generic owner input

## Purpose

`POC-GENERIC-OWNER-INPUT-001` replaces repeated local flags and the former
CI-only configuration with one reviewable repository-owned file. It records
the semantic inputs BuildOpt cannot infer safely: repository and pipeline
identity, the original Gradle workflow, the exact Git change source, required
outputs, global fallback paths, Gradle options, and timeout.

The default path is `.buildopt/profile.json`. The same file is consumed by the
local CLI and by the repository-root GitHub Action. It is POC input, not an
active optimization profile.

## Creation

An owner first executes and reviews the real Gradle workflow:

```bash
mkdir -p .buildopt
buildopt profile outputs \
  --repository-id owner/repository \
  --pipeline-class classes \
  --entrypoint classes \
  --required-output 'module/build/classes/**' \
  --output .buildopt/output-contract.json
```

Only a `VALIDATED_REQUIRED_OUTPUTS` result can be converted. The explicit
`--confirm` flag is required:

```bash
buildopt profile input \
  --output-contract .buildopt/output-contract.json \
  --confirm \
  --gradle-command ./gradlew \
  --output .buildopt/profile.json
buildopt profile input --check .buildopt/profile.json
```

The generated file binds the observed repository revision and exact SHA-256
of the reviewed output contract. `changeSource` is fixed to
`GIT_DIFF_BASE_TO_HEAD`; BuildOpt therefore derives the complete no-rename Git
diff rather than trusting a hand-maintained path list.

## Consumption and drift

Local proposal generation needs only the immutable base revision:

```bash
buildopt profile propose \
  --owner-input .buildopt/profile.json \
  --base-revision "$BASE_SHA"
```

The GitHub Action reads the same file. CI also preserves its derived
`changes.txt` in the review artifact so the dynamic input remains auditable.
The proposal binds the owner-file path and SHA-256.

Every proposal reexecutes the declared Gradle workflow and validates the
confirmed outputs against the current checkout. Missing, empty, symlinked, or
ambiguously owned outputs retain `NATIVE_FULL_GRAPH` and report the current
Gradle-owned candidates. Candidate graph documents and active profiles are not
written after drift.

The former `buildopt.poc/profile-ci-input/v1` remains readable only for
existing replay evidence and consumers during migration. New configurations
use `buildopt.poc/profile-owner-input/v1`.

## Conformance

```bash
./dev/check-generic-owner-input
./dev/check-generic-profile-ci
./dev/check-generic-profile-ci-replay
```

The first checker proves explicit confirmation, deterministic round trip,
automatic Git change derivation, proposal digest binding, output drift
diagnostics, and native fallback. The second proves the same schema through
the Action. The third preserves compatibility with the frozen five-repository
CI replay.

This block creates no timing, speedup percentage, automatic activation,
production authority, soak requirement, design-partner dependency, or Test
Optimization behavior.
