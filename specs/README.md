# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

Specifications are normative executable behavior, not the recommended learning
path. Start with the [documentation portal](../docs/README.md) or
[architecture overview](../docs/architecture/overview.md), then return here for
the exact cross-component contract.

| Specification | Owning item |
|---|---|
| [`poc-aggregate-workflow-partition-v1.md`](./poc-aggregate-workflow-partition-v1.md) | `POC-AGGREGATE-WORKFLOW-PARTITION-001` |
| [`poc-verified-output-materialization-v1.md`](./poc-verified-output-materialization-v1.md) | `POC-VERIFIED-OUTPUT-MATERIALIZATION-001` |
| [`poc-central-storage-contract-v1.md`](./poc-central-storage-contract-v1.md) and [`poc-central-storage-contract-v1.json`](./poc-central-storage-contract-v1.json) | `POC-CENTRAL-STORAGE-CONTRACT-001` |
| [`poc-central-state-storage-v1.md`](./poc-central-state-storage-v1.md) and [`poc-central-state-storage-v1.json`](./poc-central-state-storage-v1.json) | `POC-CENTRAL-STATE-STORAGE-001` |
| [`poc-central-https-auth-v1.md`](./poc-central-https-auth-v1.md) and [`poc-central-https-auth-v1.json`](./poc-central-https-auth-v1.json) | `POC-CENTRAL-HTTPS-AUTH-001` |
| [`poc-central-gradle-cache-v1.md`](./poc-central-gradle-cache-v1.md) and [`poc-central-gradle-cache-v1.json`](./poc-central-gradle-cache-v1.json) | `POC-CENTRAL-GRADLE-CACHE-001` |
| [`poc-central-state-sync-v1.md`](./poc-central-state-sync-v1.md) and [`poc-central-state-sync-v1.json`](./poc-central-state-sync-v1.json) | `POC-CENTRAL-STATE-SYNC-001` |
| [`poc-central-optimize-integration-v1.md`](./poc-central-optimize-integration-v1.md) and [`poc-central-optimize-integration-v1.json`](./poc-central-optimize-integration-v1.json) | `POC-CENTRAL-OPTIMIZE-INTEGRATION-001` |
| [`poc-central-two-machine-v1.md`](./poc-central-two-machine-v1.md) and [`poc-central-two-machine-v1.json`](./poc-central-two-machine-v1.json) | `POC-CENTRAL-TWO-MACHINE-001` |
| [`poc-generic-workflow-breadth-v1.md`](./poc-generic-workflow-breadth-v1.md) and [`poc-generic-workflow-breadth-v1.json`](./poc-generic-workflow-breadth-v1.json) | `POC-GENERIC-WORKFLOW-BREADTH-001` |
| [`poc-generic-owner-input-v1.md`](./poc-generic-owner-input-v1.md) and [`poc-generic-owner-input-v1.json`](./poc-generic-owner-input-v1.json) | `POC-GENERIC-OWNER-INPUT-001` |
| [`poc-generic-profile-ci-replay-v1.md`](./poc-generic-profile-ci-replay-v1.md) and [`poc-generic-profile-ci-replay-v1.json`](./poc-generic-profile-ci-replay-v1.json) | `POC-GENERIC-PROFILE-CI-REPLAY-001` |
| [`poc-generic-profile-ci-v1.md`](./poc-generic-profile-ci-v1.md) | `POC-GENERIC-PROFILE-CI-001` |
| [`poc-generic-profile-matrix-v4.md`](./poc-generic-profile-matrix-v4.md) and [`poc-generic-profile-matrix-v4.json`](./poc-generic-profile-matrix-v4.json) | `POC-GENERIC-PROFILE-MATRIX-003` |
| [`poc-generic-profile-matrix-v3.md`](./poc-generic-profile-matrix-v3.md) and [`poc-generic-profile-matrix-v3.json`](./poc-generic-profile-matrix-v3.json) | `POC-GENERIC-PROFILE-MATRIX-002` |
| [`poc-generic-profile-matrix-v2.md`](./poc-generic-profile-matrix-v2.md) and [`poc-generic-profile-matrix-v2.json`](./poc-generic-profile-matrix-v2.json) | `POC-GENERIC-PROFILE-MATRIX-002` |
| [`poc-generic-profile-matrix-v1.md`](./poc-generic-profile-matrix-v1.md) and [`poc-generic-profile-matrix-v1.json`](./poc-generic-profile-matrix-v1.json) | `POC-GENERIC-PROFILE-MATRIX-001` |
| [`poc-generic-profile-realworld-v1.md`](./poc-generic-profile-realworld-v1.md) and [`poc-generic-profile-realworld-v1.json`](./poc-generic-profile-realworld-v1.json) | `POC-GENERIC-PROFILE-REALWORLD-001` |
| [`poc-generic-profile-onboarding-v1.md`](./poc-generic-profile-onboarding-v1.md) | `POC-GENERIC-PROFILE-ONBOARDING-001` |
| [`poc-real-world-compatibility-v1.md`](./poc-real-world-compatibility-v1.md) and [`poc-real-world-compatibility-v1.json`](./poc-real-world-compatibility-v1.json) | `POC-REALWORLD-001` / `POC-REALWORLD-G01` |
| [`poc-real-world-performance-v1.md`](./poc-real-world-performance-v1.md) and [`poc-real-world-performance-v1.json`](./poc-real-world-performance-v1.json) | `POC-REALWORLD-002` / `POC-REALWORLD-G02` |
| [`poc-real-world-diagnostics-v1.md`](./poc-real-world-diagnostics-v1.md) and [`poc-real-world-diagnostics-v1.json`](./poc-real-world-diagnostics-v1.json) | `POC-REALWORLD-DIAGNOSTICS-001` / `POC-REALWORLD-G03` |
| [`poc-public-build-tasks-v1.md`](./poc-public-build-tasks-v1.md) and [`poc-public-build-tasks-v1.json`](./poc-public-build-tasks-v1.json) | `POC-PUBLIC-BUILD-TASKS-001` / `POC-PUBLIC-BUILD-TASKS-G01` |
| [`poc-mockito-test-build-v1.json`](./poc-mockito-test-build-v1.json) | `POC-MOCKITO-TEST-BUILD-001` / `POC-MOCKITO-TEST-BUILD-G01` |
| [`poc-spring-framework-v1.md`](./poc-spring-framework-v1.md) and [`poc-spring-framework-v1.json`](./poc-spring-framework-v1.json) | `POC-SPRING-PREREG-001` / `POC-SPRING-PREREG-G01` |
| [`poc-spring-test-preparation-v2.json`](./poc-spring-test-preparation-v2.json) | `POC-SPRING-TEST-PREPARATION-001` |
| [`poc-spring-test-build-optimization-v1.json`](./poc-spring-test-build-optimization-v1.json) | `POC-TEST-BUILD-OPTIMIZATION-001` |
| [`poc-optimization-overhead-ablation-v1.json`](./poc-optimization-overhead-ablation-v1.json) | `POC-OPTIMIZATION-OVERHEAD-ABLATION-001` |
| [`poc-runtime-research-v1.json`](./poc-runtime-research-v1.json) | Historical retired Runtime Tuning evidence |
| [`poc-remote-cache-value-v1.md`](./poc-remote-cache-value-v1.md) and [`poc-remote-cache-value-v1.json`](./poc-remote-cache-value-v1.json) | `POC-REMOTE-CACHE-VALUE-001` |
| [`poc-remote-cache-transfer-v1.md`](./poc-remote-cache-transfer-v1.md) and [`poc-remote-cache-transfer-v1.json`](./poc-remote-cache-transfer-v1.json) | `POC-REMOTE-CACHE-TRANSFER-001` |
| [`poc-qualified-remote-composition-v1.md`](./poc-qualified-remote-composition-v1.md) and [`poc-qualified-remote-composition-v1.json`](./poc-qualified-remote-composition-v1.json) | `POC-QUALIFIED-REMOTE-COMPOSITION-001` |
| [`poc-kafka-impact-edge-composition-v1.md`](./poc-kafka-impact-edge-composition-v1.md) and [`poc-kafka-impact-edge-composition-v1.json`](./poc-kafka-impact-edge-composition-v1.json) | `POC-KAFKA-IMPACT-EDGE-COMPOSITION-001` |
| [`poc-kafka-shadowjar-reproducibility-v1.md`](./poc-kafka-shadowjar-reproducibility-v1.md) and [`poc-kafka-shadowjar-reproducibility-v1.json`](./poc-kafka-shadowjar-reproducibility-v1.json) | `POC-KAFKA-SHADOWJAR-REPRODUCIBILITY-001` |
| [`poc-kafka-impact-edge-composition-v2.md`](./poc-kafka-impact-edge-composition-v2.md) and [`poc-kafka-impact-edge-composition-v2.json`](./poc-kafka-impact-edge-composition-v2.json) | `POC-KAFKA-IMPACT-EDGE-COMPOSITION-002` |
| [`poc-kafka-composition-usability-v1.md`](./poc-kafka-composition-usability-v1.md) and [`poc-kafka-composition-usability-v1.json`](./poc-kafka-composition-usability-v1.json) | `POC-KAFKA-COMPOSITION-USABILITY-001` |
| [`poc-kafka-installed-profile-value-v1.md`](./poc-kafka-installed-profile-value-v1.md) and [`poc-kafka-installed-profile-value-v1.json`](./poc-kafka-installed-profile-value-v1.json) | `POC-KAFKA-INSTALLED-PROFILE-VALUE-001` |
| [`poc-qualified-profile-matrix-v1.md`](./poc-qualified-profile-matrix-v1.md) and [`poc-qualified-profile-matrix-v1.json`](./poc-qualified-profile-matrix-v1.json) | `POC-QUALIFIED-PROFILE-MATRIX-001` |
| [`poc-spring-installed-impact-v1.md`](./poc-spring-installed-impact-v1.md) and [`poc-spring-installed-impact-v1.json`](./poc-spring-installed-impact-v1.json) | `POC-SPRING-INSTALLED-IMPACT-001` |
| [`poc-spring-impact-breadth-v1.md`](./poc-spring-impact-breadth-v1.md) and [`poc-spring-impact-breadth-v1.json`](./poc-spring-impact-breadth-v1.json) | `POC-SPRING-IMPACT-BREADTH-PREREG-001` / `POC-SPRING-IMPACT-BREADTH-G01` |
| [`poc-otel-test-preparation-v1.json`](./poc-otel-test-preparation-v1.json) | `POC-OTEL-TRANSFER-001` |
| [`poc-otel-spring-family-v2.md`](./poc-otel-spring-family-v2.md), [`poc-otel-spring-family-v2.json`](./poc-otel-spring-family-v2.json), and the [fixed entrypoints](./poc-otel-spring-family-v2.tasks.txt) | `POC-OTEL-SPRING-FAMILY-PREREG-001` |
| [`build-impact-poc-phase-timings-v1.md`](./build-impact-poc-phase-timings-v1.md) | `POC-OTEL-OVERHEAD-001` |
| [`poc-otel-graph-reduction-v1.md`](./poc-otel-graph-reduction-v1.md) | `POC-OTEL-GRAPH-001` |
| [`build-impact-poc-hot-state-v1.md`](./build-impact-poc-hot-state-v1.md) | Historical retired Hot State evidence |
| [`poc-otel-optimization-v1.md`](./poc-otel-optimization-v1.md) and [`poc-otel-optimization-v1.json`](./poc-otel-optimization-v1.json) | `POC-OTEL-STABILITY-001` |
| [`poc-otel-optimization-v2.md`](./poc-otel-optimization-v2.md) and [`poc-otel-optimization-v2.json`](./poc-otel-optimization-v2.json) | `POC-OTEL-STABILITY-002` |
| [`poc-full-path-ablation-v1.md`](./poc-full-path-ablation-v1.md) and [`poc-full-path-ablation-v1.json`](./poc-full-path-ablation-v1.json) | `POC-FULL-PATH-ABLATION-001` |
| [`poc-otel-clean-composition-v1.md`](./poc-otel-clean-composition-v1.md) and [`poc-otel-clean-composition-v1.json`](./poc-otel-clean-composition-v1.json) | `POC-FULL-PATH-CLEAN-001` |
| [`poc-impact-generalization-v1.md`](./poc-impact-generalization-v1.md) and [`poc-impact-generalization-v1.json`](./poc-impact-generalization-v1.json) | `POC-IMPACT-GENERALIZATION-001` |
| [`poc-verification-distribution-graph-v1.md`](./poc-verification-distribution-graph-v1.md) and [`poc-verification-distribution-graph-v1.json`](./poc-verification-distribution-graph-v1.json) | `POC-VERIFICATION-DISTRIBUTION-GRAPH-001` |
| [`poc-verification-overhead-attribution-v1.md`](./poc-verification-overhead-attribution-v1.md) and [`poc-verification-overhead-attribution-v1.json`](./poc-verification-overhead-attribution-v1.json) | `POC-VERIFICATION-OVERHEAD-ATTRIBUTION-001` |
| [`poc-stability-validation-v1.md`](./poc-stability-validation-v1.md) and [`poc-stability-validation-v1.json`](./poc-stability-validation-v1.json) | `POC-STABILITY-001` |
| [`poc-pairing-validation-v1.md`](./poc-pairing-validation-v1.md) and [`poc-pairing-validation-v1.json`](./poc-pairing-validation-v1.json) | `POC-PAIRING-001` |
| [`poc-groovy-validation-v1.md`](./poc-groovy-validation-v1.md) and [`poc-groovy-validation-v1.json`](./poc-groovy-validation-v1.json) | `POC-GROOVY-001` |
| [`poc-shared-groovy-validation-v1.md`](./poc-shared-groovy-validation-v1.md) and [`poc-shared-groovy-validation-v1.json`](./poc-shared-groovy-validation-v1.json) | `POC-SHARED-GROOVY-001` |
| [`poc-leaf-kotlin-validation-v1.md`](./poc-leaf-kotlin-validation-v1.md) and [`poc-leaf-kotlin-validation-v1.json`](./poc-leaf-kotlin-validation-v1.json) | `POC-LEAF-KOTLIN-001` |
| [`poc-kotlin-stability-validation-v1.md`](./poc-kotlin-stability-validation-v1.md) and [`poc-kotlin-stability-validation-v1.json`](./poc-kotlin-stability-validation-v1.json) | `POC-KOTLIN-STABILITY-001` |
| [`poc-kotlin-boundary-decision-v1.md`](./poc-kotlin-boundary-decision-v1.md) and [`poc-kotlin-boundary-decision-v1.json`](./poc-kotlin-boundary-decision-v1.json) | `POC-KOTLIN-BOUNDARY-001` |
| [`poc-overhead-attribution-v1.md`](./poc-overhead-attribution-v1.md) and [`poc-overhead-attribution-v1.json`](./poc-overhead-attribution-v1.json) | `POC-OVERHEAD-001` |
| [`poc-breadth-validation-v1.md`](./poc-breadth-validation-v1.md) and [`poc-breadth-validation-v1.json`](./poc-breadth-validation-v1.json) | `POC-BREADTH-001` / `POC-BREADTH-G01` |
| [`poc-value-validation-v1.md`](./poc-value-validation-v1.md) and [`poc-value-validation-v1.json`](./poc-value-validation-v1.json) | `POC-VALUE-001..004` / `POC-VALUE-G01` |
| [`ci-orchestration-v1.md`](./ci-orchestration-v1.md) | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| [`benchmark-beta-v1.md`](./benchmark-beta-v1.md) | `F0-032` |
| [`beta-benchmark-harness-v1.md`](./beta-benchmark-harness-v1.md) | `OPS-001/A1` |
| [`beta-disk-faults-v1.md`](./beta-disk-faults-v1.md) | `A1-003` / `A1-G03` |
| [`beta-shared-faults-v1.md`](./beta-shared-faults-v1.md) | `OPS-001/A1` |
| [`beta-system-faults-v1.md`](./beta-system-faults-v1.md) | `OPS-001/A1` / `A1-G04` |
| [`beta-sustained-v1.md`](./beta-sustained-v1.md) | `OPS-001/A1` |
| [`beta-soak-v1.md`](./beta-soak-v1.md) | Historical productization harness; excluded from the active POC |
| [`beta-circuit-breaker-v1.md`](./beta-circuit-breaker-v1.md) | `OPS-001/A1` / `A1-G02` |
| [`beta-gradle-fixtures-v1.md`](./beta-gradle-fixtures-v1.md) | `OPS-001/A1` / `A1-G02` |
| [`private-beta-token-isolation-v1.md`](./private-beta-token-isolation-v1.md) | `A1-002` / `A1-G01` |
| [`onboarding-performance-v1.md`](./onboarding-performance-v1.md) | Product onboarding |
| [`cache-parity-v1.md`](./cache-parity-v1.md) | Safe-cache performance |
| [`poc-standard-jar-cache-v1.md`](./poc-standard-jar-cache-v1.md) | OpenTelemetry POC repeated-work reduction |
| [`poc-standard-copy-cache-v1.md`](./poc-standard-copy-cache-v1.md) and [`poc-standard-copy-cache-v1.json`](./poc-standard-copy-cache-v1.json) | Historical retired standard-`Copy` evidence |
| [`poc-magic-onboarding-contract-v1.md`](./poc-magic-onboarding-contract-v1.md) and [`poc-magic-onboarding-contract-v1.json`](./poc-magic-onboarding-contract-v1.json) | `POC-MAGIC-ONBOARDING-CONTRACT-001` |
| [`poc-magic-auto-discovery-v1.md`](./poc-magic-auto-discovery-v1.md) and [`poc-magic-auto-discovery-v1.json`](./poc-magic-auto-discovery-v1.json) | `POC-MAGIC-AUTO-DISCOVERY-001` |
| [`poc-magic-calibration-v1.md`](./poc-magic-calibration-v1.md) and [`poc-magic-calibration-v1.json`](./poc-magic-calibration-v1.json) | `POC-MAGIC-CALIBRATION-001` |
| [`poc-magic-profile-portfolio-v1.md`](./poc-magic-profile-portfolio-v1.md) and [`poc-magic-profile-portfolio-v1.json`](./poc-magic-profile-portfolio-v1.json) | `POC-MAGIC-PROFILE-PORTFOLIO-001` |
| [`poc-magic-auto-replay-v1.md`](./poc-magic-auto-replay-v1.md) and [`poc-magic-auto-replay-v1.json`](./poc-magic-auto-replay-v1.json) | `POC-MAGIC-AUTO-REPLAY-001` |
| [`poc-magic-ci-onboarding-v1.md`](./poc-magic-ci-onboarding-v1.md) and [`poc-magic-ci-onboarding-v1.json`](./poc-magic-ci-onboarding-v1.json) | `POC-MAGIC-CI-ONBOARDING-001` |
| [`poc-magic-wow-report-v1.md`](./poc-magic-wow-report-v1.md) and [`poc-magic-wow-report-v1.json`](./poc-magic-wow-report-v1.json) | `POC-MAGIC-WOW-REPORT-001` |
| [`poc-magic-end-to-end-value-v1.md`](./poc-magic-end-to-end-value-v1.md) and [`poc-magic-end-to-end-value-v1.json`](./poc-magic-end-to-end-value-v1.json) | `POC-MAGIC-END-TO-END-VALUE-001` historical diagnostic evidence |
| [`poc-magic-end-to-end-value-v2.md`](./poc-magic-end-to-end-value-v2.md) and [`poc-magic-end-to-end-value-v2.json`](./poc-magic-end-to-end-value-v2.json) | `POC-MAGIC-END-TO-END-VALUE-001` terminal public-package evidence |
| [`poc-magic-calibration-cost-v1.md`](./poc-magic-calibration-cost-v1.md) | `POC-MAGIC-CALIBRATION-COST-001` |
| [`poc-central-end-to-end-value-v1.md`](./poc-central-end-to-end-value-v1.md) and [`poc-central-end-to-end-value-v1.json`](./poc-central-end-to-end-value-v1.json) | `POC-CENTRAL-END-TO-END-VALUE-001` |
| [`poc-profile-lifetime-v1.md`](./poc-profile-lifetime-v1.md) and [`poc-profile-lifetime-v1.json`](./poc-profile-lifetime-v1.json) | `POC-PROFILE-LIFETIME-001` |
| [`poc-economic-prequalification-v1.md`](./poc-economic-prequalification-v1.md) and [`poc-economic-prequalification-v1.json`](./poc-economic-prequalification-v1.json) | `POC-ECONOMIC-PREQUALIFICATION-001` |
| [`poc-automatic-breadth-transfer-v1.md`](./poc-automatic-breadth-transfer-v1.md) and [`poc-automatic-breadth-transfer-v1.json`](./poc-automatic-breadth-transfer-v1.json) | `POC-AUTOMATIC-BREADTH-TRANSFER-001` |
| [`poc-automatic-breadth-transfer-v2.md`](./poc-automatic-breadth-transfer-v2.md) and [`poc-automatic-breadth-transfer-v2.json`](./poc-automatic-breadth-transfer-v2.json) | `POC-AUTOMATIC-BREADTH-TRANSFER-V2-001` |
| [`poc-materialization-economics-v2.md`](./poc-materialization-economics-v2.md) and [`poc-materialization-economics-v2.json`](./poc-materialization-economics-v2.json) | `POC-MATERIALIZATION-ECONOMICS-V2-001` |
| [`poc-qualified-lifetime-v2.md`](./poc-qualified-lifetime-v2.md) and [`poc-qualified-lifetime-v2.json`](./poc-qualified-lifetime-v2.json) | `POC-QUALIFIED-LIFETIME-V2-001` |
| [`poc-incremental-learning-v1.md`](./poc-incremental-learning-v1.md) and [`poc-incremental-learning-v1.json`](./poc-incremental-learning-v1.json) | `POC-INCREMENTAL-LEARNING-001` |
| [`poc-normal-build-tail-expansion-v1.md`](./poc-normal-build-tail-expansion-v1.md) and [`poc-normal-build-tail-expansion-v1.json`](./poc-normal-build-tail-expansion-v1.json) | `POC-NORMAL-BUILD-TAIL-EXPANSION-001` |
| [`private-beta-data-lifecycle-v1.md`](./private-beta-data-lifecycle-v1.md) | `A1-004` / `A1-G05` |
| [`private-beta-operations-v1.md`](./private-beta-operations-v1.md) | `A1-005` |
| [`owner-controlled-pilot-deployment-v1.md`](./owner-controlled-pilot-deployment-v1.md) | `A1-001` |
| [`owner-poc-evaluation-v1.md`](./owner-poc-evaluation-v1.md) | `A1-006` / `A1-G06` |
| [`runtime-owner-evaluation-v1.md`](./runtime-owner-evaluation-v1.md) | Historical retired Runtime Tuning evidence |
| [`task-intelligence-poc-v1.md`](./task-intelligence-poc-v1.md) | `MVP-C1` |
| [`build-impact-manifest-v1.md`](./build-impact-manifest-v1.md) | `C3-001` |
| [`build-impact-declared-graph-v1.md`](./build-impact-declared-graph-v1.md) | `C3-002` |
| [`build-impact-shadow-validation-v1.md`](./build-impact-shadow-validation-v1.md) | `C3-003` |
| [`build-impact-promotion-gate-v1.md`](./build-impact-promotion-gate-v1.md) | `C3-004` / `BIA-002` |
| [`build-impact-selection-v1.md`](./build-impact-selection-v1.md) | `C3-005` |
| [`build-impact-gate-v1.md`](./build-impact-gate-v1.md) | `C3-G01` / MVP-C3 |
| [`build-impact-automatic-v1.md`](./build-impact-automatic-v1.md) | `BIA-F4-001..003` / `BIA-F4-G01` |
| [`build-impact-performance-v1.md`](./build-impact-performance-v1.md) | Build Impact performance |
| [`build-impact-poc-onboarding-v1.md`](./build-impact-poc-onboarding-v1.md) and [`build-impact-poc-onboarding-v1.json`](./build-impact-poc-onboarding-v1.json) | `POC-INSTALLED-IMPACT-001` |
| [`platform-compatibility-v1.md`](./platform-compatibility-v1.md) | `PLAT-F5-001..004` / `PLAT-F5-G01` |
| [`platform-runtime-parity-v2.md`](./platform-runtime-parity-v2.md) | `PLAT-F6-001..004` / `PLAT-F6-G01` |
| [`edge-cache-config-v1.md`](./edge-cache-config-v1.md) | `C2-001` |
| [`edge-cache-committed-read-v1.md`](./edge-cache-committed-read-v1.md) | `C2-002` |
| [`edge-cache-capacity-slru-v1.md`](./edge-cache-capacity-slru-v1.md) | `C2-003` |
| [`edge-cache-pending-replication-v1.md`](./edge-cache-pending-replication-v1.md) | `C2-004` |
| [`edge-cache-two-node-proxy-v1.md`](./edge-cache-two-node-proxy-v1.md) | `C2-005` |
| [`edge-cache-gate-v1.md`](./edge-cache-gate-v1.md) | `C2-G01` / MVP-C2 |
| [`edge-operability-v1.md`](./edge-operability-v1.md) | `O1-001..004` / `O1-G01` / POC-O1 |
| [`owner-poc-lab-v1.md`](./owner-poc-lab-v1.md) | `O2-001..004` / `O2-G01` / POC-O2 |
| [`build-history-api-v1.md`](./build-history-api-v1.md) | `UX-F1-001` |
| [`build-history-dashboard-v1.md`](./build-history-dashboard-v1.md) | `UX-F1-002` |
| [`custom-task-contract-java-recipe-v1.md`](./custom-task-contract-java-recipe-v1.md) | `C4-004` / `C4-G06` |
| [`test-optimization-integration-v1.md`](./test-optimization-integration-v1.md) | `F0-033` |
| [`full-relevant-validation-gate-v1.md`](./full-relevant-validation-gate-v1.md) | `C4-006` / `C4-G02` |
| [`customer-patch-workflow-v1.md`](./customer-patch-workflow-v1.md) | `C4-007` / `C4-G01` / `C4-G03` / `C4-G04` |
| [`patch-delivery-recovery-v1.md`](./patch-delivery-recovery-v1.md) | `C4-008` |
| [`post-merge-patch-monitor-v1.md`](./post-merge-patch-monitor-v1.md) | `C4-009` / `C4-G05` |
| [`patch-bundle-v1.md`](./patch-bundle-v1.md) | `F0-034` |
| [`bandit-policy-v1.md`](./bandit-policy-v1.md) | `F0-035` |
| [`capability-matrix-v1.md`](./capability-matrix-v1.md) | `F0-036` |
| [`data-lifecycle-v1.md`](./data-lifecycle-v1.md) | `F0-037` |
| [`tier-one-cache-policy-v1.md`](./tier-one-cache-policy-v1.md) | `A0-002` |
| [`tier-one-cache-conformance-v1.md`](./tier-one-cache-conformance-v1.md) | `A0-G01` |
| [`l1-l2-revocation-v1.md`](./l1-l2-revocation-v1.md) | `A0-G02` |
| [`gateway-rotation-v1.md`](./gateway-rotation-v1.md) | `A0-G03` |
| [`gateway-spool-v1.md`](./gateway-spool-v1.md) | `A0-G04` |
| [`shared-commit-recovery-v1.md`](./shared-commit-recovery-v1.md) | `A0-G05` |
| [`shared-capacity-slru-v1.md`](./shared-capacity-slru-v1.md) | `A1-003` / `A1-G03` |
| [`no-hit-overhead-v1.md`](./no-hit-overhead-v1.md) | `A0-G06` |
| [`test-cache-isolation-v1.md`](./test-cache-isolation-v1.md) | `A0-G08` |
| [`managed-l1-v1.md`](./managed-l1-v1.md) | `A0-003` |
| [`single-node-shared-storage-v1.md`](./single-node-shared-storage-v1.md) | `A0-004` |
| [`self-hosted-single-node-config-v1.md`](./self-hosted-single-node-config-v1.md) | `A2-001` |
| [`self-hosted-service-install-v1.md`](./self-hosted-service-install-v1.md) | `A2-002` |
| [`self-hosted-upgrade-restart-v1.md`](./self-hosted-upgrade-restart-v1.md) | `A2-003` |
| [`self-hosted-manual-restore-v1.md`](./self-hosted-manual-restore-v1.md) | `A2-004` |
| [`self-hosted-single-node-gate-v1.md`](./self-hosted-single-node-gate-v1.md) | `A2-G01` / MVP-A2 |
| [`pending-commit-cas-v1.md`](./pending-commit-cas-v1.md) | `A0-005` |
| [`local-authenticated-cache-v1.md`](./local-authenticated-cache-v1.md) | `A0-006` |
| [`gradle-bootstrap-cache-v1.md`](./gradle-bootstrap-cache-v1.md) | `A0-007` |
| [`export-gateway-v1.md`](./export-gateway-v1.md) | `A0-008` |
| [`causal-pilot-v1.md`](./causal-pilot-v1.md) | `A0-009` |
| [`jvm-agent-spike-v1.md`](./jvm-agent-spike-v1.md) | `SPK-002` |
| [`hermetic-helper-spike-v1.md`](./hermetic-helper-spike-v1.md) | `SPK-003` |
| [`release-bundle-v1.md`](./release-bundle-v1.md) | `F0-038` / `DEPLOY-001` |
| [`deployment-lifecycle-v1.md`](./deployment-lifecycle-v1.md) | `DEPLOY-001` |
| [`ops-readiness-v1.md`](./ops-readiness-v1.md) | `OPS-001/A1` |
| [`ops-alerts-v1.md`](./ops-alerts-v1.md) | `OPS-001/A1` |
| [`walking-skeleton-faults-v1.md`](./walking-skeleton-faults-v1.md) | `WS-008` |
| [`walking-skeleton-overhead-v1.md`](./walking-skeleton-overhead-v1.md) | `WS-009` |

