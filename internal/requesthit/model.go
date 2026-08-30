// Package requesthit defines the fail-closed safety contract for satisfying a
// previously successful Gradle request without starting Gradle. It owns only
// canonical evidence and verification; selection, materialization, execution,
// and performance measurement remain separate later POC blocks.
package requesthit

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
)

const (
	SchemaVersion = "buildopt.poc/verified-request-hit-safety-record/v1"
	RecordType    = "VERIFIED_REQUEST_HIT_SAFETY_RECORD"

	ArgumentEncoding = "UINT64_BE_LENGTH_PREFIXED_UTF8_V1"

	DispositionContractComplete = "SAFETY_CONTRACT_COMPLETE"
	DispositionRetainNative     = "RETAIN_NATIVE_GRADLE"
)

// SafetyRecord is immutable evidence captured from one successful native
// Gradle request. Digest fields bind complete inventories rather than samples.
type SafetyRecord struct {
	SchemaVersion   string           `json:"schemaVersion"`
	RecordType      string           `json:"recordType"`
	RecordID        string           `json:"recordId"`
	CapturedAt      string           `json:"capturedAt"`
	ExpiresAt       string           `json:"expiresAt"`
	RevocationEpoch int64            `json:"revocationEpoch"`
	Request         RequestBinding   `json:"request"`
	Execution       ExecutionBinding `json:"execution"`
	Inputs          InputBinding     `json:"inputs"`
	Outputs         OutputContract   `json:"outputs"`
	Tasks           []TaskSafety     `json:"tasks"`
	PriorResult     PriorResult      `json:"priorResult"`
}

// RequestBinding preserves the exact length-framed argument vector while
// keeping raw command arguments out of durable evidence.
type RequestBinding struct {
	ArgumentEncoding         string `json:"argumentEncoding"`
	ArgumentCount            int    `json:"argumentCount"`
	ArgumentsSHA256          string `json:"argumentsSha256"`
	WorkingDirectory         string `json:"workingDirectory"`
	RepositoryScopeSHA256    string `json:"repositoryScopeSha256"`
	RepositoryIdentitySHA256 string `json:"repositoryIdentitySha256"`
}

// ExecutionBinding binds every toolchain and finalized build-model fact that
// may change the meaning of the same command.
type ExecutionBinding struct {
	WrapperSHA256            string `json:"wrapperSha256"`
	GradleVersion            string `json:"gradleVersion"`
	GradleDistributionSHA256 string `json:"gradleDistributionSha256"`
	JDKVendor                string `json:"jdkVendor"`
	JDKVersion               string `json:"jdkVersion"`
	JDKRuntimeSHA256         string `json:"jdkRuntimeSha256"`
	SafeEnvironmentSHA256    string `json:"safeEnvironmentSha256"`
	RequestGraphSHA256       string `json:"requestGraphSha256"`
	TaskImplementationSHA256 string `json:"taskImplementationSha256"`
	BuildLogicSHA256         string `json:"buildLogicSha256"`
}

// InputBinding represents complete transitive repository and external input
// inventories. Completeness flags are explicit so absence cannot mean empty.
type InputBinding struct {
	RepositoryInputsComplete bool            `json:"repositoryInputsComplete"`
	RepositoryManifestSHA256 string          `json:"repositoryManifestSha256"`
	ExternalInputsComplete   bool            `json:"externalInputsComplete"`
	ExternalInputs           []ExternalInput `json:"externalInputs"`
}

// ExternalInput is one immutable non-repository input identity included in the
// complete transitive request inventory.
type ExternalInput struct {
	Kind     string `json:"kind"`
	Identity string `json:"identity"`
	Present  bool   `json:"present"`
	SHA256   string `json:"sha256"`
}

// OutputContract lists both present and expected-absent repository-relative
// output states. Present states point to immutable content-addressed bytes.
type OutputContract struct {
	Complete bool          `json:"complete"`
	States   []OutputState `json:"states"`
}

