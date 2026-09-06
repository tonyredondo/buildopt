// Command complete-native-correction-validator checks the static CNC v1 freeze.
// It never starts Gradle, changes subject source, or consumes historical results.
package main

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
	"reflect"
	"strings"
)

// Independent freeze literals are intentional: deriving expected policies or
// identities from the document under test would let that document approve its
// own drift. Row counts and command profiles are reconstructed separately.
const frozenPolicy = `{
  "schemaVersion": "buildopt.specs/poc-complete-native-correction/v1",
  "experiment": "COMPLETE_NATIVE_CORRECTION_V1",
  "status": "STATIC_CONTRACT_ONLY",
  "authority": {
    "localScope": true,
    "budgetApproved": true,
    "realGradleExecution": false,
    "publicSourceMutation": false,
    "publication": false,
    "phaseB": false,
    "phaseC": false,
    "repositoryOrTaskNameRules": false,
    "predecessorRowsAsEvidence": false
  },
  "budget": {
    "maximumStarts": 60,
    "maximumElapsedSeconds": 7200,
    "progressReviewSeconds": 1800,
    "maximumSecondsPerStart": 1200,
    "maximumAdditionalDiskGiB": 20,
    "minimumFreeDiskGiB": 10,
    "maximumWorkers": 4,
    "concurrentStarts": 1,
    "maximumCandidates": 1,
    "maximumInfrastructureReplacements": 2,
    "clock": "MONOTONIC_FROM_EXECUTION_PREPARATION_INCLUDING_DOWNLOADS_WAITS_AND_VALIDATION",
    "resetOnResume": false
  },
  "environment": {
    "scope": "LOCAL_NON_CI",
    "absent": [
      "CI",
      "RELEASE_VERSION"
    ],
    "os": "linux",
    "architecture": "amd64",
    "mainsPower": true,
    "cpuAffinity": "0-3",
    "powerProfile": "performance",
    "scalingDriver": "intel_pstate",
    "energyPerformancePreference": "performance",
    "autoToolchainDownload": false,
    "autoToolchainDiscovery": false,
    "quiescenceSeconds": 120,
    "stabilitySamples": 7,
    "maximumStabilityRatio": 1.15
  },
  "gates": {
    "completeBlockerClosure": true,
    "exactOutputBytes": true,
    "maximumProductFailures": 0,
    "minimumMaterialMilliseconds": 500,
    "minimumMaterialPercent": 2,
    "balancedPairs": 8,
    "requiredPositivePairs": 8,
    "minimumMeanSavingMilliseconds": 500,
    "minimumMeanSavingPercent": 2,
    "bootstrapResamples": 4096,
    "bootstrapSeedMultiplier": 2654435761,
    "bootstrapMultiplier": 1664525,
    "bootstrapIncrement": 1013904223,
    "bootstrapModulus": 4294967296,
    "bootstrapIndexDivisor": 536870912,
    "intervalIndices": [
      102,
      3993
    ],
    "positiveIntervalLower": true,
    "p95": "NEAREST_RANK_MAX_OF_EIGHT",
    "nonRegressiveP95": true,
    "maximumMachinePaybackBuilds": 300,
    "humanCost": "SEPARATE_LEDGER",
    "ciWallTimeGate": false,
    "relocationClaim": false
  },
  "executionPrerequisites": [
    "COMMITTED_CONTRACT_IDENTITY",
    "REGISTERED_SHARED_GIT_WORKTREES",
    "VERIFIED_SOURCE_ARCHIVE_AND_FILES",
    "VERIFIED_RUNTIME_ARCHIVES_AND_IDENTITIES",
    "CNC_003_PACKAGE_AND_FAKE_CHILD_PROOF",
    "FRESH_NATIVE_MATERIALITY_BEFORE_RECIPE",
    "COMPLETE_FIXTURE_PROOF_BEFORE_PUBLIC_CANDIDATE",
    "EXACT_NATIVE_NATIVE_AND_CANDIDATE_OUTPUTS_BEFORE_VALUE"
  ],
  "executionOrder": [
    "P01",
    "P02",
    "D01",
    "D02",
    "M01",
    "M02",
    "F01",
    "F02",
    "F03",
    "F04",
    "F05",
    "F06",
    "F07",
    "F08",
    "F09",
    "F10",
    "F11",
    "F12",
    "F13",
    "F14",
    "F15",
    "F16",
    "F17",
    "F18",
    "F19",
    "F20",
    "F21",
    "F22",
    "F23",
    "F24",
    "C01",
    "C02",
    "C03",
    "C04",
    "C05",
    "C06",
    "C07",
    "C08",
    "C09",
    "C10",
    "V01",
    "V02",
    "V03",
    "V04",
    "V05",
    "V06",
    "V07",
    "V08",
    "V09",
    "V10",
    "V11",
    "V12",
    "V13",
    "V14",
    "V15",
    "V16",
    "V17",
    "V18"
  ],
  "stateProtocol": {
    "worktrees": "REGISTERED_SHARED_GIT_DETACHED_NO_CLONES_OR_SOURCE_COPIES",
    "publicRoots": [
      "native-a",
      "native-b",
      "candidate"
    ],
    "initialState": "EMPTY_PRIVATE_GRADLE_HOMES_AND_BUILD_DIRECTORIES",
    "preparation": "P01_NATIVE_A_AND_P02_NATIVE_B_THEN_IMMUTABLE_DEPENDENCY_ONLY_SEED_TO_ALL_ARMS",
    "seedExclusions": [
      "configuration-cache",
      "build-cache",
      "task-history",
      "outputs",
      "daemon-state"
    ],
    "networkAfterPreparation": "OFFLINE_BOTH_ARMS",
    "correctness": "C01_NATIVE_A_C02_NATIVE_B_C03_TO_C09_CANDIDATE_EVOLVING_C10_REVERT_CANDIDATE_ROOT",
    "value": "RESTORE_DECLARED_SOURCE_REMOVE_CORRECTNESS_OUTPUT_AND_EXECUTION_STATE_KEEP_DEPENDENCIES_BOTH_ARMS",
    "valueOutputs": "REMOVE_ONLY_DECLARED_PRODUCER_OUTPUTS_SYMMETRICALLY_BEFORE_EACH_ROW_KEEP_NATIVE_CACHES",
    "reserve": "REPEAT_EXACT_FAILED_INFRASTRUCTURE_ROW_AT_MOST_ONCE_PER_FAILURE_RETAIN_BOTH",
    "deadlines": "CHECK_BEFORE_EVERY_CHILD_AND_ENFORCE_REMAINING_WINDOW_ON_CHILD_GROUP",
    "zeroStartRefusal": "RETAIN_REASON_WITHOUT_COUNTING_AS_STARTED",
    "startedAttempt": "ATOMIC_RESERVATION_BEFORE_SPAWN_AND_IMMUTABLE_TERMINAL_CAPTURE",
    "bootstrap": "PAIRED_DELTA_MEANS_USING_FIXED_LCG_AND_FROZEN_ORDER_NOT_HISTORICAL_ROWS",
    "fixtureDependencies": "REUSE_OWNER_RESOLVED_TEST_CLASSPATH_NO_NEW_DEPENDENCIES",
    "commandExpansion": "EXACT_PROFILE_PLUS_PRIVATE_GRADLE_HOME_PINNED_JDK_PATHS_AND_OFFLINE_AFTER_PREPARATION",
    "correctnessBuildCache": "DISABLED_SYMMETRICALLY_NO_RERUN_TASKS",
    "correctnessOutputReset": "REMOVE_CAPTURED_PRODUCER_OUTPUTS_BEFORE_C01_TO_C04_PRESERVE_CANDIDATE_STATE_FOR_C05_TO_C09"
  },
  "fixtureInputProofs": [
    {
      "row": "F03",
      "previous": "F02",
      "change": "NONE",
      "expected": "REUSE",
      "maskingChangeAllowed": false
    },
    {
      "row": "F05",
      "previous": "F03",
      "change": "GIT_BRANCH_OUTPUT_ONLY",
      "expected": "INVALIDATE",
      "maskingChangeAllowed": false
    },
    {
      "row": "F07",
      "previous": "F05",
      "change": "HEAD_COMMIT_ONLY_BRANCH_OUTPUT_UNCHANGED",
      "expected": "REUSE",
      "maskingChangeAllowed": false
    },
    {
      "row": "F09",
      "previous": "F07",
      "change": "RELEASE_VERSION_ONLY",
      "expected": "INVALIDATE",
      "maskingChangeAllowed": false
    }
  ],
  "publicMutations": [
    {
      "id": "unrelated",
      "path": "cnc-unrelated-note.txt",
      "operation": "ADD_UTF8_CNC_UNRELATED_NEWLINE",
      "correctnessRows": [
        "C05"
      ],
      "expected": "NO_REQUIRED_OUTPUT_OR_CONFIGURATION_CHANGE"
    },
    {
      "id": "source",
      "path": "src/main/java/graphql/Assert.java",
      "operation": "INSERT_PRIVATE_STATIC_FINAL_INT_CNC_PROOF_SENTINEL_EQUALS_1729_IN_TOP_LEVEL_CLASS",
      "correctnessRows": [
        "C06",
        "C07"
      ],
      "expected": "RECOMPILE_AND_CHANGED_CLASS_BYTES_IDENTICAL_BETWEEN_ARMS"
    },
    {
      "id": "compile-failure",
      "path": "src/main/java/graphql/Assert.java",
      "operation": "REPLACE_CNC_PROOF_SENTINEL_INT_INITIALIZER_WITH_STRING_CNC_INTENTIONAL_TYPE_ERROR",
      "correctnessRows": [
        "C08",
        "C09"
      ],
      "expected": "SAME_DECLARED_COMPILE_FAILURE"
    },
    {
      "id": "revert",
      "path": "ALL_TRANSACTION_AND_PROBE_FILES",
      "operation": "EXACT_INVERSE_NO_GIT_RESET_OR_FORCED_CLEANUP",
      "correctnessRows": [
        "C10"
      ],
      "expected": "ORIGINAL_SOURCE_AND_REQUIRED_OUTPUTS"
    }
  ]
}`
const frozenSubject = `{
  "schemaVersion": "buildopt.specs/poc-complete-native-correction-subjects/v1",
  "selection": "DIRECTED_KNOWN_SUBJECT_NOT_PREVALENCE",
  "subjects": [
    {
      "repository": "https://github.com/graphql-java/graphql-java.git",
      "revision": "f2d8c9126f898c084b176631b7346bc6fbec296a",
      "archiveSha256": "83e80bbaa2b3e3308dd35e0b7120399f02149507674d958b0e2e4cc81718d051",
      "archiveVerification": "REQUIRED_BEFORE_EXECUTION_NOT_YET_OBSERVED",
      "workingDirectory": ".",
      "ownerArguments": [
        "assemble"
      ],
      "files": [
        {
          "path": "build.gradle",
          "sha256": "27cfbb5dc699619d3398eb651b9e6cb4dda1d6fd47f87591849d01284841bbbc"
        },
        {
          "path": ".github/workflows/pull_request.yml",
          "sha256": "05478c6b84875e6be640c96fc4fda2c4a4383f7a11d777b03710be3430e5ee97"
        },
        {
          "path": "gradle/wrapper/gradle-wrapper.properties",
          "sha256": "1d126b2f29d1a87825e57ae7a7a1b1cc06f8dcbc89dd614dc6a5b493244ba2cc"
        },
        {
          "path": "settings.gradle",
          "sha256": "b2df82d2e3c57a8ae2b916389aeeb50fad13fac8d9a9c5f0d2411bda0d812034"
        },
        {
          "path": "gradlew",
          "sha256": "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"
        },
        {
          "path": "src/main/java/graphql/Assert.java",
          "sha256": "915c50745bec947434f97443cddd604737455fa27971e820523f0f00ecadf6cf"
        },
        {
          "path": "performance-results-page/build.gradle",
          "sha256": "592097005031ce85c981febc39a51d39748b6195c4e554272e5cc04d9a0044d1"
        },
        {
          "path": "gradle.properties",
          "sha256": "6d4fd01f004c17b30d9c4374c6fc7cba69c4462a4a8dfec7314cfb57a6ccfb94"
        }
      ],
      "bindings": [
        {
          "id": "B1",
          "path": "build.gradle",
          "startLine": 144,
          "endLine": 144,
          "sha256": "f3f00a9b11758c5356f5dac6817929646653f157936ae794a39eae452689f8b5",
          "owner": "getDevelopmentVersion/git-worktree",
          "proof": [
            "F01",
            "F02",
            "F03",
            "F04",
            "F05",
            "F12",
            "F13",
            "F14",
            "F15",
            "F16",
            "F17"
          ]
        },
        {
          "id": "B2",
          "path": "build.gradle",
          "startLine": 140,
          "endLine": 173,
          "sha256": "02c563e01c795bc1dbe4f4087cd88e8d14cec9b88dddb103e82ea7c87323300e",
          "owner": "getDevelopmentVersion/version-consumers",
          "proof": [
            "F03",
            "F05",
            "F07",
            "F09",
            "F10",
            "F11",
            "F16",
            "F17"
          ]
        },
        {
          "id": "B3",
          "path": "build.gradle",
          "startLine": 186,
          "endLine": 197,
          "sha256": "4ba1d2d651280f4eaebe6ba7e7956bc81e4012b63a229c394f5e6956d6927645",
          "owner": "build-result-summary",
          "proof": [
            "F01",
            "F03",
            "F18",
            "F19",
            "F20",
            "F21",
            "F22",
            "F23"
          ]
        },
        {
          "id": "B4",
          "path": "build.gradle",
          "startLine": 938,
          "endLine": 980,
          "sha256": "104afb47711254e5e886d3e9fcb9bf245f6434703cc0dfafaa21e88999593f72",
          "owner": "test-summary",
          "proof": [
            "F01",
            "F03",
            "F18",
            "F19",
            "F22",
            "F23",
            "F24"
          ]
        },
        {
          "id": "B4-state",
          "path": "build.gradle",
          "startLine": 751,
          "endLine": 788,
          "sha256": "5f4405ce448bbbddd2b1403a687dd99e6f6c995056b183f7bde41fbe9e6b49c0",
          "owner": "test-listener-state",
          "proof": [
            "F01",
            "F03",
            "F24"
          ]
        },
        {
          "id": "B5-consumer",
          "path": "build.gradle",
          "startLine": 468,
          "endLine": 495,
          "sha256": "0ebff2808ad0f53f8e8d121754df3364bcfeb88a866ce2a2e36b14a9c28246d2",
          "owner": "final-jar-and-outgoing-artifact-consumers",
          "proof": [
            "F01",
            "F02",
            "F03",
            "C01",
            "C02",
            "C03",
            "C04"
          ]
        },
        {
          "id": "B5-plain",
          "path": "build.gradle",
          "startLine": 204,
          "endLine": 218,
          "sha256": "95d39640882dd379002da55a3a727b8907004f09a1df9c355ce1e636d6cf4cef",
          "owner": "plain-jar-manifest-configuration",
          "proof": [
            "F01",
            "F02",
            "F03",
            "C01",
            "C02",
            "C03",
            "C04"
          ]
        },
        {
          "id": "B5-shadow",
          "path": "build.gradle",
          "startLine": 311,
          "endLine": 331,
          "sha256": "33d1ea33bd03f0a3dc3a8b5f9a0dc033a32c9c06c8b51c38d9ab7cbfba8afe82",
          "owner": "shadow-bnd-instructions",
          "proof": [
            "F01",
            "F02",
            "F03",
            "C01",
            "C02",
            "C03",
            "C04"
          ]
        }
      ],
      "plugin": {
        "repository": "https://github.com/bndtools/bnd.git",
        "revision": "47e504d7881ba466703c55a8dca7b0578561582d",
        "version": "7.1.0",
        "path": "gradle-plugins/biz.aQute.bnd.gradle/src/main/java/aQute/bnd/gradle/BundleTaskExtension.java",
        "sha256": "5d9532d8ee2b82580748f225b0114378d1cbe646d21c4abf513177bf15f520c0",
        "startLine": 393,
        "endLine": 408,
        "spanSha256": "718118dc57c9464100722d199024df69f77bdcd41a2fb03cbe948bb8e29b92f9",
        "runtimeBinaryVerification": "REQUIRED_BEFORE_FIXTURES"
      },
      "outputs": {
        "selectors": [
          "build/libs/*.jar",
          "performance-results-page/build/libs/*.jar",
          "build/intermediates/plain-jar/*.jar",
          "build/intermediates/shadow-jar/*.jar"
        ],
        "producers": [
          ":buildFinalJar",
          ":jcstressJar",
          ":performance-results-page:jar",
          ":jar",
          ":shadowJar"
        ],
        "comparison": "COMPLETE_PATH_SIZE_SHA256_BYTE_EXACT",
        "inventory": "CAPTURE_NATIVE_TWICE_BEFORE_CANDIDATE",
        "allowMissing": false,
        "allowSymlinks": false,
        "allowUnowned": false,
        "allowNormalization": false
      }
    }
  ],
  "runtimes": [
    {
      "role": "daemon-and-root",
      "version": "25.0.4.8.1",
      "url": "https://corretto.aws/downloads/resources/25.0.4.8.1/amazon-corretto-25.0.4.8.1-linux-x64.tar.gz",
      "sha256": "b838e42c8e915019ed34e4cc54c7cda2e7e00d2a2a49be44578814735fc9accc"
    },
    {
      "role": "subproject",
      "version": "21.0.12.9.1",
      "url": "https://corretto.aws/downloads/resources/21.0.12.9.1/amazon-corretto-21.0.12.9.1-linux-x64.tar.gz",
      "sha256": "f79824540cef882da0cdf1369f9d1d69afc14b5a9bc3a771fd5bb795793ce2f2"
    }
  ]
}`

