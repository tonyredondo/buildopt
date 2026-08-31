# Critical-Path Successor Selection v1

Status: `CPSS-001` complete. The evidence-only decision is
`SELECT_CHANGE_SCOPED_CRITICAL_PATH_DISCOVERY_V1`; successor execution is not
part of this block.

The selection reconstructs terminal BuildOpt evidence without public source
writes, candidate Gradle builds or new timing. A successor may proceed to
source/trace discovery only if it preserves optimized native Gradle, exact
required outputs, zero additional product failures and no repository/task-name
rules. Timing remains closed until discovery is conclusive for all five public
families and at least three families expose causally avoidable critical-path
work of at least 500 ms and 2% of native wall time.

The selected hypothesis is change-scoped critical-path discovery for Build
Impact. Historical end-to-end results provide substantial signals in Spring,
OpenTelemetry and Kafka, but only Spring and Kafka isolate Build Impact. The
later longitudinal detector activated 0/71 eligible builds. The successor must
therefore repair discovery breadth before executing an action; it may not
import historical performance rows as fresh proof.

Fixed rejections:

- remote-cache locality: 0/3 eligible timed families qualified in v3;
- broad fragment learning: 0/71 activations and 0/5 positive families;
- normalization-aware cacheability: public correctness stopped at 4/9
  candidates;
- opportunity-first recurrence: only 1/5 families reached the native-ceiling
  probe gate;
- Gradle-free request hits: current public breadth did not clear its frozen
  safety/completeness gates.

Runtime tuning, Hot State, standard Copy and generic cache transport remain
retired or rejected absent materially new causal evidence.
