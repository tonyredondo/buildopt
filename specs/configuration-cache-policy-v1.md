# Configuration Cache policy v1

This POC contract closes `B-002`. A compatible policy starts only in a
noncritical CI cohort and uses Gradle's strict problem mode. A persistent
workspace that preserves its encryption key enters `CI_CANARY`; an ephemeral
workspace may exercise compatibility but is labelled `COMPATIBILITY_ONLY` and
never claims savings. Promotion requires the configured number of idempotently identified, correct,
natural entry reuses. Entry creation alone and synthetic repetition do not
count.

An attributable failure immediately moves the rollout to `SUSPENDED`, disables
Configuration Cache, increments the invalidation generation, and changes the
configuration-policy digest. Promotion itself does not change that digest, so
rollout metadata cannot destroy an otherwise compatible local hit. Warning
mode is rejected rather than retained as permanent compatibility policy.

The distributable decision contains only enablement, contract version,
configuration-policy digest, and invalidation generation. A local identity
binds repository, trust domain, host, absolute workspace, Gradle version,
encryption strategy, contract, digest, and generation to the in-place
`.gradle/configuration-cache` directory. Neither that path, entry bytes, nor an
encryption key is serialized into the decision or sent through Shared Cache.

For an authenticated Gradle Wrapper invocation, the launcher replaces any
caller Configuration Cache toggle with the signed decision. Enabled policies
inject `--configuration-cache`, strict `--configuration-cache-problems=fail`,
the configuration-policy digest, and the contract version. Disabled or
suspended policies inject `--no-configuration-cache` plus the same versioned
identity inputs. Non-Gradle commands remain unchanged.

Run `./dev/check-configuration-cache-policy` for the executable contract.
