package schemavalidator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type capabilityMatrix struct {
	SchemaVersion             string                  `json:"schemaVersion"`
	OperatingSystem           string                  `json:"operatingSystem"`
	Architecture              string                  `json:"architecture"`
	UnknownCombinationProfile string                  `json:"unknownCombinationProfile"`
	Profiles                  []capabilityProfile     `json:"profiles"`
	Combinations              []capabilityCombination `json:"combinations"`
}

type capabilityProfile struct {
	ID           string             `json:"id"`
	Capabilities []capabilityRecord `json:"capabilities"`
}

type capabilityRecord struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Method   string   `json:"method"`
	Reason   string   `json:"reason"`
	Fallback string   `json:"fallback"`
	Evidence []string `json:"evidence"`
}

type capabilityCombination struct {
	Gradle  string `json:"gradle"`
	JDK     int    `json:"jdk"`
	DSL     string `json:"dsl"`
	Profile string `json:"profile"`
}

func TestCapabilityMatrixV1(t *testing.T) {
	t.Parallel()

	matrix := loadCapabilityMatrix(t)
	if matrix.SchemaVersion != "buildopt.specs/capability-matrix/v1" ||
		matrix.OperatingSystem != "linux" ||
		matrix.Architecture != "amd64" ||
		matrix.UnknownCombinationProfile != "UNTESTED" {
		t.Errorf("invalid matrix identity")
	}
	profiles := validateCapabilityProfiles(t, matrix.Profiles)
	if _, exists := profiles[matrix.UnknownCombinationProfile]; !exists {
		t.Errorf("unknown-combination profile is absent")
	}
	validateTierOneCombinations(t, matrix.Combinations, profiles)
}

func loadCapabilityMatrix(t *testing.T) capabilityMatrix {
	t.Helper()
	path := filepath.Join(
		findRepositoryRoot(t),
		"specs",
		"capability-matrix-v1.json",
	)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var matrix capabilityMatrix
	if err := decoder.Decode(&matrix); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s has trailing data: %v", path, err)
	}
	return matrix
}

func validateCapabilityProfiles(
	t *testing.T,
	profiles []capabilityProfile,
) map[string]capabilityProfile {
	t.Helper()
	wantCapabilities := []string{
		"PLUGIN_HANDSHAKE",
		"CONFIGURATION_CACHE_REUSE",
		"TASK_OUTCOMES",
		"TASK_TO_NATIVE_PUT",
		"CRITICAL_PATH",
		"CACHE_MISS_REASONS",
		"MANAGED_SHARED_CACHE",
		"JVM_AGENT",
		"HERMETIC_PRODUCER",
		"PATCH_BUNDLE_APPLIER",
	}
	result := make(map[string]capabilityProfile, len(profiles))
	for _, profile := range profiles {
		if _, duplicate := result[profile.ID]; duplicate || profile.ID == "" {
			t.Fatalf("empty or duplicate profile %q", profile.ID)
		}
		result[profile.ID] = profile
		if len(profile.Capabilities) != len(wantCapabilities) {
			t.Errorf(
				"%s capability count = %d",
				profile.ID,
				len(profile.Capabilities),
			)
			continue
		}
		for index, capability := range profile.Capabilities {
			if capability.ID != wantCapabilities[index] {
				t.Errorf(
					"%s capability %d = %s, want %s",
					profile.ID,
					index,
					capability.ID,
					wantCapabilities[index],
				)
			}
			switch capability.Status {
			case "EXACT":
				if capability.Method == "" || capability.Reason != "" ||
					len(capability.Evidence) == 0 {
					t.Errorf("incomplete exact capability: %+v", capability)
				}
			case "APPROXIMATED":
				if capability.Method == "" || capability.Reason != "" ||
					len(capability.Evidence) == 0 {
					t.Errorf("incomplete approximated capability: %+v", capability)
				}
			case "UNAVAILABLE":
				if capability.Method != "" || capability.Reason == "" ||
					capability.Fallback == "" {
					t.Errorf("incomplete unavailable capability: %+v", capability)
				}
			default:
				t.Errorf("invalid capability status: %+v", capability)
			}
		}
	}
	return result
}