The terminal [economic prequalification evidence](../benchmarks/results/poc-economic-prequalification-v1/README.md)
applies the contract to public Ktor history. A low-recurrence CORS owner is
rejected in 192.442 ms before discovery/calibration, reducing the observed
fallback penalty from 220.761 to 13.896 seconds across runs. The matching
Jetty replay still saves 100.744 seconds, but its 1,386.764-second learning
cost remains unpaid in the observed window.

The immutable [V1 automatic breadth evidence](../benchmarks/results/poc-automatic-breadth-transfer-v1/README.md)
records the synchronous-learning blocker. The current
[V2 evidence](../benchmarks/results/poc-automatic-breadth-transfer-v2/README.md)
applies the unchanged zero-manual-file command after incremental learning,
verified output materialization and aggregate partitioning are composed. It
records zero product failures, exact outputs and faster candidates on all five
repositories; OpenTelemetry, Kafka, Micronaut and Groovy qualify while Spring
safely retains native under the unchanged gates.

The terminal [materialization-economics V2 evidence](../benchmarks/results/poc-materialization-economics-v2/README.md)
keeps those repositories and gates but measures complete Gradle-plus-wrapper
wall time, derives candidate tasks from the observed task graph and replaces
per-file durable state with one manifest-bound pack. All five rows qualify,
all 40 pairs improve, exact outputs and fallback pass, and one-time learning
cost repays in one to four matching builds. The earlier V2 bundle remains
immutable before-evidence.

