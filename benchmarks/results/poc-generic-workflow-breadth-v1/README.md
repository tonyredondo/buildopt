# Generic workflow breadth evidence

This directory preserves the terminal hosted capability result for the generic
owner-input workflow matrix.

- BuildOpt revision: `0c1b64fef5e210526a41d487f4ed4cea865c4777`
- Hosted run: [31598631537](https://github.com/tonyredondo/buildopt/actions/runs/31598631537)
- Supported cells: packaging, typed verification, distribution, and
  build-owned test preparation
- Result: 4/4 exact structural candidates with byte-identical declared outputs
- Fallback: 1/1 unsupported executable workflow retained native before timing
- Gradle `Test` tasks executed: 0
- Performance observations: 0

Validate the preserved result from the repository root:

```bash
./dev/check-generic-workflow-breadth-result \
  benchmarks/results/poc-generic-workflow-breadth-v1/workflow-breadth.json
```

This result proves capability and fail-closed behavior. It does not prove that
any workflow family is faster than optimized native Gradle.
