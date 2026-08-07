package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	qualifiedPOCProfileUsage             = "usage: buildopt poc --changes-file PATH [--config PATH] [--timings-file PATH]\n"
	qualifiedPOCProfileSchemaVersion     = "buildopt.poc/qualified-profile/v1"
	qualifiedPOCProfilePlanSchemaVersion = "buildopt.poc/qualified-profile-plan/v1"
	qualifiedPOCProfileID                = "clean-build-impact-plus-exact-standard-jar"
	qualifiedPOCProfileOwnership         = "REPOSITORY_COMMITTED"
	qualifiedPOCProfileClaimScope        = "DECLARED_OUTPUTS_ONLY"
	qualifiedPOCProfileFallback          = "NATIVE_FULL_GRAPH"
	qualifiedPOCProfileDefaultPath       = "buildopt-qualified-profile.json"
	maximumQualifiedPOCProfileBytes      = 64 << 10
)

type qualifiedPOCProfile struct {
	SchemaVersion  string                        `json:"schemaVersion"`
	ProfileVersion uint64                        `json:"profileVersion"`
	ProfileID      string                        `json:"profileId"`
	Ownership      string                        `json:"ownership"`
	ClaimScope     string                        `json:"claimScope"`
	RepositoryID   string                        `json:"repositoryId"`
	PipelineClass  string                        `json:"pipelineClass"`
	Fallback       string                        `json:"fallback"`
	Impact         qualifiedPOCProfileImpact     `json:"impact"`
	Mechanisms     qualifiedPOCProfileMechanisms `json:"mechanisms"`
	GradleOptions  []string                      `json:"gradleOptions"`
}

type qualifiedPOCProfileImpact struct {
	Manifest          string `json:"manifest"`
	Graph             string `json:"graph"`
	GeneratedManifest string `json:"generatedManifest"`
}

type qualifiedPOCProfileMechanisms struct {
	BuildImpact         bool `json:"buildImpact"`
	StandardJarAdapter  bool `json:"standardJarAdapter"`
	SafeCache           bool `json:"safeCache"`
	RuntimeTuning       bool `json:"runtimeTuning"`
	HotState            bool `json:"hotState"`
	StandardCopyAdapter bool `json:"standardCopyAdapter"`
	SharedEdgeCache     bool `json:"sharedEdgeCache"`
}

type qualifiedPOCProfilePlan struct {
	SchemaVersion         string   `json:"schemaVersion"`
	ProfileID             string   `json:"profileId"`
	ClaimScope            string   `json:"claimScope"`
	RepositoryID          string   `json:"repositoryId"`
	PipelineClass         string   `json:"pipelineClass"`
	SelectionMode         string   `json:"selectionMode"`
	SelectionReason       string   `json:"selectionReason"`
	AlternativeID         string   `json:"alternativeId,omitempty"`
	Entrypoints           []string `json:"entrypoints"`
	FallbackEntrypoints   []string `json:"fallbackEntrypoints"`
	AffectedProjects      []string `json:"affectedProjects"`
	OmittedProjectCount   int      `json:"omittedProjectCount"`
	ExpectedOutputs       []string `json:"expectedOutputs"`
	PreservedTestCheckIDs []string `json:"preservedTestCheckIds"`
	EnabledAdapters       []string `json:"enabledAdapters"`
	DisabledMechanisms    []string `json:"disabledMechanisms"`
	ProductionAuthorized  bool     `json:"productionAuthorized"`
}

