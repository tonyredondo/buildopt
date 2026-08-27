# Sticky-wrapper durable catalog fixtures

`catalog.json` is the checked-in SWL-012 report produced from the current
paired native Gradle evidence. `pass-false.json` is a negative vector: changing
the report gate must be rejected before a catalog can be consumed.

The fixtures verify the report shape only. They do not authorize applying a
recipe, merging a source change, or claiming customer coverage.
