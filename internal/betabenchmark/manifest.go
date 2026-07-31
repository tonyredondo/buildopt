// Package betabenchmark executes and validates reproducible private-beta
// workload slices without promoting partial evidence to the full operational
// gate.
package betabenchmark

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

const (
	manifestSchemaVersion = "buildopt.benchmarks/beta/v1"
	maximumManifestBytes  = 1 << 20
)

type manifest struct {
	SchemaVersion string `json:"schemaVersion"`
	Seed          int64  `json:"seed"`
	Runner        struct {
		Spec                    string `json:"spec"`
		SpecSHA256              string `json:"specSha256"`
		RunnerClass             string `json:"runnerClass"`
		ContainerPlatformDigest string `json:"containerPlatformDigest"`
	} `json:"runner"`
	Components components `json:"components"`
	Clients    []int      `json:"clients"`

	ObjectCycleCount int         `json:"objectCycleCount"`
	ObjectMix        []objectMix `json:"objectMix"`
	Phases           []phase     `json:"phases"`
	GradleFixtures   []struct {
		ID                   string `json:"id"`
		SizeClass            string `json:"sizeClass"`
		RequiredKnownOutputs bool   `json:"requiredKnownOutputs"`
		RequiredCriticalPath bool   `json:"requiredCriticalPath"`
	} `json:"gradleFixtures"`
	Faults []struct {
		ID             string `json:"id"`
		ExpectedSafety string `json:"expectedSafety"`
	} `json:"faults"`
	RequiredReportFields []string `json:"requiredReportFields"`
	RequiredAlerts       []string `json:"requiredAlerts"`
}

type components struct {
	Gradle               string `json:"gradle"`
	JDK                  string `json:"jdk"`
	MetricsCatalog       string `json:"metricsCatalog"`
	MetricsCatalogSHA256 string `json:"metricsCatalogSha256"`
}

type objectMix struct {
	SizeBytes int64 `json:"sizeBytes"`
	Percent   int   `json:"percent"`
}

type phase struct {
	ID               string `json:"id"`
	DurationSeconds  int    `json:"durationSeconds"`
	TargetHitPercent int    `json:"targetHitPercent"`
	AllClientCounts  bool   `json:"allClientCounts"`
}

