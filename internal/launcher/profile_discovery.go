package launcher

import (
	"flag"
	"fmt"
	"io"

	"github.com/tonyredondo/buildopt/internal/profilediscovery"
)

const profileDiscoveryUsage = "usage: buildopt profile discover --manifest PATH --graph PATH --generated-manifest PATH --matrix-summary PATH --cell-evidence PATH --profile-contract PATH\n"

func runProfileDiscovery(args []string, stdout, stderr io.Writer) int {
	if (len(args) == 1 && isHelp(args)) ||
		(len(args) == 2 && args[0] == "discover" && isHelp(args[1:])) {
		_, _ = io.WriteString(stdout, profileDiscoveryUsage)
		return 0
	}
	if len(args) == 0 || args[0] != "discover" {
		_, _ = io.WriteString(stderr, profileDiscoveryUsage)
		return exitUsage
	}
	flags := flag.NewFlagSet("buildopt profile discover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifest := flags.String("manifest", "", "checked Build Impact manifest")
	graph := flags.String("graph", "", "checked Build Impact graph")
	generated := flags.String("generated-manifest", "", "checked generated-state binding")
	matrix := flags.String("matrix-summary", "", "terminal installed-profile matrix")
	evidence := flags.String("cell-evidence", "", "one matrix cell evidence document")
	contract := flags.String("profile-contract", "", "reviewed POC profile contract")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *manifest == "" || *graph == "" || *generated == "" || *matrix == "" || *evidence == "" || *contract == "" {
		_, _ = io.WriteString(stderr, profileDiscoveryUsage)
		return exitUsage
	}
	repositoryRoot, err := canonicalWorkingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	report, err := profilediscovery.Discover(profilediscovery.Options{
		RepositoryRoot: repositoryRoot, ManifestPath: *manifest, GraphPath: *graph,
		GeneratedPath: *generated, MatrixSummaryPath: *matrix,
		CellEvidencePath: *evidence, ProfileContractPath: *contract,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	raw, err := profilediscovery.Render(report)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: profile discovery unavailable: %v\n", err)
		return exitConfiguration
	}
	if _, err := stdout.Write(raw); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt: write profile discovery: %v\n", err)
		return exitConfiguration
	}
	return 0
}