func prepareQualifiedPOCProfileInvocation(args []string, bypass bool) (impactInvocation, error) {
	flags := flag.NewFlagSet("buildopt poc", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", qualifiedPOCProfileDefaultPath, "repository-owned qualified POC profile")
	changesPath := flags.String("changes-file", "", "repository-relative newline-delimited changed paths")
	timingsPath := flags.String("timings-file", "", "repository-relative machine-readable phase timing output")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *changesPath == "" {
		return impactInvocation{}, errors.New("invalid qualified POC profile arguments")
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		return impactInvocation{}, err
	}
	profile, err := loadQualifiedPOCProfile(repositoryRoot, *configPath)
	if err != nil {
		return impactInvocation{}, err
	}
	impactArgs := []string{
		"--repository-id", profile.RepositoryID,
		"--pipeline-class", profile.PipelineClass,
		"--changes-file", *changesPath,
		"--manifest", profile.Impact.Manifest,
		"--graph", profile.Impact.Graph,
		"--generated-manifest", profile.Impact.GeneratedManifest,
	}
	if *timingsPath != "" {
		impactArgs = append(impactArgs, "--timings-file", *timingsPath)
	}
	for _, option := range profile.GradleOptions {
		impactArgs = append(impactArgs, "--gradle-option="+option)
	}
	impactArgs = append(impactArgs, "--cache-standard-jar-producers")
	invocation, err := prepareImpactInvocation(impactArgs, bypass)
	if err != nil {
		return impactInvocation{}, err
	}
	manifest, err := buildimpact.LoadRepositoryManifest(
		repositoryRoot,
		profile.Impact.Manifest,
		profile.RepositoryID,
		profile.PipelineClass,
	)
	if err != nil {
		return impactInvocation{}, err
	}
	expectedOutputs := make([]string, 0, len(manifest.Manifest.RequiredArtifacts))
	for _, artifact := range manifest.Manifest.RequiredArtifacts {
		expectedOutputs = append(expectedOutputs, artifact.Path)
	}
	enabledAdapters := []string{}
	invocation.standardJarCache = invocation.plan.CandidateSelected
	if invocation.standardJarCache {
		enabledAdapters = []string{"STANDARD_JAR"}
	}
	invocation.standardCopyCache = false
	invocation.qualifiedProfile = &qualifiedPOCProfilePlan{
		SchemaVersion:         qualifiedPOCProfilePlanSchemaVersion,
		ProfileID:             profile.ProfileID,
		ClaimScope:            profile.ClaimScope,
		RepositoryID:          profile.RepositoryID,
		PipelineClass:         profile.PipelineClass,
		SelectionMode:         invocation.plan.Mode,
		SelectionReason:       invocation.plan.Reason,
		AlternativeID:         invocation.plan.AlternativeID,
		Entrypoints:           append([]string(nil), invocation.plan.Entrypoints...),
		FallbackEntrypoints:   append([]string(nil), manifest.Manifest.OriginalEntrypoints...),
		AffectedProjects:      append([]string{}, invocation.plan.AffectedProjects...),
		OmittedProjectCount:   len(invocation.plan.OmittedProjects),
		ExpectedOutputs:       expectedOutputs,
		PreservedTestCheckIDs: append([]string(nil), invocation.plan.PreservedTestCheckIDs...),
		EnabledAdapters:       enabledAdapters,
		DisabledMechanisms: []string{
			"HOT_STATE",
			"RUNTIME_TUNING",
			"SAFE_CACHE",
			"SHARED_EDGE_CACHE",
			"STANDARD_COPY",
		},
		ProductionAuthorized: false,
	}
	return invocation, nil
}

func canonicalWorkingDirectory() (string, error) {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	repositoryRoot, err = filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("canonicalize repository root: %w", err)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("make repository root absolute: %w", err)
	}
	return repositoryRoot, nil
}

