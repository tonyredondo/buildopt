package schemavalidator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPatchBundleV1Vectors(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schema := compileContractSchema(
		t,
		repositoryRoot,
		"patch-bundle.v1.schema.json",
		PatchBundleSchemaID,
	)
	fixtureRoot := filepath.Join(
		repositoryRoot,
		"contracts",
		"jsonschema",
		"testdata",
		"patch-bundle.v1",
	)
	expectedValid := map[string]struct{}{
		"archive-reproducibility.json": {},
		"custom-task-contract.json":    {},
	}
	validFixtures := fixtureFiles(t, filepath.Join(fixtureRoot, "valid"))
	if len(validFixtures) != len(expectedValid) {
		t.Fatalf(
			"found %d valid PatchBundle fixtures, want %d",
			len(validFixtures),
			len(expectedValid),
		)
	}
	for _, fixturePath := range validFixtures {
		if _, ok := expectedValid[filepath.Base(fixturePath)]; !ok {
			t.Fatalf("undocumented valid fixture %s", filepath.Base(fixturePath))
		}
		bundle := readJSONObject(t, fixturePath)
		if err := schema.Validate(bundle); err != nil {
			t.Fatalf(
				"%s must validate: %v",
				relativePath(repositoryRoot, fixturePath),
				err,
			)
		}
		if err := validatePatchBundleSemantics(fixtureRoot, bundle); err != nil {
			t.Fatalf(
				"%s has invalid semantics: %v",
				relativePath(repositoryRoot, fixturePath),
				err,
			)
		}
	}

	mutationPaths := fixtureFiles(t, filepath.Join(fixtureRoot, "invalid"))
	if len(mutationPaths) != 13 {
		t.Fatalf("found %d invalid PatchBundle mutations, want 13", len(mutationPaths))
	}
	for _, mutationPath := range mutationPaths {
		mutationPath := mutationPath
		t.Run("invalid/"+filepath.Base(mutationPath), func(t *testing.T) {
			mutation := readPatchMutation(t, mutationPath)
			basePath := filepath.Clean(
				filepath.Join(filepath.Dir(mutationPath), mutation.Base),
			)
			if !strings.HasPrefix(basePath, filepath.Join(fixtureRoot, "valid")+string(os.PathSeparator)) {
				t.Fatalf("mutation base escapes valid fixture directory: %s", mutation.Base)
			}
			bundle := readJSONObject(t, basePath)
			applyPatchMutation(t, bundle, mutation)

			schemaErr := schema.Validate(bundle)
			switch mutation.Layer {
			case "schema":
				if schemaErr == nil {
					t.Fatal("schema-layer mutation must be rejected")
				}
				if !strings.Contains(schemaErr.Error(), mutation.Diagnostic) {
					t.Fatalf(
						"schema diagnostic = %v, want %q",
						schemaErr,
						mutation.Diagnostic,
					)
				}
			case "semantic":
				if schemaErr != nil {
					t.Fatalf("semantic mutation must remain schema-valid: %v", schemaErr)
				}
				err := validatePatchBundleSemantics(fixtureRoot, bundle)
				if err == nil {
					t.Fatal("semantic mutation must be rejected")
				}
				if !strings.Contains(err.Error(), mutation.Diagnostic) {
					t.Fatalf(
						"semantic diagnostic = %v, want %q",
						err,
						mutation.Diagnostic,
					)
				}
			default:
				t.Fatalf("unknown mutation layer %q", mutation.Layer)
			}
		})
	}
}

func TestPatchBundleValidatorEntrypoint(t *testing.T) {
	t.Parallel()

	repositoryRoot := findRepositoryRoot(t)
	schemaRoot := filepath.Join(repositoryRoot, "contracts", "jsonschema")
	if err := ValidatePatchBundleV1(
		filepath.Join(schemaRoot, "patch-bundle.v1.schema.json"),
		filepath.Join(
			schemaRoot,
			"testdata",
			"patch-bundle.v1",
			"valid",
			"archive-reproducibility.json",
		),
	); err != nil {
		t.Fatalf("PatchBundle validator entrypoint: %v", err)
	}
}

