# Cross-commit value recovery evidence

This bundle compares the same Apache Kafka public history window before and
after removing redundant full-workflow observation from an already-selected
structural replay. Both captures use installed packages, the same qualifier,
the same six descendants, exact required outputs, and the qualified-lifetime
v2 subject contract.

| Measurement | Before | After |
| --- | ---: | ---: |
| Selected replay control | 160.895 s | 147.552 s |
| Selected replay candidate | 166.299 s | 42.577 s |
| Attributable selected saving | **-5.404 s** | **+104.975 s / 71.14%** |
| Six-build cumulative net after learning/publication | **-22.040 s** | **+66.772 s** |
| Terminal decision | Not paid back | Paid back in observed window |

The after candidate is 123.722 seconds faster than the before candidate on the
same selected revision. Its attributable selected-replay value improves by
110.379 seconds. The five native-retained after observations total -31.441
seconds of arm delta; that uncontrolled timing variation is included in window
economics but is not claimed as BuildOpt mechanism value.

The selected replay restores the intended graph reduction: the candidate sees
four cache hits versus 32 in control while preserving all 4,449 required output
files exactly. Qualification itself saves 5.557 seconds/21.21% with 8/8
positive pairs and costs 3.576 seconds; publication costs 3.186 seconds.

Validate the complete bundle with:

```bash
./dev/check-cross-commit-value-recovery
```

The checker validates both raw captures with the existing qualified-lifetime
v2 subject checker, recomputes every comparison from those files, and rejects
any mutation of the selected replay, economics, output gates, frozen revisions,
or implementation bindings.

This is bounded proof-of-concept evidence from one Kafka workflow and public
history window. It is not a universal 71.14% claim or production authority.