func loadQualifiedPOCProfile(repositoryRoot, relativePath string) (qualifiedPOCProfile, error) {
	raw, err := readQualifiedPOCProfileFile(repositoryRoot, relativePath)
	if err != nil {
		return qualifiedPOCProfile{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var profile qualifiedPOCProfile
	if err := decoder.Decode(&profile); err != nil {
		return qualifiedPOCProfile{}, fmt.Errorf("decode qualified POC profile: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return qualifiedPOCProfile{}, errors.New("qualified POC profile has trailing content")
	}
	if err := validateQualifiedPOCProfile(profile); err != nil {
		return qualifiedPOCProfile{}, err
	}
	return profile, nil
}

func readQualifiedPOCProfileFile(repositoryRoot, relativePath string) ([]byte, error) {
	if !validQualifiedPOCProfilePath(relativePath) {
		return nil, errors.New("qualified POC profile path must be clean and repository relative")
	}
	path := filepath.Join(repositoryRoot, relativePath)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect qualified POC profile: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumQualifiedPOCProfileBytes {
		return nil, errors.New("qualified POC profile must be a bounded regular file")
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve qualified POC profile: %w", err)
	}
	relative, err := filepath.Rel(repositoryRoot, canonicalPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return nil, errors.New("qualified POC profile escapes the repository")
	}
	raw, err := os.ReadFile(canonicalPath)
	if err != nil {
		return nil, fmt.Errorf("read qualified POC profile: %w", err)
	}
	return raw, nil
}

func validateQualifiedPOCProfile(profile qualifiedPOCProfile) error {
	if profile.SchemaVersion != qualifiedPOCProfileSchemaVersion || profile.ProfileVersion != 1 ||
		profile.ProfileID != qualifiedPOCProfileID {
		return errors.New("unsupported qualified POC profile identity")
	}
	if profile.Ownership != qualifiedPOCProfileOwnership ||
		profile.ClaimScope != qualifiedPOCProfileClaimScope ||
		profile.Fallback != qualifiedPOCProfileFallback {
		return errors.New("qualified POC profile must remain repository-owned, output-scoped, and native-full-graph fallback")
	}
	if profile.RepositoryID == "" || profile.PipelineClass == "" {
		return errors.New("qualified POC profile requires repository and pipeline bindings")
	}
	for _, path := range []string{
		profile.Impact.Manifest,
		profile.Impact.Graph,
		profile.Impact.GeneratedManifest,
	} {
		if !validQualifiedPOCProfilePath(path) {
			return errors.New("qualified POC profile Build Impact paths must be clean and repository relative")
		}
	}
	if !profile.Mechanisms.BuildImpact || !profile.Mechanisms.StandardJarAdapter ||
		profile.Mechanisms.SafeCache || profile.Mechanisms.RuntimeTuning ||
		profile.Mechanisms.HotState || profile.Mechanisms.StandardCopyAdapter ||
		profile.Mechanisms.SharedEdgeCache {
		return errors.New("qualified POC profile must enable only Build Impact and the exact standard Jar adapter")
	}
	if len(profile.GradleOptions) > 32 {
		return errors.New("qualified POC profile has too many Gradle options")
	}
	seen := map[string]bool{}
	for _, option := range profile.GradleOptions {
		if !validImpactGradleOption(option) || seen[option] {
			return errors.New("qualified POC profile Gradle options must be unique bounded execution options")
		}
		seen[option] = true
	}
	return nil
}

func validQualifiedPOCProfilePath(candidate string) bool {
	return candidate != "" && !filepath.IsAbs(candidate) &&
		filepath.Clean(candidate) == candidate && candidate != "." && candidate != ".." &&
		!strings.HasPrefix(candidate, ".."+string(filepath.Separator))
}

func writeQualifiedPOCProfilePlan(writer io.Writer, plan qualifiedPOCProfilePlan) error {
	raw, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("encode qualified POC profile plan: %w", err)
	}
	_, err = fmt.Fprintf(writer, "buildopt: qualified POC profile plan %s\n", raw)
	return err
}

func qualifiedPOCProfileEnvironment(getenv func(string) string, standardJarEnabled bool) func(string) string {
	return func(name string) string {
		switch name {
		case gradleStandardJarCacheEnvironment:
			if standardJarEnabled {
				return "1"
			}
			return "0"
		case gradleSafeCacheEnvironment, gradleStandardCopyCacheEnvironment:
			return "0"
		case gradleCheckstyleHeapEnvironment,
			gradleInitScriptEnvironment,
			gradlePluginJarEnvironment,
			sessioningest.ServerURLEnvironment,
			sessioningest.ServerTokenEnvironment,
			sessioningest.ExportContextEnvironment,
			localAuthorityPathEnvironment,
			localTrustRootPathEnvironment,
			localCredentialPathEnvironment,
			sharedCacheTokenPathEnvironment,
			sharedCacheURLEnvironment,
			managedL1StateRootEnvironment,
			managedL1TenantEnvironment,
			managedL1RepositoryEnvironment,
			managedL1TrustDomainEnvironment,
			managedL1CompatibilityEnvironment,
			managedL1GenerationEnvironment,
			managedL1L2WriterEnvironment,
			gradleBootstrapConfigPathEnvironment:
			return ""
		default:
			return getenv(name)
		}
	}
}
