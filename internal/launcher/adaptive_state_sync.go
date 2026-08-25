package launcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/adaptivefragment"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const adaptiveStateSyncSchema = "buildopt.poc/adaptive-state-sync-result/v1"

type adaptiveStateSyncResult struct {
	SchemaVersion        string            `json:"schemaVersion"`
	Operation            string            `json:"operation"`
	LocalStatus          string            `json:"localStatus"`
	LocalHeadSHA256      string            `json:"localHeadSha256,omitempty"`
	Central              centralSyncResult `json:"central"`
	RestoredFromCentral  bool              `json:"restoredFromCentral"`
	UsedVerifiedLocal    bool              `json:"usedVerifiedLocal"`
	NativeFallback       bool              `json:"nativeFallback"`
	Reason               string            `json:"reason"`
	ProductionAuthorized bool              `json:"productionAuthorized"`
	TestOptimization     string            `json:"testOptimization"`
}

// synchronizeAdaptiveState is the AF-012 adapter over the existing central
// state plane. It deliberately does not expose a public CLI; AF-014 owns the
// installed one-command flow after the longitudinal gate closes.
func synchronizeAdaptiveState(
	ctx context.Context,
	operation string,
	localRoot string,
	connectionDirectory string,
	connection centralConnection,
	client *centralStateClient,
	origin sharedcache.StateOrigin,
) (adaptiveStateSyncResult, error) {
	result := adaptiveStateSyncResult{
		SchemaVersion: adaptiveStateSyncSchema, Operation: operation,
		LocalStatus: "ABSENT", Reason: "NO_VERIFIED_ADAPTIVE_STATE",
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	local := map[sharedcache.StateKind]*centralLocalPublication{}
	snapshot, localErr := adaptivefragment.LoadLocalState(localRoot)
	switch {
	case localErr == nil:
		result.LocalStatus = "VALID"
		result.LocalHeadSHA256 = snapshot.HeadSHA256
		publications, err := collectAdaptiveStatePublications(snapshot, origin)
		if err != nil {
			result.LocalStatus = "CORRUPT"
			result.NativeFallback = true
			result.Reason = "LOCAL_PUBLICATION_REJECTED"
			return result, err
		}
		local = publications
	case errors.Is(localErr, adaptivefragment.ErrLocalStateAbsent):
		// A clean machine is allowed to pull the exact typed state.
	case localErr != nil:
		result.LocalStatus = "CORRUPT"
		result.NativeFallback = true
		result.Reason = "LOCAL_STATE_REJECTED"
		return result, localErr
	}

	central, syncErr := synchronizeCentralStateWithLocal(
		ctx, operation, connectionDirectory, connection, client, local, nil,
	)
	result.Central = central
	restored, current, restoreErr := restoreAdaptiveStateFromSnapshots(
		localRoot, connectionDirectory, connection.RepositoryScopeSHA256,
	)
	if restoreErr == nil {
		result.LocalStatus = "VALID"
		result.LocalHeadSHA256 = current.HeadSHA256
		result.RestoredFromCentral = restored
		result.UsedVerifiedLocal = true
		result.NativeFallback = false
		result.Reason = "VERIFIED_ADAPTIVE_STATE_AVAILABLE"
		return result, syncErr
	}
	if localErr == nil {
		current, reloadErr := adaptivefragment.LoadLocalState(localRoot)
		if reloadErr == nil && current.HeadSHA256 == snapshot.HeadSHA256 {
			result.LocalStatus = "VALID"
			result.LocalHeadSHA256 = current.HeadSHA256
			result.UsedVerifiedLocal = true
			result.NativeFallback = false
			result.Reason = "VERIFIED_LOCAL_STATE_RETAINED"
			return result, syncErr
		}
	}
	result.NativeFallback = true
	result.Reason = "NATIVE_FALLBACK_NO_VERIFIED_STATE"
	if syncErr != nil {
		return result, syncErr
	}
	return result, restoreErr
}

func collectAdaptiveStatePublications(
	snapshot adaptivefragment.LocalStateSnapshot,
	origin sharedcache.StateOrigin,
) (map[sharedcache.StateKind]*centralLocalPublication, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, snapshot.Head.CreatedAt)
	if err != nil {
		return nil, errors.New("adaptive state head has an invalid creation time")
	}
	evidenceFiles := map[string][]byte{}
	portfolioFiles := map[string][]byte{}
	for path, raw := range snapshot.Files {
		switch filepath.Base(filepath.FromSlash(path)) {
		case "head.json", "portfolio.json":
			portfolioFiles[path] = raw
		default:
			evidenceFiles[path] = raw
		}
	}
	evidence, err := newCentralLocalPublication(
		sharedcache.StateKindEvidence, snapshot.Head.RepositoryScopeSHA256,
		snapshot.Head.CompatibilitySHA256, snapshot.Head.BindingsSHA256,
		origin, createdAt, evidenceFiles,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare adaptive evidence publication: %w", err)
	}
	portfolio, err := newCentralLocalPublication(
		sharedcache.StateKindPortfolio, snapshot.Head.RepositoryScopeSHA256,
		snapshot.Head.CompatibilitySHA256, snapshot.Head.BindingsSHA256,
		origin, createdAt, portfolioFiles,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare adaptive portfolio publication: %w", err)
	}
	return map[sharedcache.StateKind]*centralLocalPublication{
		sharedcache.StateKindEvidence:  evidence,
		sharedcache.StateKindPortfolio: portfolio,
	}, nil
}

