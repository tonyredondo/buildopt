package sharedcache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCloseWaitsForAndReportsProtectedAccessBatchFailure(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	fault := errors.New("protected access persistence fault")
	entered := make(chan struct{})
	release := make(chan struct{})
	storage.testHooks.beforeProtectedAccessBatch = func(int) error {
		close(entered)
		<-release
		return fault
	}
	if err := storage.batchProtectedAccess(
		ctx,
		CommittedObject{
			RepositoryTenant:    "tenant-test",
			NamespaceGeneration: 1,
			Key:                 "key-test",
			Blob: Blob{
				Digest: "sha256:" + strings.Repeat("a", 64),
				Size:   1,
			},
			DecisionDigest: "sha256:" + strings.Repeat("b", 64),
		},
		time.Now().UTC(),
	); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		_ = storage.Close()
		t.Fatal("protected access batch did not start")
	}
	closed := make(chan error, 1)
	go func() {
		closed <- storage.Close()
	}()
	select {
	case err := <-closed:
		close(release)
		t.Fatalf("storage closed before access batch finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-closed:
		if !errors.Is(err, fault) {
			t.Fatalf("storage close error = %v, want %v", err, fault)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("storage did not close after access batch finished")
	}
}

func TestOperationalSnapshotReportsBoundedStorageHealth(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = storage.Close()
	})

	snapshot, err := storage.OperationalSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.CapturedAt.IsZero() ||
		!snapshot.DiskProbeSucceeded ||
		snapshot.DiskTotalBytes == 0 ||
		snapshot.DiskAvailableBytes == 0 ||
		snapshot.DiskAvailableBytes > snapshot.DiskTotalBytes ||
		snapshot.PendingAttempts != 0 ||
		snapshot.ExpiredPendingAttempts != 0 ||
		snapshot.QuarantineRecords != 0 ||
		!snapshot.IntegrityHealthy ||
		!snapshot.SQLiteProbeSucceeded ||
		snapshot.SQLiteProbeDuration < 0 {
		t.Fatalf("operational snapshot = %+v", snapshot)
	}
}

func TestCacheMetadataUsesBoundedConcurrentWALConnections(t *testing.T) {
	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if actual := storage.cache.database.Stats().MaxOpenConnections; actual !=
		cacheMetadataConnections {
		t.Fatalf(
			"cache metadata connection limit = %d, want %d",
			actual,
			cacheMetadataConnections,
		)
	}
	if actual := storage.control.database.Stats().MaxOpenConnections; actual !=
		controlMetadataConnections {
		t.Fatalf(
			"control metadata connection limit = %d, want %d",
			actual,
			controlMetadataConnections,
		)
	}

	connections := make([]*sql.Conn, 0, cacheMetadataConnections-1)
	defer func() {
		for _, connection := range connections {
			_ = connection.Close()
		}
	}()
	for range cacheMetadataConnections - 1 {
		connection, err := storage.cache.database.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, connection)
	}
	queryContext, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var count int
	if err := storage.cache.database.QueryRowContext(
		queryContext,
		"SELECT count(*) FROM committed_objects",
	).Scan(&count); err != nil {
		t.Fatalf("concurrent cache metadata reader: %v", err)
	}
	if count != 0 {
		t.Fatalf("committed object count = %d, want 0", count)
	}
}

