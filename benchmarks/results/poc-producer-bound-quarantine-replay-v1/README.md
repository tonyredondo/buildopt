# Producer-bound quarantine replay evidence

This directory contains the bounded public-repository evidence for
`POC-PRODUCER-BOUND-QUARANTINE-REPLAY-001`.

The run used the published Spring Framework JMS change frozen by the contract,
captured a producer-bound output inventory, rebuilt the same native workflow in
an independent checkout with Gradle build cache disabled, quarantined complete
Gradle producers whose bytes changed, and then recalibrated eight alternating
control/candidate pairs.

The evidence establishes the following result:

- 8,216 outputs were compared across independent native roots;
- 14 volatile paths caused two complete producers and 352 outputs to be
  quarantined;
- 7,864 stable outputs remained transportable;
- the candidate rebuilt the quarantined producers locally and preserved the
  exact required-output digest in all 16 measured observations;
- control averaged 8,643.875 ms and the quarantined candidate averaged
  7,029.375 ms, saving 1,614.5 ms or 18.68%;
- seven of eight pairs were positive and the 95% saved-time interval was
  +549.375..+2,702.25 ms;
- the measured calibration cost projects one matching build to break even.

Files:

- `summary.json` is the compact aggregate and contains every measured arm;
- `quarantine-result.json` retains the complete producer/output decision;
- `calibration-evidence.json` is the qualification evidence used to publish the
  POC profile;
- `output-contract.json` retains the Gradle producer attribution captured before
  filtering;
- `result.json` is the final launcher result.

The run is proof-of-concept evidence only. It does not authorize production
activation, generalize the percentage to unrelated repositories, average
repository percentages, add mechanism percentages, require soak/design-partner
work, or change Test Optimization.
