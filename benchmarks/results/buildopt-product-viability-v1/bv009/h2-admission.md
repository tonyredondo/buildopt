# BV-009: Conditional invalidation-repair admission

Date: 2026-09-08. Status: **verified, NO_ADMITTED_INVALIDATION_REPAIR**.
[Machine decision](./h2-decision.json), [native prefix](../bv002/opportunity.md).

This was a read-only admission check of two concrete signatures observed in the
20-transition prefix. No additional Gradle starts or H2 patch were made.

| Inspected cause | Current source and native evidence | Outcome |
|---|---|---|
| Over-broad module-path fingerprint | `CompileModulePathArgumentProvider.getModulePath()` already has `@CompileClasspath`. At ordinal 19 the private-constant/static-initializer change leaves ESQL and core module compilers `UP-TO-DATE`. Other reasons cite changed/removed class API inputs. | Not admitted: the proposed missing normalization is already present. Internal compiler rebuild minimality remains unproved. |
| Checkstyle invalidated by unrelated compiler outputs | `CheckstylePrecommitPlugin` explicitly sets an empty classpath. Main lint is up-to-date at ordinal 18 and executes on genuine source edits at 19/20. | Not admitted: required source inputs cannot be omitted. Per-file processing and native engine caching are a separate investigation. |

The three inspected source files have one Git blob version across the anchor
and every prefix revision; exact SHA-256 pins and native result objects are in
the machine decision. Source shape, a long compiler duration or an
`incremental=false` task result does not by itself identify a correct repair.
The Spotless Configuration Cache failure is a native compatibility finding,
not evidence of unnecessary successful-task invalidation.

Keep H2 closed for these inspected causes. Reopening needs a distinct observed
source cause and a fresh bounded admission, not another broad annotation or
source inventory. The conditional 20-start H2 experiment allocation was never
started because no cause passed admission.
