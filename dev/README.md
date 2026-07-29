# Development tools

Reproducible entrypoints for bootstrap, diagnostics, and local execution.

`ENV-001..012` will add `toolchains.lock.yaml`, `bootstrap`, `doctor`, and `run`. No script may require `sudo`, replace global toolchains, or download an artifact without verifying its digest.