The [qualified-lifetime V2 contract](./poc-qualified-lifetime-v2.md) follows
those same five qualified profiles over frozen public first-parent descendant
commits. It adds verified CAS transport for materialized output packs and
measures cumulative wall-time value against persistent optimized-native arms.
Exact native retention is a valid outcome; the experiment does not require a
positive result, average repositories or weaken the qualification gates. A
later mismatch between two native-retained builds is recorded as native output
nondeterminism and rejects that subject without a timing claim; selected output
differences remain product failures.

The terminal [qualified-lifetime V2 evidence](../benchmarks/results/poc-qualified-lifetime-v2/README.md)
records 4/5 current qualifications, 2/4 portable output sets, zero selected
replays across seven public descendants and zero paid-back subjects. All seven
later builds retain exact optimized-native outputs with zero product failures.
This negative result remains immutable before-evidence and does not authorize a
weaker gate or repeat-until-positive rerun.

The [cross-commit value recovery contract](./poc-cross-commit-value-recovery-v1.md)
then holds Kafka's qualifier and six-descendant public window fixed. It requires
cheap eligibility before central work, verified local refresh, distinct target,
revalidation and output revisions, and no full-workflow observation after a
selective replay is chosen. Its checked before/after evidence requires a
regressive selected replay before the repair, positive attributable selected
value after it, exact outputs, zero product failures and positive cumulative net
after qualification/publication cost. Native-retained arm variation stays
visible but cannot prove selected-replay value.

