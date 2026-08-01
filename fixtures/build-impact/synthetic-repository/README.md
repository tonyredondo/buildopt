# Build Impact synthetic repository

This owner-controlled fixture is a three-project Gradle repository used only by
the C3-005 proof. `service-a` depends on `library-c`; `service-b` is
independent. The customer manifest authorizes `:service-a:assemble` as the one
alternative that preserves its required JAR and Build-owned compile check.

The declared graph maps a library-c source change through its reverse dependent
service-a. The selected build therefore omits only service-b. The
`testOwnedCheck` task models a Test Optimization-owned check and is executed
separately in both workspaces; it is not part of the Build Impact-selected
entrypoints.

The checker copies this directory to temporary isolated workspaces and runs the
repository wrapper with `--offline`. No generated output is written here and
the fixture makes no timing, soak, external-user, or production claim.
