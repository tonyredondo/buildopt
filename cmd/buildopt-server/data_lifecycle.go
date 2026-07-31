package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

type repeatedStringFlag []string

func (values *repeatedStringFlag) String() string {
	if values == nil {
		return ""
	}
	content, _ := json.Marshal([]string(*values))
	return string(content)
}

func (values *repeatedStringFlag) Set(value string) error {
	if value == "" {
		return errors.New("empty repeated value")
	}
	*values = append(*values, value)
	return nil
}

func runDataLifecycle(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 || args[0] != "delete" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	flags := flag.NewFlagSet("buildopt-server data delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dataRoot := flags.String("data-root", "", "marked deployment data root")
	deletionID := flags.String("deletion-id", "", "idempotent deletion ID")
	tenant := flags.String("tenant", "", "tenant identity")
	repository := flags.String("repository", "", "repository identity")
	trustDomain := flags.String("trust-domain", "", "trust-domain identity")
	nextNamespaceGeneration := flags.Int64(
		"next-namespace-generation",
		0,
		"minimum namespace generation after deletion",
	)
	nextL1SecurityGeneration := flags.Uint64(
		"next-l1-security-generation",
		0,
		"minimum managed-L1 generation after deletion",
	)
	tokenKeyPath := flags.String(
		"token-key",
		"",
		"private 32-byte HMAC tokenization key",
	)
	tokenKeyVersion := flags.String(
		"token-key-version",
		"",
		"public tokenization key version",
	)
	var externalDestinations repeatedStringFlag
	flags.Var(
		&externalDestinations,
		"external-destination",
		"customer-controlled destination requiring the tombstone",
	)
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	key, err := loadLifecycleTokenKey(*tokenKeyPath)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid managed-data deletion key",
		)
		return exitConfiguration
	}
	defer clear(key)
	report, err := datalifecycle.DeleteManagedData(
		ctx,
		datalifecycle.DeletionRequest{
			DataRoot:                 *dataRoot,
			DeletionID:               *deletionID,
			Tenant:                   *tenant,
			Repository:               *repository,
			TrustDomain:              *trustDomain,
			NextNamespaceGeneration:  *nextNamespaceGeneration,
			NextL1SecurityGeneration: *nextL1SecurityGeneration,
			TokenKey:                 key,
			TokenKeyVersion:          *tokenKeyVersion,
			ExternalDestinations:     []string(externalDestinations),
			RequestedAt:              time.Now().UTC(),
		},
	)
	if err != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt-server: managed-data deletion failed: %v\n",
			err,
		)
		return exitConfiguration
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: encode managed-data deletion report",
		)
		return 1
	}
	return 0
}

func loadLifecycleTokenKey(path string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("token key path must be absolute and canonical")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok ||
		stat.Uid != uint32(os.Geteuid()) ||
		!info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 ||
		info.Size() != datalifecycle.RedactionKeyBytes {
		return nil, errors.New("token key is not a private fixed-size file")
	}
	key := make([]byte, datalifecycle.RedactionKeyBytes)
	if _, err := io.ReadFull(file, key); err != nil {
		return nil, err
	}
	return key, nil
}
