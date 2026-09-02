# Economics-Gated Reviewed Native Patch v1 evidence

The fresh source audit is 10/10 conclusive and finds action rows in five
families. Registration/effect binding admits two Elasticsearch diagnostics and
no candidate build in the other families. `:server:forbiddenPatterns` executes
for 7,141 ms on the hard-dependency critical path and exceeds both the 500-ms
and 2% gates; `:server:filepermissions` takes 317 ms and rejects.

The two-line marker-only patch for `ForbiddenPatternsTask` passes five bounded
correctness starts: execution, same-root restore, cross-root restore, content
invalidation and restore after exact input revert. The output SHA-256 remains
`a4c3ed04...2211`, the public source reverts exactly and product failures are
zero.

V1 stops before a value result because its single stabilization covered the
candidate root but not the control root's build-logic classpath transition.
The contaminated rows are retained only in `invalid-value-attempt.json`, have
zero accepted timing samples and are forbidden inputs to v2. No v1 speedup is
claimed.