func restoreAdaptiveStateFromSnapshots(
	localRoot string,
	connectionDirectory string,
	repositoryScope string,
) (bool, adaptivefragment.LocalStateSnapshot, error) {
	evidence, err := loadCentralSnapshot(connectionDirectory, repositoryScope, sharedcache.StateKindEvidence)
	if err != nil {
		return false, adaptivefragment.LocalStateSnapshot{}, err
	}
	portfolio, err := loadCentralSnapshot(connectionDirectory, repositoryScope, sharedcache.StateKindPortfolio)
	if err != nil {
		return false, adaptivefragment.LocalStateSnapshot{}, err
	}
	if !centralPortfolioReferencesEvidence(portfolio, evidence.manifestSHA256) ||
		evidence.bundle.CompatibilitySHA256 != portfolio.bundle.CompatibilitySHA256 ||
		evidence.bundle.BindingsSHA256 != portfolio.bundle.BindingsSHA256 {
		return false, adaptivefragment.LocalStateSnapshot{}, errors.New("adaptive central snapshots are not one linked generation")
	}
	bundle, transportedFiles, err := decodeAdaptiveStateSnapshots(evidence.bundle, portfolio.bundle)
	if err != nil {
		return false, adaptivefragment.LocalStateSnapshot{}, err
	}
	prepared, err := adaptivefragment.PrepareLocalState(bundle)
	if err != nil || !equalAdaptiveStateFiles(prepared.Files, transportedFiles) {
		return false, adaptivefragment.LocalStateSnapshot{}, errors.New("adaptive central state failed exact local-head verification")
	}
	current, loadErr := adaptivefragment.LoadLocalState(localRoot)
	if errors.Is(loadErr, adaptivefragment.ErrLocalStateAbsent) {
		restored, restoreErr := adaptivefragment.RestoreLocalState(localRoot, bundle)
		return restoreErr == nil, restored, restoreErr
	}
	if loadErr != nil {
		return false, adaptivefragment.LocalStateSnapshot{}, loadErr
	}
	if current.HeadSHA256 == prepared.HeadSHA256 {
		return false, current, nil
	}
	if prepared.Head.PortfolioGeneration == current.Head.PortfolioGeneration+1 &&
		prepared.Head.LedgerGeneration == current.Head.LedgerGeneration+1 {
		next, saveErr := adaptivefragment.SaveLocalState(localRoot, bundle, current.HeadSHA256)
		return saveErr == nil, next, saveErr
	}
	return false, adaptivefragment.LocalStateSnapshot{}, errors.New("adaptive central state is not the next local generation")
}

func decodeAdaptiveStateSnapshots(
	evidence centralStateBundle,
	portfolio centralStateBundle,
) (adaptivefragment.StateBundle, map[string][]byte, error) {
	files := map[string][]byte{}
	for _, bundle := range []centralStateBundle{evidence, portfolio} {
		for _, file := range bundle.Files {
			raw, err := base64.RawStdEncoding.DecodeString(file.ContentBase64)
			if err != nil {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central file encoding is invalid")
			}
			if _, exists := files[file.Path]; exists {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central file path is duplicated")
			}
			files[file.Path] = raw
		}
	}
	var bundle adaptivefragment.StateBundle
	observations := map[string]adaptivefragment.Observation{}
	seen := map[string]bool{}
	for path, raw := range files {
		base := filepath.Base(filepath.FromSlash(path))
		switch base {
		case "fragment.json":
			if seen[base] || decodeAdaptiveStrict(raw, &bundle.Fragment) != nil {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central fragment is invalid")
			}
			seen[base] = true
		case "portfolio.json":
			if seen[base] || decodeAdaptiveStrict(raw, &bundle.Portfolio) != nil {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central portfolio is invalid")
			}
			seen[base] = true
		case "ledger.json":
			if seen[base] || decodeAdaptiveStrict(raw, &bundle.Ledger) != nil {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central ledger is invalid")
			}
			seen[base] = true
		case "head.json":
			if seen[base] {
				return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central head is duplicated")
			}
			seen[base] = true
		default:
			if strings.Contains(path, "/observations/") {
				var observation adaptivefragment.Observation
				if decodeAdaptiveStrict(raw, &observation) != nil {
					return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central observation is invalid")
				}
				observations[path] = observation
				continue
			}
			return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central state contains an unknown file")
		}
	}
	paths := make([]string, 0, len(observations))
	for path := range observations {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		bundle.Observations = append(bundle.Observations, observations[path])
	}
	if !seen["head.json"] || adaptivefragment.ValidateStateBundle(bundle) != nil {
		return adaptivefragment.StateBundle{}, nil, errors.New("adaptive central state bundle is incomplete or invalid")
	}
	return bundle, files, nil
}

func equalAdaptiveStateFiles(expected, actual map[string][]byte) bool {
	if len(expected) != len(actual) {
		return false
	}
	for path, raw := range expected {
		if !bytes.Equal(raw, actual[path]) {
			return false
		}
	}
	return true
}

func decodeAdaptiveStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("adaptive central document contains trailing data")
	}
	return nil
}
