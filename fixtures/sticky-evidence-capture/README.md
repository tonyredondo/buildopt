# Generic evidence producer fixtures

These fixtures exercise the public `SWL-FRESH-001` producer without using a
repository name, task name or path-extension rule:

- `kotlin-task-action.json` proves a complete Kotlin DSL capture can emit a
  review-only task-contract action while the graph producer completes with no
  opportunity;
- `groovy-graph-action.json` proves the same API can emit a review-only graph
  action for Groovy DSL while the Java task detector is not applicable; and
- Go conformance tests cover conclusive no-opportunity, not-applicable,
  unavailable input, producer failure and corrupt source bindings.

The DSL field is provenance. The implementation never branches on it except to
validate that the capture came from one of the two supported fixture families.
Every action leaves costs explicitly unavailable until a later measurement
block and grants no patch or runtime authority.
