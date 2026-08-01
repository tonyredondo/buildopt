# Patch Autopilot validation and exact revert v1

Every registered recipe is bound to the existing six isolated
candidate/control runs and its registry-owned artifact adapter. Promotion still
requires current FULL_RELEVANT_VALIDATION evidence before the canonical
Ed25519 PatchBundle can create a new draft branch.

Exact reversal is one common capability rather than recipe-name special
casing. The generator accepts only verified registry entries declaring
EXACT_MODIFY_ONLY, requires every merged path to equal its signed postimage,
reads every restoration byte from the signed base revision, creates a new
canonical signed bundle, and routes it through the production verifier and
applier. Reviewed-adapter identity, digest, and evidence are preserved.

The real-Git proof covers Kotlin and Groovy archive reproducibility, root build
cache properties, and the reviewed custom-task recipe. Each forward patch
creates an immutable draft path, each inverse restores the original bytes in a
new draft path, and neither changes the default branch or customer checkout.
ADD operations and changed postimages remain fail-closed.

Run `./dev/check-patch-autopilot-validation-revert`.
