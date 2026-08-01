package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildimpact"
)

const usage = "usage: buildopt-impact <generate|check> --repository ROOT --manifest PATH --repository-id OWNER/REPO --pipeline-class CLASS --graph PATH --generated-manifest PATH\n"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || (args[0] != "generate" && args[0] != "check") {
		fmt.Fprint(os.Stderr, usage)
		return 64
	}
	mode := args[0]
	flags := flag.NewFlagSet("buildopt-impact "+mode, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repository := flags.String("repository", ".", "Gradle repository root")
	manifest := flags.String("manifest", "buildopt-impact-manifest.json", "repository-relative customer manifest")
	repositoryID := flags.String("repository-id", "", "bound owner/repository identity")
	pipelineClass := flags.String("pipeline-class", "", "bound pipeline class")
	graphPath := flags.String("graph", "buildopt-impact-graph.generated.json", "repository-relative generated graph")
	generatedPath := flags.String("generated-manifest", "buildopt-impact.generated.json", "repository-relative generated manifest")
	gradleCommand := flags.String("gradle-command", "", "repository-relative or absolute Gradle command")
	timeout := flags.Duration("timeout", 5*time.Minute, "Gradle discovery timeout")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *repositoryID == "" || *pipelineClass == "" || *timeout <= 0 {
		fmt.Fprint(os.Stderr, usage)
		return 64
	}
	root, err := filepath.Abs(*repository)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	generated, err := buildimpact.Discover(ctx, buildimpact.DiscoveryOptions{
		RepositoryRoot: root,
		ManifestPath:   *manifest,
		RepositoryID:   *repositoryID,
		PipelineClass:  *pipelineClass,
		GradleCommand:  *gradleCommand,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	graphOutput, err := safeOutputPath(root, *graphPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	generatedOutput, err := safeOutputPath(root, *generatedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	if mode == "check" {
		if err := checkExact(graphOutput, generated.GraphJSON); err != nil {
			fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
			return 1
		}
		if err := checkExact(generatedOutput, generated.GeneratedJSON); err != nil {
			fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
			return 1
		}
		fmt.Printf("Build Impact generated state is current (%s)\n", generated.Graph.Digest)
		return 0
	}
	if err := writeAtomic(graphOutput, generated.GraphJSON); err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	if err := writeAtomic(generatedOutput, generated.GeneratedJSON); err != nil {
		fmt.Fprintf(os.Stderr, "buildopt-impact: %v\n", err)
		return 1
	}
	fmt.Printf("Generated conservative Build Impact state (%s)\n", generated.Graph.Digest)
	return 0
}

func safeOutputPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == "." || relative == ".." {
		return "", fmt.Errorf("output path %q must be clean and repository relative", relative)
	}
	output := filepath.Join(root, relative)
	rel, err := filepath.Rel(root, output)
	if err != nil || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("output path %q escapes the repository", relative)
	}
	return output, nil
}

func checkExact(path string, expected []byte) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated state %s: %w", path, err)
	}
	if !bytes.Equal(raw, expected) {
		return fmt.Errorf("generated state drift at %s; run generate and review the diff", path)
	}
	return nil
}

func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".buildopt-impact-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryPath, 0o644); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
