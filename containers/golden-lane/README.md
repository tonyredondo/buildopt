# Golden lane container

The executable image for golden lane v1 is:

```text
docker.io/library/eclipse-temurin@sha256:a5418a1fcf440bb273e1db3bce5b0794eb78bfc9d044ba740de76dcbe6075f50
```

It corresponds to the Linux amd64 manifest of source tag `21.0.11_10-jdk-jammy` and runs Eclipse Temurin 21.0.11+10.

We do not build a derived image yet: doing so would require a registry, signing, an SBOM, and the `F0-038` lifecycle. The runner mounts the checkout into this immutable image and runs only the repository's Gradle Wrapper.

Functional smoke test:

```bash
./dev/run-golden-lane-container --smoke
```

Contractual evidence:

```bash
./dev/run-golden-lane-container --require-runner-class
```

The second command rejects hosts that cannot provide the 4 vCPU/16 GiB development runner class.

Both modes run the `F0-040A` fixture and the Gradle 9.6.1 half of `SPK-001`
after the base Java artifact checks. Strict mode therefore covers the parallel
equivalent tasks, Worker API isolation modes, child JVM, remote HTTP cache,
failure/cancellation, Configuration Cache reuse, and the tested
`UNATTRIBUTED` whole-attempt fallback inside the pinned 4-CPU/16-GiB container.
The checksum-pinned Gradle 8.14.3 half runs through the host-side `--full`
matrix.

Both modes also execute the four-pair `WS-009` neutral-envelope smoke. Only
strict mode qualifies the runner class and may create the initial
`benchmarks/results/ws-009-golden-lane.json`; once present, that historical
report is validated but never rewritten.
