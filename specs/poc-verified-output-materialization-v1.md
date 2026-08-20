# Verified unaffected-output materialization POC

## Question

Can `buildopt optimize` preserve the complete output contract of an aggregate
Gradle workflow while omitting unaffected producers in a clean workspace?

This is a bounded POC question. It does not authorize production rollout,
remote multi-tenant storage, a soak, or Test Optimization.

## Contract

Automatic discovery keeps two distinct sets:

1. `requiredOutputs` is the complete terminal output contract observed from
   the original Gradle workflow;
2. `candidateOutputs` is the subset produced by the structurally selected
   candidate entrypoints.

The difference may be captured only after a successful authoritative full
graph. Each regular repository-owned file is stored in private content-addressed
state and bound to its path, SHA-256, size, mode, repository identity, target
revision, complete output contract, and candidate output subset. Symlinks,
external paths, empty matches, unsafe parents, excessive file/byte counts and
ambiguous ownership retain native Gradle.

Before a candidate starts, BuildOpt verifies every manifest and blob. Missing
outputs are written atomically. Existing matching bytes are reused; existing
different bytes, a missing blob, a corrupt blob, a changed manifest, or any
binding drift rejects materialization without overwriting the workspace. The
ordinary invocation then executes the full native graph. Candidate completion
still hashes the complete required-output set; drift also triggers full-graph
recovery.

The initial POC stores payloads in repository-local private BuildOpt state. A
qualified profile that needs these payloads is not remotely replayable until a
later contract carries the blobs through the central state bundle; remote
absence therefore retains native Gradle rather than weakening the output
contract.

## Executable proof

[`dev/run-verified-output-materialization`](../dev/run-verified-output-materialization)
creates the existing three-project Gradle fixture and runs four useful
invocations:

1. a full-graph baseline observes and captures the complete workflow output;
2. a full-graph control contributes the first paired observation;
3. all project build directories are removed, then the structural candidate
   rebuilds the changed `service-a` output while verified state restores the
   required `library-c` and `service-b` JARs;
4. one content-addressed blob is corrupted, all build directories are removed
   again, and BuildOpt must reject the candidate before it starts and execute
   the full native graph.

The baseline, clean candidate and corruption fallback must each contain the
same three JARs with one identical aggregate digest. Unit tests separately
prove that stale workspace bytes are never overwritten and that corrupt state
writes no partial output.

Validate committed evidence with:

```bash
./dev/check-verified-output-materialization
```

This proof establishes correctness and generic orchestration only. It makes no
wall-time claim; the five-repository transfer is repeated only after aggregate
workflow partitioning is also implemented.
