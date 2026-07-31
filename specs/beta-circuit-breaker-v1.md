# Private-beta circuit breaker v1

This contract materializes the circuit-breaker portion of `A1-G02` without
claiming the separate eight-hour soak or small/medium/large fixture evidence.
The managed gateway owns one breaker per runner slot and stores only its
reason and UTC retry window in private state; upstream authority and
credentials remain invocation-only.

## Opening conditions

The gateway opens the circuit when concurrent verified GET reservations
exhaust the spool quota, a verified object exceeds 100 MiB, an upstream PUT
returns `413`, or the verified spool observes `ENOSPC`/`EDQUOT`. GET-side
pressure remains a byte-free `404`; the existing `Expect: 100-continue` path
preserves an upstream `413` without consuming the rejected request body.

Opening the breaker removes the active cache binding immediately. During the
five-minute cooldown, the next registration receives the stable local gateway
identity but no L2 binding. The launcher omits the managed Shared authority
from Gradle and converts an authorized L2 writer into a private read/write L1
invocation. Invalid, public, oversized, symlinked, or unknown-field state
fails closed by continuing to suppress L2.

After the canonical retry time, a valid state file is removed and the state
directory is synced before a later invocation may retry L2. A gateway restart
does not lose an unexpired circuit.

## Gradle preservation

The real Gradle 9.6.1/JDK 21 TestKit runs Kotlin and Groovy fixtures twice for
each of `FLOOD`, `OBJECT_TOO_LARGE`, and `DISK_PRESSURE`. The fallback build
must succeed with Shared absent and populate the native managed L1; the next
clean build must be `FROM_CACHE` and reuse Configuration Cache.

Run the complete bounded check:

```bash
./dev/check-beta-circuit-breaker
```

This closes only `OPS-001/A1-CIRCUIT-BREAKER-MATRIX`. The exact eight-hour
soak, the benchmark's small/medium/large Gradle fixture matrix, `A1-G02`, and
the complete `OPS-001/A1` profile remain open.
