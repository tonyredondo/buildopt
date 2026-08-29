package launcher

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/tonyredondo/buildopt/internal/requestaligned"
	"github.com/tonyredondo/buildopt/internal/requestportfolio"
)

const maximumRequestPortfolioCaptureBytes = 64 << 20

//go:embed request_portfolio.init.gradle
var requestPortfolioInitScript []byte

// prepareChild appends an internal init script to the same Gradle invocation
// that the wrapper already received. The durable request identity was derived
// before this method runs, so the observer cannot replace or widen the user's
// command. Observation setup is fail-open and never starts another build.
func (state *requestPortfolioState) prepareChild(childArgs []string) []string {
	if state == nil || state.evidencePath == "" || state.bypassed || state.capturePrepared || state.captureErr != nil {
		return childArgs
	}
	arguments, err := json.Marshal(childArgs[1:])
	if err != nil {
		state.captureErr = fmt.Errorf("encode observed request arguments: %w", err)
		return childArgs
	}
	capturePath := os.Getenv(requestPortfolioCaptureEnvironment)
	state.preserveCapture = capturePath != ""
	if capturePath == "" {
		capturePath = state.evidencePath + ".capture.json"
	}
	initScriptPath := state.evidencePath + ".init.gradle"
	for _, path := range []string{state.evidencePath, capturePath, initScriptPath} {
		if err := prepareNewPrivateObservationPath(path); err != nil {
			state.captureErr = err
			return childArgs
		}
	}
	file, err := os.OpenFile(initScriptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		state.captureErr = fmt.Errorf("create observed request init script: %w", err)
		return childArgs
	}
	if _, err := file.Write(requestPortfolioInitScript); err != nil {
		_ = file.Close()
		_ = os.Remove(initScriptPath)
		state.captureErr = fmt.Errorf("write observed request init script: %w", err)
		return childArgs
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(initScriptPath)
		state.captureErr = fmt.Errorf("sync observed request init script: %w", err)
		return childArgs
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(initScriptPath)
		state.captureErr = fmt.Errorf("close observed request init script: %w", err)
		return childArgs
	}
	state.capturePath = capturePath
	state.initScriptPath = initScriptPath
	state.captureArguments = string(arguments)
	state.capturePrepared = true
	prepared := append([]string(nil), childArgs...)
	return append(prepared, "--init-script", initScriptPath)
}

func prepareNewPrivateObservationPath(path string) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("observed request capture paths must be clean and absolute")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create observed request capture directory: %w", err)
	}
	info, err := os.Lstat(parent)
	if err != nil || !privateManagedDirectoryInfo(info) {
		return errors.New("observed request capture directory is not private")
	}
	if _, err := os.Lstat(path); err == nil {
		return errors.New("observed request capture path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect observed request capture path: %w", err)
	}
	return nil
}

func (state *requestPortfolioState) materializeEvidence() error {
	if state == nil || !state.capturePrepared {
		return nil
	}
	capture, err := loadRequestPortfolioCapture(state.capturePath)
	if err != nil {
		return err
	}
	observation, err := requestaligned.Produce(capture)
	if err != nil {
		return fmt.Errorf("produce observed request evidence: %w", err)
	}
	if observation.Status != requestaligned.StatusComplete || observation.CompatibilityIdentitySHA256 == "" ||
		observation.RequestGraphIdentitySHA256 == "" {
		return fmt.Errorf("observed request evidence is %s: %s", observation.Status, observation.Reason)
	}
	requestedTasks := append([]string(nil), observation.RequestedTasks...)
	sort.Strings(requestedTasks)
	return requestportfolio.WriteEvidence(state.evidencePath, requestportfolio.Evidence{
		SchemaVersion:               requestportfolio.EvidenceSchemaVersion,
		ObservationID:               state.observationID,
		ArgumentsSHA256:             state.argumentsSHA,
		CompatibilityIdentitySHA256: observation.CompatibilityIdentitySHA256,
		RequestedTasks:              requestedTasks,
		RequestGraphIdentitySHA256:  observation.RequestGraphIdentitySHA256,
	})
}

func (state *requestPortfolioState) cleanupCaptureArtifacts() {
	if state == nil || !state.capturePrepared {
		return
	}
	_ = os.Remove(state.initScriptPath)
	if !state.preserveCapture {
		_ = os.Remove(state.capturePath)
	}
}

func loadRequestPortfolioCapture(path string) (requestaligned.Capture, error) {
	info, err := os.Lstat(path)
	if err != nil || !privateGatewayFileInfo(info) || info.Size() < 1 || info.Size() > maximumRequestPortfolioCaptureBytes {
		return requestaligned.Capture{}, errors.New("observed request capture file is unsafe or too large")
	}
	file, err := os.Open(path)
	if err != nil {
		return requestaligned.Capture{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return requestaligned.Capture{}, errors.New("observed request capture file changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maximumRequestPortfolioCaptureBytes+1))
	if err != nil || len(raw) > maximumRequestPortfolioCaptureBytes {
		return requestaligned.Capture{}, errors.New("read observed request capture")
	}
	var capture requestaligned.Capture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&capture); err != nil {
		return requestaligned.Capture{}, fmt.Errorf("decode observed request capture: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return requestaligned.Capture{}, errors.New("observed request capture has trailing data")
	}
	return capture, nil
}
