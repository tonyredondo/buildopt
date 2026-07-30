package sharedcache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	cacheMetadataRole   = "CACHE_VISIBILITY"
	controlMetadataRole = "CONTROL_AUDIT_INDEX"
)

type metadataDefinition struct {
	role       string
	path       string
	migrations []schemaMigration
	objects    []schemaObject
}

type schemaMigration struct {
	version    int
	name       string
	statements []string
}

type schemaObject struct {
	objectType string
	name       string
	statement  string
}

type sqliteMetadata struct {
	owner      *Storage
	definition metadataDefinition
	database   *sql.DB
}

var _ MetadataStore = (*sqliteMetadata)(nil)

func cacheMetadataDefinition(path string) metadataDefinition {
	versionOne := schemaMigration{
		version: 1,
		name:    "cache-v1",
		statements: []string{
			`CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY CHECK (version > 0),
    name TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL CHECK (length(checksum) = 64),
    applied_at_unix_ms INTEGER NOT NULL CHECK (applied_at_unix_ms >= 0)
)`,
			`CREATE TABLE commit_decisions (
    decision_digest TEXT PRIMARY KEY
        CHECK (
            length(decision_digest) = 71
            AND substr(decision_digest, 1, 7) = 'sha256:'
            AND substr(decision_digest, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    attempt_id TEXT NOT NULL UNIQUE CHECK (length(attempt_id) BETWEEN 1 AND 256),
    canonical_document BLOB NOT NULL,
    committed_at_unix_ms INTEGER NOT NULL CHECK (committed_at_unix_ms >= 0)
)`,
			`CREATE TABLE committed_objects (
    tenant_id TEXT NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 256),
    namespace_generation INTEGER NOT NULL CHECK (namespace_generation >= 0),
    cache_key TEXT NOT NULL CHECK (length(cache_key) BETWEEN 1 AND 512),
    blob_digest TEXT NOT NULL
        CHECK (
            length(blob_digest) = 71
            AND substr(blob_digest, 1, 7) = 'sha256:'
            AND substr(blob_digest, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0 AND size_bytes <= 104857600),
    decision_digest TEXT NOT NULL REFERENCES commit_decisions(decision_digest),
    committed_at_unix_ms INTEGER NOT NULL CHECK (committed_at_unix_ms >= 0),
    last_access_unix_ms INTEGER NOT NULL CHECK (last_access_unix_ms >= committed_at_unix_ms),
    PRIMARY KEY (tenant_id, namespace_generation, cache_key)
) WITHOUT ROWID`,
			`CREATE INDEX committed_objects_blob_digest
ON committed_objects (blob_digest)`,
			`CREATE INDEX committed_objects_last_access
ON committed_objects (last_access_unix_ms, size_bytes)`,
		},
	}
	versionTwo := schemaMigration{
		version: 2,
		name:    "cache-v2",
		statements: []string{
			`CREATE TABLE cache_attempts (
    attempt_id TEXT PRIMARY KEY CHECK (length(attempt_id) BETWEEN 1 AND 256),
    request_fingerprint TEXT NOT NULL
        CHECK (
            length(request_fingerprint) = 71
            AND substr(request_fingerprint, 1, 7) = 'sha256:'
            AND substr(request_fingerprint, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    tenant_id TEXT NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 256),
    repository_id TEXT NOT NULL CHECK (length(repository_id) BETWEEN 1 AND 256),
    trust_domain TEXT NOT NULL CHECK (length(trust_domain) BETWEEN 1 AND 256),
    namespace_generation INTEGER NOT NULL CHECK (namespace_generation > 0),
    source_revision TEXT NOT NULL CHECK (length(source_revision) BETWEEN 7 AND 64),
    source_state_digest TEXT NOT NULL CHECK (length(source_state_digest) BETWEEN 71 AND 76),
    policy_digest TEXT NOT NULL CHECK (length(policy_digest) = 71),
    configuration_policy_digest TEXT NOT NULL CHECK (length(configuration_policy_digest) = 71),
    cache_contract_digest TEXT NOT NULL CHECK (length(cache_contract_digest) = 71),
    owner_id TEXT NOT NULL CHECK (length(owner_id) BETWEEN 1 AND 256),
    lease_id TEXT NOT NULL CHECK (length(lease_id) BETWEEN 1 AND 256),
    lease_expires_at_unix_ms INTEGER NOT NULL CHECK (lease_expires_at_unix_ms >= 0),
    state TEXT NOT NULL CHECK (state IN ('PENDING', 'COMMITTED', 'ABORTED')),
    state_version INTEGER NOT NULL CHECK (state_version > 0),
    terminal_id TEXT,
    terminal_fingerprint TEXT,
    decision_digest TEXT,
    abort_reason TEXT,
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    updated_at_unix_ms INTEGER NOT NULL CHECK (updated_at_unix_ms >= created_at_unix_ms),
    CHECK (
        (state = 'PENDING' AND terminal_id IS NULL AND terminal_fingerprint IS NULL
            AND decision_digest IS NULL AND abort_reason IS NULL)
        OR (state = 'COMMITTED' AND terminal_id IS NOT NULL
            AND terminal_fingerprint IS NOT NULL AND decision_digest IS NOT NULL
            AND abort_reason IS NULL)
        OR (state = 'ABORTED' AND terminal_id IS NOT NULL
            AND terminal_fingerprint IS NOT NULL AND decision_digest IS NULL
            AND abort_reason IS NOT NULL)
    )
)`,
			`CREATE TABLE pending_objects (
    attempt_id TEXT NOT NULL REFERENCES cache_attempts(attempt_id) ON DELETE CASCADE,
    cache_key TEXT NOT NULL CHECK (length(cache_key) BETWEEN 1 AND 256),
    blob_digest TEXT NOT NULL
        CHECK (
            length(blob_digest) = 71
            AND substr(blob_digest, 1, 7) = 'sha256:'
            AND substr(blob_digest, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0 AND size_bytes <= 104857600),
    created_at_unix_ms INTEGER NOT NULL CHECK (created_at_unix_ms >= 0),
    PRIMARY KEY (attempt_id, cache_key)
) WITHOUT ROWID`,
			`CREATE TABLE quarantine_records (
    record_id INTEGER PRIMARY KEY,
    decision_digest TEXT,
    attempt_id TEXT,
    blob_digest TEXT NOT NULL
        CHECK (
            length(blob_digest) = 71
            AND substr(blob_digest, 1, 7) = 'sha256:'
            AND substr(blob_digest, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0 AND size_bytes <= 104857600),
    reason TEXT NOT NULL CHECK (reason IN ('MISSING', 'CORRUPT')),
    quarantined_at_unix_ms INTEGER NOT NULL CHECK (quarantined_at_unix_ms >= 0)
)`,
			`CREATE INDEX cache_attempts_lease_state
ON cache_attempts (state, lease_expires_at_unix_ms)`,
			`CREATE INDEX pending_objects_blob_digest
ON pending_objects (blob_digest)`,
			`CREATE INDEX quarantine_records_digest
ON quarantine_records (blob_digest, quarantined_at_unix_ms)`,
		},
	}
	versionThree := schemaMigration{
		version: 3,
		name:    "cache-v3",
		statements: []string{
			`CREATE TABLE attempt_authorities (
    attempt_id TEXT PRIMARY KEY
        REFERENCES cache_attempts(attempt_id) ON DELETE CASCADE,
    authority_digest TEXT NOT NULL
        CHECK (
            length(authority_digest) = 71
            AND substr(authority_digest, 1, 7) = 'sha256:'
            AND substr(authority_digest, 8) NOT GLOB '*[^0-9a-f]*'
        )
) WITHOUT ROWID`,
			`CREATE INDEX attempt_authorities_authority_digest
ON attempt_authorities (authority_digest)`,
		},
	}
	definition := metadataDefinition{
		role: cacheMetadataRole,
		path: path,
		migrations: []schemaMigration{
			versionOne,
			versionTwo,
			versionThree,
		},
	}
	definition.objects = []schemaObject{
		{
			objectType: "index",
			name:       "attempt_authorities_authority_digest",
			statement:  versionThree.statements[1],
		},
		{
			objectType: "index",
			name:       "cache_attempts_lease_state",
			statement:  versionTwo.statements[3],
		},
		{
			objectType: "index",
			name:       "committed_objects_blob_digest",
			statement:  versionOne.statements[3],
		},
		{
			objectType: "index",
			name:       "committed_objects_last_access",
			statement:  versionOne.statements[4],
		},
		{
			objectType: "index",
			name:       "pending_objects_blob_digest",
			statement:  versionTwo.statements[4],
		},
		{
			objectType: "index",
			name:       "quarantine_records_digest",
			statement:  versionTwo.statements[5],
		},
		{
			objectType: "table",
			name:       "attempt_authorities",
			statement:  versionThree.statements[0],
		},
		{
			objectType: "table",
			name:       "cache_attempts",
			statement:  versionTwo.statements[0],
		},
		{
			objectType: "table",
			name:       "commit_decisions",
			statement:  versionOne.statements[1],
		},
		{
			objectType: "table",
			name:       "committed_objects",
			statement:  versionOne.statements[2],
		},
		{
			objectType: "table",
			name:       "pending_objects",
			statement:  versionTwo.statements[1],
		},
		{
			objectType: "table",
			name:       "quarantine_records",
			statement:  versionTwo.statements[2],
		},
		{
			objectType: "table",
			name:       "schema_migrations",
			statement:  versionOne.statements[0],
		},
	}
	return definition
}

