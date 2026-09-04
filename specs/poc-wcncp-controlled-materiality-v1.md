# WCNCP controlled materiality successor v1

`WCNCP-009A` is a prospective, versioned successor for the three families that
remain incomplete after `WCNCP-009`. The original report and all 16 selected
diagnostics remain immutable. In particular, this successor does not relabel
`LOCAL_FUNCTIONAL` duration as controlled evidence and does not extend the
original two-diagnostics-per-family budget after observing its outcome.

The machine contract freezes a new six-start budget before dependency prefetch
or measurement. It binds the exact GraphQL Java, Apache Groovy, and Test Retry
revisions and source facts already identified prospectively by WCNCP-009. One
unmeasured dependency prefetch per family is setup cost. Each family then gets
exactly two fresh optimized-native diagnostic starts from separate clean source
roots and a shared warm dependency/cache root.

The runner is this owner-operated Linux AMD64 machine. It need not be a
physically dedicated host, but it must be connected to mains power, expose at
least 15 GiB of memory, run all online CPUs under the `performance` governor,
pin Gradle to CPUs 0-3 and four workers, and run no concurrent agent-launched
build or benchmark. Seven pre-outcome SHA-256 samples over the same 128 MiB
memory-backed buffer must have a maximum/minimum duration ratio no greater than
1.15. A failed preflight produces `INCOMPLETE_PERFORMANCE_ENVIRONMENT`; it is
never relaxed or rerun after family outcomes are visible.

For Configuration Cache possibilities, materiality is the non-overlapping
union of Gradle's root `Load build`, `Configure build`, and `Calculate build
tree task graph` operations. This is the serial configuration critical-path
work the correction might unlock; it is not a speedup claim. For Apache Groovy,
materiality is only the duration of executed
`org.apache.groovy.gradle.ReleaseInfoGenerator` tasks that are members of the
longest hard-dependency task chain. Cumulative parallel task time never counts.

Both rows must independently reach 500 ms and 2% of the complete root `Run
build` operation. Test Retry remains `UNSUPPORTED_PROBLEM_CLASS` even when its
configuration work is material because two external License plugin tasks still
block Configuration Cache and dependency changes are forbidden. GraphQL Java
may enter correctness only if one later reviewed proposal closes every bound
repository-owned blocker. Apache Groovy may enter only when its exact affected
task class is material in both rows.

This block runs no candidate, changes no public source, collects no paired
control/candidate value samples, and makes no speedup claim. A conclusive result
is folded back into WCNCP-009. WCNCP-010 opens only if all ten original families
are then conclusive and at least three are actionable and material.
