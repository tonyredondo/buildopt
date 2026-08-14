# Groovy candidate target-convergence incident

The first capture on BuildOpt revision
`eb873f512fb2bda0e0d1d2a9beee7cb224287563` stopped before pair 1 because
the final two candidate target-workload confirmations did not converge. Both
observed 25 tasks, but one recorded 15 executed and 9 restored from cache while
the next recorded 16 executed and 8 restored from cache. The control arm had
already converged on one exact 218-task fingerprint.

This is retained as an unresolved pre-measurement incident, not a performance
or output-equivalence result. The generic measurement error now reports a
bounded task-path and terminal-outcome diff so the oscillation can be
attributed before deciding whether any correction is justified. No timed pair
was produced, no result was discarded, and no workflow, task, output rule,
threshold, pair order, fallback, product branch, or POC boundary changed.
