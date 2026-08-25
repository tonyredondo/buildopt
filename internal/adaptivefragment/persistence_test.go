package adaptivefragment

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tonyredondo/buildopt/internal/filelock"
)

func TestLocalStateRoundTripIsCanonicalAndMachineIndependent(t *testing.T) {
	bundle := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))
	leftRoot := filepath.Join(t.TempDir(), "adaptive")
	rightRoot := filepath.Join(t.TempDir(), "adaptive")
	left, err := SaveLocalState(leftRoot, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	right, err := RestoreLocalState(rightRoot, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if left.HeadSHA256 != right.HeadSHA256 || left.Head.CompatibilitySHA256 != right.Head.CompatibilitySHA256 ||
		left.Head.BindingsSHA256 != right.Head.BindingsSHA256 || len(left.Files) != len(right.Files) {
		t.Fatalf("machine-independent state drifted: %+v / %+v", left.Head, right.Head)
	}
	for path, expected := range left.Files {
		actual, found := right.Files[path]
		if !found || string(actual) != string(expected) {
			t.Fatalf("portable file %q drifted", path)
		}
	}
	loaded, err := LoadLocalState(filepath.Join(t.TempDir(), "adaptive"))
	if err == nil || !errors.Is(err, ErrLocalStateAbsent) {
		t.Fatalf("absent local state = %+v, %v", loaded, err)
	}
}

func TestLocalStateCASIdempotencyAndNextGeneration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "adaptive")
	bundle := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))
	first, err := SaveLocalState(root, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := SaveLocalState(root, bundle, first.HeadSHA256)
	if err != nil || replayed.HeadSHA256 != first.HeadSHA256 {
		t.Fatalf("idempotent replay = %+v, %v", replayed.Head, err)
	}
	if _, err := SaveLocalState(root, bundle, digest("stale-head")); !errors.Is(err, ErrLocalStateConflict) {
		t.Fatalf("stale local CAS = %v", err)
	}

	next := bundle
	next.Portfolio.Generation = 2
	next.Portfolio.SupersedesSHA256 = first.Head.PortfolioSHA256
	next.Ledger.Generation = 2
	next.Ledger.SupersedesSHA256 = first.Head.LedgerSHA256
	next.Ledger.PortfolioGeneration = 2
	second, err := SaveLocalState(root, next, first.HeadSHA256)
	if err != nil || second.Head.PortfolioGeneration != 2 || second.Head.LedgerGeneration != 2 ||
		second.HeadSHA256 == first.HeadSHA256 {
		t.Fatalf("next local generation = %+v, %v", second.Head, err)
	}
	loaded, err := LoadLocalState(root)
	if err != nil || loaded.HeadSHA256 != second.HeadSHA256 {
		t.Fatalf("loaded next local generation = %+v, %v", loaded.Head, err)
	}
}

func TestLocalStateRejectsCorruptionAndBusyWriter(t *testing.T) {
	root := filepath.Join(t.TempDir(), "adaptive")
	bundle := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))
	snapshot, err := SaveLocalState(root, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(root, localLockFile), os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := filelock.Try(lock, filelock.Exclusive); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLocalState(root); !errors.Is(err, ErrLocalStateBusy) {
		t.Fatalf("busy local reader = %v", err)
	}
	if err := filelock.Unlock(lock); err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}

	ledgerPath := ""
	for path := range snapshot.Files {
		if filepath.Base(path) == "ledger.json" {
			ledgerPath = path
		}
	}
	if ledgerPath == "" {
		t.Fatal("persisted ledger path is absent")
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(ledgerPath)), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLocalState(root); !errors.Is(err, ErrLocalStateCorrupt) {
		t.Fatalf("corrupt local ledger = %v", err)
	}
}

func TestLocalStateUsesPrivateFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows privacy is enforced by ACL-aware platform storage")
	}
	root := filepath.Join(t.TempDir(), "adaptive")
	bundle := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))
	snapshot, err := SaveLocalState(root, bundle, "")
	if err != nil {
		t.Fatal(err)
	}
	for path := range snapshot.Files {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private adaptive file %q = %v/%v", path, info, err)
		}
	}
}

func TestLocalStateRejectsSymlinkLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows symlink creation requires privileges not granted to ordinary users")
	}
	root := filepath.Join(t.TempDir(), "adaptive")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "unrelated")
	if err := os.WriteFile(target, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, localLockFile)); err != nil {
		t.Fatal(err)
	}
	bundle := readStateBundle(t, filepath.Join(stateFixtureRoot(t), "valid", "active-lifecycle.json"))
	if _, err := SaveLocalState(root, bundle, ""); err == nil {
		t.Fatal("symlinked adaptive state lock was accepted")
	}
	raw, err := os.ReadFile(target)
	if err != nil || string(raw) != "preserve" {
		t.Fatalf("symlink target changed: %q, %v", raw, err)
	}
}
