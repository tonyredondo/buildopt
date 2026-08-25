package adaptivefragment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonyredondo/buildopt/internal/contractcrypto"
	"github.com/tonyredondo/buildopt/internal/filelock"
)

const (
	// LocalHeadSchemaVersion identifies the local-first AF-012 head document.
	LocalHeadSchemaVersion = "buildopt.adaptive/local-state-head/v1"
	localHeadFile          = "head.json"
	localLockFile          = "state.lock"
	localGenerationsDir    = "generations"
	localMaximumDocument   = 1 << 20
)

var (
	// ErrLocalStateAbsent means no adaptive state has been persisted yet.
	ErrLocalStateAbsent = errors.New("adaptive local state is absent")
	// ErrLocalStateConflict means the expected head no longer owns the store.
	ErrLocalStateConflict = errors.New("adaptive local state head conflicts")
	// ErrLocalStateCorrupt means persisted bytes or links failed verification.
	ErrLocalStateCorrupt = errors.New("adaptive local state is corrupt")
	// ErrLocalStateBusy means another process currently owns the state writer.
	ErrLocalStateBusy = errors.New("adaptive local state writer is busy")
)

// LocalStateHead is the sole mutable pointer in one local adaptive-state
// store. Every referenced document remains immutable and content addressed.
type LocalStateHead struct {
	SchemaVersion         string   `json:"schemaVersion"`
	RecordType            string   `json:"recordType"`
	RepositoryScopeSHA256 string   `json:"repositoryScopeSha256"`
	PortfolioGeneration   uint64   `json:"portfolioGeneration"`
	LedgerGeneration      uint64   `json:"ledgerGeneration"`
	GenerationDirectory   string   `json:"generationDirectory"`
	FragmentSHA256        string   `json:"fragmentSha256"`
	ObservationSHA256     []string `json:"observationSha256"`
	PortfolioSHA256       string   `json:"portfolioSha256"`
	LedgerSHA256          string   `json:"ledgerSha256"`
	CompatibilitySHA256   string   `json:"compatibilitySha256"`
	BindingsSHA256        string   `json:"bindingsSha256"`
	CreatedAt             string   `json:"createdAt"`
	ProductionAuthorized  bool     `json:"productionAuthorized"`
	TestOptimization      string   `json:"testOptimization"`
}

// LocalStateSnapshot is one fully verified local generation and its exact
// canonical source bytes. Callers may transport the bytes but must revalidate
// the bundle before granting any activation authority.
type LocalStateSnapshot struct {
	Head       LocalStateHead
	HeadSHA256 string
	Bundle     StateBundle
	Files      map[string][]byte
}

// PrepareLocalState validates one bundle and returns the exact portable bytes
// that SaveLocalState would persist, without touching the filesystem. Remote
// consumers use it to verify a transported head before restoring any state.
func PrepareLocalState(bundle StateBundle) (LocalStateSnapshot, error) {
	if err := ValidateStateBundle(bundle); err != nil {
		return LocalStateSnapshot{}, fmt.Errorf("validate adaptive state before preparation: %w", err)
	}
	return buildLocalStateSnapshot(bundle)
}

// SaveLocalState validates and atomically persists one immutable state bundle.
// expectedHeadSHA256 implements optimistic concurrency; an empty value is
// accepted only for the first generation.
func SaveLocalState(root string, bundle StateBundle, expectedHeadSHA256 string) (LocalStateSnapshot, error) {
	if err := ValidateStateBundle(bundle); err != nil {
		return LocalStateSnapshot{}, fmt.Errorf("validate adaptive state before persistence: %w", err)
	}
	if err := ensureLocalStateDirectory(root); err != nil {
		return LocalStateSnapshot{}, err
	}
	lock, err := openLocalStateLock(root)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Exclusive); err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return LocalStateSnapshot{}, ErrLocalStateBusy
		}
		return LocalStateSnapshot{}, err
	}
	defer filelock.Unlock(lock)

	current, currentErr := loadLocalStateUnlocked(root)
	if currentErr != nil && !errors.Is(currentErr, ErrLocalStateAbsent) {
		return LocalStateSnapshot{}, currentErr
	}
	if errors.Is(currentErr, ErrLocalStateAbsent) {
		if expectedHeadSHA256 != "" || bundle.Portfolio.Generation != 1 || bundle.Ledger.Generation != 1 {
			return LocalStateSnapshot{}, ErrLocalStateConflict
		}
	} else {
		if expectedHeadSHA256 != current.HeadSHA256 {
			return LocalStateSnapshot{}, ErrLocalStateConflict
		}
	}

	snapshot, err := buildLocalStateSnapshot(bundle)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	if currentErr == nil && snapshot.HeadSHA256 == current.HeadSHA256 {
		return current, nil
	}
	if currentErr == nil {
		if bundle.Portfolio.Generation != current.Head.PortfolioGeneration+1 ||
			bundle.Ledger.Generation != current.Head.LedgerGeneration+1 ||
			bundle.Portfolio.SupersedesSHA256 != current.Head.PortfolioSHA256 ||
			bundle.Ledger.SupersedesSHA256 != current.Head.LedgerSHA256 {
			return LocalStateSnapshot{}, ErrLocalStateConflict
		}
	}
	if err := persistLocalStateSnapshot(root, snapshot); err != nil {
		return LocalStateSnapshot{}, err
	}
	return loadLocalStateUnlocked(root)
}

