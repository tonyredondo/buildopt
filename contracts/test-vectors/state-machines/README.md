# State-machine vectors

`state-machines.v1.json` is the executable `F0-023` contract for the separate
task-qualification, action-rollout, and durable-attempt lifecycles.

The 12 scenarios cover valid paths, `INCONCLUSIVE` without promotion, task
contract drift atomically suspending a dependent action, revalidation,
rollback, a lost response plus idempotent replay, skipped transitions, stale
CAS, conflicting command reuse, post-task cancellation, dead-owner
reconciliation, and terminal-state safety. Incomplete evidence never becomes
authorization.

Validate the machine definitions against the existing action/attempt schemas
and execute every transition with:

```bash
./dev/check-state-machines
```
