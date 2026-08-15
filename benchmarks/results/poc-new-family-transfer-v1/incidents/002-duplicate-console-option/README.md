# Rejected pre-measurement attempt: duplicate console option

After the generic project-property correction, the next fresh attempt reached
the output-contract launcher but stopped before configuring or executing
Ktor's owner `assemble` tasks. Gradle rejected two identical
`--console=plain` arguments: one came from the preregistered owner option vector
and one from BuildOpt's controlled output-contract invocation.

The repository-independent correction now removes owner daemon/console values
only for this untimed preflight and appends BuildOpt's single controlled
`--no-daemon --console=plain` pair. All other owner options, including the
reviewed `-Pname=value` properties, remain unchanged. Timed control and
candidate invocations still receive the exact preregistered vector. No partial
proposal, warm-up or duration from this attempt is reused.