// LoadLocalState returns one fully verified local adaptive-state generation.
func LoadLocalState(root string) (LocalStateSnapshot, error) {
	if err := ensureLocalStateDirectory(root); err != nil {
		return LocalStateSnapshot{}, err
	}
	lock, err := openLocalStateLock(root)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	defer lock.Close()
	if err := filelock.Try(lock, filelock.Shared); err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return LocalStateSnapshot{}, ErrLocalStateBusy
		}
		return LocalStateSnapshot{}, err
	}
	defer filelock.Unlock(lock)
	return loadLocalStateUnlocked(root)
}

// RestoreLocalState persists a verified remote generation into an empty local
// store. Existing local state is never overwritten implicitly.
func RestoreLocalState(root string, bundle StateBundle) (LocalStateSnapshot, error) {
	return SaveLocalState(root, bundle, "")
}

func buildLocalStateSnapshot(bundle StateBundle) (LocalStateSnapshot, error) {
	fragmentRaw, err := MarshalCanonicalDocument(bundle.Fragment)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	portfolioRaw, err := MarshalCanonicalDocument(bundle.Portfolio)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	ledgerRaw, err := MarshalCanonicalDocument(bundle.Ledger)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	fragmentSHA := digestCanonicalBytes(fragmentRaw)
	portfolioSHA := digestCanonicalBytes(portfolioRaw)
	ledgerSHA := digestCanonicalBytes(ledgerRaw)
	generationDirectory := localGenerationDirectory(bundle.Portfolio.Generation, portfolioSHA)
	files := map[string][]byte{
		filepath.ToSlash(filepath.Join(generationDirectory, "fragment.json")):  fragmentRaw,
		filepath.ToSlash(filepath.Join(generationDirectory, "portfolio.json")): portfolioRaw,
		filepath.ToSlash(filepath.Join(generationDirectory, "ledger.json")):    ledgerRaw,
	}
	observationSHA := make([]string, 0, len(bundle.Observations))
	for index, observation := range bundle.Observations {
		raw, marshalErr := MarshalCanonicalDocument(observation)
		if marshalErr != nil {
			return LocalStateSnapshot{}, marshalErr
		}
		sha := digestCanonicalBytes(raw)
		observationSHA = append(observationSHA, sha)
		name := fmt.Sprintf("%020d-%s.json", index+1, sha)
		files[filepath.ToSlash(filepath.Join(generationDirectory, "observations", name))] = raw
	}
	compatibilityRaw, err := MarshalCanonicalDocument(struct {
		Domain                string                `json:"domain"`
		RepositoryScopeSHA256 string                `json:"repositoryScopeSha256"`
		FamilyID              string                `json:"familyId"`
		RevisionID            string                `json:"revisionId"`
		Kind                  Kind                  `json:"kind"`
		SelectorSHA256        string                `json:"selectorSha256"`
		AuthoritySHA256       string                `json:"authoritySha256"`
		Bindings              map[BindingKey]string `json:"bindings"`
	}{
		Domain:                "buildopt-adaptive-local-compatibility-v1",
		RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
		FamilyID:              bundle.Fragment.FamilyID, RevisionID: bundle.Fragment.RevisionID,
		Kind: bundle.Fragment.Kind, SelectorSHA256: bundle.Fragment.SelectorSHA256,
		AuthoritySHA256: bundle.Fragment.AuthoritySHA256, Bindings: bundle.Fragment.Bindings,
	})
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	bindingsRaw, err := MarshalCanonicalDocument(struct {
		Domain            string   `json:"domain"`
		FragmentSHA256    string   `json:"fragmentSha256"`
		ObservationSHA256 []string `json:"observationSha256"`
		PortfolioSHA256   string   `json:"portfolioSha256"`
		LedgerSHA256      string   `json:"ledgerSha256"`
	}{
		Domain: "buildopt-adaptive-local-bindings-v1", FragmentSHA256: fragmentSHA,
		ObservationSHA256: observationSHA, PortfolioSHA256: portfolioSHA, LedgerSHA256: ledgerSHA,
	})
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	head := LocalStateHead{
		SchemaVersion: LocalHeadSchemaVersion, RecordType: "ADAPTIVE_LOCAL_STATE_HEAD",
		RepositoryScopeSHA256: bundle.Fragment.RepositoryScopeSHA256,
		PortfolioGeneration:   bundle.Portfolio.Generation, LedgerGeneration: bundle.Ledger.Generation,
		GenerationDirectory: generationDirectory, FragmentSHA256: fragmentSHA,
		ObservationSHA256: observationSHA, PortfolioSHA256: portfolioSHA, LedgerSHA256: ledgerSHA,
		CompatibilitySHA256: digestCanonicalBytes(compatibilityRaw),
		BindingsSHA256:      digestCanonicalBytes(bindingsRaw), CreatedAt: bundle.Portfolio.CreatedAt,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	headRaw, err := MarshalCanonicalDocument(head)
	if err != nil {
		return LocalStateSnapshot{}, err
	}
	files[localHeadFile] = headRaw
	return LocalStateSnapshot{
		Head: head, HeadSHA256: digestCanonicalBytes(headRaw), Bundle: bundle, Files: files,
	}, nil
}

func persistLocalStateSnapshot(root string, snapshot LocalStateSnapshot) error {
	paths := make([]string, 0, len(snapshot.Files))
	for path := range snapshot.Files {
		if path != localHeadFile {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := writeImmutableLocalFile(path, snapshot.Files[relative]); err != nil {
			return err
		}
	}
	return writeAtomicLocalFile(filepath.Join(root, localHeadFile), snapshot.Files[localHeadFile])
}

func loadLocalStateUnlocked(root string) (LocalStateSnapshot, error) {
	headRaw, err := readLocalStateFile(filepath.Join(root, localHeadFile))
	if errors.Is(err, os.ErrNotExist) {
		return LocalStateSnapshot{}, ErrLocalStateAbsent
	}
	if err != nil || !canonicalJSONBytes(headRaw) {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	var head LocalStateHead
	if decodeStrictLocalJSON(headRaw, &head) != nil || validateLocalHead(head) != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	files := map[string][]byte{localHeadFile: headRaw}
	fragmentPath := filepath.Join(root, filepath.FromSlash(head.GenerationDirectory), "fragment.json")
	portfolioPath := filepath.Join(root, filepath.FromSlash(head.GenerationDirectory), "portfolio.json")
	ledgerPath := filepath.Join(root, filepath.FromSlash(head.GenerationDirectory), "ledger.json")
	fragmentRaw, err := readAndVerifyLocalDocument(fragmentPath, head.FragmentSHA256)
	if err != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	portfolioRaw, err := readAndVerifyLocalDocument(portfolioPath, head.PortfolioSHA256)
	if err != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	ledgerRaw, err := readAndVerifyLocalDocument(ledgerPath, head.LedgerSHA256)
	if err != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	files[localRelative(root, fragmentPath)] = fragmentRaw
	files[localRelative(root, portfolioPath)] = portfolioRaw
	files[localRelative(root, ledgerPath)] = ledgerRaw
	observations := make([]Observation, 0, len(head.ObservationSHA256))
	observationDirectory := filepath.Join(root, filepath.FromSlash(head.GenerationDirectory), "observations")
	entries, err := os.ReadDir(observationDirectory)
	if err != nil || len(entries) != len(head.ObservationSHA256) {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	for index, expectedSHA := range head.ObservationSHA256 {
		prefix := fmt.Sprintf("%020d-%s.json", index+1, expectedSHA)
		path := filepath.Join(observationDirectory, prefix)
		raw, readErr := readAndVerifyLocalDocument(path, expectedSHA)
		if readErr != nil {
			return LocalStateSnapshot{}, ErrLocalStateCorrupt
		}
		var observation Observation
		if decodeStrictLocalJSON(raw, &observation) != nil {
			return LocalStateSnapshot{}, ErrLocalStateCorrupt
		}
		observations = append(observations, observation)
		files[localRelative(root, path)] = raw
	}
	var fragment PersistedFragment
	var portfolio Portfolio
	var ledger EconomicLedger
	if decodeStrictLocalJSON(fragmentRaw, &fragment) != nil ||
		decodeStrictLocalJSON(portfolioRaw, &portfolio) != nil ||
		decodeStrictLocalJSON(ledgerRaw, &ledger) != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	bundle := StateBundle{Fragment: fragment, Observations: observations, Portfolio: portfolio, Ledger: ledger}
	if ValidateStateBundle(bundle) != nil {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	rebuilt, err := buildLocalStateSnapshot(bundle)
	if err != nil || rebuilt.HeadSHA256 != digestCanonicalBytes(headRaw) ||
		!bytes.Equal(rebuilt.Files[localHeadFile], headRaw) {
		return LocalStateSnapshot{}, ErrLocalStateCorrupt
	}
	// Preserve the exact persisted bytes after reconstructing the typed state.
	rebuilt.Head = head
	rebuilt.HeadSHA256 = digestCanonicalBytes(headRaw)
	rebuilt.Files = files
	return rebuilt, nil
}

func validateLocalHead(head LocalStateHead) error {
	if head.SchemaVersion != LocalHeadSchemaVersion || head.RecordType != "ADAPTIVE_LOCAL_STATE_HEAD" ||
		head.PortfolioGeneration == 0 || head.LedgerGeneration == 0 ||
		!validSHA(head.RepositoryScopeSHA256) || !validSHA(head.FragmentSHA256) ||
		!validSHA(head.PortfolioSHA256) || !validSHA(head.LedgerSHA256) ||
		!validSHA(head.CompatibilitySHA256) || !validSHA(head.BindingsSHA256) ||
		head.ProductionAuthorized || head.TestOptimization != "OUT_OF_SCOPE" ||
		len(head.ObservationSHA256) == 0 {
		return ErrLocalStateCorrupt
	}
	for _, digest := range head.ObservationSHA256 {
		if !validSHA(digest) {
			return ErrLocalStateCorrupt
		}
	}
	expectedDirectory := localGenerationDirectory(head.PortfolioGeneration, head.PortfolioSHA256)
	if head.GenerationDirectory != expectedDirectory || strings.Contains(head.GenerationDirectory, "..") {
		return ErrLocalStateCorrupt
	}
	return nil
}

func localGenerationDirectory(generation uint64, portfolioSHA string) string {
	return filepath.ToSlash(filepath.Join(
		localGenerationsDir,
		"generation-"+fmt.Sprintf("%020d", generation)+"-"+portfolioSHA,
	))
}

func ensureLocalStateDirectory(root string) error {
	if root == "" || !filepath.IsAbs(root) {
		return errors.New("adaptive local state root must be absolute")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(root)
	if err != nil || !privateLocalDirectoryInfo(info) {
		return errors.New("adaptive local state root is unsafe")
	}
	return nil
}

func openLocalStateLock(root string) (*os.File, error) {
	return openPrivateLocalLock(filepath.Join(root, localLockFile))
}

func writeImmutableLocalFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, raw) {
			return nil
		}
		return ErrLocalStateConflict
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeAtomicLocalFile(path, raw)
}

func writeAtomicLocalFile(path string, raw []byte) error {
	if len(raw) < 1 || len(raw) > localMaximumDocument {
		return errors.New("adaptive local state document size is invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".buildopt-adaptive-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceLocalStateFile(temporaryPath, path); err != nil {
		return err
	}
	return syncLocalStateDirectory(filepath.Dir(path))
}

func readAndVerifyLocalDocument(path, expectedSHA string) ([]byte, error) {
	raw, err := readLocalStateFile(path)
	if err != nil || !canonicalJSONBytes(raw) || digestCanonicalBytes(raw) != expectedSHA {
		return nil, ErrLocalStateCorrupt
	}
	return raw, nil
}

func readLocalStateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !privateLocalFileInfo(info) || info.Size() < 1 || info.Size() > localMaximumDocument {
		return nil, ErrLocalStateCorrupt
	}
	return os.ReadFile(path)
}

func canonicalJSONBytes(raw []byte) bool {
	canonical, err := contractcrypto.CanonicalizeJCS(raw)
	return err == nil && bytes.Equal(canonical, raw)
}

func decodeStrictLocalJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("adaptive local state contains a trailing JSON value")
	}
	return nil
}

func digestCanonicalBytes(raw []byte) string {
	digest, _ := CanonicalDocumentSHA256(raw)
	return digest
}

func localRelative(root, path string) string {
	relative, _ := filepath.Rel(root, path)
	return filepath.ToSlash(relative)
}
