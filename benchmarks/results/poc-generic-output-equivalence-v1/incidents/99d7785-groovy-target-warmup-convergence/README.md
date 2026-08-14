# Groovy target-warmup convergence incident

The first terminal attempt on BuildOpt revision
`99d7785d662a1fc0a1779857c08b4c4d7cb1fb01` completed eight positive,
semantically equivalent pairs and a full fallback. It saved 55,250.125 ms on
average (73.86%), but correctly retained native because the single changed-
target warmup had one more cache hit than every measured candidate sample.
All eight measured control fingerprints and all eight measured candidate
fingerprints were internally stable.

This is a bounded convergence problem, not output drift or measured-pair
instability. The generic runner now performs three changed-target warmups. It
requires the last two fingerprints to match before pair 1 and rejects a
non-convergent arm without timing. The first changed-target warmup remains in
the evidence as a diagnostic but does not define the steady-state fingerprint.

No workflow, task, output rule, value threshold, pair order, fallback,
repository-specific product branch, or POC boundary changed. All terminal
captures restart from zero on one later immutable BuildOpt revision.
