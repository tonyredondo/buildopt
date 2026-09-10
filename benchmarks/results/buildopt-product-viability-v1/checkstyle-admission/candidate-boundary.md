# Next candidate: content-aware native Checkstyle execution

Status: **pending implementation**. Admission is scoped to the pinned
Elasticsearch owner/configuration in [the contract](./contract.json).
This file defines acceptance work; it is not an implemented optimization.

The correction should avoid repeating successful per-file checking while
retaining the complete native Gradle input collection and ordered native
reports. An unchanged timestamp is insufficient evidence. A changed rule's
bytecode must invalidate prior checking even when XML configuration is unchanged.
Do not fork Checkstyle, introduce a central service, omit a requested task,
weaken a report, or attribute native cacheability benefits to this candidate.

| Order | Implement or verify | Required outcome |
|---|---|---|
| 1 | Bind the owner integration and native task APIs to exact source/version hashes. Identify every consumer of Checkstyle task type, reports, properties and dependencies before choosing the smallest adapter. | Exact patch and inverse, complete public/native task contract, refusal of unsupported source/version drift. A code shape alone is insufficient. |
| 2 | Define prior successful state keyed by file content and path plus checking implementation/classpath, rule configuration, suppressions, charset and roots. Retain full declared Gradle inputs. | Changed bytes with an unchanged timestamp, changed rule bytecode and changed path cannot reuse an invalid success. State is separate per task/root; unsupported cases use native checking. |
| 3 | Reuse only established successes. Incomplete/failed state, removed outputs, cache restoration without matching local history, malformed state and global semantic changes require a complete native check. | Atomic successful state publication and deterministic recovery. No partial run becomes trusted history; no hidden failure or automatic success. |
| 4 | Preserve native file enumeration and every required XML entry, field and diagnostic. Retain exclusion semantics and exact exception/failure behavior. | Byte-identical reports where deterministic; every permitted path normalization declared in advance. On failure, any full-check fallback must preserve diagnostics and charge repeated work. |
| 5 | Exercise positive reuse and all BV-004 boundaries: edits/additions/deletions/renames, empty input, same-mtime changes, rule/config/suppression/charset/root changes, malformed source, failed-then-fixed input, output/history loss, restored cache, cross-root state and unsupported versions. | Complete native-versus-candidate correctness matrix and owner integration; no mocks replacing the actual checker. The three retained native-cache counterexamples become required regression cases. |
| 6 | Verify exact inverse application and ordinary native execution after removal, including pre-existing configuration and reports. | Reviewed local correction with reversible behavior; BV-003/BV-004 may become verified only after actual proof. |
| 7 | Qualify the real replay runner, isolation, accounting, source transitions, graph/output capture, injected fault detection and restart recovery against BV-005. | A checked runner, not reuse of the admission scripts as a qualified measurement harness. |
| 8 | Run the engineering prefix, freeze candidate/runner/cost classification, then execute the original confirmation protocol. | Measured net workflow value on all scheduled transitions; two complete replications, payback, interval and latency criteria unchanged. Only then proceed to other owners and adaptation. |

The native control contains the prior reviewed ForbiddenPatterns cacheability
delta wherever its pinned applicability holds. It appears in both arms and
receives no new benefit attribution. Ordinary Gradle build-cache and up-to-date
behavior remain enabled; the unqualified raw Checkstyle engine cache remains off.

Treat the 5.9452% optimistic dependency-model result as a narrow admission
margin. The 4.74555-second headroom over the discovery prefix is a sensitivity,
not an allowance to omit report/state/qualification costs or an estimate of
future validation savings. A correct but uneconomic candidate must stop.

Use the existing task-state locator and owned Checkstyle worktree. Declare a
candidate allocation, including owner-test nested Gradle starts, before native
implementation runs; preserve all admission and predecessor charges. The owner's
standing continuation/budget authorization covers local work, while publication
and customer contact remain outside this task.