The [cross-commit breadth replication contract](./poc-cross-commit-breadth-replication-v1.md)
freezes two non-Kafka public families under the unchanged recovery mechanism.
Its immutable subject file binds Spring root `classes` and OpenTelemetry JMX
`testClasses`; a preregistered addendum binds Spring JMS after the broad
Spring change exposed a different owner graph. The addendum cannot rewrite the
original observations.

The terminal
[breadth evidence](../benchmarks/results/poc-cross-commit-breadth-replication-v1/README.md)
records one failed calibration, one pre-calibration ownership rejection and one
positive calibration rejected for 14 native-output differences. Zero profiles
are portable, zero replays are selected, zero product failures occur and the
cross-repository value claim remains bounded to the prior Kafka evidence.

The [workflow-input ownership contract](./poc-workflow-input-ownership-v1.md)
then addresses the OpenTelemetry blocker generically. A changed unowned path
may be ignored only when the exact requested workflow produces complete
finalized task-input evidence and zero tasks consume it. Owned paths remain
relevant; consumed, deleted, missing, incomplete, ambiguous and all-unowned
cases retain native Gradle. The frozen JMX proof ignores only root
`CHANGELOG.md` and completes a 1,027->8-project structural discovery. It skips
calibration and makes no wall-time or broadened-value claim.

The terminal [incremental-learning evidence](../benchmarks/results/poc-incremental-learning-v1/README.md)
then replaces that synchronous transaction with one exact-bound observation
per useful invocation. Its fixture completes eight pairs with zero
measurement-only workflows and retains native Gradle when 0.90% does not pass
the unchanged value and 30-build gates.

Each specification must link fixtures or conformance tests and the RFC decision it refines. `F0-010` reserves these paths without creating empty specifications.