func controlMetadataDefinition(path string) metadataDefinition {
	versionOne := schemaMigration{
		version: 1,
		name:    "control-v1",
		statements: []string{
			`CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY CHECK (version > 0),
    name TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL CHECK (length(checksum) = 64),
    applied_at_unix_ms INTEGER NOT NULL CHECK (applied_at_unix_ms >= 0)
)`,
			`CREATE TABLE decision_audit_index (
    decision_digest TEXT PRIMARY KEY
        CHECK (
            length(decision_digest) = 71
            AND substr(decision_digest, 1, 7) = 'sha256:'
            AND substr(decision_digest, 8) NOT GLOB '*[^0-9a-f]*'
        ),
    indexed_at_unix_ms INTEGER NOT NULL CHECK (indexed_at_unix_ms >= 0)
)`,
			`CREATE INDEX decision_audit_index_indexed_at
ON decision_audit_index (indexed_at_unix_ms)`,
		},
	}
	versionTwo := schemaMigration{
		version: 2,
		name:    "control-v2",
		statements: []string{
			`CREATE TABLE reconciliation_runs (
    run_id INTEGER PRIMARY KEY,
    started_at_unix_ms INTEGER NOT NULL CHECK (started_at_unix_ms >= 0),
    completed_at_unix_ms INTEGER NOT NULL CHECK (completed_at_unix_ms >= started_at_unix_ms),
    expired_attempts INTEGER NOT NULL CHECK (expired_attempts >= 0),
    invalidated_decisions INTEGER NOT NULL CHECK (invalidated_decisions >= 0),
    quarantined_blobs INTEGER NOT NULL CHECK (quarantined_blobs >= 0),
    deleted_orphan_blobs INTEGER NOT NULL CHECK (deleted_orphan_blobs >= 0),
    repaired_audit_rows INTEGER NOT NULL CHECK (repaired_audit_rows >= 0),
    status TEXT NOT NULL CHECK (status = 'COMPLETE')
)`,
			`CREATE INDEX reconciliation_runs_completed_at
ON reconciliation_runs (completed_at_unix_ms)`,
		},
	}
	versionThree := schemaMigration{
		version: 3,
		name:    "control-v3",
		statements: []string{
			`CREATE TABLE local_authority_state (
    scope_digest TEXT PRIMARY KEY CHECK (length(scope_digest) = 64),
    tenant_id TEXT NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 256),
    repository_id TEXT NOT NULL CHECK (length(repository_id) BETWEEN 1 AND 256),
    trust_domain TEXT NOT NULL CHECK (length(trust_domain) BETWEEN 1 AND 256),
    policy_id TEXT NOT NULL CHECK (length(policy_id) BETWEEN 1 AND 256),
    policy_version INTEGER NOT NULL CHECK (policy_version > 0),
    policy_digest TEXT NOT NULL CHECK (length(policy_digest) = 71),
    configuration_policy_digest TEXT NOT NULL CHECK (length(configuration_policy_digest) = 71),
    revocation_epoch INTEGER NOT NULL CHECK (revocation_epoch > 0),
    revocation_digest TEXT NOT NULL CHECK (length(revocation_digest) = 71),
    l1_security_generation INTEGER NOT NULL CHECK (l1_security_generation > 0),
    gateway_connection_generation INTEGER NOT NULL CHECK (gateway_connection_generation > 0),
    namespace TEXT NOT NULL CHECK (length(namespace) BETWEEN 1 AND 512),
    namespace_generation INTEGER NOT NULL CHECK (namespace_generation > 0),
    policy_expires_at_unix_ms INTEGER NOT NULL CHECK (policy_expires_at_unix_ms >= 0),
    installed_at_unix_ms INTEGER NOT NULL CHECK (installed_at_unix_ms >= 0)
) WITHOUT ROWID`,
			`CREATE TABLE local_authority_documents (
    authority_digest TEXT PRIMARY KEY CHECK (length(authority_digest) = 71),
    scope_digest TEXT NOT NULL
        REFERENCES local_authority_state(scope_digest),
    attempt_id TEXT NOT NULL CHECK (length(attempt_id) BETWEEN 1 AND 256),
    credential_digest TEXT NOT NULL CHECK (length(credential_digest) = 71),
    allow_read INTEGER NOT NULL CHECK (allow_read IN (0, 1)),
    allow_write INTEGER NOT NULL CHECK (allow_write IN (0, 1)),
    canonical_document BLOB NOT NULL,
    expires_at_unix_ms INTEGER NOT NULL CHECK (expires_at_unix_ms >= 0),
    registered_at_unix_ms INTEGER NOT NULL CHECK (registered_at_unix_ms >= 0),
    CHECK (allow_read = 1 OR allow_write = 1)
) WITHOUT ROWID`,
			`CREATE INDEX local_authority_documents_attempt
ON local_authority_documents (attempt_id)`,
			`CREATE INDEX local_authority_documents_expires
ON local_authority_documents (expires_at_unix_ms)`,
		},
	}
	definition := metadataDefinition{
		role: controlMetadataRole,
		path: path,
		migrations: []schemaMigration{
			versionOne,
			versionTwo,
			versionThree,
		},
	}
	definition.objects = []schemaObject{
		{
			objectType: "index",
			name:       "decision_audit_index_indexed_at",
			statement:  versionOne.statements[2],
		},
		{
			objectType: "index",
			name:       "local_authority_documents_attempt",
			statement:  versionThree.statements[2],
		},
		{
			objectType: "index",
			name:       "local_authority_documents_expires",
			statement:  versionThree.statements[3],
		},
		{
			objectType: "index",
			name:       "reconciliation_runs_completed_at",
			statement:  versionTwo.statements[1],
		},
		{
			objectType: "table",
			name:       "decision_audit_index",
			statement:  versionOne.statements[1],
		},
		{
			objectType: "table",
			name:       "local_authority_documents",
			statement:  versionThree.statements[1],
		},
		{
			objectType: "table",
			name:       "local_authority_state",
			statement:  versionThree.statements[0],
		},
		{
			objectType: "table",
			name:       "reconciliation_runs",
			statement:  versionTwo.statements[0],
		},
		{
			objectType: "table",
			name:       "schema_migrations",
			statement:  versionOne.statements[0],
		},
	}
	return definition
}