type row struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Scenario string `json:"scenario"`
	Arm      string `json:"arm"`
	Command  string `json:"command"`
	Cache    string `json:"cache"`
	Expect   string `json:"expect"`
}

func main() {
	contractPath := flag.String("contract", "specs/poc-complete-native-correction-v1.json", "static contract")
	subjectsPath := flag.String("subjects", "specs/poc-complete-native-correction-v1.subjects.json", "subject manifest")
	sourceRoot := flag.String("sources-root", "", "optional exact subject root; file/span check only")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(errors.New("unexpected positional arguments"))
	}
	contract, err := os.ReadFile(*contractPath)
	if err != nil {
		fail(err)
	}
	subjects, err := os.ReadFile(*subjectsPath)
	if err != nil {
		fail(err)
	}
	if err = validate(contract, subjects); err != nil {
		fail(err)
	}
	if *sourceRoot != "" {
		if err = verifySources(*sourceRoot, subjects); err != nil {
			fail(err)
		}
	}
	fmt.Println("CNC v1 static contract OK: 60 reconstructed slots, 7200-second ceiling, 1800-second review; no execution proof")
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

// decode rejects duplicate keys as well as trailing JSON. A normal Unmarshal
// would silently replace a duplicate budget or authority field.
func decode(data []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	value, err := readValue(d)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON")
	}
	return value, nil
}

