# Repository-independent structural POC profile

## Purpose

This contract converts exact installed-path evidence into a reviewable
Build-Impact-only profile. It is deliberately independent of repository names,
Gradle DSLs, plugins and known project layouts. A repository is eligible only
when its own complete manifest, graph, generated binding and measured evidence
agree byte for byte.

Structural analysis alone remains a measurement proposal. It cannot create a
profile because graph reduction does not prove lower wall-clock time. The
qualification step additionally requires eight alternating comparisons against
the repository's optimized native Gradle path, at least 500 ms and 2% mean
savings, a positive deterministic 95% paired lower bound, eight positive pairs,
identical stable outputs, zero product-attributable failures and a successful
native-full-graph fallback.

## Commands

Analyze a checked repository graph without granting authority:

```bash
buildopt profile analyze \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

After the proposed scope has been measured under the evidence contract,
materialize a deterministic profile for review:

```bash
buildopt profile qualify \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json \
  --evidence buildopt-structural-qualification.json \
  > buildopt-qualified-profile.json
```

Run the installed POC with a repository-relative change list:

```bash
buildopt poc --changes-file .buildopt-changes
```

The qualifier reads bounded repository-relative regular files and writes only
JSON to standard output. Review and repository ownership remain explicit; it
does not mutate the checkout or activate production behavior.

## Fail-closed behavior

The qualifier refuses to emit a profile when timing, output, fallback,
mechanism, subject, plan or source binding is incomplete or inconsistent. The
v4 profile records the evidence digest and binds the manifest, graph and
generated state with SHA-256 preconditions. At execution time, drift, unknown
ownership, ambiguous attribution, a global change or local bypass retains the
native full graph.

Only Build Impact may be enabled. Safe Cache, Runtime Tuning, Hot State,
standard task adapters, Shared/Edge Cache and Test Optimization remain disabled
unless separately measured and qualified. Percentages from those mechanisms
are never added to this result.

## POC boundary

This proves that BuildOpt can carry one repository's measured structural value
through a generic installed command. It does not claim automatic activation,
universal savings, production readiness, a public release, soak completion or
design-partner validation.

Validate the contract, deterministic materialization, identity independence,
negative evidence and runtime drift fallback with:

```bash
./dev/check-poc-structural-profile
```
