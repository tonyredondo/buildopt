# Spring Messaging Candidate Correctness v1 Tracker

| Block | Outcome | State |
|---|---|---|
| `SMCC-001` | Freeze the exact untimed three-request correctness contract | `DONE` |
| `SMCC-002` | Execute native/native/candidate from empty state | `TODO` |
| `SMCC-003` | Reconstruct exact outputs and authorize value or stop | `WAITING` |

SMGC supplies selection and expected-output bindings only. This route permits
one natural candidate after two native requests, zero timing samples and zero
public-source writes. A pass may authorize only a separate paired-value
contract.
