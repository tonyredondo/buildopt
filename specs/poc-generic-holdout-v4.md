# Hibernate ORM target-workload and host-pressure diagnosis

The v3 diagnostic proved that Hibernate's original 7/8 result was recoverable:
separating cache seeding from base daemon stabilization changed the first pair
from 1.118 seconds slower to 11.883 seconds faster. The complete fresh run did
not qualify, however. Its first four pairs were positive, its last four were
negative, both arms continued becoming faster, and the control task count
changed from 301 to 300 in pair eight.

Version 3 remains valid and immutable. Its base-revision stabilization did not
execute the changed target workload under the same frozen seed used by measured
pairs. Counts and complete-log digests also cannot reveal which task or outcome
changed, while an exploratory high-IO PSI sample was taken outside the arm
boundaries and cannot explain any specific observation.

Version 4 corrects those generic diagnostic limitations before collecting new
timing:

- each private arm keeps the excluded cache-seed invocation at the base
  revision;
- each arm then runs an excluded base-revision daemon stabilization from the
  frozen seed;
- each arm finally runs the exact target revision and measured workflow from
  the same frozen seed before pair one;
- every warm-up and measured arm binds a sorted normalized fingerprint of its
  Gradle task paths and outcomes in addition to the existing bounded counts and
  complete-log digest;
- Linux CPU, memory and IO pressure-stall totals are sampled immediately before
  and after each Gradle process, outside the wall-time interval, and only their
  non-negative deltas are retained;
- a measured task fingerprint that changes within either arm makes the result
  ineligible even when timing thresholds would otherwise pass;
- all eight alternating pairs are rerun from zero.

The public Hibernate revision, fixed mutation, root workflow, required JARs,
12-worker optimized-native control, Build-Impact-only candidate, timing
boundaries, pair order, output equality, fallback requirement, 500-ms/2%
minimum effect, positive interval and 8/8 repeatability gate are unchanged.
Host pressure is diagnostic and cannot turn a failing pair into a success.

The untimed fallback uses the same 12-worker parallel scheduling as the measured
arms and only replaces the persistent daemon with `--no-daemon`; this records
the behavior of the implementation exactly and does not alter timed evidence.

No v2/v3 warm-up, proposal or timing is reused. This remains a
repository-independent POC measurement correction, not a Hibernate-specific
product rule, threshold relaxation, automatic activation, production hardening,
Test Optimization, soak testing or design-partner work.
