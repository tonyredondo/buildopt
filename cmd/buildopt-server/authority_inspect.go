package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
)

type authorityInspection struct {
	SchemaVersion        string                            `json:"schemaVersion"`
	AuthorityDigest      string                            `json:"authorityDigest"`
	Repository           localauthority.RepositoryIdentity `json:"repository"`
	PolicyID             string                            `json:"policyId"`
	PolicyVersion        int64                             `json:"policyVersion"`
	RevocationEpoch      int64                             `json:"revocationEpoch"`
	L1SecurityGeneration int64                             `json:"l1SecurityGeneration"`
	Namespace            string                            `json:"namespace"`
	NamespaceGeneration  int64                             `json:"namespaceGeneration"`
}

func runAuthorityInspect(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 || args[0] != "inspect" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	flags := flag.NewFlagSet(
		"buildopt-server authority inspect",
		flag.ContinueOnError,
	)
	flags.SetOutput(io.Discard)
	authorityPath := flags.String("authority", "", "signed authority document")
	trustRootPath := flags.String("trust-root", "", "pinned authority trust root")
	credentialPath := flags.String("credential", "", "authority data credential")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 ||
		*authorityPath == "" || *trustRootPath == "" || *credentialPath == "" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	verified, _, credential, err := localauthority.LoadFiles(
		ctx,
		*authorityPath,
		*trustRootPath,
		*credentialPath,
		time.Now().UTC(),
	)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: local authority inspection rejected",
		)
		return exitConfiguration
	}
	defer clear(credential)
	document := verified.Document()
	result := authorityInspection{
		SchemaVersion:        "buildopt.server/authority-inspection/v1",
		AuthorityDigest:      document.AuthorityDigest,
		Repository:           document.Repository,
		PolicyID:             document.Policy.PolicyID,
		PolicyVersion:        document.Policy.PolicyVersion,
		RevocationEpoch:      document.Revocation.RevocationEpoch,
		L1SecurityGeneration: document.Revocation.L1SecurityGeneration,
		Namespace:            document.Policy.RemoteCache.Namespace,
		NamespaceGeneration:  document.Policy.RemoteCache.NamespaceGeneration,
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: encode authority inspection failed",
		)
		return 1
	}
	return 0
}
