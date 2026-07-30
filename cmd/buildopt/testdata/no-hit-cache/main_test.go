package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

func TestPrepareAuthorityAndAuthenticatedMiss(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	attemptID := "11111111-1111-4111-8111-000000000001"
	if err := prepareAuthority(root, attemptID, now); err != nil {
		t.Fatalf("prepare authority: %v", err)
	}
	authorityPath := filepath.Join(root, "authority.json")
	trustRootPath := filepath.Join(root, "trust-root.json")
	credentialPath := filepath.Join(root, "credential")
	for _, path := range []string{
		authorityPath,
		trustRootPath,
		credentialPath,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private fixture %s = %v/%v", path, info, err)
		}
	}
	verified, _, credential, err := localauthority.LoadFiles(
		context.Background(),
		authorityPath,
		trustRootPath,
		credentialPath,
		now,
	)
	if err != nil {
		t.Fatalf("load authority: %v", err)
	}
	defer clear(credential)
	document := verified.Document()
	if document.Attempt.AttemptID != attemptID ||
		!document.Attempt.AllowRead ||
		document.Attempt.AllowWrite ||
		document.Policy.Budgets.MaxSynchronousOverheadMs != 500 ||
		document.Policy.Budgets.MaxSynchronousOverheadRatio != 0.02 {
		t.Fatalf("authority selection = %+v", document)
	}

	logPath := filepath.Join(root, "requests.jsonl")
	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := &missHandler{
		authorityPath:  authorityPath,
		trustRootPath:  trustRootPath,
		credentialPath: credentialPath,
		bearer: base64.RawURLEncoding.EncodeToString(
			credential,
		),
		log: logFile,
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"http://127.0.0.1/cache/compile-key",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+handler.bearer)
	request.Header.Set(
		"X-BuildOpt-Authority-Digest",
		document.AuthorityDigest,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("authenticated miss status = %d", response.Code)
	}
	if err := logFile.Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(logPath)
	if err != nil ||
		!strings.Contains(string(content), `"outcome":"MISS"`) ||
		!strings.Contains(string(content), `"path":"/cache/compile-key"`) {
		t.Fatalf("miss log = %q/%v", content, err)
	}
}

func TestPrepareAuthorityRejectsRelativeRoot(t *testing.T) {
	if err := prepareAuthority(
		"relative",
		"11111111-1111-4111-8111-000000000001",
		time.Now().UTC(),
	); err == nil {
		t.Fatal("relative authority root passed")
	}
}