func validateTierOneCombinations(
	t *testing.T,
	combinations []capabilityCombination,
	profiles map[string]capabilityProfile,
) {
	t.Helper()
	want := make(map[string]struct{})
	for _, gradle := range []string{"8.14.3", "9.6.1"} {
		jdks := []int{17, 21}
		if gradle == "9.6.1" {
			jdks = append(jdks, 25)
		}
		for _, jdk := range jdks {
			for _, dsl := range []string{"GROOVY", "KOTLIN"} {
				want[fmt.Sprintf("%s/%d/%s", gradle, jdk, dsl)] = struct{}{}
			}
		}
	}
	if len(combinations) != len(want) {
		t.Fatalf("combination count = %d, want %d", len(combinations), len(want))
	}
	seen := make(map[string]struct{}, len(combinations))
	for _, combination := range combinations {
		key := fmt.Sprintf(
			"%s/%d/%s",
			combination.Gradle,
			combination.JDK,
			combination.DSL,
		)
		if _, expected := want[key]; !expected {
			t.Errorf("unexpected Tier 1 combination %s", key)
		}
		if _, duplicate := seen[key]; duplicate {
			t.Errorf("duplicate Tier 1 combination %s", key)
		}
		seen[key] = struct{}{}
		if _, exists := profiles[combination.Profile]; !exists {
			t.Errorf("%s references unknown profile %s", key, combination.Profile)
		}
	}
	golden := findCapabilityCombination(
		combinations,
		"9.6.1",
		21,
		"KOTLIN",
	)
	if golden.Profile != "GOLDEN_PROVEN" {
		t.Errorf("golden profile = %s", golden.Profile)
	}
	correlation := findCapabilityCombination(
		combinations,
		"8.14.3",
		21,
		"KOTLIN",
	)
	if correlation.Profile != "CORRELATION_ONLY" {
		t.Errorf("8.14.3 correlation profile = %s", correlation.Profile)
	}
	executed := 0
	for _, combination := range combinations {
		if combination.JDK == 25 {
			if combination.Profile != "UNTESTED" {
				t.Errorf("unexecuted JDK 25 combination claims profile: %+v", combination)
			}
			continue
		}
		executed++
		if combination != golden && combination != correlation &&
			combination.Profile != "TIER_ONE_FIXTURE_PROVEN" {
			t.Errorf("executed fixture combination has wrong profile: %+v", combination)
		}
	}
	if executed != 8 {
		t.Errorf("executed Tier 1 rows = %d, want 8", executed)
	}
	fixture := findCapability(
		profiles["TIER_ONE_FIXTURE_PROVEN"].Capabilities,
		"CONFIGURATION_CACHE_REUSE",
	)
	if fixture.Status != "EXACT" {
		t.Errorf("fixture Configuration Cache capability: %+v", fixture)
	}
	taskPut := findCapability(
		profiles[golden.Profile].Capabilities,
		"TASK_TO_NATIVE_PUT",
	)
	if taskPut.Status != "UNAVAILABLE" ||
		taskPut.Fallback != "ABORT_PENDING" {
		t.Errorf("unsafe task-to-PUT capability: %+v", taskPut)
	}
}

func findCapabilityCombination(
	combinations []capabilityCombination,
	gradle string,
	jdk int,
	dsl string,
) capabilityCombination {
	for _, combination := range combinations {
		if combination.Gradle == gradle &&
			combination.JDK == jdk &&
			combination.DSL == dsl {
			return combination
		}
	}
	return capabilityCombination{}
}

func findCapability(
	capabilities []capabilityRecord,
	id string,
) capabilityRecord {
	index := slices.IndexFunc(
		capabilities,
		func(capability capabilityRecord) bool {
			return capability.ID == id
		},
	)
	if index < 0 {
		return capabilityRecord{}
	}
	return capabilities[index]
}