func TestOpenCreatesPrivateWALStorageAndOwnsWriter(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage, err := Open(ctx, root)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			_ = storage.Close()
		}
	})

	layout := storage.Layout()
	if layout != (Layout{
		Root:            root,
		Blobs:           filepath.Join(root, "blobs", "sha256"),
		Spool:           filepath.Join(root, "spool"),
		Quarantine:      filepath.Join(root, "quarantine"),
		CacheDatabase:   filepath.Join(root, "cache.sqlite"),
		ControlDatabase: filepath.Join(root, "control.sqlite"),
		WriterLock:      filepath.Join(root, "writer.lock"),
	}) {
		t.Fatalf("layout = %+v", layout)
	}
	for _, directory := range []string{
		layout.Root,
		filepath.Join(layout.Root, "blobs"),
		layout.Blobs,
		layout.Spool,
		layout.Quarantine,
	} {
		assertMode(t, directory, true, 0o700)
	}
	for _, file := range []string{
		layout.CacheDatabase,
		layout.ControlDatabase,
		layout.WriterLock,
	} {
		assertMode(t, file, false, 0o600)
	}

	assertMetadataHealth(
		t,
		storage.cache,
		cacheMetadataRole,
		"wal",
	)
	assertMetadataHealth(
		t,
		storage.control,
		controlMetadataRole,
		"wal",
	)
	assertSchemaObjects(
		t,
		storage.cache.database,
		cacheMetadataDefinition(layout.CacheDatabase).objects,
	)
	assertSchemaObjects(
		t,
		storage.control.database,
		controlMetadataDefinition(layout.ControlDatabase).objects,
	)

	second, err := Open(ctx, root)
	if second != nil || !errors.Is(err, ErrWriterBusy) {
		if second != nil {
			_ = second.Close()
		}
		t.Fatalf("second writer = %+v/%v, want busy", second, err)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}
	closed = true
	if _, err := storage.CacheMetadata().SchemaVersion(ctx); !errors.Is(
		err,
		ErrClosed,
	) {
		t.Fatalf("closed schema access = %v, want ErrClosed", err)
	}
	reopened, err := Open(ctx, root)
	if err != nil {
		t.Fatalf("reopen released storage: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close reopened storage: %v", err)
	}
}

func TestSQLiteLifecyclesRemainSeparateAndPersistent(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "shared")
	storage, err := Open(ctx, root)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	if _, err := storage.cache.database.ExecContext(
		ctx,
		`INSERT INTO commit_decisions
    (decision_digest, attempt_id, canonical_document, committed_at_unix_ms)
VALUES (?, ?, ?, ?)`,
		digest,
		"attempt-test",
		[]byte(`{"fixture":true}`),
		1,
	); err != nil {
		t.Fatalf("insert cache decision fixture: %v", err)
	}
	if _, err := storage.control.database.ExecContext(
		ctx,
		`INSERT INTO decision_audit_index
    (decision_digest, indexed_at_unix_ms)
VALUES (?, ?)`,
		digest,
		2,
	); err != nil {
		t.Fatalf("insert control index fixture: %v", err)
	}
	if _, err := storage.cache.database.ExecContext(
		ctx,
		"SELECT * FROM decision_audit_index",
	); err == nil {
		t.Fatal("control table leaked into cache.sqlite")
	}
	if _, err := storage.control.database.ExecContext(
		ctx,
		"SELECT * FROM commit_decisions",
	); err == nil {
		t.Fatal("cache table leaked into control.sqlite")
	}
	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}

	reopened, err := Open(ctx, root)
	if err != nil {
		t.Fatalf("reopen storage: %v", err)
	}
	defer reopened.Close()
	for name, database := range map[string]*sql.DB{
		"cache":   reopened.cache.database,
		"control": reopened.control.database,
	} {
		var count int
		table := "commit_decisions"
		if name == "control" {
			table = "decision_audit_index"
		}
		if err := database.QueryRowContext(
			ctx,
			"SELECT count(*) FROM "+table,
		).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s persisted row count = %d/%v", name, count, err)
		}
	}
}

