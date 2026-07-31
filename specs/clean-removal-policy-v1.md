# Contractual clean removal v1

This POC contract closes `B-004`. The runtime optimizer accepts immutable
Gradle Wrapper argv plus exact task-position evidence from a versioned model.
It removes every `clean` task only when the action is authorized and all
relevant positions resolve to allowlisted core `org.gradle.api.tasks.Delete`
tasks that delete declared outputs exclusively and have no customization,
added actions, dependencies, finalizers, side effects, or observed failure
semantics.

The workspace must either be newly created and verified empty, or have a
proven persistent lifecycle that prevents stale outputs. A CI barrier,
unavailable or incomplete model, release contract, reproducibility validation,
unknown lifecycle, unmodeled clean task, or any unsafe task contract preserves
the complete original command. Multiple clean tasks are all-or-nothing.

When clean is the only modeled task, the decision marks the invocation to be
skipped; it never runs a bare Wrapper command that could select default tasks.
The engine retains an immutable original argv and returns stable reason codes
for every preservation path. This block implements the disabled action and its
contract fixtures; it does not activate it for an owner repository or close
`B-G04`, which also requires invocation-merging coverage.

Run `./dev/check-clean-removal-policy` for the executable contract.
