package launcher

import (
	"bytes"
	"crypto/sha256"
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
	qualifiedPOCProfileUsage             = "usage: buildopt poc --changes-file PATH [--config PATH] [--timings-file PATH] [--edge-url LOOPBACK_ORIGIN]\n"
	qualifiedPOCProfileSchemaVersionV1   = "buildopt.poc/qualified-profile/v1"
	qualifiedPOCProfileSchemaVersionV2   = "buildopt.poc/qualified-profile/v2"
	qualifiedPOCProfileSchemaVersionV3   = "buildopt.poc/qualified-profile/v3"
	qualifiedPOCProfilePlanSchemaV1      = "buildopt.poc/qualified-profile-plan/v1"
	qualifiedPOCProfilePlanSchemaV2      = "buildopt.poc/qualified-profile-plan/v2"
	qualifiedPOCProfilePlanSchemaV3      = "buildopt.poc/qualified-profile-plan/v3"
	qualifiedPOCProfileIDV1              = "clean-build-impact-plus-exact-standard-jar"
	qualifiedPOCProfileIDV2              = "normalized-impact-plus-read-only-edge"
	qualifiedPOCProfileIDV3              = "build-impact-only"
	qualifiedPOCProfileOwnership         = "REPOSITORY_COMMITTED"
	qualifiedPOCProfileClaimScope        = "DECLARED_OUTPUTS_ONLY"
	qualifiedPOCProfileFallback          = "NATIVE_FULL_GRAPH"
	qualifiedPOCProfileDefaultPath       = "buildopt-qualified-profile.json"
	maximumQualifiedPOCProfileBytes      = 64 << 10
	maximumQualifiedPOCPreconditionBytes = 16 << 20
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
	Preconditions  []qualifiedPOCPrecondition    `json:"preconditions,omitempty"`
	EdgeCache      *qualifiedPOCEdgeCache        `json:"edgeCache,omitempty"`
}

type qualifiedPOCPrecondition struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type qualifiedPOCEdgeCache struct {
	Mode string `json:"mode"`
}

type qualifiedPOCPreconditionResult struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Status string `json:"status"`
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
	SchemaVersion         string                           `json:"schemaVersion"`
	ProfileID             string                           `json:"profileId"`
	ClaimScope            string                           `json:"claimScope"`
	RepositoryID          string                           `json:"repositoryId"`
	PipelineClass         string                           `json:"pipelineClass"`
	SelectionMode         string                           `json:"selectionMode"`
	SelectionReason       string                           `json:"selectionReason"`
	AlternativeID         string                           `json:"alternativeId,omitempty"`
	Entrypoints           []string                         `json:"entrypoints"`
	FallbackEntrypoints   []string                         `json:"fallbackEntrypoints"`
	AffectedProjects      []string                         `json:"affectedProjects"`
	OmittedProjectCount   int                              `json:"omittedProjectCount"`
	ExpectedOutputs       []string                         `json:"expectedOutputs"`
	PreservedTestCheckIDs []string                         `json:"preservedTestCheckIds"`
	EnabledAdapters       []string                         `json:"enabledAdapters"`
	DisabledMechanisms    []string                         `json:"disabledMechanisms"`
	Preconditions         []qualifiedPOCPreconditionResult `json:"preconditions,omitempty"`
	EdgeCacheMode         string                           `json:"edgeCacheMode,omitempty"`
	EdgeCacheEndpoint     string                           `json:"edgeCacheEndpoint,omitempty"`
	ProductionAuthorized  bool                             `json:"productionAuthorized"`
}

