# Single-node Shared storage v1

This specification materializes `A0-004` and the storage substrate of
`STORAGE-001`. It installs immutable content-addressed blobs plus separate
WAL-mode `cache.sqlite` and `control.sqlite` lifecycles inside
`buildopt-server`. It does not expose a cache GET/PUT route, make pending
objects visible, authenticate a `CommitDecision`, or close an A0 exit gate.

## Activation and layout

The internal pilot opts in with an absolute private path:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token> \
  buildopt-server serve \
  --listen 127.0.0.1:8042 \
  --state-dir /var/lib/buildopt
```

The server owns this complete layout:

```text
<state-dir>/
  writer.lock
  blobs/sha256/<first-2-hex>/<remaining-62-hex>
  spool/
  quarantine/
  cache.sqlite
  control.sqlite
```

Every directory is current-user-owned mode `0700`; every managed file is
regular, singly linked after publication, and mode `0600`. The root, blob,
spool, and quarantine directories must be on the same local filesystem.
Only an explicit allowlist of proven-local Linux filesystem types is accepted;
network, clustered, FUSE, and unknown types fail closed. A mode-`0600`
non-blocking exclusive `writer.lock` permits one server process for the
complete root; a second process fails before opening a listener.
An existing root containing any entry outside this layout is rejected before
BuildOpt adds files, so a broad private directory cannot silently become a
storage root.

## Immutable blob boundary

The filesystem blob implementation streams at most 100 MiB into a private
same-filesystem spool file while calculating SHA-256. It checks cancellation,
syncs the complete file, publishes it at
`blobs/sha256/<first-2-hex>/<remaining-62-hex>` with a no-replacement hard
link, syncs the digest directory, removes the spool link, and syncs the spool
directory. A concurrent identical write reuses only bytes that pass a complete
size-and-digest verification. It never replaces a corrupt digest path.

`OpenVerified` hashes and sizes the complete private regular file before
returning a rewound handle. No caller receives unverified bytes. Blob presence
alone grants no read or cache-hit authority; A0-005 must first find a valid
committed metadata record.

## Independent SQLite lifecycles

Both databases use the pinned CGO-free SQLite driver, WAL journal mode,
`synchronous=FULL`, foreign keys, a 5-second busy timeout, and
`trusted_schema=OFF`. Each has a transactional schema-v1 migration with an
exact binary-owned checksum, `PRAGMA user_version`, an exact object inventory,
`PRAGMA quick_check`, and `PRAGMA foreign_key_check`.

`cache.sqlite` reserves the sole future visibility transaction:

- `commit_decisions` retains immutable canonical decision bytes;
- `committed_objects` has the first-writer key
  `(tenant_id, namespace_generation, cache_key)`;
- blob-digest and byte-weighted last-access indexes support later integrity
  reconciliation and retention.

`control.sqlite` contains only the independently repairable
`decision_audit_index`. The files never share a transaction, and this block
does not write a decision or visibility row. A0-005 will implement the
all-or-nothing `CommitDecision + COMMITTED` operation and later audit indexing
defined by ADR 0002.

An unsafe path, busy writer, unknown or drifted migration, invalid file mode,
or SQLite integrity failure prevents server startup. The server reports that
the storage substrate is initialized while leaving cache routes absent.

## Executable evidence

Run:

```bash
./dev/check-shared-storage
```

The checker validates the exact machine contract and runs race-enabled
conformance for private layout, local-filesystem classification, exclusive
ownership, WAL/migration drift, database separation and persistence, atomic
blob publication, concurrent deduplication, corruption, cancellation,
oversize cleanup, close/reopen, and the real statically linked server
lifecycle. It also reruns the implementation-independent ADR 0002 model and
the capability matrix. `A0-005` and `A0-006` now compose this substrate with
pending visibility and locally authenticated routing, while the complete A0
cache conformance gates remain open.
