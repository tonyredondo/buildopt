package sharedcache

// StateSchemaVersion is the independent typed BuildOpt-state schema version.
const StateSchemaVersion = 2

func stateMetadataDefinition(path string) metadataDefinition {
	versionOne := schemaMigration{
		version: 1,
		name:    "typed-state-v1",
		statements: []string{
			`CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY CHECK (version > 0),
    name TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL CHECK (length(checksum) = 64),
    applied_at_unix_ms INTEGER NOT NULL CHECK (applied_at_unix_ms >= 0)
)`,
			`CREATE TABLE state_objects (
    repository_scope_sha256 TEXT NOT NULL CHECK (
        length(repository_scope_sha256) = 64
        AND repository_scope_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    kind TEXT NOT NULL CHECK (kind IN ('PORTFOLIO', 'EVIDENCE', 'CHECKPOINT')),
    blob_digest TEXT NOT NULL CHECK (
        length(blob_digest) = 71
        AND substr(blob_digest, 1, 7) = 'sha256:'
        AND substr(blob_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 16777216),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    PRIMARY KEY (repository_scope_sha256, kind, blob_digest)
) WITHOUT ROWID`,
			`CREATE INDEX state_objects_created_at
ON state_objects (created_at_unix_ms, repository_scope_sha256, kind)`,
			`CREATE TABLE state_manifests (
    repository_scope_sha256 TEXT NOT NULL CHECK (
        length(repository_scope_sha256) = 64
        AND repository_scope_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    kind TEXT NOT NULL CHECK (kind IN ('PORTFOLIO', 'EVIDENCE', 'CHECKPOINT')),
    generation INTEGER NOT NULL CHECK (generation > 0),
    manifest_digest TEXT NOT NULL CHECK (
        length(manifest_digest) = 71
        AND substr(manifest_digest, 1, 7) = 'sha256:'
        AND substr(manifest_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    canonical_document BLOB NOT NULL CHECK (length(canonical_document) BETWEEN 1 AND 1048576),
    compatibility_sha256 TEXT NOT NULL CHECK (
        length(compatibility_sha256) = 64
        AND compatibility_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    bindings_sha256 TEXT NOT NULL CHECK (
        length(bindings_sha256) = 64
        AND bindings_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    status TEXT NOT NULL CHECK (status IN ('COMPLETE', 'RESUMABLE')),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    expires_at_unix_ms INTEGER,
    published_at_unix_ms INTEGER,
    retention_started_at_unix_ms INTEGER,
    PRIMARY KEY (repository_scope_sha256, kind, generation),
    UNIQUE (repository_scope_sha256, kind, manifest_digest),
    CHECK (expires_at_unix_ms IS NULL OR expires_at_unix_ms > created_at_unix_ms),
    CHECK (published_at_unix_ms IS NULL OR published_at_unix_ms >= created_at_unix_ms),
    CHECK (retention_started_at_unix_ms IS NULL OR published_at_unix_ms IS NOT NULL)
) WITHOUT ROWID`,
			`CREATE INDEX state_manifests_retention
ON state_manifests (
    kind, published_at_unix_ms, retention_started_at_unix_ms,
    expires_at_unix_ms, created_at_unix_ms
)`,
			`CREATE TABLE state_manifest_artifacts (
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL,
    manifest_digest TEXT NOT NULL,
    role TEXT NOT NULL CHECK (length(role) BETWEEN 1 AND 64),
    blob_digest TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 16777216),
    PRIMARY KEY (repository_scope_sha256, kind, manifest_digest, role, blob_digest),
    FOREIGN KEY (repository_scope_sha256, kind, manifest_digest)
        REFERENCES state_manifests(repository_scope_sha256, kind, manifest_digest)
        ON DELETE CASCADE,
    FOREIGN KEY (repository_scope_sha256, kind, blob_digest)
        REFERENCES state_objects(repository_scope_sha256, kind, blob_digest)
) WITHOUT ROWID`,
			`CREATE TABLE state_manifest_references (
    repository_scope_sha256 TEXT NOT NULL,
    source_kind TEXT NOT NULL,
    source_manifest_digest TEXT NOT NULL,
    target_kind TEXT NOT NULL CHECK (target_kind = 'EVIDENCE'),
    target_manifest_digest TEXT NOT NULL,
    relation TEXT NOT NULL CHECK (relation = 'QUALIFICATION'),
    PRIMARY KEY (
        repository_scope_sha256, source_kind, source_manifest_digest,
        target_kind, target_manifest_digest, relation
    ),
    FOREIGN KEY (repository_scope_sha256, source_kind, source_manifest_digest)
        REFERENCES state_manifests(repository_scope_sha256, kind, manifest_digest)
        ON DELETE CASCADE,
    FOREIGN KEY (repository_scope_sha256, target_kind, target_manifest_digest)
        REFERENCES state_manifests(repository_scope_sha256, kind, manifest_digest)
) WITHOUT ROWID`,
			`CREATE INDEX state_manifest_references_target
ON state_manifest_references (
    repository_scope_sha256, target_kind, target_manifest_digest
)`,
			`CREATE TABLE state_heads (
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL,
    generation INTEGER NOT NULL CHECK (generation > 0),
    head_digest TEXT NOT NULL CHECK (
        length(head_digest) = 71
        AND substr(head_digest, 1, 7) = 'sha256:'
        AND substr(head_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    canonical_document BLOB NOT NULL CHECK (length(canonical_document) BETWEEN 1 AND 1048576),
    manifest_digest TEXT NOT NULL,
    compatibility_sha256 TEXT NOT NULL CHECK (length(compatibility_sha256) = 64),
    updated_at_unix_ms INTEGER NOT NULL CHECK (updated_at_unix_ms >= 0),
    PRIMARY KEY (repository_scope_sha256, kind),
    UNIQUE (repository_scope_sha256, kind, head_digest),
    FOREIGN KEY (repository_scope_sha256, kind, manifest_digest)
        REFERENCES state_manifests(repository_scope_sha256, kind, manifest_digest)
) WITHOUT ROWID`,
			`CREATE TABLE state_cas_requests (
    idempotency_key TEXT PRIMARY KEY CHECK (
        length(idempotency_key) = 64
        AND idempotency_key NOT GLOB '*[^0-9a-f]*'
    ),
    request_digest TEXT NOT NULL CHECK (
        length(request_digest) = 71
        AND substr(request_digest, 1, 7) = 'sha256:'
        AND substr(request_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL,
    generation INTEGER NOT NULL CHECK (generation > 0),
    head_digest TEXT NOT NULL,
    head_canonical_document BLOB NOT NULL CHECK (length(head_canonical_document) BETWEEN 1 AND 1048576),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    CHECK (kind IN ('PORTFOLIO', 'EVIDENCE', 'CHECKPOINT'))
) WITHOUT ROWID`,
			`CREATE INDEX state_cas_requests_created_at
ON state_cas_requests (created_at_unix_ms)`,
		},
	}
	definition := metadataDefinition{
		role:       stateMetadataRole,
		path:       path,
		migrations: []schemaMigration{versionOne, versionTwoWCNCP},
	}
	definition.objects = []schemaObject{
		{objectType: "index", name: "state_cas_requests_created_at", statement: versionOne.statements[10]},
		{objectType: "index", name: "state_manifest_references_target", statement: versionOne.statements[7]},
		{objectType: "index", name: "state_manifests_retention", statement: versionOne.statements[4]},
		{objectType: "index", name: "state_objects_created_at", statement: versionOne.statements[2]},
		{objectType: "index", name: "wcncp_cas_requests_created_at", statement: versionTwoWCNCP.statements[9]},
		{objectType: "index", name: "wcncp_manifest_references_target", statement: versionTwoWCNCP.statements[6]},
		{objectType: "index", name: "wcncp_manifests_retention", statement: versionTwoWCNCP.statements[3]},
		{objectType: "index", name: "wcncp_objects_created_at", statement: versionTwoWCNCP.statements[1]},
		{objectType: "table", name: "schema_migrations", statement: versionOne.statements[0]},
		{objectType: "table", name: "state_cas_requests", statement: versionOne.statements[9]},
		{objectType: "table", name: "state_heads", statement: versionOne.statements[8]},
		{objectType: "table", name: "state_manifest_artifacts", statement: versionOne.statements[5]},
		{objectType: "table", name: "state_manifest_references", statement: versionOne.statements[6]},
		{objectType: "table", name: "state_manifests", statement: versionOne.statements[3]},
		{objectType: "table", name: "state_objects", statement: versionOne.statements[1]},
		{objectType: "table", name: "wcncp_cas_requests", statement: versionTwoWCNCP.statements[8]},
		{objectType: "table", name: "wcncp_heads", statement: versionTwoWCNCP.statements[7]},
		{objectType: "table", name: "wcncp_manifest_artifacts", statement: versionTwoWCNCP.statements[4]},
		{objectType: "table", name: "wcncp_manifest_references", statement: versionTwoWCNCP.statements[5]},
		{objectType: "table", name: "wcncp_manifests", statement: versionTwoWCNCP.statements[2]},
		{objectType: "table", name: "wcncp_objects", statement: versionTwoWCNCP.statements[0]},
	}
	return definition
}

