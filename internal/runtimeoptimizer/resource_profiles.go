package runtimeoptimizer

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	GoldenResourceCatalogVersion = "golden-linux-amd64-4c-16g-v1"
	GoldenRunnerClass            = "linux-amd64-4c-16g-v1"
	GoldenBuildClass             = "LARGE"
	GoldenCompatibilityClass     = "gradle-9.6-jdk21-linux-amd64"
	GoldenJDKVendor              = "temurin"
	GoldenJDKVersion             = "21.0.8+9"
	GoldenJDKArchitecture        = "linux-amd64"
	goldenMemoryBytes            = int64(16 * 1024 * 1024 * 1024)
	goldenMinimumHeadroomBytes   = int64(2 * 1024 * 1024 * 1024)
)

// ResourceProfile is one immutable arm from the finite golden catalog.
type ResourceProfile struct {
	ProfileID            string
	ProfileDigest        string
	CatalogVersion       string
	RunnerClass          string
	BuildClass           string
	CompatibilityClass   string
	JDKVendor            string
	JDKVersion           string
	JDKArchitecture      string
	MaxWorkers           int
	GradleHeapMB         int
	GradleJVMArgs        []string
	CompilerForking      bool
	CompilerMaxProcesses int
	ProcessLimit         int
	OpenFileLimit        int
	CgroupCPUCount       int
	CgroupMemoryBytes    int64
	MinimumHeadroomBytes int64
	StartupPassed        bool
	MemoryPassed         bool
	RollbackPassed       bool
}

// ResourceProfileContext is the complete pre-outcome eligibility input.
type ResourceProfileContext struct {
	RunnerClass          string
	BuildClass           string
	CompatibilityClass   string
	JDKVendor            string
	JDKVersion           string
	JDKArchitecture      string
	CgroupCPUCount       int
	CgroupMemoryBytes    int64
	AvailableMemoryBytes int64
}

// ResourceProfileSelection is a validated profile and its daemon-start arguments.
type ResourceProfileSelection struct {
	Profile   ResourceProfile
	Arguments []string
}

// GoldenResourceProfiles returns defensive copies of the four initial arms.
func GoldenResourceProfiles() []ResourceProfile {
	profiles := []ResourceProfile{
		goldenResourceProfile("STABLE_CONTROL", "a", 2, 4096),
		goldenResourceProfile("W2_H3G", "b", 2, 3072),
		goldenResourceProfile("W3_H4G", "c", 3, 4096),
		goldenResourceProfile("W4_H6G", "d", 4, 6144),
	}
	for index := range profiles {
		profiles[index].GradleJVMArgs = slices.Clone(profiles[index].GradleJVMArgs)
	}
	return profiles
}

// SelectGoldenResourceProfile validates an exact reference and runner context.
func SelectGoldenResourceProfile(profileID, profileDigest, catalogVersion string, context ResourceProfileContext) (ResourceProfileSelection, error) {
	if catalogVersion != GoldenResourceCatalogVersion || !validDigest(profileDigest) {
		return ResourceProfileSelection{}, errors.New("select resource profile: unknown catalog or digest")
	}
	var selected *ResourceProfile
	for _, profile := range GoldenResourceProfiles() {
		if profile.ProfileID == profileID {
			candidate := profile
			selected = &candidate
			break
		}
	}
	if selected == nil || selected.ProfileDigest != profileDigest {
		return ResourceProfileSelection{}, errors.New("select resource profile: profile is not in the finite catalog")
	}
	if reason := resourceProfileIneligibility(*selected, context); reason != "" {
		return ResourceProfileSelection{}, fmt.Errorf("select resource profile: %s", reason)
	}
	return ResourceProfileSelection{
		Profile: *selected,
		Arguments: []string{
			"--max-workers=" + strconv.Itoa(selected.MaxWorkers),
			"-Dorg.gradle.jvmargs=-Xmx" + strconv.Itoa(selected.GradleHeapMB) + "m " + strings.Join(selected.GradleJVMArgs, " "),
		},
	}, nil
}

// ApplyResourceProfileArguments replaces only workers and daemon heap inputs.
func ApplyResourceProfileArguments(arguments []string, selection ResourceProfileSelection) ([]string, error) {
	if len(arguments) == 0 || selection.Profile.ProfileID == "" || len(selection.Arguments) != 2 {
		return nil, errors.New("apply resource profile: invalid arguments or selection")
	}
	result := make([]string, 0, len(arguments)+2)
	result = append(result, arguments[0])
	result = append(result, selection.Arguments...)
	skipNext := false
	for _, argument := range arguments[1:] {
		if skipNext {
			skipNext = false
			continue
		}
		if argument == "--max-workers" {
			skipNext = true
			continue
		}
		if strings.HasPrefix(argument, "--max-workers=") || strings.HasPrefix(argument, "-Dorg.gradle.jvmargs=") {
			continue
		}
		result = append(result, argument)
	}
	if skipNext {
		return nil, errors.New("apply resource profile: --max-workers lacks a value")
	}
	return result, nil
}

func goldenResourceProfile(profileID, digestCharacter string, workers, heapMB int) ResourceProfile {
	return ResourceProfile{
		ProfileID: profileID, ProfileDigest: "sha256:" + strings.Repeat(digestCharacter, 64), CatalogVersion: GoldenResourceCatalogVersion,
		RunnerClass: GoldenRunnerClass, BuildClass: GoldenBuildClass, CompatibilityClass: GoldenCompatibilityClass,
		JDKVendor: GoldenJDKVendor, JDKVersion: GoldenJDKVersion, JDKArchitecture: GoldenJDKArchitecture,
		MaxWorkers: workers, GradleHeapMB: heapMB, GradleJVMArgs: []string{"-Dfile.encoding=UTF-8"},
		CompilerForking: false, CompilerMaxProcesses: 1, ProcessLimit: 1024, OpenFileLimit: 65536,
		CgroupCPUCount: 4, CgroupMemoryBytes: goldenMemoryBytes, MinimumHeadroomBytes: goldenMinimumHeadroomBytes,
		StartupPassed: true, MemoryPassed: true, RollbackPassed: true,
	}
}

func resourceProfileIneligibility(profile ResourceProfile, context ResourceProfileContext) string {
	if context.RunnerClass != profile.RunnerClass || context.BuildClass != profile.BuildClass || context.CompatibilityClass != profile.CompatibilityClass {
		return "runner, build, or compatibility class mismatch"
	}
	if context.JDKVendor != profile.JDKVendor || context.JDKVersion != profile.JDKVersion || context.JDKArchitecture != profile.JDKArchitecture {
		return "JDK identity mismatch"
	}
	if context.CgroupCPUCount != profile.CgroupCPUCount || context.CgroupMemoryBytes != profile.CgroupMemoryBytes {
		return "cgroup identity mismatch"
	}
	if context.AvailableMemoryBytes < int64(profile.GradleHeapMB)*1024*1024+profile.MinimumHeadroomBytes {
		return "insufficient memory headroom"
	}
	if !profile.StartupPassed || !profile.MemoryPassed || !profile.RollbackPassed {
		return "profile eligibility fixture failed"
	}
	return ""
}
