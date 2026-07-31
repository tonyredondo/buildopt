# Private-beta token isolation v1

This contract closes `A1-002` and `A1-G01` for the isolated private-beta
profile. It does not create users, RBAC, OIDC, multi-tenancy, KMS/HSM, DSSE,
HA, or a pilot deployment.

Every remote token is an independently generated 32-byte opaque value. The
operator receives it once from `buildopt-server token issue`; the backend
persists only a domain-separated SHA-256 digest in `control.sqlite`. Its row
binds tenant, repository, trust domain, exact namespace and generation, one of
`STABLE`/`QUARANTINE`/`CONTROL`, `READ` or `READ_WRITE`, issue time, an expiry
no more than 30 days away, and optional revocation time. Reusing one token in
another plane is structurally impossible because a digest is unique and the
plane is part of its single scope.

The stable HTTP handler combines that token scope with the already verified
signed local authority. A cross-repository, cross-namespace,
cross-generation, quarantine, or control token receives `401`; a valid read
token receives `403` on `PUT`. The handler consults the durable registry on
every request, so `buildopt-server token revoke` invalidates the token before
the next request and therefore before the next build. Expired tokens also
receive `401`.

The launcher accepts the remote secret only through a private mode-`0600`
file named by `BUILDOPT_SHARED_CACHE_TOKEN_PATH`. That credential is distinct
from the signed authority's invocation credential, is installed only in the
gateway's upstream binding, and is removed before Gradle starts. Remote HTTP
is rejected; only loopback may remain HTTP and any non-loopback endpoint must
use TLS.

The GitHub fixture has disjoint jobs. An untrusted fork runs the baseline and
contains neither a secret expression nor a BuildOpt token path. A same-repo
pull request receives only the stable read token. A push to protected `main`,
behind the `buildopt-private-beta` environment, receives the distinct stable
read-write token. `pull_request_target` is deliberately absent.

Run the complete negative gate with:

```bash
./dev/check-private-beta-token-isolation
```
