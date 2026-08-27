# Sticky-wrapper customer status and explanation v1

This contract closes `SWL-013`. It gives a customer a small, read-only view of
what the committed wrapper knows and why the next build will retain native
Gradle or consider a sticky action. It is a reporting surface, not an
authorization path and not a performance claim.

## Commands and routing

The generated wrapper reserves management commands behind the explicit prefix:

```text
./buildoptw --buildopt status [--json]
./buildoptw --buildopt explain [--json]
```

The unprefixed forms remain ordinary Gradle tasks, so a project task named
`status` or `explain` is not shadowed:

```text
./buildoptw status
./buildoptw explain
```

`--gradle` can always force the latter interpretation. Status and explanation
run after the pinned distribution has been verified, but they do not start
Gradle, contact a service, write observations, or modify the repository.

## Report model

Both commands build one `buildopt.sticky/status/v1` report. `status` prints a
short summary and `explain` prints the same model as a more verbose explanation;
`--json` serializes that exact model for scripts. The sections are:

- `wrapper`: pinned distribution version, mode and whether a project/server
  binding is configured;
- `decision`: the safe next execution state and plain-language reason;
- `observations`: counts, outcomes and measured wall/Gradle/cache time;
- `trials`: verified candidate/control count, or an explicit unavailable value;
- `cache`: transport and hit/miss counts only when they were actually recorded;
- `economics`: signed gross saving, BuildOpt cost and net saving when a verified
  ledger exists;
- `fallback`: whether native Gradle is retained and why;
- `bindings`: exact repository, source, Gradle, Wrapper, arguments and BuildOpt
  digests from the latest validated observation; and
- `explanation`: sentences rendered from the other fields, never from a second
  source of truth.

Every numeric measurement has `state: AVAILABLE` with a value and unit, or
`state: UNAVAILABLE` with a reason and no value. Missing evidence is never
turned into zero. Ordinary observation timing is not a cache hit/miss count,
and an observation is not a trial or an economic ledger entry.

## Safety and tamper behavior

Status validates the committed wrapper files before reading private state. The
ordinary JSONL log is loaded with its canonical-byte, scope and timing checks;
malformed, cross-repository or arithmetic-inconsistent records fail the
management command rather than being skipped. A local decision that cannot be
verified with the configured authority key registry is shown as unavailable and
retains native Gradle. No URL, token, credential name, checkout path or other
secret is copied into the report.

The report is deliberately POC-scoped. It does not authorize an action, merge a
patch, claim customer-wide speedup or replace the frozen campaign and terminal
scorecard.

## Validation

Run the focused checker:

```bash
./dev/check-sticky-wrapper-status
```

The checker exercises a clean wrapper, a validated ordinary observation, an
empty state, read-only behavior, management routing and a tampered log. The
same Go tests reject an unavailable measurement that tries to carry numeric
zero, so arithmetic and missing-evidence semantics cannot be silently changed.
