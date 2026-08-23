// Command native-volatility-evaluation derives and optionally verifies a
// producer-atomic output transport plan from two independent native builds.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tonyredondo/buildopt/internal/nativevolatility"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "native-volatility-evaluation: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("native-volatility-evaluation", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	firstPath := flags.String("first", "", "first native observation")
	secondPath := flags.String("second", "", "second native observation")
	reusedPath := flags.String("reused", "", "candidate transported-output observation")
	rebuiltPath := flags.String("rebuilt", "", "candidate locally-rebuilt observation")
	observeRoot := flags.String("observe-root", "", "native workspace to observe")
	manifestPath := flags.String("materialization-manifest", "", "producer-bound materialization manifest")
	binding := flags.String("binding", "", "observation environment binding SHA-256")
	portfolioContextPath := flags.String("portfolio-context", "", "cross-revision portfolio context")
	portfolioPath := flags.String("portfolio", "", "existing cross-revision portfolio")
	learnResultPath := flags.String("learn-result", "", "authoritative native result to learn")
	learnRevision := flags.String("learn-revision", "", "learning source revision SHA-256")
	applyCurrentPath := flags.String("apply-current", "", "current native result to partition")
	applyRevision := flags.String("apply-revision", "", "current source revision SHA-256")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("usage: native-volatility-evaluation --first FILE --second FILE [--reused FILE --rebuilt FILE] | --observe-root DIR --materialization-manifest FILE --binding SHA256 | --portfolio-context FILE [--portfolio FILE] --learn-result FILE --learn-revision SHA256 | --portfolio-context FILE --portfolio FILE --apply-current FILE --apply-revision SHA256 [--reused FILE --rebuilt FILE]")
	}
	portfolioMode := *portfolioContextPath != "" || *portfolioPath != "" ||
		*learnResultPath != "" || *learnRevision != "" ||
		*applyCurrentPath != "" || *applyRevision != ""
	if portfolioMode {
		if *firstPath != "" || *secondPath != "" || *observeRoot != "" ||
			*manifestPath != "" || *binding != "" || *portfolioContextPath == "" {
			return errors.New("portfolio mode cannot be combined with observation or analysis mode")
		}
		return runPortfolioMode(
			stdout, *portfolioContextPath, *portfolioPath, *learnResultPath, *learnRevision,
			*applyCurrentPath, *applyRevision, *reusedPath, *rebuiltPath,
		)
	}
	if *observeRoot != "" || *manifestPath != "" || *binding != "" {
		if *observeRoot == "" || *manifestPath == "" || *binding == "" ||
			*firstPath != "" || *secondPath != "" || *reusedPath != "" || *rebuiltPath != "" {
			return errors.New("observe mode requires root, materialization manifest and binding only")
		}
		observation, err := observeMaterialization(*observeRoot, *manifestPath, *binding)
		if err != nil {
			return err
		}
		return writeJSON(stdout, observation)
	}
	if *firstPath == "" || *secondPath == "" {
		return errors.New("usage: native-volatility-evaluation --first FILE --second FILE [--reused FILE --rebuilt FILE]")
	}
	if (*reusedPath == "") != (*rebuiltPath == "") {
		return errors.New("both --reused and --rebuilt are required for candidate verification")
	}
	first, err := readObservation(*firstPath)
	if err != nil {
		return fmt.Errorf("read first observation: %w", err)
	}
	second, err := readObservation(*secondPath)
	if err != nil {
		return fmt.Errorf("read second observation: %w", err)
	}
	result := nativevolatility.Analyze(first, second)
	if *reusedPath != "" {
		reused, readErr := readObservation(*reusedPath)
		if readErr != nil {
			return fmt.Errorf("read reused observation: %w", readErr)
		}
		rebuilt, readErr := readObservation(*rebuiltPath)
		if readErr != nil {
			return fmt.Errorf("read rebuilt observation: %w", readErr)
		}
		if err := nativevolatility.VerifyCandidate(result, reused, rebuilt); err != nil {
			return err
		}
	}
	return writeJSON(stdout, result)
}