`ci-orchestration-v1.json` is the machine-readable scheduling, isolation,
budget, and recovery corpus consumed by the F0-030 conformance checker.
`commit-atomicity-v1.json` is the F0-031 transaction fault/replay plan backing
ADR 0002.
`test-optimization-integration-v1.json` is the shared F0-033
producer/consumer scenario corpus.
`patch-bundle-v1.json` is the ordered F0-034 application and recovery plan
consumed by the Java patcher spike.
`bandit-policy-v1.json` is the deterministic F0-035 policy/replay corpus.
`capability-matrix-v1.json` is the current evidence-backed F0-036 Tier 1
status matrix.
`data-lifecycle-v1.json` is the F0-037 retention, redaction, buffering, and
deletion contract.
`tier-one-cache-policy-v1.json` is the restriction-only A0-002 runtime,
task/action, transform, and fallback allowlist.
`tier-one-cache-conformance-v1.json` is the A0-G01 backend, gateway, Gradle
client, retry, corruption, and default-deny compatibility matrix.
`l1-l2-revocation-v1.json` is the A0-G02 committed-L2/native-L1 generation,
authenticated revocation, miss/rotation, and aborted-writer isolation
contract.
`gateway-rotation-v1.json` is the A0-G03 stable process restart, complete local
identity rotation, Configuration Cache, transient upstream authority, and
concurrent-slot isolation contract.
`gateway-spool-v1.json` is the A0-G04 complete pre-200 verification, bounded
reservation, disk/cancellation/checksum fault, and managed-process crash
cleanup contract.
`shared-commit-recovery-v1.json` is the A0-G05 real filesystem/SQLite WAL
contract for concurrent commit CAS, all-object visibility atomicity, digest
audit repair, and safe orphan/missing/expired recovery.
`shared-capacity-slru-v1.json` is the A1-003/A1-G03 hard-quota, durable TTL,
byte-weighted SLRU, conservative reservation, and high/low watermark contract.
`edge-cache-config-v1.json` is the C2-001 private single-node configuration,
origin transport, bounded storage, and immutable Shared-only authority
contract. It deliberately opens no Edge route yet.
`edge-cache-committed-read-v1.json` is the C2-002 authenticated Shared hit,
complete durable publication, exact current-revocation offline-read, corruption
fallback, and restart contract.
`edge-cache-capacity-slru-v1.json` is the C2-003 hard byte quota,
conservative reservation, durable TTL, byte-SLRU pressure, and schema migration
contract.
`edge-cache-pending-replication-v1.json` is the C2-004 exact-attempt pending
visibility, authenticated asynchronous Shared replication, durable
retry/restart, TTL, and no-local-promotion contract.
`edge-cache-two-node-proxy-v1.json` is the C2-005 IPv4-loopback route,
committed-first and exact-attempt fallback, and owner-controlled two-Edge
central-collision proof.
`edge-cache-gate-v1.json` is the C2-G01 current-tree constituent, invariant,
boundary, and final two-node runtime composition contract.
`edge-operability-v1.json` is the POC-O1 standalone process, signed hot reload,
private aggregate status, signed-bundle, reproducible-systemd-unit, and
graceful-lifecycle composition contract.
`owner-poc-lab-v1.json` composes the synthetic Gradle lane, repeated Shared
fault evidence, two-Edge Shared collision proof, and complete Edge operability
gate into one JSON-reporting POC-O2 command that base CI runs without owner
repositories, external partners, or the deferred soak.
`build-history-api-v1.json` is the UX-F1-001 authenticated loopback read model
over immutable redacted BUILD_SESSION exports, including exact list/detail
operations, filters, stable cursor pagination, private-file limits, and the
explicit boundary that leaves Test Optimization unchanged.

`build-history-dashboard-v1.json` is the UX-F1-002 embedded local interface
contract, including memory-only API authentication, exact/loaded-only filter
semantics, source-backed summaries/details, security headers, responsive and
accessible states, and the explicit absence of fabricated analytics or Test
Optimization behavior.

`no-hit-overhead-v1.json` is the A0-G06 paired strict-runner contract for
authenticated read-only L2 misses, fresh L1/output state, long-session p95
budgets, and pre-outcome L2 omission with zero short-session requests.
`test-cache-isolation-v1.json` is the A0-G08 fail-closed no-grant contract for
root, actual `buildSrc`, and included-plugin `Test` tasks, including a usable
authenticated remote-cache control and exact zero-request guarded proof.
`deployment-lifecycle-v1.json` is the DEPLOY-001 contract for externally
verified immutable versions, atomic selection, persistent data, and the
install/upgrade/rollback/uninstall lifecycle.
`ops-readiness-v1.json` is the first OPS-001/A1 slice for live-before-ready
startup, fail-closed application routing, shutdown draining, and signed
authority reload within 60 seconds.
`ops-alerts-v1.json` is the OPS-001/A1 ten-class local alert contract for
bounded storage, authority, export, and acceptance signals plus deterministic
activation/recovery without exposing sensitive values.
`private-beta-token-isolation-v1.json` is the A1-002/A1-G01 contract for
hashed 30-day credentials, exact repository/namespace/plane/operation scopes,
per-request revocation, remote TLS, gateway-only token handling, and GitHub
fork isolation.
`private-beta-data-lifecycle-v1.json` is the A1-004/A1-G05 contract for
pre-persistence HMAC redaction, explicit bounded diagnostic profiles,
logical-before-physical whole-deployment deletion, active managed leases,
tokenized downstream obligations, and enforced Shared/L1 generation floors.
`private-beta-operations-v1.json` is the A1-005 composition contract for the
isolated profile's readiness/revocation, ten-class local alert surface,
runner circuit fallback, and bypass/rollback/uninstall procedures. Its bounded
exercise explicitly excludes the eight-hour soak and external pilot evidence.
`owner-controlled-pilot-deployment-v1.json` binds the private synthetic pilot
repository, signed installed release, deterministic workload, authenticated
managed-L1 replay, schema-valid sessions, and explicit non-causal boundary
that closes A1-001 without closing A1-006 or A1-G06.
`owner-poc-evaluation-v1.json` binds the two immutable public pilot revisions,
the paired alternating design, exact required distribution, causal lower bound,
p95 limit, and zero-divergence/failure acceptance used to close A1-006/A1-G06.
`beta-benchmark-harness-v1.json` is the OPS-001/A1 executable smoke contract
for all phase/client strata, real Shared HTTP publication/read paths, private
raw observations, digest-bound summaries, and explicit non-qualification.
`beta-disk-faults-v1.json` is the exact benchmark-bound high-watermark and
out-of-space fault slice with raw trigger/recovery observations, zero-body-read
admission rejection, byte eviction to low, and tamper-evident validation.
`beta-shared-faults-v1.json` is the exact benchmark-bound cancellation,
integrity, SQLite contention, lease-expiry, and pending/commit process-death
slice with 17 private trigger/recovery observations.
`beta-system-faults-v1.json` is the exact benchmark-bound gateway/server
restart, network latency/loss, and signed policy/grant revocation slice with
18 private trigger/recovery observations.
`beta-sustained-v1.json` is the exact one-hour 1/8/32-client benchmark slice
through the real managed gateway and Shared data plane, with 30,000 private
observations, strict golden-runner qualification, and boundary-specific p95
targets.
`beta-soak-v1.json` is the exact eight-hour 1/8/32-client stability slice
through one long-lived managed gateway, Shared store, and authority, with
30,000 private observations and the same strict runner and p95 boundaries.
`beta-circuit-breaker-v1.json` is the A1-G02 flood, oversized-object, and
disk-pressure circuit slice: private per-slot state suppresses Shared between
invocations, preserves writable managed L1, and proves Kotlin/Groovy Gradle
fallback and replay without claiming the separate soak or fixture-size matrix.
`beta-gradle-fixtures-v1.json` is the benchmark-bound small/medium/large
Kotlin DSL build matrix: deterministic multi-project repositories prove exact
known outputs, ordered critical paths, managed-L1 replay, and Configuration
Cache reuse without claiming performance qualification or the separate soak.
`managed-l1-v1.json` is the A0-003 launcher/settings-plugin contract for
opaque scope binding, native retention, generation directories, exclusive
leases, and L2-writer local disablement.
`single-node-shared-storage-v1.json` is the A0-004 server/filesystem contract
for private immutable blobs, one process writer, separate WAL-mode
cache/control schemas, and fail-closed startup.
`self-hosted-single-node-config-v1.json` is the A2-001 strict declarative
configuration and pre-listener storage-preflight contract for the isolated
single-node profile.
`self-hosted-service-install-v1.json` is the A2-002 signed-release, private
layout, path-only secret, deterministic systemd-unit, and reproducible fresh
installation contract.
`self-hosted-upgrade-restart-v1.json` is the A2-003 serialized signed-upgrade,
rollback-safe descriptor composition, unchanged persistent-data restart, and
pending-object invisibility contract.
`self-hosted-manual-restore-v1.json` is the A2-004 absent-target offline
snapshot, cryptographic recovery-authority comparison, strict generation
rotation, atomic publication, and fail-closed admission contract.
`self-hosted-single-node-gate-v1.json` is the A2-G01 current-source composite
that closes MVP-A2 only when configuration, installation, upgrade/restart, and
manual restore all pass together.
`build-impact-manifest-v1.json` is the C3-001 strict customer-authority
boundary for repository/pipeline binding, enumerated original and alternative
entrypoint sets, required artifacts/checks, global paths, and mandatory
`FULL_GRAPH` fallback.
`build-impact-declared-graph-v1.json` is the C3-002 strict
manifest-digest-bound Gradle graph and shadow decision contract: affected
projects include reverse dependents, every required artifact/Build-owned check
must remain reachable, Test-owned checks stay untouched, and all unknown or
global cases run the original full graph.
`build-impact-shadow-validation-v1.json` is the C3-003 immutable observation
and result contract for full-baseline shadow evidence, isolated paired
controls, exact project/artifact/check comparison, explicit false negatives,
and infrastructure/baseline `INCONCLUSIVE` outcomes; every result keeps active
selection disabled.
`build-impact-promotion-gate-v1.json` is the C3-004 exact BIA-002 evidence
gate: current manifest/graph/adapter binding, 30-day and 3,000-decision
minimums, 99% validation coverage, 100 controls per mandatory stratum,
one-sided zero-failure confidence bounds, and immediate suspension on one false
negative. Current checked-in evidence remains honestly `INCONCLUSIVE`, and a
qualified report still cannot activate selection.
`build-impact-selection-v1.json` is the C3-005 sole active omission boundary:
it rechecks canonical manifest/graph digests and bound BIA-002 observations,
selects only a customer-authorized graph alternative, preserves Test-owned
checks separately, and restores original entrypoints on disablement, bypass,
kill switch, drift, insufficient/suspended evidence, or conservative graph
fallback.
`pending-commit-cas-v1.json` is the A0-005 lifecycle contract for durable
pending attempts, canonical Ed25519 decisions, atomic first-writer visibility,
context-bound opaque HTTP GET/PUT, quarantine, and startup reconciliation.
`build-impact-gate-v1.json` is the C3-G01 current-tree composite that closes
owner-operated MVP-C3 only when manifest, graph, shadow/control, unchanged
BIA-002, fail-closed selection, and the offline synthetic full-versus-selected
proof all pass together without mutating source.
`local-authenticated-cache-v1.json` is the A0-006 trust and routing contract
for canonical local authority, monotonic policy/revocation generations,
current-state Shared authorization, gateway credential translation, and the
managed Gradle `HttpBuildCache`.
`gradle-bootstrap-cache-v1.json` is the A0-007 launcher contract for signed
read-only dependency snapshots, per-runner writable homes and leases,
independent Wrapper checksum verification, and native distribution reuse.
`export-gateway-v1.json` is the A0-008 complete/partial BUILD_SESSION contract
for private bounded JSONL, deterministic at-least-once replay, startup
recovery, and stdout export.
`causal-pilot-v1.json` is the A0-009 pre-outcome paired-assignment, neutral
observation, deterministic bootstrap, preliminary result, and internal
net-savings gate contract.

