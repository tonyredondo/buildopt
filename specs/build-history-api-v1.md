# Build history API v1

This contract closes `UX-F1-001` with an authenticated read-only view over
the immutable `BUILD_SESSION v1` documents already published by the export
gateway. The API never reconstructs raw repository, task, or trust-domain
identities: it returns only the HMAC-redacted values stored on disk.

Set an independent read credential while the existing loopback server has an
export directory:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<write-only-ingest-token> \
BUILDOPT_HISTORY_API_TOKEN=<independent-read-token> \
  buildopt-server serve \
    --listen 127.0.0.1:8042 \
    --export-dir /private/buildopt-exports
```

List newest sessions with optional exact filters and stable cursor pagination:

```text
GET /api/v1/build-sessions?repository=TOKEN&outcome=SUCCESS&limit=25&cursor=OPAQUE
```

Resolve one exact session identity, including identities containing `/`, with:

```text
GET /api/v1/build-session?id=SESSION_ID
```

Both operations require `Authorization: Bearer ...`, emit `Cache-Control:
no-store`, reject unknown or repeated query parameters, and fail closed when a
candidate history file is permissive, oversized, malformed, trailing, or not
bound by filename to its session identity. The API is absent unless its
separate token is configured, and that token cannot ingest sessions.

Run the complete contract and real-server proof with:

```bash
./dev/check-build-history-api
```

This block supplies the data boundary for the later embedded dashboard. It
does not add a remote API, restore raw identities, select tests, or modify the
separate Test Optimization product.
