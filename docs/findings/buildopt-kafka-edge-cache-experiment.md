# Edge Cache on Kafka

Edge Cache keeps a copy of saved build results near the machine running a
build. This Kafka experiment asked whether avoiding a slow remote read could
reduce the time needed to compile client code and tests. It saved 15.21% under
the network conditions tested.

## What changed

Kafka ran `:clients:testClasses`, which compiles production and test code without
executing the tests. Both versions used Gradle's remote build cache and restored
the same files. One read directly from the shared cache; the other read from a
nearby BuildOpt cache that had already been filled.

The remote connection was simulated locally, with a 337-ms delay per response
and a transfer rate of about 7 MB/s. Those settings came from three downloads
of a fixed source archive. This was a controlled network experiment, not a
measurement of a deployed remote cache.

Dependencies and the nearby cache were prepared before timing. Filling that
cache transferred 8,431,390 bytes in six requests; that preparation was not
included in the measured saving.

## Results

Four comparisons alternated which version ran first:

| Measurement | Result |
| --- | ---: |
| Average reading directly from the shared cache | 8,885.25 ms |
| Average reading from the nearby cache | 7,534 ms |
| Average saving | 1,351.25 ms / 15.21% |
| Comparisons with a saving | 4 of 4 |
| 95% interval for the saving | 788.25–1,883 ms |

Each direct run made four remote requests and transferred 7,652,777 bytes. The
nearby cache served those results without another remote request. All 4,062
required output files matched, as did the recorded task outcomes. There were
no failures attributed to BuildOpt.

## What the result supports

Keeping results nearby helped when remote reads had this delay and bandwidth
limit. The experiment did not test whether the saving would repay filling and
maintaining that nearby cache through everyday code changes.

The later [five-project Edge Cache study](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/remote-cache-locality-value-v3/README.md)
did not find a qualifying saving on any of the three projects that reached
timing. That limits the broader claim while leaving this Kafka result valid
for its tested conditions.

The [original Kafka measurements](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/poc-remote-cache-transfer-v1.json)
retain all four comparisons, file checks and network settings. The
[experiment specification](https://github.com/tonyredondo/buildopt/blob/main/specs/poc-remote-cache-transfer-v1.json)
records the source version, command and requirements.