func openSQLiteMetadata(
	ctx context.Context,
	owner *Storage,
	definition metadataDefinition,
) (*sqliteMetadata, error) {
	if err := preparePrivateDatabase(definition.path); err != nil {
		return nil, err
	}
	if err := validatePrivateDatabaseFiles(definition.path); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", sqliteDataSource(definition.path))
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	metadata := &sqliteMetadata{
		owner:      owner,
		definition: definition,
		database:   database,
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := metadata.applyOrValidateSchema(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := validatePrivateDatabaseFiles(definition.path); err != nil {
		_ = database.Close()
		return nil, err
	}
	return metadata, nil
}

func sqliteDataSource(path string) string {
	databaseURL := &url.URL{
		Scheme: "file",
		Path:   path,
	}
	query := url.Values{}
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "wal")
	query.Set("_synchronous", "full")
	query.Add("_pragma", "trusted_schema(0)")
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}

func (metadata *sqliteMetadata) applyOrValidateSchema(
	ctx context.Context,
) error {
	if err := metadata.validatePragmas(ctx); err != nil {
		return err
	}
	var version int
	if err := metadata.database.QueryRowContext(
		ctx,
		"PRAGMA user_version",
	).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version < 0 || version > SchemaVersion {
		return fmt.Errorf(
			"unsupported %s schema version %d",
			metadata.definition.role,
			version,
		)
	}
	if version == 0 {
		if err := metadata.requireEmptySchema(ctx); err != nil {
			return err
		}
	} else if err := metadata.validateMigrationRecords(
		ctx,
		version,
	); err != nil {
		return err
	}
	for version < SchemaVersion {
		migration := metadata.definition.migrations[version]
		if migration.version != version+1 {
			return errors.New("non-contiguous schema migration definition")
		}
		if err := metadata.applyMigration(ctx, migration); err != nil {
			return err
		}
		version = migration.version
	}
	if err := metadata.validateMigrationRecords(
		ctx,
		SchemaVersion,
	); err != nil {
		return err
	}
	if err := metadata.validateSchemaObjects(ctx); err != nil {
		return err
	}
	return metadata.integrityCheck(ctx)
}

func (metadata *sqliteMetadata) validatePragmas(ctx context.Context) error {
	testCases := []struct {
		query string
		want  string
	}{
		{query: "PRAGMA journal_mode", want: "wal"},
		{query: "PRAGMA foreign_keys", want: "1"},
		{query: "PRAGMA synchronous", want: "2"},
		{query: "PRAGMA busy_timeout", want: "5000"},
		{query: "PRAGMA trusted_schema", want: "0"},
	}
	for _, testCase := range testCases {
		var actual string
		if err := metadata.database.QueryRowContext(
			ctx,
			testCase.query,
		).Scan(&actual); err != nil {
			return fmt.Errorf("%s: %w", testCase.query, err)
		}
		if !strings.EqualFold(actual, testCase.want) {
			return fmt.Errorf(
				"%s = %q, want %q",
				testCase.query,
				actual,
				testCase.want,
			)
		}
	}
	return nil
}

func (metadata *sqliteMetadata) requireEmptySchema(ctx context.Context) error {
	rows, err := metadata.database.QueryContext(
		ctx,
		`SELECT type, name
FROM sqlite_master
WHERE name NOT LIKE 'sqlite_%'
ORDER BY type, name`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var objectType string
		var name string
		if err := rows.Scan(&objectType, &name); err != nil {
			return err
		}
		return fmt.Errorf(
			"unversioned schema contains %s %s",
			objectType,
			name,
		)
	}
	return rows.Err()
}

func (metadata *sqliteMetadata) applyMigration(
	ctx context.Context,
	migration schemaMigration,
) error {
	transaction, err := metadata.database.BeginTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelSerializable},
	)
	if err != nil {
		return err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = transaction.Rollback()
		}
	}()
	for _, statement := range migration.statements {
		if _, err := transaction.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf(
				"apply schema migration %d: %w",
				migration.version,
				err,
			)
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO schema_migrations
    (version, name, checksum, applied_at_unix_ms)
VALUES (?, ?, ?, ?)`,
		migration.version,
		migration.name,
		migrationChecksum(metadata.definition.role, migration),
		time.Now().UTC().UnixMilli(),
	); err != nil {
		return fmt.Errorf("record schema migration: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA user_version = %d", migration.version),
	); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	rollback = false
	return nil
}

func (metadata *sqliteMetadata) validateMigrationRecords(
	ctx context.Context,
	expectedVersion int,
) error {
	rows, err := metadata.database.QueryContext(
		ctx,
		`SELECT version, name, checksum
FROM schema_migrations
ORDER BY version`,
	)
	if err != nil {
		return fmt.Errorf("read migration records: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var (
			version  int
			name     string
			checksum string
		)
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return fmt.Errorf("scan migration record: %w", err)
		}
		count++
		if version < 1 ||
			version > len(metadata.definition.migrations) {
			return errors.New(
				"schema migration identity does not match this binary",
			)
		}
		migration := metadata.definition.migrations[version-1]
		if version != count ||
			migration.version != version ||
			name != migration.name ||
			checksum != migrationChecksum(
				metadata.definition.role,
				migration,
			) {
			return errors.New(
				"schema migration identity does not match this binary",
			)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != expectedVersion {
		return errors.New("schema migration identity does not match this binary")
	}
	return nil
}

func (metadata *sqliteMetadata) validateSchemaObjects(
	ctx context.Context,
) error {
	rows, err := metadata.database.QueryContext(
		ctx,
		`SELECT type, name, sql
FROM sqlite_master
WHERE name NOT LIKE 'sqlite_%'
ORDER BY type, name`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var actual []schemaObject
	for rows.Next() {
		var object schemaObject
		if err := rows.Scan(
			&object.objectType,
			&object.name,
			&object.statement,
		); err != nil {
			return err
		}
		actual = append(actual, object)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !slices.Equal(actual, metadata.definition.objects) {
		return fmt.Errorf(
			"schema objects = %+v, want %+v",
			actual,
			metadata.definition.objects,
		)
	}
	return nil
}

func migrationChecksum(role string, migration schemaMigration) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("buildopt-single-node-schema-v1"))
	for _, value := range append(
		[]string{role, migration.name},
		migration.statements...,
	) {
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(value))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func (metadata *sqliteMetadata) Role() string {
	return metadata.definition.role
}

func (metadata *sqliteMetadata) SchemaVersion(
	ctx context.Context,
) (int, error) {
	finish, err := metadata.owner.beginOperation()
	if err != nil {
		return 0, err
	}
	defer finish()
	var version int
	err = metadata.database.QueryRowContext(
		ctx,
		"PRAGMA user_version",
	).Scan(&version)
	return version, err
}

func (metadata *sqliteMetadata) IntegrityCheck(ctx context.Context) error {
	finish, err := metadata.owner.beginOperation()
	if err != nil {
		return err
	}
	defer finish()
	return metadata.integrityCheck(ctx)
}

func (metadata *sqliteMetadata) integrityCheck(ctx context.Context) error {
	var result string
	if err := metadata.database.QueryRowContext(
		ctx,
		"PRAGMA quick_check(1)",
	).Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite quick check: %s", result)
	}
	rows, err := metadata.database.QueryContext(
		ctx,
		"PRAGMA foreign_key_check",
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("SQLite foreign-key check found a violation")
	}
	return rows.Err()
}

func (metadata *sqliteMetadata) close() error {
	if metadata == nil || metadata.database == nil {
		return nil
	}
	err := metadata.database.Close()
	metadata.database = nil
	return err
}

func validatePrivateDatabaseFiles(path string) error {
	for _, candidate := range []string{
		path,
		path + "-journal",
		path + "-wal",
		path + "-shm",
	} {
		if err := validatePrivateSidecar(candidate); err != nil {
			return fmt.Errorf("unsafe SQLite file %s: %w", candidate, err)
		}
	}
	return nil
}