func readValue(d *json.Decoder) (any, error) {
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return token, nil
	}
	switch delimiter {
	case '{':
		object := map[string]any{}
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("non-string key")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("duplicate key %q", key)
			}
			value, err := readValue(d)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		if _, err := d.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case '[':
		array := []any{}
		for d.More() {
			value, err := readValue(d)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if _, err := d.Token(); err != nil {
			return nil, err
		}
		return array, nil
	default:
		return nil, errors.New("unexpected delimiter")
	}
}

func normalized(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	result, err := decode(data)
	if err != nil {
		panic(err)
	}
	return result
}

func validate(contractData, subjectData []byte) error {
	value, err := decode(contractData)
	if err != nil {
		return fmt.Errorf("contract: %w", err)
	}
	contract, ok := value.(map[string]any)
	if !ok {
		return errors.New("contract must be an object")
	}
	expectedPolicy, err := decode([]byte(frozenPolicy))
	if err != nil {
		return err
	}
	for key, expected := range expectedPolicy.(map[string]any) {
		if !reflect.DeepEqual(contract[key], expected) {
			return fmt.Errorf("changed or missing policy %q", key)
		}
	}
	if len(contract) != len(expectedPolicy.(map[string]any))+3 {
		return errors.New("unknown contract fields")
	}
	if !reflect.DeepEqual(contract["commands"], normalized(expectedCommands())) {
		return errors.New("command profile drift")
	}

	expectedRows := expectedSchedule()
	if !reflect.DeepEqual(contract["rows"], normalized(expectedRows)) {
		return errors.New("missing, reordered, unknown or changed proof row")
	}
	rowBytes, err := json.Marshal(contract["rows"])
	if err != nil {
		return err
	}
	var actualRows []row
	if err := json.Unmarshal(rowBytes, &actualRows); err != nil {
		return err
	}
	// Reconstruct counts from the checked individual rows, never summary fields.
	counts := map[string]int{}
	ids := map[string]bool{}
	for _, r := range actualRows {
		if ids[r.ID] {
			return fmt.Errorf("duplicate row %s", r.ID)
		}
		ids[r.ID] = true
		counts[r.Category]++
	}
	if len(ids) != 60 || counts["fixtures"] != 24 || counts["correctness"] != 10 ||
		counts["value"] != 18 || counts["reserve"] != 2 {
		return errors.New("invalid independent allocation")
	}
	if !reflect.DeepEqual(contract["allocation"], normalized(counts)) {
		return errors.New("allocation disagrees with reconstructed rows")
	}

	subjectValue, err := decode(subjectData)
	if err != nil {
		return fmt.Errorf("subjects: %w", err)
	}
	subjectObject, ok := subjectValue.(map[string]any)
	if !ok {
		return errors.New("subjects must be an object")
	}
	subjects, ok := subjectObject["subjects"].([]any)
	if !ok || len(subjects) != 1 {
		return errors.New("exactly one directed subject required")
	}
	subject, ok := subjects[0].(map[string]any)
	if !ok {
		return errors.New("invalid subject")
	}
	label, ok := subject["label"].(string)
	if !ok || strings.TrimSpace(label) == "" {
		return errors.New("subject label required")
	}
	// Labels cannot determine admission. All other source/runtime/output bindings
	// are immutable inputs, not name-derived heuristics.
	delete(subject, "label")
	expectedSubject, err := decode([]byte(frozenSubject))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(subjectObject, expectedSubject) {
		return errors.New("source, runtime, proof binding or output contract drift")
	}
	return nil
}

