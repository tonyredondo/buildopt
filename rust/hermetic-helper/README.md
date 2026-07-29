# Linux Hermetic Helper

Experimental optional Rust helper for C1.

It applies the OS boundary for a supported task-specific producer. Rust is not a source of hermeticity by itself, and this component is not required for A1/B. Its viability is decided through `SPK-003`.

`ENV-009` pins the repository compiler through the root `rust-toolchain.toml` and validates it with `./dev/check-rust-toolchain`. No Cargo package or helper behavior is activated here; those artifacts begin only when the spike has executable isolation and coverage contracts.
