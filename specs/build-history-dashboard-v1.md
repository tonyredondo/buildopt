# Build history dashboard v1

This contract closes `UX-F1-002` with a dependency-free interface embedded in
`buildopt-server`. When the history API is configured, open:

```text
http://127.0.0.1:8042/buildopt/
```

The page asks for the independent `BUILDOPT_HISTORY_API_TOKEN` and keeps it
only in JavaScript memory for the lifetime of that page. It does not put the
credential in the URL, HTML, local storage, session storage, cookies, or any
external request. Forgetting the token clears all loaded data.

The dashboard exposes only facts returned by the existing redacted history
API:

- counts and median duration describe the currently loaded rows, while the
  matching count comes from the active exact filters;
- repository and outcome filters are server-side and exact;
- session/revision search is explicitly limited to loaded rows;
- selecting a row loads immutable outcome, timing, workload, measurement,
  requested-work, and recovery facts;
- no chart or percentage invents longitudinal, cache-hit, or optimization
  evidence that `BUILD_SESSION v1` does not contain.

The UI includes authentication, loading, ready, empty, error, detail-loading,
and detail-error states. Semantic landmarks, labeled controls, keyboard
buttons, visible focus, live status, responsive layouts, reduced-motion
handling, and a WCAG AA contrast target are part of the contract.

Static resources are embedded in the Go binary and served with no-store,
same-origin-only CSP, frame denial, and no-referrer headers. The dashboard
performs no remote request and remains absent when the history token is not
configured.

Run the executable contract with:

```bash
./dev/check-build-history-dashboard
```

This interface remains local and read-only. It does not recover raw identities,
mutate exports, add analytics not present in the source documents, or implement
any Test Optimization behavior.
