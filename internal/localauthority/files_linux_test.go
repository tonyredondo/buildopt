//go:build linux

package localauthority

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrivateAuthorityFilesAndMonotonicStateStore(t *testing.T) {
	document, credential, privateKey, publicKey := authorityTestFixture()
	authority, err := Sign(document, "deployment-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	trustRoot, err := EncodeTrustRoot(TrustRoot{
		Keys: []PublicKey{{
			KeyID: "deployment-key-1",
			PublicKey: base64.RawURLEncoding.EncodeToString(
				publicKey,
			),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	authorityPath := writePrivateAuthorityTestFile(
		t,
		root,
		"authority.json",
		authority,
	)
	trustRootPath := writePrivateAuthorityTestFile(
		t,
		root,
		"trust-root.json",
		trustRoot,
	)
	credentialPath := writePrivateAuthorityTestFile(
		t,
		root,
		"credential",
		[]byte(base64.RawURLEncoding.EncodeToString(credential)+"\n"),
	)
	verified, keys, loadedCredential, err := LoadFiles(
		context.Background(),
		authorityPath,
		trustRootPath,
		credentialPath,
		authorityTestNow,
	)
	if err != nil ||
		!bytes.Equal(keys["deployment-key-1"], publicKey) ||
		!bytes.Equal(loadedCredential, credential) {
		t.Fatalf("load files = %+v/%v/%v", verified, loadedCredential, err)
	}

	signingPath := writePrivateAuthorityTestFile(
		t,
		root,
		"signing-key.json",
		mustEncodeSigningKey(t, privateKey),
	)
	keyID, loadedPrivate, err := LoadSigningKeyFile(signingPath, keys)
	if err != nil ||
		keyID != "deployment-key-1" ||
		!bytes.Equal(loadedPrivate, privateKey) {
		t.Fatalf("load signing key = %q/%v/%v", keyID, loadedPrivate, err)
	}

	stateRoot := filepath.Join(root, "state")
	store, err := NewFileStateStore(stateRoot)
	if err != nil {
		t.Fatalf("create state store: %v", err)
	}
	previous, current, changed, err := store.Install(
		verified,
		authorityTestNow,
	)
	if err != nil || previous != (State{}) || !changed {
		t.Fatalf("initial install = %+v/%+v/%t/%v", previous, current, changed, err)
	}
	previous, replay, changed, err := store.Install(
		verified,
		authorityTestNow.Add(time.Minute),
	)
	if err != nil || changed || replay != previous || replay != current {
		t.Fatalf("exact replay = %+v/%+v/%t/%v", previous, replay, changed, err)
	}
	statePath := filepath.Join(
		stateRoot,
		"policy",
		"scopes",
		current.ScopeDigest+".json",
	)
	info, err := os.Lstat(statePath)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 {
		t.Fatalf("state file = %v/%v", info, err)
	}

	rollbackDocument := document
	rollbackDocument.Policy.PolicyVersion--
	rollbackAuthority, err := Sign(
		rollbackDocument,
		"deployment-key-1",
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	rollbackVerified, err := Verify(
		context.Background(),
		rollbackAuthority,
		keys,
		credential,
		authorityTestNow,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.Install(
		rollbackVerified,
		authorityTestNow.Add(2*time.Minute),
	); !errors.Is(err, ErrRollback) {
		t.Fatalf("rollback install = %v, want ErrRollback", err)
	}
}

func TestPrivateAuthorityFilesRejectModeAndSymlink(t *testing.T) {
	root := t.TempDir()
	publicPath := filepath.Join(root, "public")
	if err := os.WriteFile(publicPath, []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPrivateFile(publicPath, 64); err == nil ||
		!strings.Contains(err.Error(), "private") {
		t.Fatalf("public file error = %v", err)
	}

	target := writePrivateAuthorityTestFile(
		t,
		root,
		"target",
		[]byte("target"),
	)
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPrivateFile(link, 64); err == nil {
		t.Fatal("symlink authority file was accepted")
	}
}

func writePrivateAuthorityTestFile(
	t *testing.T,
	root string,
	name string,
	content []byte,
) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustEncodeSigningKey(
	t *testing.T,
	privateKey ed25519.PrivateKey,
) []byte {
	t.Helper()
	encoded, err := EncodeSigningKey("deployment-key-1", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