func prepareQualifiedPOCProfileInvocation(args []string, bypass bool) (impactInvocation, error) {
	flags := flag.NewFlagSet("buildopt poc", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", qualifiedPOCProfileDefaultPath, "repository-owned qualified POC profile")
	changesPath := flags.String("changes-file", "", "repository-relative newline-delimited changed paths")
	timingsPath := flags.String("timings-file", "", "repository-relative machine-readable phase timing output")
	edgeURL := flags.String("edge-url", "", "read-only IPv4 loopback Edge origin for the qualified POC")
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
	if profile.Mechanisms.StandardJarAdapter {
		impactArgs = append(impactArgs, "--cache-standard-jar-producers")
	}
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
	preconditions := qualifiedPOCPreconditionResults(profile.Preconditions, "NOT_EVALUATED")
	pocEdgeURL := ""
	if profile.SchemaVersion != qualifiedPOCProfileSchemaVersionV2 {
		if *edgeURL != "" {
			return impactInvocation{}, errors.New("qualified POC profile does not accept an Edge endpoint")
		}
	} else if invocation.plan.CandidateSelected {
		var satisfied bool
		preconditions, satisfied = evaluateQualifiedPOCPreconditions(repositoryRoot, profile.Preconditions)
		if !satisfied {
			invocation.plan = qualifiedPOCNativeFallback(invocation.plan, manifest.Manifest.OriginalEntrypoints, "PROFILE_PRECONDITION_FAILED")
		} else if canonical, edgeErr := canonicalLoopbackHTTPOrigin(*edgeURL); edgeErr != nil {
			invocation.plan = qualifiedPOCNativeFallback(invocation.plan, manifest.Manifest.OriginalEntrypoints, "PROFILE_EDGE_UNAVAILABLE")
		} else {
			pocEdgeURL = canonical
		}
		invocation.gradleArgs = append(append([]string(nil), profile.GradleOptions...), invocation.plan.Entrypoints...)
	}
	expectedOutputs := make([]string, 0, len(manifest.Manifest.RequiredArtifacts))
	for _, artifact := range manifest.Manifest.RequiredArtifacts {
		expectedOutputs = append(expectedOutputs, artifact.Path)
	}
	enabledAdapters := []string{}
	invocation.standardJarCache = invocation.plan.CandidateSelected
	invocation.standardJarCache = invocation.standardJarCache && profile.Mechanisms.StandardJarAdapter
	if invocation.standardJarCache {
		enabledAdapters = []string{"STANDARD_JAR"}
	}
	if invocation.plan.CandidateSelected && profile.Mechanisms.SharedEdgeCache {
		enabledAdapters = []string{"READ_ONLY_EDGE"}
		invocation.pocEdgeCacheURL = pocEdgeURL
	}
	invocation.standardCopyCache = false
	planSchema := qualifiedPOCProfilePlanSchemaV1
	disabledMechanisms := []string{"HOT_STATE", "RUNTIME_TUNING", "SAFE_CACHE", "SHARED_EDGE_CACHE", "STANDARD_COPY"}
	if profile.SchemaVersion == qualifiedPOCProfileSchemaVersionV2 {
		planSchema = qualifiedPOCProfilePlanSchemaV2
		disabledMechanisms = []string{"HOT_STATE", "RUNTIME_TUNING", "SAFE_CACHE", "STANDARD_COPY", "STANDARD_JAR"}
	} else if profile.SchemaVersion == qualifiedPOCProfileSchemaVersionV3 {
		planSchema = qualifiedPOCProfilePlanSchemaV3
		disabledMechanisms = []string{"HOT_STATE", "RUNTIME_TUNING", "SAFE_CACHE", "SHARED_EDGE_CACHE", "STANDARD_COPY", "STANDARD_JAR"}
	}
	invocation.qualifiedProfile = &qualifiedPOCProfilePlan{
		SchemaVersion:         planSchema,
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
		DisabledMechanisms:    disabledMechanisms,
		Preconditions:         preconditions,
		EdgeCacheMode:         qualifiedPOCEdgeMode(profile),
		EdgeCacheEndpoint:     pocEdgeURL,
		ProductionAuthorized:  false,
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
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve qualified POC repository root: %w", err)
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
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
	versionOne := profile.SchemaVersion == qualifiedPOCProfileSchemaVersionV1 &&
		profile.ProfileVersion == 1 && profile.ProfileID == qualifiedPOCProfileIDV1
	versionTwo := profile.SchemaVersion == qualifiedPOCProfileSchemaVersionV2 &&
		profile.ProfileVersion == 2 && profile.ProfileID == qualifiedPOCProfileIDV2
	versionThree := profile.SchemaVersion == qualifiedPOCProfileSchemaVersionV3 &&
		profile.ProfileVersion == 3 && profile.ProfileID == qualifiedPOCProfileIDV3
	if !versionOne && !versionTwo && !versionThree {
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
	if versionOne && (!profile.Mechanisms.BuildImpact || !profile.Mechanisms.StandardJarAdapter ||
		profile.Mechanisms.SafeCache || profile.Mechanisms.RuntimeTuning ||
		profile.Mechanisms.HotState || profile.Mechanisms.StandardCopyAdapter ||
		profile.Mechanisms.SharedEdgeCache || len(profile.Preconditions) != 0 || profile.EdgeCache != nil) {
		return errors.New("qualified POC profile v1 must enable only Build Impact and the exact standard Jar adapter")
	}
	if versionTwo && (!profile.Mechanisms.BuildImpact || profile.Mechanisms.StandardJarAdapter ||
		profile.Mechanisms.SafeCache || profile.Mechanisms.RuntimeTuning ||
		profile.Mechanisms.HotState || profile.Mechanisms.StandardCopyAdapter ||
		!profile.Mechanisms.SharedEdgeCache || profile.EdgeCache == nil ||
		profile.EdgeCache.Mode != "READ_ONLY_LOOPBACK" || len(profile.Preconditions) == 0 ||
		len(profile.Preconditions) > 8) {
		return errors.New("qualified POC profile v2 must enable only Build Impact and read-only loopback Edge with bounded preconditions")
	}
	if versionThree && (!profile.Mechanisms.BuildImpact || profile.Mechanisms.StandardJarAdapter ||
		profile.Mechanisms.SafeCache || profile.Mechanisms.RuntimeTuning ||
		profile.Mechanisms.HotState || profile.Mechanisms.StandardCopyAdapter ||
		profile.Mechanisms.SharedEdgeCache || len(profile.Preconditions) != 0 || profile.EdgeCache != nil) {
		return errors.New("qualified POC profile v3 must enable only Build Impact")
	}
	for _, precondition := range profile.Preconditions {
		if precondition.Type != "FILE_SHA256" || !validQualifiedPOCProfilePath(precondition.Path) ||
			len(precondition.SHA256) != sha256.Size*2 {
			return errors.New("qualified POC profile preconditions must be repository-relative SHA-256 files")
		}
		for _, character := range precondition.SHA256 {
			if !strings.ContainsRune("0123456789abcdef", character) {
				return errors.New("qualified POC profile precondition SHA-256 must be lowercase hexadecimal")
			}
		}
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

func qualifiedPOCPreconditionResults(preconditions []qualifiedPOCPrecondition, status string) []qualifiedPOCPreconditionResult {
	results := make([]qualifiedPOCPreconditionResult, 0, len(preconditions))
	for _, precondition := range preconditions {
		results = append(results, qualifiedPOCPreconditionResult{
			Type: precondition.Type, Path: precondition.Path,
			SHA256: precondition.SHA256, Status: status,
		})
	}
	return results
}

func evaluateQualifiedPOCPreconditions(repositoryRoot string, preconditions []qualifiedPOCPrecondition) ([]qualifiedPOCPreconditionResult, bool) {
	results := qualifiedPOCPreconditionResults(preconditions, "FAILED")
	satisfied := true
	for index, precondition := range preconditions {
		digest, err := hashQualifiedPOCPreconditionFile(repositoryRoot, precondition.Path)
		if err != nil {
			satisfied = false
			continue
		}
		if digest == precondition.SHA256 {
			results[index].Status = "SATISFIED"
		} else {
			satisfied = false
		}
	}
	return results, satisfied
}

func hashQualifiedPOCPreconditionFile(repositoryRoot, relativePath string) (string, error) {
	if !validQualifiedPOCProfilePath(relativePath) {
		return "", errors.New("qualified POC precondition path must be clean and repository relative")
	}
	path := filepath.Join(repositoryRoot, relativePath)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumQualifiedPOCPreconditionBytes {
		return "", errors.New("qualified POC precondition must be a bounded regular file")
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	canonicalRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("qualified POC precondition escapes the repository")
	}
	file, err := os.Open(canonicalPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func qualifiedPOCNativeFallback(plan buildimpact.POCCandidatePlan, originalEntrypoints []string, reason string) buildimpact.POCCandidatePlan {
	plan.Mode = buildimpact.DecisionFullGraph
	plan.Reason = reason
	plan.Entrypoints = append([]string(nil), originalEntrypoints...)
	plan.AlternativeID = ""
	plan.AffectedProjects = nil
	plan.OmittedProjects = nil
	plan.CandidateSelected = false
	return plan
}

func qualifiedPOCEdgeMode(profile qualifiedPOCProfile) string {
	if profile.EdgeCache == nil {
		return ""
	}
	return profile.EdgeCache.Mode
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

func qualifiedPOCProfileEnvironment(getenv func(string) string, standardJarEnabled bool, pocEdgeCacheURL string) func(string) string {
	return func(name string) string {
		switch name {
		case gradleStandardJarCacheEnvironment:
			if standardJarEnabled {
				return "1"
			}
			return "0"
		case gradlePOCEdgeCacheURLEnvironment:
			return pocEdgeCacheURL
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