func expectedCommands() map[string][]string {
	flags := []string{"--console=plain", "--stacktrace", "--max-workers=4", "--parallel", "--build-cache"}
	commands := map[string][]string{
		"prefetch":             {"assemble", "testClasses", "--no-configuration-cache"},
		"fixtureNative":        {"fixtureProof", "--no-configuration-cache", "--rerun-tasks"},
		"fixtureCandidate":     {"fixtureProof", "--configuration-cache", "--configuration-cache-problems=fail", "--rerun-tasks"},
		"native":               {"assemble", "--no-configuration-cache"},
		"strict":               {"assemble", "--configuration-cache", "--configuration-cache-problems=fail"},
		"nativeCorrectness":    {"assemble", "verifyShadedClassAnnotations", "compileShadedJarConsumer", "--no-configuration-cache"},
		"candidateCorrectness": {"assemble", "verifyShadedClassAnnotations", "compileShadedJarConsumer", "--configuration-cache", "--configuration-cache-problems=fail"},
	}
	for key, args := range commands {
		commands[key] = append(args, flags...)
		if key == "nativeCorrectness" || key == "candidateCorrectness" {
			commands[key][len(commands[key])-1] = "--no-build-cache"
		}
	}
	return commands
}

func expectedSchedule() []row {
	fixtureCases := []string{"success", "success", "success", "branch", "branch", "commit", "commit", "release", "release", "ci", "ci", "no-git", "no-git", "missing-executable", "missing-executable", "process-matrix", "process-matrix", "task-failure", "task-failure", "configuration-failure", "configuration-failure", "cancellation", "cancellation", "after-cancellation"}
	rows := []row{}
	for i, scenario := range fixtureCases {
		n := i + 1
		arm, command, cache, expect := "native", "fixtureNative", "none", "success"
		candidate := n == 2 || n == 3 || n == 24 || (n >= 4 && n%2 == 1)
		if candidate {
			arm, command, cache = "candidate", "fixtureCandidate", "fresh"
		}
		switch n {
		case 2:
			cache = "store"
		case 3, 7:
			cache = "reuse"
		case 5, 9:
			cache = "invalidate"
		}
		switch scenario {
		case "missing-executable", "configuration-failure":
			expect = "configuration-failure"
		case "process-matrix":
			expect = "captured-path-results"
		case "task-failure":
			expect = "task-failure"
		case "cancellation":
			expect = "cancelled"
		}
		rows = append(rows, row{fmt.Sprintf("F%02d", n), "fixtures", scenario, arm, command, cache, expect})
	}
	for i, category := range []string{"preparation", "diagnostics", "materiality"} {
		for n := 1; n <= 2; n++ {
			command, cache, expect := "native", "none", "success"
			if category == "preparation" {
				command = "prefetch"
			}
			if category == "diagnostics" {
				command, cache, expect = "strict", "fresh", "strict-blockers"
			}
			rows = append(rows, row{fmt.Sprintf("%s%02d", []string{"P", "D", "M"}[i], n), category, category, "native", command, cache, expect})
		}
	}
	correctnessCases := []string{"baseline", "baseline", "baseline", "baseline", "unrelated", "source", "source", "compile-failure", "compile-failure", "revert"}
	for i, scenario := range correctnessCases {
		n := i + 1
		arm, command, cache, expect := "native", "nativeCorrectness", "none", "success"
		switch n {
		case 3, 4, 5, 7, 9:
			arm, command = "candidate", "candidateCorrectness"
		}
		switch n {
		case 3:
			cache = "store"
		case 4, 5:
			cache = "reuse"
		case 7, 9:
			cache = "observe-source-tracking"
		}
		if n == 8 || n == 9 {
			expect = "task-failure"
		}
		rows = append(rows, row{fmt.Sprintf("C%02d", n), "correctness", scenario, arm, command, cache, expect})
	}
	for i := 0; i < 18; i++ {
		pair := 0
		arm := "native"
		scenario := "stabilization"
		if i == 1 {
			arm = "candidate"
		}
		if i >= 2 {
			pair = (i-2)/2 + 1
			scenario = fmt.Sprintf("pair-%d", pair)
			if (pair%2 == 1 && i%2 == 1) || (pair%2 == 0 && i%2 == 0) {
				arm = "candidate"
			}
		}
		command, cache := "native", "none"
		if arm == "candidate" {
			command, cache = "strict", "reuse"
			if pair == 0 {
				cache = "store"
			}
		}
		rows = append(rows, row{fmt.Sprintf("V%02d", i+1), "value", scenario, arm, command, cache, "success"})
	}
	for i := 1; i <= 2; i++ {
		rows = append(rows, row{fmt.Sprintf("R%02d", i), "reserve", "classified-infrastructure-replacement", "original", "original", "original", "original"})
	}
	return rows
}

