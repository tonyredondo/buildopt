# Build Impact POC hot state

`buildopt impact` may reuse a previously validated POC plan only when
This is a historical contract for a retired mechanism. In the original
experiment, `--hot-state-dir` and `--repository-revision` were supplied
together. The key
binds repository/pipeline/revision, manifest, graph, generated state, changed
paths, Wrapper files, BuildOpt executable and Gradle options. State is private
and atomic; every drift is a miss followed by normal fail-closed planning.

This caches selection state, not build outputs or production authority.
