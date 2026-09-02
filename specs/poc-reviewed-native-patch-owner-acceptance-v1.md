# Reviewed Native Patch Owner Acceptance v1

`REVIEWED_NATIVE_PATCH_OWNER_ACCEPTANCE_V1` tests the remaining owner-operated
product boundary: whether a repository owner can understand and decide on each
qualified native Gradle correction from a review-only draft without BuildOpt
implementation knowledge.

The fixed cohort contains the two RNPD recipes at their exact public revisions:

- `REVIEWED_RELATIVE_CACHEABILITY_JAVA_V1` for Micronaut revision
  `428ddeb3ad2acdabef2027cc06af3bf46865956a`; and
- `REVIEWED_MARKER_ONLY_CACHEABILITY_JAVA_V1` for Spring revision
  `91eb42645e26a7ef9382b4a655bcefe5c8682fee`.

Each proposal is presented in a draft pull request inside an owner-controlled
fork. Its base branch points to the exact historical revision and its candidate
branch contains only the digest-bound source correction. No upstream pull
request, default-branch update, merge, force-push or automatic application is
allowed.

Before reading the proposal, the owner starts an active-review timer. The owner
records elapsed active review time, one decision, the number of clarification
questions and any safety or comprehension concern. BuildOpt may not infer or
fabricate those fields. The recommended POC gate is at most 15 active minutes
and one clarification per proposal, with both proposals accepted for a
controlled trial. Machine and human time remain separate quantities.

Acceptance qualifies only the review experience and measured owner-operated
economics. It does not authorize merge, upstream submission, production,
automatic application, automatic merge, soak, design-partner work or Test
Optimization.
