# Controlled remote-cache value fixture

This independent Gradle 9.6.1 fixture owns eight cacheable deterministic
producers. Each producer writes one non-compressible 4-MiB output, for eight
required files and 32 MiB per build. The local build cache is disabled and the
remote URL plus seed-only push permission come from the experiment harness.

The fixture is not a product benchmark by itself. The versioned remote-cache
protocol controls cache population, Shared/Edge topology, modeled network,
arm isolation, pair order, output hashing and terminal value decision.
