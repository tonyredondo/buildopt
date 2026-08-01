package datalifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/filelock"
)

func TestDeleteManagedDataRevokesBeforePhysicalRemoval(t *testing.T) {
	root := managedDataRootFixture(t)
	requestedAt := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	request := deletionRequestFixture(root, requestedAt)
	manager := deletionManager{
		removeAll: os.RemoveAll,
		now: func() time.Time {
			return requestedAt.Add(2 * time.Second)
		},
		afterLogicalRevocation: func(tombstone DeletionTombstone) error {
			if tombstone.State != deletionStateRevoked {
				t.Fatalf("logical state = %q", tombstone.State)
			}
			if _, err := os.Stat(filepath.Join(root, "shared", "payload")); err != nil {
				t.Fatalf("payload removed before logical revocation hook: %v", err)
			}
			boundary, err := InspectManagedPath(filepath.Join(root, "shared"))
			if err != nil {
				t.Fatalf("inspect revoked boundary: %v", err)
			}
			if !boundary.Managed || !boundary.Revoked ||
				boundary.MinimumNamespaceGeneration != 8 ||
				boundary.MinimumL1SecurityGeneration != 12 {
				t.Fatalf("revoked boundary = %+v", boundary)
			}
			return nil
		},
	}
	report, err := manager.delete(context.Background(), request)
	if err != nil {
		t.Fatalf("delete managed data: %v", err)
	}
	if report.Replay || report.Tombstone.State != deletionStateComplete ||
		report.Tombstone.RemovedComponents != len(managedComponents) ||
		report.Tombstone.PhysicalCompletedAt !=
			requestedAt.Add(2*time.Second).Format(time.RFC3339Nano) {
		t.Fatalf("deletion report = %+v", report)
	}
	for _, component := range managedComponents {
		if _, err := os.Lstat(filepath.Join(root, component.name)); !errors.Is(
			err,
			os.ErrNotExist,
		) {
			t.Fatalf("managed component %q remains: %v", component.name, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(root, deletionTombstoneName))
	if err != nil {
		t.Fatalf("read deletion tombstone: %v", err)
	}
	for _, raw := range []string{
		request.Tenant,
		request.Repository,
		request.TrustDomain,
		request.ExternalDestinations[0],
	} {
		if bytes.Contains(content, []byte(raw)) {
			t.Fatalf("tombstone contains raw identity %q", raw)
		}
	}
	request.RequestedAt = request.RequestedAt.Add(time.Minute)
	replay, err := manager.delete(context.Background(), request)
	if err != nil {
		t.Fatalf("replay deletion: %v", err)
	}
	if !replay.Replay || !reflect.DeepEqual(
		replay.Tombstone,
		report.Tombstone,
	) {
		t.Fatalf("replay = %+v, want exact prior tombstone", replay)
	}
	boundary, err := InspectManagedPath(root)
	if err != nil {
		t.Fatalf("inspect completed boundary: %v", err)
	}
	if !boundary.Managed || boundary.Revoked ||
		boundary.MinimumNamespaceGeneration != 8 ||
		boundary.MinimumL1SecurityGeneration != 12 {
		t.Fatalf("completed boundary = %+v", boundary)
	}
}

func TestDeleteManagedDataRefusesActiveLeaseBeforeRevocation(t *testing.T) {
	root := managedDataRootFixture(t)
	writerPath := filepath.Join(root, "shared", "writer.lock")
	writer, err := os.OpenFile(writerPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open writer lease: %v", err)
	}
	defer writer.Close()
	if err := filelock.Try(writer, filelock.Exclusive); err != nil {
		t.Fatalf("hold writer lease: %v", err)
	}
	defer filelock.Unlock(writer)

	_, err = DeleteManagedData(
		context.Background(),
		deletionRequestFixture(root, time.Now().UTC()),
	)
	if !errors.Is(err, ErrManagedDataBusy) {
		t.Fatalf("active lease error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, deletionTombstoneName)); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf("logical revocation written despite active lease: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "shared", "payload")); err != nil {
		t.Fatalf("payload changed despite active lease: %v", err)
	}
}

func TestDeleteManagedDataRefusesSharedLifecycleLease(t *testing.T) {
	root := managedDataRootFixture(t)
	lease, boundary, err := AcquireManagedLease(filepath.Join(root, "exports"))
	if err != nil {
		t.Fatalf("acquire managed lifecycle lease: %v", err)
	}
	defer lease.Close()
	if !boundary.Managed || boundary.DataRoot != root {
		t.Fatalf("managed lifecycle boundary = %+v", boundary)
	}
	_, err = DeleteManagedData(
		context.Background(),
		deletionRequestFixture(root, time.Now().UTC()),
	)
	if !errors.Is(err, ErrManagedDataBusy) {
		t.Fatalf("shared lifecycle lease error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, deletionTombstoneName)); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf("logical revocation written despite shared lifecycle lease: %v", err)
	}
}

func TestDeleteManagedDataRejectsImplicitRetentionAndUnknownRootData(
	t *testing.T,
) {
	root := managedDataRootFixture(t)
	request := deletionRequestFixture(root, time.Now().UTC())
	request.RetentionHold = &RetentionHold{}
	if _, err := DeleteManagedData(context.Background(), request); err == nil ||
		errors.Is(err, ErrRetentionHoldActive) {
		t.Fatalf("implicit retention error = %v", err)
	}

	request.RetentionHold = nil
	if err := os.WriteFile(
		filepath.Join(root, "unknown-data"),
		[]byte("must-not-delete"),
		0o600,
	); err != nil {
		t.Fatalf("create unknown root data: %v", err)
	}
	if _, err := DeleteManagedData(context.Background(), request); err == nil {
		t.Fatal("deleted a data root containing an unknown entry")
	}
	if _, err := os.Stat(filepath.Join(root, "unknown-data")); err != nil {
		t.Fatalf("unknown data changed: %v", err)
	}
}

func managedDataRootFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "deployment-data")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create data root: %v", err)
	}
	marker, err := json.Marshal(map[string]string{
		"deploymentRoot": filepath.Join(filepath.Dir(root), "deployment"),
		"schemaVersion":  "buildopt.dev/deployment-data/v1",
	})
	if err != nil {
		t.Fatalf("encode marker: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, deploymentDataMarkerName),
		append(marker, '\n'),
		0o600,
	); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	for _, component := range managedComponents {
		directory := filepath.Join(root, component.name)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("create component %s: %v", component.name, err)
		}
		if err := os.WriteFile(
			filepath.Join(directory, "payload"),
			[]byte("managed"),
			0o600,
		); err != nil {
			t.Fatalf("write component payload: %v", err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "l1", "locks"), 0o700); err != nil {
		t.Fatalf("create L1 locks: %v", err)
	}
	for _, lockPath := range []string{
		filepath.Join(root, "shared", "writer.lock"),
		filepath.Join(root, "l1", "locks", "scope.lock"),
	} {
		if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
			t.Fatalf("write lease fixture: %v", err)
		}
	}
	return root
}

func deletionRequestFixture(
	root string,
	requestedAt time.Time,
) DeletionRequest {
	key := make([]byte, RedactionKeyBytes)
	for index := range key {
		key[index] = byte(index + 1)
	}
	return DeletionRequest{
		DataRoot:                 root,
		DeletionID:               "delete-private-beta-001",
		Tenant:                   "tenant-sensitive",
		Repository:               "repository-sensitive",
		TrustDomain:              "trust-sensitive",
		NextNamespaceGeneration:  8,
		NextL1SecurityGeneration: 12,
		TokenKey:                 key,
		TokenKeyVersion:          "deletion-key-v1",
		ExternalDestinations:     []string{"customer-warehouse"},
		RequestedAt:              requestedAt,
	}
}
