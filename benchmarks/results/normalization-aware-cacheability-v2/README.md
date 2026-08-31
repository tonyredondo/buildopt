# Normalization-aware cacheability v2 evidence

`NAC-002` completes the fresh source-only classification in
[`source-classification.json`](./source-classification.json). All 5/5 frozen
revisions are conclusive and 4/5 families expose an action, passing the frozen
3/5 breadth threshold. The 18 typed rows contain eight marker-only candidates,
one reviewed-relative-proof-required candidate, seven already-cacheable tasks
and two incomplete/ambiguous tasks.

The report was regenerated from the five frozen Git source trees by the
versioned v2 classifier. The independent checker derives family counts from
rows, verifies source digests, reruns the classifier and rejects any DNO report
dependency. No candidate was patched, built or timed; `NAC-003` is next.
