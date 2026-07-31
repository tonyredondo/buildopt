package launcher

import (
	"errors"
	"path/filepath"
	"strconv"

	"github.com/tonyredondo/buildopt/internal/runtimeoptimizer"
)

const (
	resourceRunnerClassEnvironment     = "BUILDOPT_RUNNER_CLASS"
	resourceBuildClassEnvironment      = "BUILDOPT_BUILD_CLASS"
	resourceJDKVendorEnvironment       = "BUILDOPT_RUNNER_JDK_VENDOR"
	resourceJDKVersionEnvironment      = "BUILDOPT_RUNNER_JDK_VERSION"
	resourceJDKArchitectureEnvironment = "BUILDOPT_RUNNER_JDK_ARCHITECTURE"
	resourceCgroupCPUEnvironment       = "BUILDOPT_RUNNER_CGROUP_CPU_COUNT"
	resourceCgroupMemoryEnvironment    = "BUILDOPT_RUNNER_CGROUP_MEMORY_BYTES"
	resourceAvailableMemoryEnvironment = "BUILDOPT_RUNNER_AVAILABLE_MEMORY_BYTES"
)

func applyAuthorizedResourceProfile(childArgs []string, authority *localAuthorityContext, getenv func(string) string) ([]string, error) {
	if authority == nil || !authority.resourceProfileAuthorized || len(childArgs) == 0 || filepath.Base(childArgs[0]) != "gradlew" {
		return childArgs, nil
	}
	if getenv == nil {
		return childArgs, errors.New("resource profile: environment reader is unavailable")
	}
	cpuCount, err := parsePositiveResourceInt(getenv(resourceCgroupCPUEnvironment), resourceCgroupCPUEnvironment)
	if err != nil {
		return childArgs, err
	}
	memoryBytes, err := parsePositiveResourceInt64(getenv(resourceCgroupMemoryEnvironment), resourceCgroupMemoryEnvironment)
	if err != nil {
		return childArgs, err
	}
	availableBytes, err := parsePositiveResourceInt64(getenv(resourceAvailableMemoryEnvironment), resourceAvailableMemoryEnvironment)
	if err != nil {
		return childArgs, err
	}
	selection, err := runtimeoptimizer.SelectGoldenResourceProfile(
		authority.resourceProfile.ProfileID,
		authority.resourceProfile.ProfileDigest,
		authority.resourceProfile.CatalogVersion,
		runtimeoptimizer.ResourceProfileContext{
			RunnerClass: getenv(resourceRunnerClassEnvironment), BuildClass: getenv(resourceBuildClassEnvironment),
			CompatibilityClass: authority.managedL1Config.compatibilityClass,
			JDKVendor:          getenv(resourceJDKVendorEnvironment), JDKVersion: getenv(resourceJDKVersionEnvironment), JDKArchitecture: getenv(resourceJDKArchitectureEnvironment),
			CgroupCPUCount: cpuCount, CgroupMemoryBytes: memoryBytes, AvailableMemoryBytes: availableBytes,
		},
	)
	if err != nil {
		return childArgs, err
	}
	return runtimeoptimizer.ApplyResourceProfileArguments(childArgs, selection)
}

func parsePositiveResourceInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 || strconv.Itoa(parsed) != value {
		return 0, errors.New("resource profile: invalid " + name)
	}
	return parsed, nil
}

func parsePositiveResourceInt64(value, name string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 || strconv.FormatInt(parsed, 10) != value {
		return 0, errors.New("resource profile: invalid " + name)
	}
	return parsed, nil
}