type patchMutation struct {
	Base       string   `json:"base"`
	Operation  string   `json:"operation"`
	Path       []string `json:"path"`
	Value      any      `json:"value"`
	CopyIndex  int      `json:"copyIndex"`
	Layer      string   `json:"layer"`
	Diagnostic string   `json:"diagnostic"`
}

func readPatchMutation(t *testing.T, path string) patchMutation {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}()
	var mutation patchMutation
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&mutation); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if mutation.Base == "" || len(mutation.Path) == 0 ||
		mutation.Layer == "" || mutation.Diagnostic == "" {
		t.Fatalf("incomplete mutation fixture %s", path)
	}
	return mutation
}

func applyPatchMutation(
	t *testing.T,
	bundle map[string]any,
	mutation patchMutation,
) {
	t.Helper()

	parent := any(bundle)
	for _, component := range mutation.Path[:len(mutation.Path)-1] {
		switch value := parent.(type) {
		case map[string]any:
			next, ok := value[component]
			if !ok {
				t.Fatalf("mutation path component %q is absent", component)
			}
			parent = next
		case []any:
			index, err := strconv.Atoi(component)
			if err != nil || index < 0 || index >= len(value) {
				t.Fatalf("invalid mutation array index %q", component)
			}
			parent = value[index]
		default:
			t.Fatalf("mutation path traverses scalar at %q", component)
		}
	}
	last := mutation.Path[len(mutation.Path)-1]
	switch mutation.Operation {
	case "replace", "add":
		object, ok := parent.(map[string]any)
		if !ok {
			t.Fatalf("mutation target %q is not an object", last)
		}
		if mutation.Operation == "replace" {
			if _, exists := object[last]; !exists {
				t.Fatalf("replace target %q is absent", last)
			}
		}
		object[last] = mutation.Value
	case "append-copy":
		object, ok := parent.(map[string]any)
		if !ok {
			t.Fatalf("append parent is not an object")
		}
		values, ok := object[last].([]any)
		if !ok || mutation.CopyIndex < 0 || mutation.CopyIndex >= len(values) {
			t.Fatalf("append-copy target %q is invalid", last)
		}
		copyBytes, err := json.Marshal(values[mutation.CopyIndex])
		if err != nil {
			t.Fatalf("copy mutation value: %v", err)
		}
		var copied any
		if err := json.Unmarshal(copyBytes, &copied); err != nil {
			t.Fatalf("decode mutation copy: %v", err)
		}
		object[last] = append(values, copied)
	default:
		t.Fatalf("unknown mutation operation %q", mutation.Operation)
	}
}

