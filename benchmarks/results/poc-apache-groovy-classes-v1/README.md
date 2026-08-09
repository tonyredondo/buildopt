# Apache Groovy classes evidence bundle

This directory preserves the complete input/output handoff for the qualified
Apache Groovy 5.0.8 `groovy-json` classes experiment.

| File | Purpose |
|---|---|
| `buildopt-impact-manifest.json` | Repository-owned original/candidate task and output declaration |
| `buildopt-impact-graph.generated.json` | Generic 37-project discovered graph and direct source ownership |
| `buildopt-impact.generated.json` | Binding between the manifest and generated graph |
| `changes.txt` | Exact base-to-target source change used by the candidate |
| `fallback-changes.txt` | Global change that must restore the full graph |
| `measurement.json` | Eight alternating optimized-native versus installed-BuildOpt pairs |
| `qualified-profile.json` | Review-required v4 profile derived from the exact evidence |

The result is 50.06% mean savings with 8/8 positive pairs and 66
byte-identical class outputs. It qualifies only the declared classes scope and
does not authorize automatic or production activation.

From the repository root, validate the bundle with:

```bash
./dev/check-poc-apache-groovy-classes-v1
```
