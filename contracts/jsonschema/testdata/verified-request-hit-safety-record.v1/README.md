# Verified request hit safety record vectors

`valid/complete.json` represents every fact required by the `VRH-002`
fail-closed contract. `negative-matrix.json` applies one bounded mutation at a
time and requires a stable native-retention reason. These semantic cases are
validated by `internal/requesthit`; JSON Schema validates only the closed
document shape.

The complete verdict is classification evidence only. It does not select an
action, restore an output, start or skip Gradle, or authorize timing.