// versionTwoWCNCP adds the Wrapper-Coordinated Native Corrections control
// plane. It shares the physical CAS blobs with the Gradle data plane and the
// existing typed state, but uses independent metadata tables so no Gradle
// cache key, existing state kind, or WCNCP kind can address another
// namespace. WCNCP-001 owns persistence only; no route or wrapper depends on
// these tables yet.
var versionTwoWCNCP = schemaMigration{
	version: 2,
	name:    "wcncp-typed-state-v1",
	statements: []string{
		`CREATE TABLE wcncp_objects (
    repository_scope_sha256 TEXT NOT NULL CHECK (
        length(repository_scope_sha256) = 64
        AND repository_scope_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    kind TEXT NOT NULL CHECK (kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    blob_digest TEXT NOT NULL CHECK (
        length(blob_digest) = 71
        AND substr(blob_digest, 1, 7) = 'sha256:'
        AND substr(blob_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 16777216),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    PRIMARY KEY (repository_scope_sha256, kind, blob_digest)
) WITHOUT ROWID`,
		`CREATE INDEX wcncp_objects_created_at
ON wcncp_objects (created_at_unix_ms, repository_scope_sha256, kind)`,
		`CREATE TABLE wcncp_manifests (
    repository_scope_sha256 TEXT NOT NULL CHECK (
        length(repository_scope_sha256) = 64
        AND repository_scope_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    kind TEXT NOT NULL CHECK (kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    generation INTEGER NOT NULL CHECK (generation > 0),
    manifest_digest TEXT NOT NULL CHECK (
        length(manifest_digest) = 71
        AND substr(manifest_digest, 1, 7) = 'sha256:'
        AND substr(manifest_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    canonical_document BLOB NOT NULL CHECK (length(canonical_document) BETWEEN 1 AND 1048576),
    compatibility_sha256 TEXT NOT NULL CHECK (
        length(compatibility_sha256) = 64
        AND compatibility_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    bindings_sha256 TEXT NOT NULL CHECK (
        length(bindings_sha256) = 64
        AND bindings_sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    status TEXT NOT NULL CHECK (status IN ('COMPLETE', 'RESUMABLE')),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    expires_at_unix_ms INTEGER,
    published_at_unix_ms INTEGER,
    retention_started_at_unix_ms INTEGER,
    PRIMARY KEY (repository_scope_sha256, kind, generation),
    UNIQUE (repository_scope_sha256, kind, manifest_digest),
    CHECK (expires_at_unix_ms IS NULL OR expires_at_unix_ms > created_at_unix_ms),
    CHECK (published_at_unix_ms IS NULL OR published_at_unix_ms >= created_at_unix_ms),
    CHECK (retention_started_at_unix_ms IS NULL OR published_at_unix_ms IS NOT NULL)
) WITHOUT ROWID`,
		`CREATE INDEX wcncp_manifests_retention
ON wcncp_manifests (
    kind, published_at_unix_ms, retention_started_at_unix_ms,
    expires_at_unix_ms, created_at_unix_ms
)`,
		`CREATE TABLE wcncp_manifest_artifacts (
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL,
    manifest_digest TEXT NOT NULL,
    role TEXT NOT NULL CHECK (length(role) BETWEEN 1 AND 64),
    blob_digest TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 16777216),
    PRIMARY KEY (repository_scope_sha256, kind, manifest_digest, role, blob_digest),
    FOREIGN KEY (repository_scope_sha256, kind, manifest_digest)
        REFERENCES wcncp_manifests(repository_scope_sha256, kind, manifest_digest)
        ON DELETE CASCADE,
    FOREIGN KEY (repository_scope_sha256, kind, blob_digest)
        REFERENCES wcncp_objects(repository_scope_sha256, kind, blob_digest)
) WITHOUT ROWID`,
		`CREATE TABLE wcncp_manifest_references (
    repository_scope_sha256 TEXT NOT NULL,
    source_kind TEXT NOT NULL CHECK (source_kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    source_manifest_digest TEXT NOT NULL,
    target_kind TEXT NOT NULL CHECK (target_kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    target_manifest_digest TEXT NOT NULL,
    relation TEXT NOT NULL CHECK (relation IN ('DERIVED_FROM', 'QUALIFICATION', 'VALIDATES', 'DECIDES')),
    PRIMARY KEY (
        repository_scope_sha256, source_kind, source_manifest_digest,
        target_kind, target_manifest_digest, relation
    ),
    FOREIGN KEY (repository_scope_sha256, source_kind, source_manifest_digest)
        REFERENCES wcncp_manifests(repository_scope_sha256, kind, manifest_digest)
        ON DELETE CASCADE,
    FOREIGN KEY (repository_scope_sha256, target_kind, target_manifest_digest)
        REFERENCES wcncp_manifests(repository_scope_sha256, kind, manifest_digest)
) WITHOUT ROWID`,
		`CREATE INDEX wcncp_manifest_references_target
ON wcncp_manifest_references (
    repository_scope_sha256, target_kind, target_manifest_digest
)`,
		`CREATE TABLE wcncp_heads (
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    generation INTEGER NOT NULL CHECK (generation > 0),
    head_digest TEXT NOT NULL CHECK (
        length(head_digest) = 71
        AND substr(head_digest, 1, 7) = 'sha256:'
        AND substr(head_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    canonical_document BLOB NOT NULL CHECK (length(canonical_document) BETWEEN 1 AND 1048576),
    manifest_digest TEXT NOT NULL,
    compatibility_sha256 TEXT NOT NULL CHECK (length(compatibility_sha256) = 64),
    updated_at_unix_ms INTEGER NOT NULL CHECK (updated_at_unix_ms >= 0),
    PRIMARY KEY (repository_scope_sha256, kind),
    UNIQUE (repository_scope_sha256, kind, head_digest),
    FOREIGN KEY (repository_scope_sha256, kind, manifest_digest)
        REFERENCES wcncp_manifests(repository_scope_sha256, kind, manifest_digest)
) WITHOUT ROWID`,
		`CREATE TABLE wcncp_cas_requests (
    idempotency_key TEXT PRIMARY KEY CHECK (
        length(idempotency_key) = 64
        AND idempotency_key NOT GLOB '*[^0-9a-f]*'
    ),
    request_digest TEXT NOT NULL CHECK (
        length(request_digest) = 71
        AND substr(request_digest, 1, 7) = 'sha256:'
        AND substr(request_digest, 8) NOT GLOB '*[^0-9a-f]*'
    ),
    repository_scope_sha256 TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION')),
    generation INTEGER NOT NULL CHECK (generation > 0),
    head_digest TEXT NOT NULL,
    head_canonical_document BLOB NOT NULL CHECK (length(head_canonical_document) BETWEEN 1 AND 1048576),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    CHECK (kind IN ('WCNCP_OBSERVATION', 'WCNCP_OPPORTUNITY', 'WCNCP_PROPOSAL', 'WCNCP_VALIDATION', 'WCNCP_DECISION'))
) WITHOUT ROWID`,
		`CREATE INDEX wcncp_cas_requests_created_at
ON wcncp_cas_requests (created_at_unix_ms)`,
	},
}
