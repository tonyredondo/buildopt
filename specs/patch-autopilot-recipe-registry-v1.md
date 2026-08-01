# Patch Autopilot recipe registry v1

The registry is the single closed allowlist shared by PatchBundle verification
and candidate validation. Selection requires an exact recipe ID and version;
there is no closest-version negotiation or caller-provided metadata.

Each entry fixes applicability, risk, artifact validation, exact-inverse
capability, and whether a reviewed adapter is mandatory. Adding or changing a
recipe therefore requires a source change, conformance evidence, and a new
version rather than accepting bundle-supplied policy.

Run `./dev/check-patch-autopilot-recipe-registry`.
