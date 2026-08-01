package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExclusiveLockLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lease")
	first, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	if err := Try(first, Exclusive); err != nil {
		t.Fatal(err)
	}
	if err := Try(second, Exclusive); !errors.Is(err, ErrBusy) {
		t.Fatalf("second lock error = %v", err)
	}
	if err := Unlock(first); err != nil {
		t.Fatal(err)
	}
	if err := Try(second, Exclusive); err != nil {
		t.Fatal(err)
	}
	if err := Unlock(second); err != nil {
		t.Fatal(err)
	}
}