func loadManifest(path string) (manifest, []byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil ||
		!info.Mode().IsRegular() ||
		info.Size() < 1 ||
		info.Size() > maximumManifestBytes {
		return manifest{}, nil, "", errors.New(
			"benchmark manifest is not a bounded regular file",
		)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, nil, "", errors.New("read benchmark manifest")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var loaded manifest
	if err := decoder.Decode(&loaded); err != nil {
		return manifest{}, nil, "", fmt.Errorf(
			"decode benchmark manifest: %w",
			err,
		)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return manifest{}, nil, "", errors.New(
			"benchmark manifest must contain exactly one JSON value",
		)
	}
	if err := validateManifest(loaded); err != nil {
		return manifest{}, nil, "", err
	}
	digest := sha256.Sum256(raw)
	return loaded, raw, "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validateManifest(loaded manifest) error {
	if loaded.SchemaVersion != manifestSchemaVersion ||
		loaded.Seed != 2026072901 ||
		loaded.ObjectCycleCount != 10000 ||
		!slices.Equal(loaded.Clients, []int{1, 8, 32}) {
		return errors.New("benchmark manifest identity or load matrix drifted")
	}
	expectedMix := []objectMix{
		{SizeBytes: 65536, Percent: 70},
		{SizeBytes: 1048576, Percent: 20},
		{SizeBytes: 10485760, Percent: 8},
		{SizeBytes: 104857600, Percent: 2},
	}
	if !slices.Equal(loaded.ObjectMix, expectedMix) {
		return errors.New("benchmark object mix drifted")
	}
	expectedPhases := []phase{
		{
			ID:               "COLD",
			DurationSeconds:  300,
			TargetHitPercent: 0,
			AllClientCounts:  true,
		},
		{
			ID:               "WARM_70",
			DurationSeconds:  300,
			TargetHitPercent: 70,
			AllClientCounts:  true,
		},
		{
			ID:               "SUSTAINED",
			DurationSeconds:  3600,
			TargetHitPercent: 70,
			AllClientCounts:  true,
		},
		{
			ID:               "SOAK",
			DurationSeconds:  28800,
			TargetHitPercent: 70,
			AllClientCounts:  true,
		},
	}
	if !slices.Equal(loaded.Phases, expectedPhases) {
		return errors.New("benchmark phase matrix drifted")
	}
	expectedFaults := []string{
		"GATEWAY_RESTART",
		"SERVER_RESTART",
		"MID_PUT_CANCEL",
		"MID_GET_CANCEL",
		"TRUNCATED_BLOB",
		"CORRUPT_BLOB",
		"NETWORK_LATENCY",
		"NETWORK_LOSS",
		"SQLITE_BUSY",
		"EXPIRED_LEASE",
		"DISK_HIGH_WATERMARK",
		"DISK_OUT_OF_SPACE",
		"REVOKED_POLICY",
		"REVOKED_GRANT",
		"DEATH_BETWEEN_PENDING_AND_COMMIT",
	}
	actualFaults := make([]string, 0, len(loaded.Faults))
	for _, fault := range loaded.Faults {
		if fault.ExpectedSafety == "" {
			return errors.New("benchmark fault lacks expected safety")
		}
		actualFaults = append(actualFaults, fault.ID)
	}
	if !slices.Equal(actualFaults, expectedFaults) {
		return errors.New("benchmark fault matrix drifted")
	}
	expectedFixtures := []string{
		"TIER1_SMALL",
		"TIER1_MEDIUM",
		"TIER1_LARGE",
	}
	actualFixtures := make([]string, 0, len(loaded.GradleFixtures))
	for _, fixture := range loaded.GradleFixtures {
		if !fixture.RequiredKnownOutputs ||
			!fixture.RequiredCriticalPath {
			return errors.New("benchmark Gradle fixture is incomplete")
		}
		actualFixtures = append(actualFixtures, fixture.ID)
	}
	expectedReportFields := []string{
		"benchmarkDigest",
		"seed",
		"startedAt",
		"completedAt",
		"hardware",
		"cgroup",
		"components",
		"actualObjectDistribution",
		"p50",
		"p95",
		"p99",
		"throughput",
		"errors",
		"bytes",
		"recovery",
		"readinessTransitions",
		"faultOutcomes",
		"deviations",
		"rawObservations",
	}
	expectedAlerts := []string{
		"DISK_QUOTA",
		"CORRUPTION",
		"STUCK_ATTEMPT_OR_LEASE",
		"REVOCATION_LAG",
		"POLICY_FRESHNESS",
		"CIRCUIT_BREAKER",
		"SQLITE_CONTENTION",
		"EXPORT_BACKLOG",
		"ACCEPTANCE_ERROR_RATE",
		"ACCEPTANCE_LATENCY",
	}
	if !slices.Equal(actualFixtures, expectedFixtures) ||
		!slices.Equal(loaded.RequiredReportFields, expectedReportFields) ||
		!slices.Equal(loaded.RequiredAlerts, expectedAlerts) ||
		loaded.Runner.Spec == "" ||
		loaded.Runner.SpecSHA256 == "" ||
		loaded.Runner.RunnerClass == "" ||
		loaded.Runner.ContainerPlatformDigest == "" ||
		loaded.Components.Gradle == "" ||
		loaded.Components.JDK == "" ||
		loaded.Components.MetricsCatalog == "" ||
		loaded.Components.MetricsCatalogSHA256 == "" {
		return errors.New("benchmark evidence surface is incomplete")
	}
	return nil
}
