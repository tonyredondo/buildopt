# Request-aligned public capture

This directory is the immutable `SWL-REQUEST-003` evidence set. It was captured
from zero BuildOpt observations with the exact frozen ordinary request and the
chronological first-parent selection policy in `contract.json`.

The ledger contains 110 transitions:

| Family | Attempts | Relevant actions | Irrelevant | Unsafe | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| Apache Groovy | 7 | 5 | 2 | 0 | relevant target met |
| Apache Kafka | 30 | 0 | 0 | 30 | budget exhausted |
| Micronaut Core | 30 | 0 | 16 | 14 | budget exhausted |
| OpenTelemetry Java Instrumentation | 30 | 0 | 19 | 11 | budget exhausted |
| Spring Framework | 13 | 5 | 7 | 1 | relevant target met |

Each ledger row binds its raw base capture, target capture and reconstructed
classifier report by SHA-256. `sha256sums` binds the evidence files. Validate
the complete set from the repository root with:

```bash
./dev/check-request-aligned-public-capture
```

Only two of five family inputs are complete and only those two expose actions.
This is not the independent breadth decision and it contains no candidate
execution, wall-time measurement or activation authority. `SWL-REQUEST-004`
must rebuild every report before applying the frozen gate.
