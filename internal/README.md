# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

For the component-to-package dependency map and the owning executable,
contract, fixture, and validation path, see the
[repository map](../docs/architecture/repository-map.md#go-package-map). Each
package exposes a `go doc` package comment that states its authority and
failure boundary; exported symbols should document non-obvious lifecycle,
security, persistence, or side effects close to the code.

`adaptivefragment/` owns the `AF-001`..`AF-010` POC contracts for repository-
scoped, path-independent optimization fragment identity and immutable typed
state. It distinguishes stable families from evidence-bound revisions,
validates correctness authorities and declared semantic bindings, evaluates
only bindings consumed by each fragment, and links fragment, observation,
portfolio and economic-ledger generations through closed JSON schemas and JCS
digests. Its discardable repository-local compatibility index returns
candidates, suspension reasons or native retention from pre-Gradle
fingerprints without mutating lifecycle state. Its pure pre-Gradle planner
selects only exact directly predicted dependency-closed compositions, treats
conflicts symmetrically, retains every constituent authority and returns native
Gradle on ambiguity or insufficient net value. Its Build Impact activation
requires exact subgraph/materialization pairs with producer-specific contexts,
restores only verified current bytes, rebuilds changed producers locally and
rebuilds expired or unqualified producers without suspending unrelated pairs,
and returns to the complete native workflow before execution on unsafe state. It
does not synchronize fragment state or claim composed wall-time value. Its
frozen-history shadow replay separates
structural compatibility from output-byte freshness and economic
authorization without consuming future observations.
Its economic assessment preserves signed observed value, deduplicates stable
asynchronous cost events, reports exact recurrence, decay/payback projections
and unclipped regret without granting activation.
Its online learner accepts only exact requested-build cohorts, returns a new
canonical checkpoint generation, resumes under exact digest/repository/binding
identity and limits value-regression suspension to dependent fragments.
Its cross-repository prior ranks generic task/plugin/Gradle/structure
hypotheses without using repository identity as a feature or transferring
correctness, value or activation authority.
Its patch-opportunity detector turns repeated expensive non-cacheable task
evidence into a review-only proposal. It cannot authorize or apply a patch;
the exact recipe, transactional validation and native Gradle value gate remain
separate authorities.

`nativevolatility/` owns the POC producer-atomic portability boundary. It
compares complete, exact-bound native output observations, quarantines every
producer associated with a differing output, excludes all outputs of those
producers from transport and verifies exact bytes for everything BuildOpt
reuses. Missing or ambiguous paths, bindings and producers retain native Gradle.
It contains no repository identity or filename-extension policy and grants no
production authority.

`structuralbinding/` owns the revision- and checkout-path-independent POC
compatibility fingerprint for qualified structural profiles. It binds the
repository scope, workflow and options, Wrapper, complete candidate/producer
task lineage, exact required/candidate output ownership and change family.
Incomplete, ambiguous or cyclic evidence is rejected; evidence ancestry and
revision-bound output bytes remain launcher responsibilities.

`ordinarylearning/` owns the bounded economics applied to useful observations
from customer-requested Gradle builds. It validates structural compatibility,
exact outputs, portability and product outcomes, then permits continued
learning only when projected compatible lifetime repays within five matches.
It never authorizes an extra measurement build or production behavior.

`stickywrapper/` owns the four repository-committed wrapper files, embedded
POSIX/Windows bootstrap templates and the maintainer `init`, read-only `check`
and distribution-only `update` commands. Generation resolves immutable public
release metadata, validates strict portable configuration and publishes through
a rollback-safe transaction. The generated scripts select and checksum-verify
the native archive, reject unsafe extraction, verify the internal manifest and
atomically publish a user-cache installation. They never pass credentials to
Gradle or grant runtime/performance authority; Gradle passthrough belongs to
SWL-004.

`stickyactive/` owns the SWL-011 fail-closed execution boundary. It accepts a
signed, exact-bound runtime profile only after revalidation, runs candidate and
authoritative native commands without a shell, compares required output hashes,
and suspends a profile on failure, mismatch or regression. It deliberately
contains no repository, task-name or filename rules and makes no production or
performance claim from synthetic control-flow timings.

`stickyvalue/` owns SWL-014B's repository-independent paired value evaluator.
It uses checked signed integers, a deterministic paired bootstrap, nearest-rank
p95 and nine explicit cost categories. Missing cost or inexact/failed evidence
cannot qualify an action; qualification is evidence, never execution authority.

`launcher/` contains the dependency-free `WS-001` command passthrough, the
`WS-002` Linux process-group and signal contract, the `WS-003` plugin handshake,
and the neutral `WS-004` authenticated local rendezvous used by `cmd/buildopt`.
It forwards `SIGINT`/`SIGTERM` to the child group, preserves child status, owns
the private event socket and loopback readiness gateway, and consumes the
`F0-039` local bypass before creating either service or parsing server
configuration. The bypass uses the same process/signal contract and removes all
reserved launcher state from the child. Grace-period escalation remains with
the invoking CI environment. `A0-001` adds the opt-in managed runner-slot path:
a current-user private
state root, exclusive invocation and gateway leases, a detached idle-bounded
process, UID-authenticated invocation registration, context-gated readiness,
restart-stable identity, and complete rotation when the endpoint cannot be
recovered. `A0-003` adds the launcher-owned native L1 lifecycle: opaque
tenant/repository/trust/compatibility scoping, generation-segmented private
directories, an exclusive child-lifetime lease, and local-cache disablement
for pending L2 writers. `A0-006` authenticates canonical local policy and
cumulative revocation state before Gradle, persists anti-rollback generations,
derives L1 authority from the signed state, and gives the gateway an
invocation-only Shared credential over its same-UID control channel. The
gateway translates Gradle's local Basic credential, rejects redirects, and
routes no cache request without current context. A1-004 adds a shared
deployment-lifecycle lease and rejects L1 generations below a completed
deletion tombstone.

`sessioningest/` contains the provisional `WS-005` gateway-to-server record,
strict authenticated HTTP transport, and concurrency-safe in-memory acceptance
store. Its optional `WS-006` handoff carries only predeclared tokenized context
and facts from an authenticated Gradle invocation.

`buildsession/` is the dependency-free producer for the normative
`BUILD_SESSION v1` schema and the atomic local-file exporter. It derives only
deterministic manifest/baseline digests, declares unobserved metrics
unavailable, publishes mode-`0600` immutable complete/partial JSON, and owns a
bounded private JSONL stream with deterministic at-least-once replay and
startup recovery. A1-004 applies keyed repository/trust/task redaction before
either durable form and enforces explicit bounded profile authorization.
Runtime schema conformance remains with the isolated
validator under `dev/schema-validator/`.

`datalifecycle/` owns the isolated private-beta profile policy, HMAC
tokenization, shared lifecycle leases, durable logical revocation, coordinated
known-component removal, tokenized tombstones, and post-deletion generation
floors consumed by Shared, managed L1, and export.

`localauthority/` owns the A0-006 canonical JCS/Ed25519 authority, pinned
trust-root and private-file boundary, strict semantic validation, and durable
monotonic policy/revocation state used independently by launcher and Shared.

`sharedcache/` owns the A0-004..A0-006 single-node storage and publication
boundary used by `buildopt-server`: private same-filesystem SHA-256 blobs, a
process-lifetime writer lease, independently migrated WAL-mode
`cache.sqlite`/`control.sqlite`/`state.sqlite`, durable pending attempts,
canonical Ed25519 decisions, atomic first-writer visibility, context-bound
opaque HTTP GET/PUT, quarantine, startup reconciliation, current
local-authority records, and restart-safe typed portfolio/evidence/checkpoint
state. It verifies complete bytes before returning a cache hit or state
snapshot, persists no raw data-plane credential, rejects stale/rolled-back
authority, and never derives authority from blob presence. The optional central
POC handler exposes those two logical planes only over TLS, authenticates four
independent cache/state capabilities from domain-separated token digests, and
checks expiry/revocation on every request. Automatic client forwarding remains
outside this package boundary.

`edgecache/` owns the optional MVP-C2 boundary. C2-001 adds the strict private
single-node configuration, loopback listener and authenticated Shared-origin
rules, bounded local-storage declaration, and immutable Shared-only commit and
collision authority. C2-002 adds authenticated Shared-only committed read-
through, complete content-address verification before SQLite publication, and
per-read exact current-revocation authorization across offline restart. C2-003
adds conservative byte reservations, hard quota admission, durable TTL,
probation/protected byte-SLRU, 85/75 pressure maintenance, and transactional v1
metadata migration. C2-004 adds a separate signed write authority, exact-attempt
pending reads, durable queued/replicating/replicated/rejected metadata,
authenticated asynchronous Shared PUT with retry/restart recovery, and no local
promotion. C2-005 exposes only the Gradle-compatible GET/PUT route on an
explicit IPv4 loopback listener and proves two independent Edge roots preserve
attempt-local candidates while Shared alone selects the committed winner.

`neutralenvelope/` owns the strict `WS-009` observation and report contract. It
pairs externally timed native and optimization-off wrapper executions,
reconciles required-output digests, retains signed differences, and binds the
runner, metric catalog, envelope, launcher, server, and plugin inputs.

`buildimpact/` owns the C3 conservative graph-omission boundary. C3-001 loads
only bounded, strict, repository-contained customer manifests bound to one
repository and pipeline class, digests their canonical form, and rejects
inferred entrypoints, ambiguous ownership, unsafe paths, symlinks, and any
unknown-change policy other than `FULL_GRAPH`. C3-002 binds a complete
Gradle-declared graph to that digest, maps source changes through transitive
reverse dependents, and predicts only customer-enumerated alternatives that
cover every affected project, artifact, and Build-owned check. Test-owned work
is preserved and execution remains on the original entrypoints. Automatically
normalized cyclic components retain both their expanded conservative source
boundary and their member-specific owned source roots; production uses the
former, while explicit POC attribution may use the latter and fails closed on
equal-specificity ownership. C3-003 records
strict manifest/graph/adapter-bound full-baseline and paired-control
observations, classifies candidate build/content/check/project divergence as a
false negative, and keeps infrastructure or invalid baselines inconclusive.
C3-004 aggregates only current-binding results through the unchanged BIA-002
minimum window, sample, coverage, per-stratum controls, and exact one-sided
false-negative bounds. Binding drift resets the sample, insufficient evidence
is inconclusive, and one known false negative suspends. Every validation and
promotion result still disables selection. C3-005 owns the sole active
boundary: it revalidates loaded digests, recalculates BIA-002 from bound
observations, selects only a customer-manifest alternative, and restores the
original entrypoints for every disabled, bypassed, killed, drifted, unknown,
global, unqualified, or invalid path while preserving Test-owned checks.

`profilediscovery/` owns the read-only POC specialization boundary. It binds
terminal matrix evidence to complete Build Impact state, trace/input digests,
the exact output and reviewed profile contract, then emits either a reviewable
profile or an explicit native full-graph decision. It never writes repository
state, activates a profile, recognizes repository names, or grants production
authority. The same package also owns the earlier opportunity-analysis stage:
it can identify and quantify a smaller complete graph across repositories, but
it emits only a measurement proposal and never infers wall-clock value or
enables an additional mechanism from structure alone.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.

`generated/openapi/` contains the checked-in Go transport binding derived from
the normative OpenAPI documents. It is regenerated through
`./dev/generate-code --artifact openapi-go-client-v1` and never edited
manually.
