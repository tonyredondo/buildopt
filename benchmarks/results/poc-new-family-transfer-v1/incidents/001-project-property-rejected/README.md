# Rejected pre-measurement attempt: owner project property

The first execution of the committed new-family runner stopped before Ktor's
owner `assemble` workflow, BuildOpt proposal discovery, warm-up or timing. The
generic CLI rejected the first preregistered `-Ptarget.posix=false` Gradle
project property and returned usage code 64.

The failure exposed a repository-independent integration gap: reviewed Gradle
workflows can require ordinary project properties, while BuildOpt accepted
only a fixed execution-option allowlist. The correction accepts bounded
`-Pname=value` arguments without permitting task selectors, exclusions,
project-root changes or init scripts. No observation from this failed attempt
is reused; both preregistered captures restart from fresh checkouts after the
generic correction is committed.

The exact emitted usage line is the `buildopt profile propose` usage documented
by the CLI at implementation revision `1102b06`; no partial structural
document or duration was produced.
