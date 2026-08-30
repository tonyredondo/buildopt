package requesthit_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/requesthit"
	"github.com/tonyredondo/buildopt/internal/requesthit/testkit"
)

func TestCompleteContractClassifiesWithoutAuthority(t *testing.T) {
	record := loadRecord(t)
	probe := requesthit.MatchingProbe(record)
	verdict := requesthit.Verify(record, probe, verificationTime())
	if verdict.Disposition != requesthit.DispositionContractComplete || verdict.Reason != requesthit.ReasonNone {
		t.Fatalf("verdict = %+v", verdict)
	}
	if verdict.SelectionAuthorized || verdict.ActivationAuthorized || verdict.PerformanceMeasured {
		t.Fatalf("VRH-002 invented authority: %+v", verdict)
	}
	if len(verdict.RecordSHA256) != 64 {
		t.Fatalf("record digest = %q", verdict.RecordSHA256)
	}
}

func TestNegativeMatrixRetainsNativeWithTypedReason(t *testing.T) {
	matrix := loadMatrix(t)
	if len(matrix.Cases) != 37 {
		t.Fatalf("negative case count = %d, want 37", len(matrix.Cases))
	}
	seen := map[string]bool{}
	for _, fixture := range matrix.Cases {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			if fixture.Name == "" || fixture.Mutation == "" || fixture.ExpectedReason == requesthit.ReasonNone || seen[fixture.Mutation] {
				t.Fatalf("invalid or duplicate fixture: %+v", fixture)
			}
			seen[fixture.Mutation] = true
			record := loadRecord(t)
			probe := requesthit.MatchingProbe(record)
			if err := testkit.Apply(&record, &probe, fixture.Mutation); err != nil {
				t.Fatal(err)
			}
			verdict := requesthit.Verify(record, probe, verificationTime())
			if verdict.Disposition != requesthit.DispositionRetainNative || verdict.Reason != fixture.ExpectedReason {
				t.Fatalf("verdict = %+v, want native retention %s", verdict, fixture.ExpectedReason)
			}
			if verdict.SelectionAuthorized || verdict.ActivationAuthorized || verdict.PerformanceMeasured {
				t.Fatalf("negative fixture invented authority: %+v", verdict)
			}
		})
	}
}

func TestRecordCanonicalizationAndStrictDecode(t *testing.T) {
	record := loadRecord(t)
	first, firstDigest, err := requesthit.CanonicalRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, second, secondDigest, err := requesthit.DecodeRecord(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || firstDigest != secondDigest || decoded.RecordID != record.RecordID {
		t.Fatalf("canonical round trip drifted")
	}
	unknown := append([]byte(nil), first[:len(first)-1]...)
	unknown = append(unknown, []byte(`,"unexpected":true}`)...)
	if _, _, _, err := requesthit.DecodeRecord(unknown); err == nil {
		t.Fatal("unknown field was accepted")
	}
}

func TestArgumentVectorFramingIsUnambiguous(t *testing.T) {
	if requesthit.DigestArgumentVector([]string{"ab", "c"}) == requesthit.DigestArgumentVector([]string{"a", "bc"}) {
		t.Fatal("argument framing collision")
	}
	if requesthit.DigestArgumentVector([]string{"", "a b", "*", "--"}) == requesthit.DigestArgumentVector([]string{"a b", "*", "--"}) {
		t.Fatal("empty argument was not framed")
	}
}

func loadRecord(t *testing.T) requesthit.SafetyRecord {
	t.Helper()
	raw := readFixture(t, "valid", "complete.json")
	record, _, _, err := requesthit.DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func loadMatrix(t *testing.T) testkit.Matrix {
	t.Helper()
	var matrix testkit.Matrix
	if err := json.Unmarshal(readFixture(t, "negative-matrix.json"), &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.SchemaVersion != "buildopt.poc/verified-request-hit-negative-matrix/v1" {
		t.Fatalf("matrix schema = %q", matrix.SchemaVersion)
	}
	return matrix
}

func readFixture(t *testing.T, elements ...string) []byte {
	t.Helper()
	_, current, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	path := filepath.Join(append([]string{root, "contracts", "jsonschema", "testdata", "verified-request-hit-safety-record.v1"}, elements...)...)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func verificationTime() time.Time {
	return time.Date(2026, 8, 30, 9, 0, 0, 0, time.UTC)
}
