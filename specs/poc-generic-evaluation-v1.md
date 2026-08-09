# Generic POC evaluation decision

This contract makes structural Build Impact evaluation a single installed
decision without pretending that Gradle can infer a customer's required
outputs. The command is:

```bash
buildopt profile evaluate \
  --manifest buildopt-impact-manifest.json \
  --graph buildopt-impact-graph.generated.json \
  --generated-manifest buildopt-impact.generated.json
```

With complete, known, non-Test graph state, it emits
`MEASURE_STRUCTURAL_CANDIDATE`. It does not estimate savings or activate the
candidate. Required artifacts and original/alternative entrypoints remain a
small repository-owned contract because a fast build that omits an output the
caller needs is incorrect.

After the exact installed-path paired measurement is available, the same
command accepts `--evidence` and `--profile-output`. It independently
recomputes the eight-pair result, output equality, positive interval and native
fallback. Only qualified evidence is written atomically as a digest-bound v4
profile. Weak, inconsistent, drifted or invalid evidence emits
`NATIVE_FULL_GRAPH` and writes no profile.

The command contains no repository-name switch, does not run a parameter
search, and never activates the profile automatically. It is POC workflow
compression around the existing conservative analysis and qualification
contracts. Performance measurement remains separately isolated so control and
candidate Gradle state cannot contaminate one another.

Validate the command with:

```bash
./dev/check-poc-generic-evaluation
```
