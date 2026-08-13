package launcher

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/outputequivalence"
)

const (
	profileOwnerInputSchema = "buildopt.poc/profile-owner-input/v1"
	profileOwnerInputUsage  = "usage: buildopt profile input (--output-contract PATH --confirm [--output-equivalence PATH] [--output PATH] [--global-change GLOB ...] [--gradle-command PATH] [--gradle-option VALUE ...] [--timeout DURATION] | --check PATH)\n"
	maximumOwnerInputBytes  = 1 << 20
)

var ownerInputDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type profileOwnerInput struct {
	SchemaVersion        string                         `json:"schemaVersion"`
	RepositoryID         string                         `json:"repositoryId"`
	PipelineClass        string                         `json:"pipelineClass"`
	Entrypoints          []string                       `json:"entrypoints"`
	RequiredOutputs      []string                       `json:"requiredOutputs"`
	ChangeSource         string                         `json:"changeSource"`
	GlobalChanges        []string                       `json:"globalChanges"`
	GradleCommand        string                         `json:"gradleCommand"`
	GradleOptions        []string                       `json:"gradleOptions"`
	TimeoutMinutes       int                            `json:"timeoutMinutes"`
	OutputConfirmation   profileOwnerOutputConfirmation `json:"outputConfirmation"`
	OutputEquivalence    *profileOwnerDocumentBinding   `json:"outputEquivalence,omitempty"`
	ReviewRequired       bool                           `json:"reviewRequired"`
	ActivationAutomatic  bool                           `json:"activationAutomatic"`
	ProductionAuthorized bool                           `json:"productionAuthorized"`
	TestOptimization     string                         `json:"testOptimization"`
}

type profileOwnerOutputConfirmation struct {
	Status           string `json:"status"`
	ObservedRevision string `json:"observedRevision"`
	ContractSHA256   string `json:"contractSha256"`
}

type profileOwnerDocumentBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func runProfileOwnerInput(args []string, stdout, stderr io.Writer) int {
	if isHelp(args) {
		_, _ = io.WriteString(stdout, profileOwnerInputUsage)
		return 0
	}
	flags := flag.NewFlagSet("buildopt profile input", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outputContract := flags.String("output-contract", "", "validated output-contract review artifact")
	outputEquivalence := flags.String("output-equivalence", "", "owner-reviewed semantic output-equivalence contract")
	check := flags.String("check", "", "validate and normalize an existing owner input")
	confirm := flags.Bool("confirm", false, "explicitly confirm the validated output contract")
	output := flags.String("output", ".buildopt/profile.json", "versioned owner input output")
	gradleCommand := flags.String("gradle-command", "", "portable repository-relative Gradle command")
	timeout := flags.Duration("timeout", 5*time.Minute, "owner workflow timeout")
	var globalChanges proposalStringFlag
	var gradleOptions repeatedStringFlag
	flags.Var(&globalChanges, "global-change", "full-graph fallback glob; repeat to replace defaults")
	flags.Var(&gradleOptions, "gradle-option", "Gradle workflow option; repeat for multiple values")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, profileOwnerInputUsage)
		return exitUsage
	}
	checking := *check != ""
	creating := *outputContract != ""
	if checking == creating || (checking && (*confirm || *outputEquivalence != "" || len(globalChanges) != 0 || len(gradleOptions) != 0 || *gradleCommand != "" || *output != ".buildopt/profile.json" || *timeout != 5*time.Minute)) || (creating && !*confirm) {
		_, _ = io.WriteString(stderr, profileOwnerInputUsage)
		return exitUsage
	}
	if creating && (*output == *outputContract || (*outputEquivalence != "" && (*output == *outputEquivalence || *outputContract == *outputEquivalence))) {
		_, _ = fmt.Fprintln(stderr, "buildopt: profile owner input unavailable: output must not replace its source contract")
		return exitConfiguration
	}
	root, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	if checking {
		input, _, err := readProfileOwnerInput(root, *check)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
			return exitConfiguration
		}
		raw, err := renderProfileOwnerInput(input)
		if err == nil {
			_, err = stdout.Write(raw)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
			return exitConfiguration
		}
		return 0
	}
	contractRaw, err := readRepositoryRegularDocument(root, *outputContract, maximumOutputSnapshotBytes)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	contract, err := parseConfirmedOutputContract(contractRaw)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	if len(globalChanges) == 0 {
		globalChanges = append(globalChanges, defaultProposalGlobalChanges...)
	}
	if *timeout < time.Minute || *timeout > 30*time.Minute || *timeout%time.Minute != 0 {
		_, _ = fmt.Fprintln(stderr, "buildopt: profile owner input unavailable: timeout must be a whole number of minutes between one and 30")
		return exitConfiguration
	}
	digest := sha256.Sum256(contractRaw)
	var equivalenceBinding *profileOwnerDocumentBinding
	if *outputEquivalence != "" {
		equivalenceRaw, err := readRepositoryRegularDocument(root, *outputEquivalence, maximumOwnerInputBytes)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
			return exitConfiguration
		}
		if _, err := outputequivalence.Parse(equivalenceRaw); err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
			return exitConfiguration
		}
		equivalenceBinding = &profileOwnerDocumentBinding{
			Path: *outputEquivalence, SHA256: outputequivalence.SHA256(equivalenceRaw),
		}
	}
	input := profileOwnerInput{
		SchemaVersion: profileOwnerInputSchema,
		RepositoryID:  contract.RepositoryID, PipelineClass: contract.PipelineClass,
		Entrypoints:     append([]string(nil), contract.OriginalEntrypoints...),
		RequiredOutputs: append([]string(nil), contract.DeclaredOutputs...),
		ChangeSource:    "GIT_DIFF_BASE_TO_HEAD",
		GlobalChanges:   append([]string(nil), globalChanges...),
		GradleCommand:   *gradleCommand, GradleOptions: append([]string(nil), gradleOptions...),
		TimeoutMinutes: int(timeout.Minutes()),
		OutputConfirmation: profileOwnerOutputConfirmation{
			Status: "OWNER_CONFIRMED", ObservedRevision: contract.RepositoryRevision,
			ContractSHA256: hex.EncodeToString(digest[:]),
		},
		OutputEquivalence: equivalenceBinding,
		ReviewRequired:    true, ActivationAutomatic: false,
		ProductionAuthorized: false, TestOptimization: "OUT_OF_SCOPE",
	}
	sort.Strings(input.Entrypoints)
	sort.Strings(input.RequiredOutputs)
	sort.Strings(input.GlobalChanges)
	if err := validateProfileOwnerInput(input); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := renderProfileOwnerInput(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	if err := writeRepositoryDocument(root, *output, raw, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile owner input unavailable: %v\n", err)
		return exitConfiguration
	}
	return 0
}