The additional materialized contract `golden-lane-runner-v1.json` pins the first runner class, toolchain, image, and checksums consumed by validation scripts. `release-bundle-v1.md` fixes the first verifiable Linux AMD64 distribution; `deployment-lifecycle-v1.md` owns its local install, upgrade, rollback, and uninstall behavior without claiming publication or online revocation. `walking-skeleton-overhead-v1.md` fixes the first non-promotional baseline-versus-wrapper measurement without replacing the later beta benchmark.

`poc-third-repository-transfer-v1.json` freezes the unchanged clean Build
Impact plus exact standard-`Jar` profile on Apache Kafka 4.3.1 before timing.
It binds the public revision and inputs, Gradle 9.2.1/JDK 25 workload, complete
64-project graph, central `clients` mutation, 64-to-3 project selection,
required test-preparation outputs, full-graph fallback, four-pair budget, and
unchanged POC value gate without adding Kafka-specific product logic.

`poc-qualified-profile-v1.json` and `poc-qualified-profile-v1.md` fix the
explicit repository-owned POC activation that follows that transfer.
`buildopt poc --changes-file PATH`
loads `buildopt-qualified-profile.json`, reports the complete selected or
fallback plan before Gradle starts, enables only Build Impact plus the exact
standard-`Jar` adapter on a qualified alternative, and forces Safe Cache,
retired Runtime Tuning, retired Hot State, retired standard `Copy`, and Shared/Edge out of this path.

`poc-qualified-profile-adoption-v1.json` and
`poc-qualified-profile-adoption-v1.md` freeze the subsequent installed-package
replay on the exact OpenTelemetry and Kafka revisions. They require candidate
Jar replay, historical output digests and native full-graph fallback while
forbidding fresh timing or broader performance claims.

`poc-kafka-composition-usability-v1.json` and
`poc-kafka-composition-usability-v1.md` extend that repository-owned surface
with the already-qualified Kafka Build Impact plus read-only Edge composition.
The v2 plan exposes the exact source precondition and loopback endpoint, while
global changes, precondition drift, missing Edge, local bypass, and HTTP 503
retain explicit native fallbacks without production authorization.

`poc-profile-discovery-v1.md` defines the subsequent read-only derivation
contract. It emits the retained profile only from digest-bound qualification,
graph, generated-state, trace/input, safety and reviewed-contract evidence;
unqualified or uncertain inputs emit a native full-graph decision and never
activate a profile.

`poc-general-build-value-v1.json` and `poc-general-build-value-v1.md` open the
next POC generalization question without changing the terminal portfolio
evidence. They define repository-name-independent structural opportunity
analysis and bind the direct Spring, OpenTelemetry and Kafka whole-profile
measurements. Component percentages cannot be added, repository percentages
cannot be averaged, and only strict installed replication can justify an
explicit reviewed profile; every other target remains native by default.

`poc-structural-transfer-v1.json` and `poc-structural-transfer-v1.md` freeze the
first fresh installed replication after that foundation. They bind Micronaut
Core at an exact public revision, a 75-to-22-project binary-assembly reduction,
eight alternating pairs, byte-identical required JARs, a global-change fallback
and the unchanged 500-ms/2%/positive-bound gate before any timing is accepted.

`poc-source-ownership-v1.json` and `poc-source-ownership-v1.md` separate the
direct project owner of a changed source from the expanded conservative source
boundary of a cyclic Gradle component. Production closure remains unchanged;
the owner-operated POC path may use only subset-validated owned roots and keeps
the full graph for equal-specificity ambiguity.

`poc-structural-profile-v1.json` and `poc-structural-profile-v1.md` define the
repository-independent bridge from measured structural value to an installed
Build-Impact-only profile. Eight positive optimized-native comparisons,
identical outputs, a positive paired bound, exact source hashes and proven
full-graph fallback are required before deterministic materialization. The v4
profile remains repository-owned and rechecks all three graph inputs at runtime.

`poc-structural-profile-adoption-v1.json` and
`poc-structural-profile-adoption-v1.md` freeze the subsequent installed replay.
The same Micronaut source, mutation, outputs and optimized-native control are
retained, but the candidate is now only `buildopt poc --changes-file`. Profile
materialization is excluded while validation, planning and launcher overhead
remain inside all eight candidate observations. The checked result retains
72.16% mean savings with 8/8 positive pairs and both global and graph-drift
full-graph fallbacks.

`poc-generic-evaluation-v1.json` and `poc-generic-evaluation-v1.md` define the
single generic POC decision that combines structural analysis and exact
qualification. It writes a profile only after qualified evidence and otherwise
retains native full graph; it never invents required outputs or activates the
profile automatically.

`poc-generic-measurement-v1.json` and `poc-generic-measurement-v1.md` define
the isolated paired evidence collector between analysis and evaluation. Two
local clones, two Gradle homes and two independently restored native-cache
seeds preserve arm isolation while eight alternating pairs, exact outputs and
full-graph fallback produce either qualified or inconclusive review evidence.

`poc-apache-groovy-classes-v1.md` records the first substantial fresh-family
result produced by that generic collector. It binds Apache Groovy 5.0.8, the
exact local source mutation, optimized-native `classes`, the two-project
candidate, 66 exact class outputs, eight alternating pairs and the global
full-graph fallback. It also records why distribution and aggregate-assemble
candidate scopes were rejected before accepting timing evidence.

