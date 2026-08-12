# Generic output-contract preflight on Hibernate ORM

This frozen POC evidence exercises the generic output-contract preflight on
the original, intentionally incorrect Hibernate owner declaration. It uses
public revision `2b448a59d332326f0cd0691c868425124d55cbb5`, root workflow
`assemble`, and declared output `hibernate-core/build/libs/**`.

## Result

The preflight executed the exact owner workflow once and returned
`NATIVE_FULL_GRAPH / REQUIRED_OUTPUTS_EMPTY`. The declared pattern matched
zero files because Hibernate redirects module build directories to `target`.
The same generic Gradle inspection exposed the main, sources, and Javadoc JARs
under `hibernate-core/target/libs/`, each owned by `:hibernate-core` and its
corresponding producer task.

This is a usability and correctness result, not a performance result:

- zero measurement warm-ups ran;
- zero timed observations ran;
- no structural graph or qualified profile was written; and
- no Hibernate-specific path rule was added to BuildOpt.

The evidence demonstrates that a wrong or stale owner output declaration is
now rejected before structural discovery and paired measurement, while the
review artifact provides concrete Gradle-owned candidates for correction.

## Reproduce the checks

Validate the synthetic discovery, validation, empty-pattern, and ambiguous-
ownership cases:

```bash
./dev/check-generic-output-contract
```

Validate this frozen public-repository observation against its preregistered
contract and checksums:

```bash
./dev/check-generic-output-contract-evidence
```

The full Hibernate execution requires network access and a substantial build:

```bash
./dev/run-generic-holdout \
  /absolute/evidence/directory \
  specs/poc-generic-holdout-v1.json
```

The runner uses the current installed BuildOpt path. A failed preflight stops
before timing and preserves native Gradle as the only executable decision.

## Boundaries

This evidence is review-required proof-of-concept validation. It authorizes no
automatic activation, production use, soak requirement, design-partner work,
or Test Optimization behavior.
