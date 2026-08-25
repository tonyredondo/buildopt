# Current longitudinal cohorts v1

This contract freezes the `AF-014B` public history before the current installed
BuildOpt package observes any repository timing. It prevents favorable-window
selection, result-dependent replacement and drift between the five terminal
repository rows.

For each declared branch, the freezer takes the 31 newest commits on one
contiguous first-parent chain at freeze time. The oldest commit is the anchor;
the next 20 are primary observations and the final ten are an ordered reserve
queue. A primary may be excluded only for a declared infrastructure,
buildability or exact-output reason, and replacement consumes the next unused
reserve regardless of its later timing or BuildOpt decision.

The repository key selects evidence and presentation only. Product behavior
must not branch on it. Change shapes are recomputed solely from changed paths,
and the independent checker binds repository, branch, JDK, workflow, output
contract, commit parent/tree, ordering and path digest. Unknown JSON fields are
rejected, which prevents timing or post-result annotations from entering the
frozen manifest.

The manifest proves experimental preregistration, not performance. Builds run
sequentially in `AF-014C`; reproducible checkouts and caches may be removed only
after immutable evidence is secured. Production hardening, soak, design
partners and Test Optimization remain outside this POC.
