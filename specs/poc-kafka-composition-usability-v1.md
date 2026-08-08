# Kafka composition usability protocol

This protocol converts the already-qualified Apache Kafka Build Impact plus
prewarmed Edge composition into one repository-owned POC invocation. It does
not rerun or expand the performance claim and does not introduce production
policy, rollout, or operating requirements.

The normalized Kafka checkout commits the v2 qualified profile as
`buildopt-qualified-profile.json`, records the exact SHA-256 precondition for
`build.gradle`, and invokes:

```text
buildopt poc --changes-file .buildopt-changes \
  --edge-url http://127.0.0.1:<PORT>
```

The endpoint is an explicit read-only IPv4 loopback Edge origin. It is never
stored in the repository profile and cannot enable writes. Before Gradle
starts, `buildopt poc` emits a v2 plan containing the evaluated precondition,
the exact endpoint, selected entrypoints, required outputs, omitted-project
count, enabled composition, disabled mechanisms, and `productionAuthorized:
false`.

Only the exact repository-owned candidate may combine Build Impact with Edge.
Global or unknown changes, local bypass, a changed source precondition, and a
missing or malformed endpoint all select the native full graph before Gradle
starts and do not configure Edge. If a valid Edge endpoint later returns HTTP
503, Gradle disables that remote cache and executes the already-selected graph
locally; the required output must remain byte-identical.

The performance basis remains the corrected Kafka result in
`benchmarks/results/poc-kafka-impact-edge-composition-v2.json`: 82.35% mean
reduction, four of four positive pairs, and exact normalized ShadowJar bytes.
This block proves that the same mechanism set is usable through product-owned
CLI/profile surfaces rather than experiment-only environment variables. It
does not add those percentages to any other component result.

Run `./dev/check-poc-kafka-composition-usability` for the executable contract.