func verifySources(root string, subjectsData []byte) error {
	var manifest struct {
		Subjects []struct {
			Files []struct {
				Path   string
				SHA256 string
			}
			Bindings []struct {
				Path      string
				StartLine int
				EndLine   int
				SHA256    string
			}
		}
	}
	if err := json.Unmarshal(subjectsData, &manifest); err != nil {
		return err
	}
	if len(manifest.Subjects) != 1 {
		return errors.New("missing subject")
	}
	files := map[string][]byte{}
	for _, file := range manifest.Subjects[0].Files {
		data, err := readRegular(root, file.Path)
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != file.SHA256 {
			return fmt.Errorf("source drift: %s", file.Path)
		}
		files[file.Path] = data
	}
	for _, binding := range manifest.Subjects[0].Bindings {
		data, exists := files[binding.Path]
		if !exists {
			return errors.New("span has no file binding")
		}
		if err := verifySpan(data, binding.StartLine, binding.EndLine, binding.SHA256); err != nil {
			return fmt.Errorf("%s: %w", binding.Path, err)
		}
	}
	return nil
}

func readRegular(root, relative string) ([]byte, error) {
	if !filepath.IsLocal(relative) || filepath.ToSlash(filepath.Clean(relative)) != relative {
		return nil, errors.New("unsafe source path")
	}
	current := root
	for _, part := range strings.Split(relative, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlink source rejected")
		}
	}
	info, err := os.Stat(current)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("source is not a regular file")
	}
	return os.ReadFile(current)
}

func verifySpan(data []byte, start, end int, want string) error {
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if start < 1 || end < start || end > len(lines) {
		return errors.New("invalid source span")
	}
	got := fmt.Sprintf("%x", sha256.Sum256(bytes.Join(lines[start-1:end], nil)))
	if got != want {
		return errors.New("source span drift")
	}
	return nil
}
