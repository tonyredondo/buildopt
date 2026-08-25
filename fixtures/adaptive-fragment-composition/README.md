# Adaptive fragment composition fixture

This fixture supplies the root build scripts for the `AF-011` controlled
workflow. The runner combines them with the existing Build Impact project and
the reviewed task-contract source, then measures each mechanism independently
and both compatible mechanisms together.

The fixture is synthetic and exists only for attributable POC timing. Cache
locality is measured independently because it has a different remote-cache
object contract.
