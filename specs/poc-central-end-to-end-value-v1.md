# Central installed-path value contract

## Question

The centralized POC is useful only if it improves customer-visible wall time
beyond what the same native Gradle remote cache already provides. This contract
therefore compares two installed BuildOpt paths against the same committed
central cache objects:

- the control runs the complete owner workflow with `buildopt gradle`; and
- the candidate runs `buildopt optimize`, reuses a centrally learned structural
  profile and requests only the qualified subgraph.

Both arms use the same HTTPS service, repository scope, cache namespace and
read-only credential. They have separate workspaces, Gradle User Homes and
local BuildOpt caches. Dependency and Wrapper warm-up is not measured. Project
outputs and local build-cache entries are removed before every observation, so
both arms receive the same opportunity to fetch committed remote objects.
Before either consumer starts, an explicit no-change sync proves that the
producer's automatic post-run publication already made evidence, portfolio and
checkpoint generation 1 remotely visible. The diagnostic sync may not repair a
missing publication.

## Resource envelope

Both arms use the same maximum of eight Gradle workers. Ktor also uses the same
2 GiB Gradle heap and 1.5 GiB Kotlin daemon heap in both arms, supplied through
each isolated Gradle User Home rather than as workflow arguments. This still
exercises more than the four-CPU minimum while allowing the two hot measurement
daemons and their Kotlin compiler daemons to coexist on the 16 GiB POC host.
The producer daemon is stopped after central publication because it no longer
participates in the comparison. The candidate's first connected bootstrap can
transiently use native and replay daemons, so those bootstrap daemons are also
stopped before the final paired warm-ups. The measured state then contains one
hot daemon per arm. These controls prevent unrelated completed phases from
causing an OOM; they are not Runtime Tuning and do not advantage the candidate.

## Subjects and observations

The terminal run uses at least two different substantial public Gradle
families and eight alternating pairs per family. Each observation records wall
time, pair order, central `FROM-CACHE` outcomes, graph selection, selection and
state-sync overhead and the exact digest of every required owner output.

A family qualifies only when:

1. required outputs match exactly in every pair;
2. both arms demonstrate central cache reuse;
3. the one-time discovery and calibration cost repays within at most 50
   matching builds across the shared central scope;
4. at least seven of eight pairs are positive;
5. the paired 95% lower bound and mean saving are positive; and
6. candidate p95 is lower than control p95.

The 50-build horizon is the already measured upper bound for the complete
comparative POC, whose previous cells repaid within 19 to 50 builds. It is a
shared-team horizon because one qualified portfolio is reused by every
compatible consumer of the central scope. It does not weaken the direct
onboarding gate for an isolated developer, and changing this bound requires a
new contract revision rather than silently reinterpreting retained evidence.

The seven-of-eight count is not permission to ignore a failure. The retained
raw observation must identify every non-positive pair, while the paired
interval and p95 prevent one favorable outlier from creating a passing result.

## Honest native decision

One build-logic or global-configuration change must retain the complete native
graph before Gradle starts. The connected central cache remains usable for
that full build, but the case makes no performance claim. A non-selective
decision is a required safe result, not a failed optimization.

## Interpretation

The result attributes only the end-to-end difference between the two complete
installed paths. Cache-hit ratios and graph reduction are diagnostics; their
percentages are not added. Results remain repository-specific and are not
averaged into a universal product claim.

This is POC evidence. It does not authorize production promotion, require a
design partner or soak, add repository-name rules, or include Test
Optimization.
