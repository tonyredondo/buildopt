# Evidence renderer project-property incident

The first otherwise complete Ktor capture at BuildOpt revision
`84f7f5a128370f2b68f921b970e32c605c130897` was rejected on 2026-08-15.
The installed measurement path accepted the reviewed Gradle project property
`-Pktor.develocity.skipBuildScans=true`, executed all warm-ups and eight paired
observations, and completed the fail-closed full-graph fallback. The evidence
renderer then rejected the same property because its Gradle-option allowlist
had not been kept consistent with the launcher allowlist.

The raw observations were eight of eight positive pairs, with a control mean
of 116,159.375 ms and a candidate mean of 15,013.750 ms. They are preserved
only to diagnose the generic validation defect. They are not accepted product
evidence because the renderer returned `MEASUREMENT_UNAVAILABLE` and no failed
observation may be discarded or repaired after measurement.

The complete rejected capture is preserved under `evidence/`. Its measurement
log SHA-256 is
`0ad10bccf060326fef5f162f201591418a0da9d2a34efd5417eb8ab477d923e3`.
The correction adds the same bounded project-property grammar to the evidence
renderer and negative tests for malformed names and values. Both independent
captures must be rerun from zero after that correction is committed.