func runPortfolioMode(
	stdout io.Writer,
	contextPath string,
	portfolioPath string,
	learnResultPath string,
	learnRevision string,
	applyCurrentPath string,
	applyRevision string,
	reusedPath string,
	rebuiltPath string,
) error {
	context, err := readJSON[nativevolatility.PortfolioContext](contextPath)
	if err != nil {
		return fmt.Errorf("read portfolio context: %w", err)
	}
	learning := learnResultPath != "" || learnRevision != ""
	applying := applyCurrentPath != "" || applyRevision != ""
	if learning == applying || (reusedPath == "") != (rebuiltPath == "") {
		return errors.New("portfolio mode requires exactly one complete learn or apply request")
	}
	portfolio := nativevolatility.Portfolio{}
	if portfolioPath != "" {
		portfolio, err = readJSON[nativevolatility.Portfolio](portfolioPath)
		if err != nil {
			return fmt.Errorf("read portfolio: %w", err)
		}
	}
	if learning {
		if learnResultPath == "" || learnRevision == "" || reusedPath != "" || applyCurrentPath != "" {
			return errors.New("portfolio learning requires result and revision only")
		}
		result, readErr := readJSON[nativevolatility.Result](learnResultPath)
		if readErr != nil {
			return fmt.Errorf("read learning result: %w", readErr)
		}
		learned, learnErr := nativevolatility.LearnPortfolio(
			portfolio, context, learnRevision, result, true,
		)
		if learnErr != nil {
			return learnErr
		}
		return writeJSON(stdout, learned)
	}
	if portfolioPath == "" || applyCurrentPath == "" || applyRevision == "" || learnResultPath != "" {
		return errors.New("portfolio application requires context, portfolio, current result and revision")
	}
	current, err := readJSON[nativevolatility.Result](applyCurrentPath)
	if err != nil {
		return fmt.Errorf("read current result: %w", err)
	}
	application, err := nativevolatility.ApplyPortfolio(portfolio, context, applyRevision, current)
	if err != nil {
		return err
	}
	if reusedPath != "" {
		reused, readErr := readObservation(reusedPath)
		if readErr != nil {
			return fmt.Errorf("read reused observation: %w", readErr)
		}
		rebuilt, readErr := readObservation(rebuiltPath)
		if readErr != nil {
			return fmt.Errorf("read rebuilt observation: %w", readErr)
		}
		if err := nativevolatility.VerifyPortfolioCandidate(
			application, portfolio, current, reused, rebuilt,
		); err != nil {
			return err
		}
	}
	return writeJSON(stdout, application)
}

func observeMaterialization(root, manifestPath, binding string) (nativevolatility.Observation, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nativevolatility.Observation{}, fmt.Errorf("read materialization manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest struct {
		SchemaVersion    string   `json:"schemaVersion"`
		RepositoryID     string   `json:"repositoryId"`
		TargetRevision   string   `json:"targetRevision"`
		RequiredOutputs  []string `json:"requiredOutputs"`
		CandidateOutputs []string `json:"candidateOutputs"`
		PackFile         string   `json:"packFile"`
		PackSHA256       string   `json:"packSha256"`
		PackSize         int64    `json:"packSize"`
		Entries          []struct {
			Path          string   `json:"path"`
			SHA256        string   `json:"sha256"`
			Size          int64    `json:"size"`
			Mode          uint32   `json:"mode"`
			Offset        int64    `json:"offset"`
			ProducerTasks []string `json:"producerTasks"`
		} `json:"entries"`
	}
	if err := decoder.Decode(&manifest); err != nil {
		return nativevolatility.Observation{}, fmt.Errorf("decode materialization manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nativevolatility.Observation{}, errors.New("materialization manifest contains trailing JSON")
	}
	if manifest.SchemaVersion != "buildopt.poc/verified-output-materialization/v2" || len(manifest.Entries) == 0 {
		return nativevolatility.Observation{}, errors.New("producer-bound materialization manifest is invalid")
	}
	inventory := make([]nativevolatility.Entry, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		if len(entry.ProducerTasks) == 0 {
			return nativevolatility.Observation{}, errors.New("materialization entry has no producer attribution")
		}
		inventory = append(inventory, nativevolatility.Entry{
			Path: entry.Path, ProducerTasks: append([]string(nil), entry.ProducerTasks...),
		})
	}
	return nativevolatility.Observe(root, binding, inventory)
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func readObservation(path string) (nativevolatility.Observation, error) {
	return readJSON[nativevolatility.Observation](path)
}

func readJSON[T any](path string) (T, error) {
	var value T
	raw, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return value, errors.New("JSON document contains trailing data")
	}
	return value, nil
}