func TestOpenRejectsUnsafeOrDriftedStorage(t *testing.T) {
	ctx := context.Background()
	t.Run("relative root", func(t *testing.T) {
		if storage, err := Open(ctx, "relative/shared"); err == nil {
			_ = storage.Close()
			t.Fatal("relative root was accepted")
		}
	})
	t.Run("public root", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "mode 0700") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("public root = %+v/%v", storage, err)
		}
	})
	t.Run("symlink root", func(t *testing.T) {
		parent := t.TempDir()
		target := filepath.Join(parent, "target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(parent, "shared")
		if err := os.Symlink(target, root); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil {
			_ = storage.Close()
			t.Fatal("symlink root was accepted")
		}
	})
	t.Run("unrelated root content", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(root, "unrelated"),
			[]byte("preserve"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "unexpected root entry") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("unrelated root = %+v/%v", storage, err)
		}
		if _, err := os.Stat(filepath.Join(root, "blobs")); !errors.Is(
			err,
			os.ErrNotExist,
		) {
			t.Fatalf("rejected root was mutated: %v", err)
		}
	})
	t.Run("future schema", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		storage, err := Open(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		cachePath := storage.Layout().CacheDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		database := openTestSQLite(t, cachePath)
		if _, err := database.Exec(
			fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion+1),
		); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "unsupported") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("future schema = %+v/%v", storage, err)
		}
	})
	t.Run("schema definition drift", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		storage, err := Open(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		controlPath := storage.Layout().ControlDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		database := openTestSQLite(t, controlPath)
		if _, err := database.Exec(
			"ALTER TABLE decision_audit_index ADD COLUMN unexpected TEXT",
		); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "schema objects") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("drifted schema = %+v/%v", storage, err)
		}
	})
	t.Run("corrupt database", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		storage, err := Open(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		controlPath := storage.Layout().ControlDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			controlPath,
			[]byte("not a SQLite database"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil {
			_ = storage.Close()
			t.Fatal("corrupt database was accepted")
		}
	})
	t.Run("foreign key corruption", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		storage, err := Open(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		cachePath := storage.Layout().CacheDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		database := openTestSQLite(t, cachePath)
		if _, err := database.Exec("PRAGMA foreign_keys = OFF"); err != nil {
			t.Fatal(err)
		}
		digest := "sha256:" + strings.Repeat("b", 64)
		if _, err := database.Exec(
			`INSERT INTO committed_objects
    (tenant_id, namespace_generation, cache_key, blob_digest, size_bytes,
     decision_digest, committed_at_unix_ms, last_access_unix_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"tenant-test",
			1,
			"cache-key-test",
			digest,
			1,
			digest,
			1,
			1,
		); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "foreign-key") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("foreign-key corruption = %+v/%v", storage, err)
		}
	})
	t.Run("unsafe database mode", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "shared")
		storage, err := Open(ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		cachePath := storage.Layout().CacheDatabase
		if err := storage.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(cachePath, 0o644); err != nil {
			t.Fatal(err)
		}
		if storage, err := Open(ctx, root); err == nil ||
			!strings.Contains(err.Error(), "mode 0600") {
			if storage != nil {
				_ = storage.Close()
			}
			t.Fatalf("unsafe database mode = %+v/%v", storage, err)
		}
	})
}

func assertMetadataHealth(
	t *testing.T,
	metadata *sqliteMetadata,
	role string,
	journalMode string,
) {
	t.Helper()
	ctx := context.Background()
	if metadata.Role() != role {
		t.Errorf("role = %q, want %q", metadata.Role(), role)
	}
	if version, err := metadata.SchemaVersion(ctx); err != nil ||
		version != SchemaVersion {
		t.Errorf("schema version = %d/%v", version, err)
	}
	if err := metadata.IntegrityCheck(ctx); err != nil {
		t.Errorf("integrity check: %v", err)
	}
	var actualJournal string
	if err := metadata.database.QueryRow(
		"PRAGMA journal_mode",
	).Scan(&actualJournal); err != nil ||
		actualJournal != journalMode {
		t.Errorf("journal mode = %q/%v", actualJournal, err)
	}
}

func assertSchemaObjects(
	t *testing.T,
	database *sql.DB,
	want []schemaObject,
) {
	t.Helper()
	rows, err := database.Query(
		`SELECT type, name, sql
FROM sqlite_master
WHERE name NOT LIKE 'sqlite_%'
ORDER BY type, name`,
	)
	if err != nil {
		t.Fatal(err)
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
			t.Fatal(err)
		}
		actual = append(actual, object)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(want) {
		t.Fatalf("schema objects = %+v, want %+v", actual, want)
	}
	for index := range want {
		if actual[index] != want[index] {
			t.Fatalf("schema objects = %+v, want %+v", actual, want)
		}
	}
}

func assertMode(
	t *testing.T,
	path string,
	directory bool,
	mode os.FileMode,
) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() != directory ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != mode {
		t.Fatalf("%s mode/type = %s, want directory=%t mode=%o", path, info.Mode(), directory, mode)
	}
}

func openTestSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", sqliteDataSource(path))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	return database
}
