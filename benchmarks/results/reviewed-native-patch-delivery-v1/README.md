# Reviewed native patch delivery v1 evidence

[`result.json`](./result.json) records exact materialization of the two RNPP
preimages and the complete existing PatchBundle delivery/revert matrix. The
Micronaut and Spring postimage hashes equal the accepted proposal hashes.

The two CLI materializations took 10,250 ms including their offline Gradle
startup/compilation checks. The integrated signed-bundle, real-Git draft
delivery, replay, rejection and exact-revert spike took 13,900 ms. The observed
24,150-ms machine cost repays after four compatible portfolio builds at the
RNPP signed saving of 7,906.625 ms/build. Core in-process recipe generation was
60.049 ms total; the larger figure deliberately charges process and validation
overhead. Human review is `NOT_MEASURED` and excluded.

The draft-PR adapter was in memory. No public remote, default branch or customer
source was modified, and no public Gradle build or new speed measurement ran.
The result qualifies controlled POC delivery only.