`poc-generic-profile-matrix-v1.json` and
`poc-generic-profile-matrix-v1.md` freeze the same installed structural-only
method across Spring, OpenTelemetry, Kafka, Micronaut, and Groovy. The contract
binds each public revision, declared workflow, exact source change, output
scope, 12-worker native control, eight alternating isolated pairs, exact
fallback, and non-additive reporting boundary before timing.

`poc-generic-profile-matrix-v2.json` and
`poc-generic-profile-matrix-v2.md` preregister a complete fresh rerun after the
v1 OpenTelemetry cell exposed that checkout/cache preparation and output
hashing were incorrectly included in the inter-arm gap. Both arms are now
prepared before either measured process and outputs are checked after both;
the same strict five-second process-only gap, repositories, workflows,
thresholds, outputs, fallbacks, and non-additive reporting rules remain.

`poc-generic-profile-matrix-v3.json` and
`poc-generic-profile-matrix-v3.md` freeze the terminal five-repository rerun
after two resource-control failures showed that the untimed fallback must not
overlap both measured daemons. The v3 evidence remains immutable: Kafka,
Micronaut, and Groovy qualify; Spring retains native under the 8-of-8 rule; and
OpenTelemetry retains native because its four-worker non-parallel fallback
changed required output bytes.

`poc-generic-profile-matrix-v4.json` and
`poc-generic-profile-matrix-v4.md` preregister the OpenTelemetry-only correction.
They preserve every timed condition and threshold while making the untimed
no-daemon fallback retain measured parallel 12-worker scheduling. The fresh
terminal row qualifies at 14.43% faster with 8/8 positive pairs, exact outputs,
zero product failures, and successful full-graph fallback. No v3 timing is
reused.

`poc-generic-holdout-v1.json` and `poc-generic-holdout-v1.md` freeze the first
unseen substantial holdout before proposal or timing. Hibernate ORM uses its
root `assemble` workflow, one exact `hibernate-core` source change, declared
core-library outputs, optimized native Gradle with 12 workers, and the same
eight-pair value and correctness gates as the terminal five-repository matrix.
The contract intentionally records no expected candidate or favorable result.

`poc-generic-holdout-v2.json` and `poc-generic-holdout-v2.md` retain the v1
zero-pair failure and correct only its invalid repository-owned output path.
Hibernate's frozen build plugin sets module build directories to `target`, so
v2 replaces `hibernate-core/build/libs/**` with
`hibernate-core/target/libs/**` before a fresh proposal or timing run. No v1
warm-up, proposal or observation is reused.

`poc-generic-holdout-v3.json` and `poc-generic-holdout-v3.md` retain the full
v2 7/8 result and preregister a generic causality correction before any new
timing. The measurement now separates cache seeding from daemon stabilization
and records bounded task-outcome plus log-digest diagnostics for every excluded
warm-up and measured arm. The frozen repository, workflow, mutation, outputs,
Gradle options, eight pairs and qualification thresholds remain unchanged; no
v2 timing is reused or discarded.

`poc-generic-holdout-v4.json` and `poc-generic-holdout-v4.md` preserve the v3
4/8 diagnostic and preregister a second generic correction. Each arm now warms
the exact target workload from the frozen base cache before pair one, binds a
normalized task/outcome fingerprint, and captures interval-scoped Linux CPU,
memory, and IO PSI outside measured wall time. Task-shape drift fails closed;
PSI remains diagnostic and cannot relax the unchanged 8-of-8 value gate.

The retained v4 result completes all eight fresh pairs and the exact fallback,
but remains unqualified: 5/8 positive pairs, 3.82% mean savings, a negative
lower interval bound and native task/outcome drift between 300, 301 and 302
tasks. Its four post-hoc AB/BA blocks are all positive, which authorizes only a
new preregistration for reciprocal blocks and stronger target stabilization;
no v4 timing may be reused as qualifying evidence.

`poc-generic-holdout-v5.json` and `poc-generic-holdout-v5.md` preregister the
recoverability test implied by v4. Two independent eight-pair batches become
eight reciprocal `AB/BA` blocks; each arm must expose two target-workload
observations and exact task paths before its measured pairs. Qualification
still requires 500 ms, 2%, a positive lower bound, eight-of-eight positive
blocks, exact outputs, both full-graph fallbacks and zero product failures.
Target drift is reported path by path and retains native rather than being
discarded or special-cased.

`poc-generic-workflow-value-v1.json` and
`poc-generic-workflow-value-v1.md` freeze the first public value comparison
across four build-owned workflow families: Groovy JAR packaging, Kafka typed
Checkstyle verification, Kafka fat-JAR distribution, and Spring test-class
preparation. Every row uses the installed generic profile path, eight isolated
alternating pairs, exact outputs, and native full-graph fallback. Families are
qualified or rejected independently; no percentages are averaged or added.

`poc-statistical-qualification-v2.json` and
`poc-statistical-qualification-v2.md` preregister a fresh, comparable rerun of
Spring Framework, OpenTelemetry Java Instrumentation, Kafka, Micronaut and
Groovy. Each repository receives two independent eight-pair captures from the
same BuildOpt revision and executable. Adjacent opposite-order observations
form eight balanced blocks; qualification requires material mean saving,
positive median and bootstrap lower bound, at least six positive blocks,
non-regressive candidate p95, exact outputs, stable task shape, both full-graph
fallbacks and zero product failures. Historical v1 decisions remain immutable.

`poc-new-family-transfer-v1.json` and
`poc-new-family-transfer-v1.md` freeze a transfer test on Ktor before any
owner workflow, BuildOpt proposal or target timing. The contract binds public
revision `bc7de799`, the public unqualified `jvmJar` selector, one internal
`ktor-http` source comment, the module JVM JAR, exact-byte comparison,
optimized native Gradle and the balanced two-capture qualification gate. The
original root-`assemble` target declaration was rejected before timing because
Ktor reads target switches from `gradle.properties`, not CLI project
properties. The terminal result derives `:ktor-http:jvmJar` in both fresh
captures and qualifies without repository-name product rules.

`poc-new-family-change-breadth-v1.json` and
`poc-new-family-change-breadth-v1.md` freeze the follow-up Ktor matrix before
any new proposal or timing. It covers an upstream dependency-source edit, a
JVM resource edit, a two-module mixed-source edit and an untimed global
configuration fallback under the same public `jvmJar` selector. The generic
runner now accepts multiple changed paths while retaining the legacy
single-path contract; every selective cell keeps the balanced value gate and
the global cell must remain on native Gradle.

The terminal matrix qualifies all three selective cells: dependency source
saves 85.80%, a JVM resource saves 86.51%, and a two-module mixed-source edit
saves 77.98%. All 48 pairs and 24 reciprocal blocks improve with exact JARs;
the root-configuration cell retains the native full graph twice without a
timing claim. Evidence remains bound to the public revision and selector.

`poc-new-family-installed-profile-replay-v1.json` and
`poc-new-family-installed-profile-replay-v1.md` freeze the public-package
adoption check for those three Ktor profiles. The installed `buildopt poc`
path must select every exact reviewed plan, preserve contemporary JAR bytes,
and retain native Gradle before execution when the complete invocation option
list differs from the qualified profile. The terminal
[evidence bundle](../benchmarks/results/poc-new-family-installed-profile-replay-v1/README.md)
passes 3/3 exact selections, 3/3 option-drift native fallbacks and 3/3 exact
candidate/fallback output comparisons through public `v0.3.2`. Historical
timing remains immutable.

`poc-magic-onboarding-contract-v1.json` and
`poc-magic-onboarding-contract-v1.md` fix the stable
`buildopt optimize build` surface used by the automatic-discovery implementation.
The command requires zero manual BuildOpt files, writes private atomic state,
result and discovery documents, accepts only digest-exact context-bound resume,
preserves Gradle exit behavior and keeps production authority false.

`poc-magic-auto-discovery-v1.json` and
`poc-magic-auto-discovery-v1.md` bind the derived repository/workflow/base/
change/output/graph inputs and fail-closed reasons. Packaging, verification,
distribution and test-preparation fixtures reach a structural proposal through
the customer command; unsupported, global and ambiguous inputs retain native
Gradle. No calibration, selection or timing claim is created.

`poc-magic-calibration-v1.json` and `poc-magic-calibration-v1.md` bind the
shared discovery/calibration deadline, frozen eight-pair value gate, exact
output and full-fallback requirements, break-even decision and digest-exact
evidence resume. A qualified result remains POC learning: it creates no
profile, automatic selection or production authority.

