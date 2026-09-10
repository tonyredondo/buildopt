# Proposed next investigation: observed Checkstyle work

Date: 2026-09-08. Status: **unstarted proposal**. This is a proposed amendment
to the H1 subject selection, not authorization to bypass the failed
`ForbiddenPatternsTask` gate or restart a retired mechanism.

The [completed native prefix](./bv002/opportunity.md) identifies a stronger
place to investigate: `:server:checkstyleMain` runs for 184.178 task seconds
across three real source changes, and its checkstyle group is the last
precommit dependency at ordinals 19 and 20. Combined Checkstyle task spans are
316.072 seconds, with overlap. This is observed cost, not promised savings.

The immediate question is whether existing native Checkstyle capabilities
already avoid that work correctly. A native feature must strengthen N before
any new incremental implementation is compared against it. Enabling an
existing setting does not by itself establish a new BuildOpt product mechanism.

| Step | Work and measurements | Required outcome |
|---|---|---|
| C1: Freeze the changed subject and semantic boundary | Name the exact Gradle/Checkstyle versions, current native task, full source and configuration inputs, custom rule classes, suppressions, XML outputs and failure behavior. Keep source/classpath inputs complete. | A source-bound contract or a reason to reject the subject before coding. |
| C2: Audit native capabilities first | Inspect the actual Checkstyle engine cache/incremental behavior, its invalidation model and the owner's rules. Qualify any compatible native setting against cold/warm, changed/deleted/renamed sources, rule/config/suppression changes, failures and restarted processes. Charge setup and every native/nested start. | Stronger native baseline, `NATIVE_ALREADY_INCREMENTAL`, or evidence of work that remains avoidable. |
| C3: Admit only residual material work | Reuse the already observed engineering data as discovery evidence; keep ordinals 21-100 unconsumed. Separate action work, report construction, Gradle overhead and competing dependency paths. Compute the full-workflow ceiling after C2 and keep the 5% / 1-second floors. | A qualified G1 opportunity with exact semantics, or a negative decision. No task/output omission and no selected fast-row claim. |
| C4: Preserve the multi-repository denominator | Recover the six frozen endpoints for Groovy, Kafka, Spring Framework, OpenTelemetry Java, Micronaut and Hibernate. Complete their history prerequisites and retain every availability/opportunity outcome. Amend the tracker explicitly before changing seed/replication dependencies. | A reviewable selection rule and complete denominator; an Elasticsearch finding alone cannot establish transfer or product viability. |
| C5: Return to the existing proof gates only if admitted | Use an owned subject worktree, exact patch/inverse and all BV-003/BV-004 semantic criteria; freeze candidate and runner before chronological confirmation. Preserve the later fixed/adaptive comparison and customer gates. | A tested candidate and, eventually, actual lifecycle value; no early self-managing product claim. |

Size C1/C2 before any new native run. A starting estimate is a six-hour,
20-Gradle-start admission allocation with the existing 120 GiB footprint and
40 GiB free-space guards; include any nested starts explicitly. It is an
unstarted estimate, not a live allocation or a promise that all correctness
cases will fit. Stop admission when the native capability solves the work or
no correct material intervention remains.

Do not lower the economic floors, select only commits that touch server,
consume validation rows to choose an intervention, repair unrelated compiler
or Spotless behavior as a product thesis, or revive annotation/cache inventories.
The generic H1/H2/H3 hypotheses and the six other repository outcomes remain
unverified. There is no watcher or background investigation running.
