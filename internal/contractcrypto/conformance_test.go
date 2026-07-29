package contractcrypto

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalJSONVectors(t *testing.T) {
	t.Parallel()

	rows := readTSV(
		t,
		filepath.Join(
			findContractRepositoryRoot(t),
			"contracts",
			"test-vectors",
			"canonical-json",
			"vectors.tsv",
		),
		5,
	)
	for _, row := range rows {
		row := row
		t.Run(row[0], func(t *testing.T) {
			t.Parallel()

			input := decodeBase64Field(t, row[0], row[2])
			canonical, err := CanonicalizeJCS(input)
			if row[1] == "INVALID" {
				if err == nil {
					t.Fatalf("invalid vector unexpectedly canonicalized to %s", canonical)
				}
				if !strings.Contains(err.Error(), row[3]) {
					t.Fatalf("error %q does not contain %q", err, row[3])
				}
				return
			}
			if err != nil {
				t.Fatalf("canonicalize: %v", err)
			}
			expected := decodeBase64Field(t, row[0], row[3])
			if !bytes.Equal(canonical, expected) {
				t.Fatalf("canonical bytes = %q, want %q", canonical, expected)
			}
			digest := sha256.Sum256(canonical)
			actualDigest := "sha256:" + hex.EncodeToString(digest[:])
			if actualDigest != row[4] {
				t.Fatalf("digest = %s, want %s", actualDigest, row[4])
			}
		})
	}
}

func TestUTCTimestampVectors(t *testing.T) {
	t.Parallel()

	rows := readTSV(
		t,
		filepath.Join(
			findContractRepositoryRoot(t),
			"contracts",
			"test-vectors",
			"canonical-json",
			"timestamps.tsv",
		),
		3,
	)
	for _, row := range rows {
		got := ValidUTCTimestamp(row[2])
		want := row[1] == "VALID"
		if got != want {
			t.Errorf("%s validity = %t, want %t", row[0], got, want)
		}
	}
}

func TestEd25519Vectors(t *testing.T) {
	t.Parallel()

	rows := readTSV(
		t,
		filepath.Join(
			findContractRepositoryRoot(t),
			"contracts",
			"test-vectors",
			"signatures",
			"ed25519.tsv",
		),
		5,
	)
	for _, row := range rows {
		publicKey, publicErr := hex.DecodeString(row[2])
		payload, payloadErr := base64.StdEncoding.DecodeString(row[3])
		signature, signatureErr := hex.DecodeString(row[4])
		valid := publicErr == nil &&
			payloadErr == nil &&
			signatureErr == nil &&
			len(publicKey) == ed25519.PublicKeySize &&
			len(signature) == ed25519.SignatureSize &&
			ed25519.Verify(publicKey, payload, signature)
		want := row[1] == "VALID"
		if valid != want {
			t.Errorf("%s verification = %t, want %t", row[0], valid, want)
		}
	}
}

func readTSV(t *testing.T, path string, fieldCount int) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()

	var rows [][]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != fieldCount {
			t.Fatalf("%s row has %d fields, want %d: %q", path, len(fields), fieldCount, line)
		}
		if fields[1] != "VALID" && fields[1] != "INVALID" {
			t.Fatalf("%s has unknown outcome %q", fields[0], fields[1])
		}
		rows = append(rows, fields)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s has no vectors", path)
	}
	return rows
}

func decodeBase64Field(t *testing.T, name string, value string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("%s base64: %v", name, err)
	}
	return decoded
}

func findContractRepositoryRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(current, "contracts", "test-vectors")); err == nil {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("repository root not found")
		}
		current = parent
	}
}

func ExampleCanonicalizeJCS() {
	canonical, err := CanonicalizeJCS([]byte(`{"z":2,"a":1}`))
	fmt.Println(string(canonical), err)
	// Output: {"a":1,"z":2} <nil>
}
