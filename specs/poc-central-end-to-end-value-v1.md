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

## Subjects and observations

The terminal run uses at least two different substantial public Gradle
families and eight alternating pairs per family. Each observation records wall
time, pair order, central `FROM-CACHE` outcomes, graph selection, selection and
state-sync overhead and the exact digest of every required owner output.

A family qualifies only when:

1. required outputs match exactly in every pair;
2. both arms demonstrate central cache reuse;
3. at least seven of eight pairs are positive;
4. the paired 95% lower bound and mean saving are positive; and
5. candidate p95 is lower than control p95.

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
