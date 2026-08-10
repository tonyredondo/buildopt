# Public-repository generic profile replay evidence

This directory records a clean-checkout replay of the installed
`buildopt profile propose` workflow on Apache Groovy 5.0.8 and Micronaut Core.
Neither checkout received a retained BuildOpt manifest, graph, generated-state
binding or qualified profile.

The replay reproduced the already qualified structural plans:

| Repository | Native project reach | Proposed project reach | Omitted projects | Retained value evidence |
|---|---:|---:|---:|---:|
| Apache Groovy | 37 | 2 | 35 | 50.06% faster, 8/8 positive pairs |
| Micronaut Core | 75 | 22 | 53 | 72.16% faster, 8/8 positive pairs |

Online dependency preparation was excluded from evidence collection. The
accepted proposal pass ran offline. Because both plans are structurally equal
to the retained qualified plans, this block deliberately did not rerun timing
or create a new performance claim.

Validate every plan, binding and retained-evidence comparison without network
access:

```bash
./dev/check-generic-profile-realworld
```

The evidence remains POC-only and review-required. It grants no automatic
activation, production authority or Test Optimization behavior.
