//go:build windows

package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImpactRegularFileDigestAcceptsPlatformNeutralPathOnWindows(t *testing.T) {
	root := t.TempDir()
	wrapperRoot := filepath.Join(root, "gradle", "wrapper")
	if err := os.MkdirAll(wrapperRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(wrapperRoot, "gradle-wrapper.jar"),
		[]byte("wrapper"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := impactRegularFileDigest(
		root,
		"gradle/wrapper/gradle-wrapper.jar",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := impactRegularFileDigest(root, "../outside"); err == nil {
		t.Fatal("parent traversal was accepted")
	}
}
