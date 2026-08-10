# Corrected five-repository generic structural profile matrix

This POC repeats the complete Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core, and Apache Groovy matrix after
fixing a measurement-harness defect found by the v1 OpenTelemetry cell.

The v1 harness calculated the inter-arm gap from the end of the first Gradle
process to the start of the second, but reset and restored the second checkout
and cache inside that interval. It also traversed and hashed the first arm's
outputs before starting the second. Those operations are fixture preparation
and result verification, not scheduler delay. On OpenTelemetry's 1,024-project
graph they took more than the frozen five-second boundary, so v1 correctly
rejected the partial result but exposed an invalid gap definition.

Version 2 preserves the five-second fail-closed boundary and changes only its
placement:

1. reset both isolated arms and restore both frozen cache seeds;
2. start the first Gradle process, then the second immediately after it exits;
3. calculate the gap as second process start minus first process finish;
4. after both processes finish, hash and compare all required outputs;
5. retain native Gradle on any timeout, build failure, output drift, fallback
   failure, or gap greater than five seconds.

All five repositories are rerun from zero under this corrected method. No v1
timing is reused, no favorable OpenTelemetry pair is rescued, and the v1
bundle remains immutable historical evidence.

The candidate remains Build Impact only. The control remains each repository's
declared optimized-native Gradle workflow. Eight observations alternate order,
use independent checkouts, Gradle homes, and native-cache seeds, compare
required outputs byte for byte, and prove the native full-graph fallback.

A row qualifies only when all eight pairs are positive, mean savings exceed
500 ms and 2%, the paired lower bound is positive, required outputs are stable
and identical, and fallback succeeds. Repository percentages are never
averaged and mechanism percentages are never added.

This is POC evidence only. It does not authorize automatic activation,
production rollout, Test Optimization, soak testing, or design-partner work.
