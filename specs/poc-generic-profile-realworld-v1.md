# Public-repository generic profile replay

## Question

Can a fresh user-facing `buildopt profile propose` invocation rediscover the
already qualified Apache Groovy and Micronaut structural boundaries without
copying any retained BuildOpt manifest, graph, generated state or profile into
the checkout?

## Frozen subjects

The machine contract fixes Apache Groovy at `c1689831...1997e` and Micronaut
Core at `8de8f38...e882b`. Each replay starts from a shallow checkout of that
public commit, applies the same one-file semantic mutation used by the retained
experiment and declares only its already qualified output scope.

| Repository | Original workflow | Expected proposal | Historical boundary |
|---|---|---|---:|
| Apache Groovy | `classes` | `:groovy-json:classes` | 37 projects to 2 |
| Micronaut Core | `assemble` | `:micronaut-http-client-jdk:assemble` | 75 projects to 22 |

Online dependency preparation is excluded. The accepted proposal must be
regenerated offline from the installed CLI, report complete relationships,
retain global changes on the native full graph and expose no Test task.

## Decision rule

The replay qualifies only when both repositories reproduce their retained
entrypoint, required output and project counts. Manifest and graph bytes are
not expected to equal the historical hand-authored documents: the generic
command deliberately normalizes alternative IDs, artifact IDs and fallback
globs. The structural plan must be equal.

No timing is repeated when that plan is unchanged. The retained 50.06% Groovy
and 72.16% installed Micronaut results remain the value evidence; this block
tests onboarding transfer, not a more favorable percentage. A materially new
candidate would require a new preregistered paired experiment.

No generated proposal activates a profile automatically or claims production
readiness. Test Optimization, soak testing and design-partner work remain out
of scope.

## Reproduce and validate

Run the online preparation and final offline replay explicitly:

```bash
./dev/run-generic-profile-realworld \
  /absolute/path/to/poc-generic-profile-realworld-v1
```

Validate the retained evidence without network access:

```bash
./dev/check-generic-profile-realworld
```
