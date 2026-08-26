# Portable wrapper connection and project identity contract

Status: accepted POC contract for `SWL-005`.

This contract binds the portable endpoint and project scope committed in
`.buildopt/config.toml` to one private, repository-scoped credential. It proves
that two clean checkouts discover the same authorized project without making a
checkout path part of identity. It does not yet configure Gradle's HTTP cache,
consume BuildOpt state or authorize an optimization.

The exact machine contract is
[`poc-sticky-wrapper-connection-v1.json`](./poc-sticky-wrapper-connection-v1.json).

## Portable and private inputs

The committed file names one canonical server origin, one `owner/repository`
project scope and the name of an environment variable. It contains neither the
credential value nor a machine path. The named environment variable contains
the exact `buildopt.central/access-token/v1` JSON document returned once by
`buildopt-server central-token issue`.

The document must bind the same repository and derived repository-scope
SHA-256, be unexpired, use a canonical capability order and grant both
`CACHE_READ` and `STATE_READ`. Write capabilities may also be present, but this
block never uses them. A missing variable means that a fork or untrusted CI job
has no connection and runs native Gradle without a network request.

## Stable identities

Project identity is the domain-separated SHA-256 of the committed
`project_scope`; the absolute checkout path is never an input. The connection
identity additionally binds server origin, tenant, repository, trust domain,
cache namespace and namespace generation from the owner-issued credential.

Therefore:

- the same project and credential produce the same identity on clean machines;
- moving or recreating a checkout does not create a new project;
- a different cache namespace or generation cannot reuse connection state; and
- a token issued for another project is rejected before any request.

## Authentication probe

The verified BuildOpt binary performs bounded, non-mutating capability probes:

1. a `HEAD` request to the repository-scoped state route proves `STATE_READ`;
2. a `GET` for a connection-derived absent Gradle key proves `CACHE_READ` and
   the exact namespace binding.

External endpoints require canonical HTTPS with TLS 1.3 or newer. Numeric
loopback HTTP exists only for local fixtures. Redirects are disabled, the
request budget is ten seconds and proxy discovery uses the environment. The
server remains the live authority for token revocation.

## Native fallback and secret boundary

Absent credentials are silent native fallback. Invalid configuration,
mismatched scope, incomplete capabilities, expiry, revocation, redirect or
network failure emits one controlled diagnostic and retains native Gradle.
None of those states can read a cache object or typed state document.

The wrapper root and the dynamically named credential variable are removed
from the Gradle process environment. The token is not persisted by this block
and is not written to logs or results.

## Evidence boundary

The executable gate creates two independent checkout roots and a real central
HTTPS handler. It proves portable identity, namespace separation, pre-network
rejection, missing-secret behavior, both read capabilities, live revocation,
redirect rejection, foreign-wrapper rejection and dynamic secret scrubbing.
It also compiles the touched packages for macOS ARM64 and Windows AMD64.

Passing this contract proves only a safe connection boundary. The subsequent
`SWL-006` cache contract consumes a valid connection through the native Gradle
HTTP cache while keeping the central path read-only; typed state and decisions
belong to `SWL-007` and later blocks. No build-time or production claim is made
by this connection contract.