// OutputState records either exact present bytes or explicit absence. Present
// states must carry a content-addressed materialization reference.
type OutputState struct {
	Path               string `json:"path"`
	Kind               string `json:"kind"`
	Exists             bool   `json:"exists"`
	Tracked            bool   `json:"tracked"`
	SHA256             string `json:"sha256,omitempty"`
	Size               int64  `json:"size"`
	Mode               uint32 `json:"mode"`
	MaterializationRef string `json:"materializationRef,omitempty"`
}

// TaskSafety makes unsafe Gradle semantics explicit. Every admitted task must
// be tracked and cacheable and must declare none of the unsafe boundaries.
type TaskSafety struct {
	Path            string `json:"path"`
	Cacheable       bool   `json:"cacheable"`
	Tracked         bool   `json:"tracked"`
	AlwaysRun       bool   `json:"alwaysRun"`
	LocalState      bool   `json:"localState"`
	Destroyables    bool   `json:"destroyables"`
	UntrackedWrites bool   `json:"untrackedWrites"`
	SideEffects     bool   `json:"sideEffects"`
}

// PriorResult binds the successful native outcome that a future hit would
// return. VRH-002 admits only verified success with exit code zero.
type PriorResult struct {
	Outcome         string `json:"outcome"`
	ExitCode        int    `json:"exitCode"`
	OutputsVerified bool   `json:"outputsVerified"`
}

// Probe contains current pre-execution facts. Workspace outputs may be absent
// only when their exact materialization objects are still available.
type Probe struct {
	EvidenceComplete       bool             `json:"evidenceComplete"`
	CurrentRevocationEpoch int64            `json:"currentRevocationEpoch"`
	Request                RequestBinding   `json:"request"`
	Execution              ExecutionBinding `json:"execution"`
	Inputs                 InputBinding     `json:"inputs"`
	Outputs                []ObservedOutput `json:"outputs"`
	Tasks                  []TaskSafety     `json:"tasks"`
}

// ObservedOutput is the current workspace/materialization state for one
// contract path. Verification never writes or repairs either location.
type ObservedOutput struct {
	Path                     string `json:"path"`
	WorkspaceExists          bool   `json:"workspaceExists"`
	WorkspaceSHA256          string `json:"workspaceSha256,omitempty"`
	MaterializationAvailable bool   `json:"materializationAvailable"`
	MaterializationRef       string `json:"materializationRef,omitempty"`
	MaterializationSHA256    string `json:"materializationSha256,omitempty"`
}

// Verdict is classification evidence only. The three authority fields remain
// false in VRH-002 even when the safety contract is complete.
type Verdict struct {
	Disposition          string `json:"disposition"`
	Reason               Reason `json:"reason"`
	RecordSHA256         string `json:"recordSha256"`
	SelectionAuthorized  bool   `json:"selectionAuthorized"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	PerformanceMeasured  bool   `json:"performanceMeasured"`
}

// CanonicalRecord returns RFC 8785 bytes and the bare SHA-256 identity.
func CanonicalRecord(record SafetyRecord) ([]byte, string, error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, "", err
	}
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(canonical)
	return canonical, hex.EncodeToString(digest[:]), nil
}

// DecodeRecord rejects unknown fields and trailing JSON before normalizing the
// record to its canonical representation.
func DecodeRecord(raw []byte) (SafetyRecord, []byte, string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var record SafetyRecord
	if err := decoder.Decode(&record); err != nil {
		return SafetyRecord{}, nil, "", fmt.Errorf("decode safety record: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return SafetyRecord{}, nil, "", errors.New("decode safety record: trailing JSON value")
	}
	canonical, digest, err := CanonicalRecord(record)
	return record, canonical, digest, err
}

// DigestArgumentVector hashes an unambiguous sequence of uint64-length-framed
// UTF-8 argument bytes. It intentionally does not invoke a shell or normalize
// whitespace, empty values, quoting, or Unicode.
func DigestArgumentVector(arguments []string) string {
	hash := sha256.New()
	hash.Write([]byte("buildopt-request-argv-v1\x00"))
	var length [8]byte
	for _, argument := range arguments {
		binary.BigEndian.PutUint64(length[:], uint64(len([]byte(argument))))
		hash.Write(length[:])
		hash.Write([]byte(argument))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func parseTimestamp(value string) (time.Time, bool) {
	if !contractcrypto.ValidUTCTimestamp(value) {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}
