# Verified unaffected-output materialization result

The bounded local POC completed on 2026-08-20 with the implementation commit
`629fb711d2e4d61a3b1ca86469041da29c96a4ac`.

The three-project `assemble` workflow produced three JARs. In a workspace where
all project build directories had been removed, the structural candidate
rebuilt the changed `service-a` output and BuildOpt materialized two verified
unaffected outputs: the required `library-c` and `service-b` JARs. The complete
three-file output manifest matched the full-graph baseline exactly.

The negative path then corrupted one content-addressed blob and cleaned the
workspace again. BuildOpt rejected materialization before the candidate
started, ran the full native graph, returned exit code zero and reproduced the
same complete output digest. Unit coverage also proves that stale workspace
bytes are not overwritten and a corrupt blob writes no partial output.

| Observation | Result |
|---|---:|
| Original graph | 3 projects |
| Candidate graph | 2 selected / 1 omitted |
| Complete required outputs | 3 JARs |
| Verified materialized outputs | 2 JARs / 1,466 bytes |
| Baseline vs clean candidate | Identical |
| Baseline vs corruption fallback | Identical |
| Product-attributable failure | 0 |

This is correctness evidence, not a speed claim. The next block partitions
aggregate workflows generically; only after that will the unchanged public
five-repository transfer be repeated under the existing wall-time and payback
gates.

Validate the machine-readable evidence with:

```bash
./dev/check-verified-output-materialization
```