func validatePatchBundleSemantics(
	fixtureRoot string,
	bundle map[string]any,
) error {
	createdAt, err := parseMapTime(bundle, "createdAt")
	if err != nil {
		return err
	}
	expiresAt, err := parseMapTime(bundle, "expiresAt")
	if err != nil {
		return err
	}
	if !createdAt.Before(expiresAt) {
		return errors.New("bundle validity window is empty")
	}
	validation := mapObject(bundle, "validation")
	validationCompletedAt, err := parseMapTime(validation, "completedAt")
	if err != nil {
		return err
	}
	validationExpiresAt, err := parseMapTime(validation, "expiresAt")
	if err != nil {
		return err
	}
	if createdAt.Before(validationCompletedAt) ||
		validationExpiresAt.Before(expiresAt) {
		return errors.New("validation does not cover bundle lifetime")
	}

	operations, err := mapObjectArray(bundle, "operations")
	if err != nil {
		return err
	}
	blobs, err := mapObjectArray(bundle, "blobs")
	if err != nil {
		return err
	}
	blobByRef := make(map[string]map[string]any, len(blobs))
	blobRefs := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		ref := mapString(blob, "blobRef")
		if _, duplicate := blobByRef[ref]; duplicate {
			return fmt.Errorf("duplicate blob reference %s", ref)
		}
		blobByRef[ref] = blob
		blobRefs = append(blobRefs, ref)
		if err := validateFixtureBlob(fixtureRoot, blob); err != nil {
			return err
		}
	}
	if !sort.StringsAreSorted(blobRefs) {
		return errors.New("blobs are not sorted by blobRef")
	}

	paths := make(map[string]struct{}, len(operations))
	usedBlobs := make(map[string]struct{}, len(operations))
	for index, operation := range operations {
		order, ok := operation["order"].(float64)
		if !ok || int(order) != index+1 {
			return fmt.Errorf("operation order at index %d is not contiguous", index)
		}
		path := mapString(operation, "path")
		if _, duplicate := paths[path]; duplicate {
			return fmt.Errorf("duplicate operation path %s", path)
		}
		paths[path] = struct{}{}
		blobRef := mapString(operation, "replacementBlob")
		if _, duplicate := usedBlobs[blobRef]; duplicate {
			return fmt.Errorf("replacement blob %s is reused", blobRef)
		}
		usedBlobs[blobRef] = struct{}{}
		blob, exists := blobByRef[blobRef]
		if !exists {
			return fmt.Errorf("operation references unknown blob %s", blobRef)
		}
		if mapString(operation, "postimageDigest") !=
			mapString(blob, "blobSha256") {
			return fmt.Errorf("operation postimage differs from blob %s", blobRef)
		}
	}
	if len(usedBlobs) != len(blobByRef) {
		return errors.New("bundle contains an unreferenced blob")
	}

	calculatedDigest, err := calculatePatchBundleDigest(bundle)
	if err != nil {
		return err
	}
	if got := mapString(bundle, "bundleDigest"); got != calculatedDigest {
		return fmt.Errorf("bundleDigest = %s, want %s", got, calculatedDigest)
	}
	signature := mapObject(bundle, "signature")
	if mapString(signature, "signedBundleDigest") != calculatedDigest {
		return errors.New("signature is bound to another bundle digest")
	}
	return nil
}

func validateFixtureBlob(fixtureRoot string, blob map[string]any) error {
	ref := mapString(blob, "blobRef")
	root := filepath.Clean(fixtureRoot)
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(ref)))
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return fmt.Errorf("blob reference escapes fixture root: %s", ref)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("lstat blob %s: %w", ref, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("blob %s is not a regular file", ref)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read blob %s: %w", ref, err)
	}
	if !utf8.Valid(content) {
		return fmt.Errorf("blob %s is not valid UTF-8", ref)
	}
	size, ok := blob["sizeBytes"].(float64)
	if !ok || int64(size) != int64(len(content)) {
		return fmt.Errorf("blob %s size does not match bytes", ref)
	}
	sum := sha256.Sum256(content)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if mapString(blob, "blobSha256") != digest {
		return fmt.Errorf("blob %s digest does not match bytes", ref)
	}
	return nil
}

func calculatePatchBundleDigest(bundle map[string]any) (string, error) {
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return "", fmt.Errorf("copy bundle: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return "", fmt.Errorf("decode bundle copy: %w", err)
	}
	delete(manifest, "bundleDigest")
	delete(manifest, "signature")
	rawBlobs, ok := manifest["blobs"].([]any)
	if !ok {
		return "", errors.New("blobs are not an array")
	}
	delete(manifest, "blobs")

	blobs := make([]map[string]any, 0, len(rawBlobs))
	for _, rawBlob := range rawBlobs {
		blob, ok := rawBlob.(map[string]any)
		if !ok {
			return "", errors.New("blob is not an object")
		}
		blobs = append(blobs, map[string]any{
			"blobRef":    blob["blobRef"],
			"blobSha256": blob["blobSha256"],
			"sizeBytes":  blob["sizeBytes"],
		})
	}
	sort.Slice(blobs, func(left int, right int) bool {
		return mapString(blobs[left], "blobRef") <
			mapString(blobs[right], "blobRef")
	})
	canonicalInput := map[string]any{
		"manifest": manifest,
		"blobs":    blobs,
	}
	canonical, err := json.Marshal(canonicalInput)
	if err != nil {
		return "", fmt.Errorf("encode canonical bundle input: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
