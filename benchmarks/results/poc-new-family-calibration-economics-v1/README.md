# Ktor calibration-economics evidence

This directory contains the terminal evidence for the preregistered
`POC-NEW-FAMILY-CALIBRATION-ECONOMICS-001` study. It measures the one-time
cost of discovering and stabilizing the three qualified Ktor structural
profiles, then compares that cost with their immutable per-build savings.

The study does not rerun or rewrite the terminal performance experiment. It
uses the exact savings, outputs, source revision, Gradle options and
qualification objects from the Ktor change-breadth result.

## Terminal results

Each cell contains two fresh, isolated captures. Cold proposal cost includes
the real output preflight and combined structural discovery. Candidate
stabilization includes cache seed, base-daemon stabilization and two matching
target-workload fingerprints. Exact replay is accepted only when every
repository, Wrapper, graph, option, output and executable binding matches.

| Ktor change | Cold proposal mean | Candidate stabilization mean | Installed calibration cost | Saving/build | Installed break-even | Exact-replay evaluation cost | Replay break-even |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Upstream dependency source | 372.864 s | 168.255 s | 541.119 s | 84.314 s | **7 builds** | 168.625 s | **2 builds** |
| JVM service resource | 333.585 s | 151.175 s | 484.760 s | 49.517 s | **10 builds** | 151.501 s | **4 builds** |
| Mixed production source, two modules | 384.985 s | 219.425 s | 604.410 s | 81.781 s | **8 builds** | 219.756 s | **3 builds** |

Raw exact proposal replay itself takes 0.321–0.376 seconds. The replay
evaluation cost above deliberately also includes fresh candidate
stabilization; setup is not hidden inside the already-qualified terminal
pairs.

All six cold/replay proposal artifact sets are byte-identical, all six option
drift probes reject replay, all target workloads converge after two exact
fingerprints, and every required output matches the immutable terminal digest
and count. The global-configuration cell remains untimed native full graph.

## Interpretation

- A first reviewed Ktor profile repays discovery and stabilization after
  7–10 qualifying repetitions under the corresponding frozen change shape.
- Rechecking an exact cached proposal repays after 2–4 repetitions because
  structural discovery falls to a sub-second digest-bound replay.
- The resource cell has the highest break-even despite the lowest setup cost,
  because its immutable per-build saving is also the smallest.
- Percentages, costs and break-even counts remain cell-specific. They are not
  averaged and cannot be added to other mechanism results.

This is POC adoption evidence, not automatic or production activation. A
profile still requires review, exact bindings and native fallback on drift.

## Reproduce the checks

The phase measurements are not rerun on every CI push. CI validates every raw
artifact and recomputes the terminal summary:

```bash
./dev/check-new-family-calibration-economics
./dev/check-new-family-calibration-economics-result
./dev/test-new-family-calibration-economics-result
```

The negative fixtures prove that summary and structured-capture tampering fail
closed. Raw process logs are intentionally not published; their SHA-256 values
remain bound in each phase record. The next block must replay these reviewed
Ktor profiles through the public package before making an onboarding claim.
