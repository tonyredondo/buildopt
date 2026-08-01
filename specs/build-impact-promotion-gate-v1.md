# Build Impact promotion gate v1

This contract closes `C3-004` and implements the unchanged `BIA-002` initial
promotion threshold per repository and pipeline class. Qualification requires
at least 30 days, 3,000 eligible decisions, 99% validation coverage, 100 full
controls in every mandatory change class, no known false negatives, and exact
one-sided 95% zero-failure upper bounds no greater than 0.1% aggregate and 3%
per mandatory stratum.

Every result is bound to the manifest digest, declared-graph digest, and
adapter version. A change to any binding excludes prior evidence and resets the
current sample. Duplicate, malformed, cross-scope, incomplete, or insufficient
evidence remains `INCONCLUSIVE`; one false negative immediately produces
`SUSPENDED`.

Run:

```bash
./dev/check-build-impact-promotion-gate
```

The checker proves the threshold with a deterministic 3,000-decision corpus,
the complete negative matrix, and the exact confidence calculation. It also
evaluates the two checked-in C3-003 observations and requires their honest
state to remain `INCONCLUSIVE`. Even a `QUALIFIED` report retains
`selectionAuthorized=false`; only C3-005 may consume it to plan a
customer-owned alternative.
