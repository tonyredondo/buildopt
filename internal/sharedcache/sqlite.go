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
	role        string
	path        string
	migration   []string
	objects     []schemaObject
	migrationID string
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
	definition := metadataDefinition{
		role:        cacheMetadataRole,
		path:        path,
		migrationID: "cache-v1",
		migration: []string{
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
	definition.objects = []schemaObject{
		{
			objectType: "index",
			name:       "committed_objects_blob_digest",
			statement:  definition.migration[3],
		},
		{
			objectType: "index",
			name:       "committed_objects_last_access",
			statement:  definition.migration[4],
		},
		{
			objectType: "table",
			name:       "commit_decisions",
			statement:  definition.migration[1],
		},
		{
			objectType: "table",
			name:       "committed_objects",
			statement:  definition.migration[2],
		},
		{
			objectType: "table",
			name:       "schema_migrations",
			statement:  definition.migration[0],
		},
	}
	return definition
}

func controlMetadataDefinition(path string) metadataDefinition {
	definition := metadataDefinition{
		role:        controlMetadataRole,
		path:        path,
		migrationID: "control-v1",
		migration: []string{
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
	definition.objects = []schemaObject{
		{
			objectType: "index",
			name:       "decision_audit_index_indexed_at",
			statement:  definition.migration[2],
		},
		{
			objectType: "table",
			name:       "decision_audit_index",
			statement:  definition.migration[1],
		},
		{
			objectType: "table",
			name:       "schema_migrations",
			statement:  definition.migration[0],
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
	switch version {
	case 0:
		if err := metadata.requireEmptySchema(ctx); err != nil {
			return err
		}
		if err := metadata.applyInitialMigration(ctx); err != nil {
			return err
		}
	case SchemaVersion:
	default:
		return fmt.Errorf(
			"unsupported %s schema version %d",
			metadata.definition.role,
			version,
		)
	}
	if err := metadata.validateMigrationRecord(ctx); err != nil {
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

func (metadata *sqliteMetadata) applyInitialMigration(
	ctx context.Context,
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
	for _, statement := range metadata.definition.migration {
		if _, err := transaction.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply schema migration: %w", err)
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO schema_migrations
    (version, name, checksum, applied_at_unix_ms)
VALUES (?, ?, ?, ?)`,
		SchemaVersion,
		metadata.definition.migrationID,
		migrationChecksum(metadata.definition),
		time.Now().UTC().UnixMilli(),
	); err != nil {
		return fmt.Errorf("record schema migration: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion),
	); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	rollback = false
	return nil
}

func (metadata *sqliteMetadata) validateMigrationRecord(
	ctx context.Context,
) error {
	var (
		version  int
		name     string
		checksum string
		count    int
	)
	if err := metadata.database.QueryRowContext(
		ctx,
		`SELECT version, name, checksum FROM schema_migrations`,
	).Scan(&version, &name, &checksum); err != nil {
		return fmt.Errorf("read migration record: %w", err)
	}
	if err := metadata.database.QueryRowContext(
		ctx,
		`SELECT count(*) FROM schema_migrations`,
	).Scan(&count); err != nil {
		return fmt.Errorf("count migration records: %w", err)
	}
	if version != SchemaVersion ||
		name != metadata.definition.migrationID ||
		checksum != migrationChecksum(metadata.definition) ||
		count != 1 {
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

func migrationChecksum(definition metadataDefinition) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("buildopt-single-node-schema-v1"))
	for _, value := range append(
		[]string{definition.role, definition.migrationID},
		definition.migration...,
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