func parseConfirmedOutputContract(raw []byte) (outputContractReport, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var report outputContractReport
	if err := decoder.Decode(&report); err != nil {
		return outputContractReport{}, fmt.Errorf("decode output contract: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return outputContractReport{}, errors.New("output contract has trailing content")
	}
	if report.SchemaVersion != outputContractSchema || report.Decision != "VALIDATED_REQUIRED_OUTPUTS" || report.Reason != "DECLARED_OUTPUTS_MATCH_EXECUTED_WORKFLOW" || !report.ReviewRequired || report.ActivationAutomatic || report.ProductionAuthorized || report.TestOptimization != "OUT_OF_SCOPE" {
		return outputContractReport{}, errors.New("output contract is not an explicitly reviewable validated result")
	}
	if !outputContractRepositoryPattern.MatchString(report.RepositoryID) || !outputContractPipelinePattern.MatchString(report.PipelineClass) || !validMeasurementRevision(report.RepositoryRevision) || len(report.OriginalEntrypoints) == 0 || len(report.DeclaredOutputs) == 0 || len(report.Validations) != len(report.DeclaredOutputs) {
		return outputContractReport{}, errors.New("output contract identity is invalid")
	}
	for index, validation := range report.Validations {
		if validation.Pattern != report.DeclaredOutputs[index] || validation.Status != "VALIDATED" || validation.MatchedFiles <= 0 || len(validation.OwnerProjects) == 0 || len(validation.ProducerTasks) == 0 {
			return outputContractReport{}, errors.New("output contract contains an unconfirmed declaration")
		}
	}
	return report, nil
}

func readProfileOwnerInput(repositoryRoot, relativePath string) (profileOwnerInput, string, error) {
	raw, err := readRepositoryRegularDocument(repositoryRoot, relativePath, maximumOwnerInputBytes)
	if err != nil {
		return profileOwnerInput{}, "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input profileOwnerInput
	if err := decoder.Decode(&input); err != nil {
		return profileOwnerInput{}, "", fmt.Errorf("decode owner input: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return profileOwnerInput{}, "", errors.New("owner input has trailing content")
	}
	if err := validateProfileOwnerInput(input); err != nil {
		return profileOwnerInput{}, "", err
	}
	if input.OutputEquivalence != nil {
		equivalenceRaw, err := readRepositoryRegularDocument(repositoryRoot, input.OutputEquivalence.Path, maximumOwnerInputBytes)
		if err != nil {
			return profileOwnerInput{}, "", err
		}
		if _, err := outputequivalence.Parse(equivalenceRaw); err != nil {
			return profileOwnerInput{}, "", err
		}
		if outputequivalence.SHA256(equivalenceRaw) != input.OutputEquivalence.SHA256 {
			return profileOwnerInput{}, "", errors.New("owner input output-equivalence binding drift")
		}
	}
	digest := sha256.Sum256(raw)
	return input, hex.EncodeToString(digest[:]), nil
}

func validateProfileOwnerInput(input profileOwnerInput) error {
	if input.SchemaVersion != profileOwnerInputSchema || !outputContractRepositoryPattern.MatchString(input.RepositoryID) || !outputContractPipelinePattern.MatchString(input.PipelineClass) {
		return errors.New("owner input identity is invalid")
	}
	if len(input.Entrypoints) == 0 || len(input.Entrypoints) > 256 || !uniqueMeasurementStrings(input.Entrypoints) {
		return errors.New("owner input entrypoints must be unique and bounded")
	}
	if _, err := proposalTerminalSelectors(input.Entrypoints); err != nil {
		return err
	}
	if len(input.RequiredOutputs) == 0 || len(input.RequiredOutputs) > 256 || !uniqueMeasurementStrings(input.RequiredOutputs) {
		return errors.New("owner input outputs must be confirmed, unique and bounded")
	}
	if input.ChangeSource != "GIT_DIFF_BASE_TO_HEAD" {
		return errors.New("owner input change source must be GIT_DIFF_BASE_TO_HEAD")
	}
	if len(input.GlobalChanges) == 0 || len(input.GlobalChanges) > 256 || !uniqueMeasurementStrings(input.GlobalChanges) {
		return errors.New("owner input global changes must be explicit, unique and bounded")
	}
	for _, pattern := range append(append([]string(nil), input.RequiredOutputs...), input.GlobalChanges...) {
		if !validOutputContractPattern(pattern) {
			return fmt.Errorf("owner input pattern %q is unsafe", pattern)
		}
	}
	if input.GradleCommand != "" && (!strings.HasPrefix(input.GradleCommand, "./") || !validObservedOutputPath(strings.TrimPrefix(input.GradleCommand, "./"))) {
		return errors.New("owner input Gradle command must be empty or a portable ./ repository path")
	}
	if len(input.GradleOptions) > 32 || !uniqueMeasurementStrings(input.GradleOptions) {
		return errors.New("owner input Gradle options must be unique and bounded")
	}
	for _, option := range input.GradleOptions {
		if option == "" || len(option) > 1024 || strings.ContainsAny(option, "\r\n\x00") {
			return errors.New("owner input Gradle option is invalid")
		}
	}
	if input.TimeoutMinutes < 1 || input.TimeoutMinutes > 30 {
		return errors.New("owner input timeout must be between one and 30 minutes")
	}
	if input.OutputConfirmation.Status != "OWNER_CONFIRMED" || !validMeasurementRevision(input.OutputConfirmation.ObservedRevision) || !ownerInputDigestPattern.MatchString(input.OutputConfirmation.ContractSHA256) {
		return errors.New("owner input output confirmation is invalid")
	}
	if input.OutputEquivalence != nil {
		binding := input.OutputEquivalence
		if binding.Path == "" || filepath.IsAbs(binding.Path) || path.Clean(binding.Path) != binding.Path ||
			binding.Path == "." || binding.Path == ".." || !validObservedOutputPath(binding.Path) ||
			!ownerInputDigestPattern.MatchString(binding.SHA256) {
			return errors.New("owner input output-equivalence binding is invalid")
		}
	}
	if !input.ReviewRequired || input.ActivationAutomatic || input.ProductionAuthorized || input.TestOptimization != "OUT_OF_SCOPE" {
		return errors.New("owner input POC boundaries are invalid")
	}
	return nil
}

func renderProfileOwnerInput(input profileOwnerInput) ([]byte, error) {
	raw, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func readRepositoryRegularDocument(repositoryRoot, relativePath string, maximumBytes int64) ([]byte, error) {
	if relativePath == "" || strings.Contains(relativePath, `\`) || filepath.IsAbs(relativePath) || path.Clean(relativePath) != relativePath || relativePath == "." || relativePath == ".." || !validObservedOutputPath(relativePath) {
		return nil, errors.New("document must be a clean repository-relative path")
	}
	platformPath := filepath.FromSlash(relativePath)
	current := repositoryRoot
	for _, segment := range strings.Split(platformPath, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect repository document: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("repository document path must not contain symlinks")
		}
	}
	info, err := os.Stat(current)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumBytes {
		return nil, errors.New("repository document must be a bounded non-empty regular file")
	}
	return os.ReadFile(current)
}