`poc-magic-profile-portfolio-v1.json` and
`poc-magic-profile-portfolio-v1.md` bind the bounded repository-scoped family
portfolio and its private exact artifacts. `poc-magic-auto-replay-v1.json` and
`poc-magic-auto-replay-v1.md` then authorize only exact owner-invoked replay:
all repository, revision, executable, Wrapper, workflow, option, graph, output,
evidence and profile bindings pass before Gradle or the original optimized
native workflow runs. Selection overhead is part of invocation wall time;
production authority remains false.

`poc-magic-ci-onboarding-v1.json` and
`poc-magic-ci-onboarding-v1.md` expose the same command through GitHub Actions
and GitLab with one ordinary input. Provider repository identity replaces the
ephemeral checkout path only in the repository-scope digest; executable,
Wrapper, argv, base/target, discovery and budget bindings remain exact before
restored state can be accepted. Both providers publish a checksummed result
without state, command text, logs, credentials or absolute paths. Cache loss,
corruption or drift retains native Gradle and no service is required.

`poc-new-family-calibration-economics-v1.json` and
`poc-new-family-calibration-economics-v1.md` freeze the follow-up economics
study before fresh phase timing. The study binds the three terminal Ktor value
cells, separately records installed cold discovery, exact replay and candidate
stabilization, and computes repeated-build break-even against each unchanged
terminal saving. The global fallback remains untimed and no result is averaged
across change shapes.

The terminal [economics evidence](../benchmarks/results/poc-new-family-calibration-economics-v1/README.md)
contains two fresh captures per cell. First-time calibration repays after 7,
10 and 8 qualifying builds; exact proposal replay plus fresh stabilization
repays after 2, 4 and 3. Replay artifacts, drift rejection, target fingerprints
and terminal output bindings are checked without creating measured pairs or a
new qualification decision.

`poc-generic-output-equivalence-v1.json` and
`poc-generic-output-equivalence-v1.md` preregister the three public workflows
whose outputs were semantically stable but not byte-reproducible. Exact bytes
remain the implicit contract. Explicit reviewed rules may relocate only the
isolated repository root in UTF-8 reports or compare canonical ZIP contents;
one exact properties key in one exact archive entry may be declared volatile.
Every other payload, path, mode, output shape, fallback, and wall-time gate
remains bound. Two fresh captures per workflow use the balanced qualification
method and no repository-name product rule.

The terminal [output-equivalence evidence](../benchmarks/results/poc-generic-output-equivalence-v1/README.md)
qualifies all three subjects independently: Groovy `jar` saves
52,864.125 ms/73.10%, Kafka Checkstyle saves 24,626.875 ms/29.73%, and Kafka
`shadowJar` saves 27,102.875 ms/66.55%. All 48 raw pairs improve, every output
passes its reviewed semantic contract, task shapes and tails are stable, both
fallbacks pass per subject, and product failures remain zero. The result is
review-only POC evidence and does not widen the contract or authorize
automatic activation.

`poc-generic-change-breadth-v1.json` and
`poc-generic-change-breadth-v1.md` preregister ten independent cells across
Groovy JAR packaging, Kafka Checkstyle verification, and Kafka shadow-JAR
distribution. Six selective cells cover distinct leaf and shared-source edits;
four build-logic/global cells must retain the complete owner workflow without a
timing claim. Candidate lifecycle tasks come from reviewed output owners and
must still cover the changed source in the generated graph. Each selective
cell keeps the balanced two-capture value gate and every fallback is repeated
twice through the installed proposal path.

The terminal [change-breadth evidence](../benchmarks/results/poc-generic-change-breadth-v1/README.md)
qualifies all six selective cells: Groovy `jar` saves 73.54% and 65.80%, Kafka
Checkstyle saves 28.00% and 30.10%, and Kafka `shadowJar` saves 66.64% and
79.54% for their two distinct source changes. All 96 raw pairs and 48 blocks
improve. Outputs, tails, task shapes, 12 selective fallbacks and zero-failure
gates pass. The four build-logic/global cells retain native Gradle in all eight
captures and make no timing claim. The result remains review-only POC evidence.

`poc-calibration-economics-v1.json` and
`poc-calibration-economics-v1.md` preregister the first-run economics follow-up
for all six qualified change-breadth cells. Two fresh captures per cell time
repository checkout, exact offline Wrapper preparation, fixture preparation,
and the real combined `buildopt profile propose` preflight/discovery command.
The assembler then binds the existing immutable control/candidate warm-ups and
terminal savings. It reports installed-workflow, complete POC-validation, and
conservative cold-single-workflow break-even separately for every cell; it
never averages unrelated percentages or hides setup inside terminal pairs.

The terminal [calibration economics evidence](../benchmarks/results/poc-calibration-economics-v1/README.md)
preserves 12 fresh phase captures and recomputes all six cells. Discovery plus
candidate warm-ups repays after 10–31 qualifying builds; the complete
comparative POC, including native-control warm-ups, repays after 20–55.
Checkout is visible but excluded as shared work and exact offline distribution
materialization changes no rounded POC break-even.

The [calibration efficiency protocol](./poc-calibration-efficiency-v1.md)
preregisters one-pass discovery, digest-bound proposal replay, and bounded
adaptive candidate stabilization. It keeps cold calibration and exact replay
economics separate and does not change terminal build savings.

The terminal [calibration efficiency evidence](../benchmarks/results/poc-calibration-efficiency-v1/README.md)
passes all six cells. Cold discovery is 8.01%–21.08% lower, exact replay takes
0.281–1.261 seconds, and installed break-even improves from 10–31 to 9–26
qualifying builds. All 12 drift probes miss the cache, replay artifacts are
byte-identical, and existing terminal savings and correctness evidence remain
unchanged.

The [public installed-profile replay protocol](./poc-installed-profile-replay-v1.md)
then freezes the adoption check for those six profiles. Its terminal
[evidence bundle](../benchmarks/results/poc-installed-profile-replay-v1/README.md)
installs public `v0.3.1`, drives the user-facing `buildopt poc` command in clean
external checkouts, and proves six exact candidate plans, six digest-drift
native fallbacks, six same-replay semantic-output matches, and six unchanged
historical qualifications. This block performs no new timing.

The [cross-date output-equivalence protocol](./poc-cross-date-output-equivalence-v1.md)
freezes the narrow follow-up to that public replay. It adds only Groovy's
`BuildDate` beside the already reviewed `BuildTime`, requires the old contract
to reject the controlled date change, and requires undeclared property and ZIP
payload drift to remain visible. Historical profiles and timing qualifications
remain immutable.

The retained [v5 evidence](../benchmarks/results/poc-generic-holdout-v5/README.md)
passes that unchanged contract: native averages 216.724 seconds and BuildOpt
203.991 seconds, saving 12.733 seconds/5.88% with interval
+6.808..+19.859 seconds and eight of eight positive reciprocal blocks. Exact
target shapes remain 300/32 tasks, required outputs are byte-identical and both
full-graph fallbacks pass. The decision is review-only.

`poc-generic-output-contract-v1.json` and
`poc-generic-output-contract-v1.md` freeze the fail-early correction implied by
Hibernate's original empty `build/libs` declaration. The exact owner workflow
runs once before structural discovery; Gradle-declared outputs become
repository-contained review candidates with unambiguous project ownership.
The frozen public evidence must report the wrong declaration as empty, expose
an owned `hibernate-core/target/libs` candidate, retain native, and start no
warm-up or timing. No repository-specific path rule or performance claim is
authorized.

The terminal v2 evidence reduces the 29-project root graph to one project and
saves 19,386.25 ms/7.80% on average with exact outputs and zero product
failures. Seven of eight pairs improve, so the unchanged 8-of-8 gate retains
native Gradle and the full-graph fallback succeeds.

`poc-trace-hypothesis-v1.json` and `poc-trace-hypothesis-v1.md` bind the final
trace-gated optimization decision. They require at least 500 ms of causally
recoverable product-owned critical-path work in two families, reject overlapping
task-duration sums and already-qualified mechanisms as new hypotheses, and
emit `NO_ACTIONABLE_HYPOTHESIS` when no phase clears every condition.

`poc-portfolio-decision-v1.json` and `poc-portfolio-decision-v1.md` bind the
terminal POC synthesis. The exact installed matrix, qualified Kafka cell,
deterministic discovery result and trace decision select `CONTINUE`,
`SPECIALIZE`, or `STOP/REFRAME` without averaging repository percentages or
adding mechanism effects. The checked result specializes to the bounded Kafka
profile and withdraws the general accelerator claim.

`poc-kafka-packaging-v1.json` and `poc-kafka-packaging-v1.md` freeze the next
real value experiment before timing. They compare Kafka's optimized native
root `assemble` with the installed qualified three-project client-Jar path,
bind the exact output and global fallback, and retain the existing four-pair
500-ms/2%/positive-bound qualification gate.
