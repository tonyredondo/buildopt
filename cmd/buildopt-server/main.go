package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildsession"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	exitUsage         = 64
	exitConfiguration = 78
	serverUsage       = "usage: buildopt-server serve [--listen 127.0.0.1:8042] [--export-dir PATH] [--state-dir ABSOLUTE_PATH] [--cache-authority ABSOLUTE_PATH --cache-trust-root ABSOLUTE_PATH --cache-credential ABSOLUTE_PATH]\n       buildopt-server export --export-dir PATH [--format jsonl]\n"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdout io.Writer,
	stderr io.Writer,
) (exitCode int) {
	if len(args) == 1 && isHelp(args[0]) {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if len(args) == 2 &&
		(args[0] == "serve" || args[0] == "export") &&
		isHelp(args[1]) {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	if args[0] == "export" {
		return runExport(args[1:], stdout, stderr)
	}
	if args[0] != "serve" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}

	flags := flag.NewFlagSet("buildopt-server serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	listenAddress := flags.String(
		"listen",
		"127.0.0.1:8042",
		"loopback address for the WS-005 ingest server",
	)
	exportDirectory := flags.String(
		"export-dir",
		"",
		"local directory for atomic BUILD_SESSION v1 JSON exports",
	)
	stateDirectory := flags.String(
		"state-dir",
		"",
		"absolute private directory for single-node Shared storage",
	)
	cacheAuthorityPath := flags.String(
		"cache-authority",
		"",
		"private canonical local cache authority document",
	)
	cacheTrustRootPath := flags.String(
		"cache-trust-root",
		"",
		"private pinned local cache trust-root document",
	)
	cacheCredentialPath := flags.String(
		"cache-credential",
		"",
		"private local cache data-plane credential",
	)
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	if err := validateListenAddress(*listenAddress); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-server: invalid listen address: %v\n", err)
		return exitConfiguration
	}

	var exporter *buildsession.Exporter
	if *exportDirectory != "" {
		configuredExporter, exportErr := buildsession.NewExporter(
			*exportDirectory,
		)
		if exportErr != nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: invalid BUILD_SESSION export configuration",
			)
			return exitConfiguration
		}
		exporter = configuredExporter
	}

	var sharedStorage *sharedcache.Storage
	if *stateDirectory != "" {
		if !filepath.IsAbs(*stateDirectory) {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: invalid Shared storage configuration: state directory must be absolute",
			)
			return exitConfiguration
		}
		configuredStorage, storageErr := sharedcache.Open(
			ctx,
			filepath.Clean(*stateDirectory),
		)
		if storageErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: invalid Shared storage configuration: %v\n",
				storageErr,
			)
			return exitConfiguration
		}
		sharedStorage = configuredStorage
		defer func() {
			if closeErr := sharedStorage.Close(); closeErr != nil {
				_, _ = fmt.Fprintf(
					stderr,
					"buildopt-server: Shared storage shutdown incomplete: %v\n",
					closeErr,
				)
				if exitCode == 0 {
					exitCode = 1
				}
			}
		}()
	}

	cacheConfigurationValues := []string{
		*cacheAuthorityPath,
		*cacheTrustRootPath,
		*cacheCredentialPath,
	}
	cacheConfigured := false
	cacheComplete := true
	for _, value := range cacheConfigurationValues {
		cacheConfigured = cacheConfigured || value != ""
		cacheComplete = cacheComplete && value != ""
	}
	if cacheConfigured && (!cacheComplete || sharedStorage == nil) {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: authenticated Shared cache requires state directory, authority, trust root, and credential",
		)
		return exitConfiguration
	}

	var cacheHandler http.Handler
	if cacheConfigured {
		now := time.Now().UTC()
		verified, _, credential, authorityErr := localauthority.LoadFiles(
			ctx,
			*cacheAuthorityPath,
			*cacheTrustRootPath,
			*cacheCredentialPath,
			now,
		)
		if authorityErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: invalid local cache authority: %v\n",
				authorityErr,
			)
			return exitConfiguration
		}
		binding, _, authorityErr := sharedStorage.InstallLocalAuthority(
			ctx,
			verified,
			credential,
			now,
		)
		if authorityErr == nil {
			cacheHandler, authorityErr =
				sharedcache.NewLocalAuthorityHTTPHandler(
					sharedStorage,
					binding,
					credential,
				)
		}
		for index := range credential {
			credential[index] = 0
		}
		if authorityErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: local cache authority unavailable: %v\n",
				authorityErr,
			)
			return exitConfiguration
		}
	}

	ingestStore := sessioningest.NewStore()
	logger := log.New(stdout, "buildopt-server: ", 0)
	ingestHandler, err := sessioningest.NewHandler(
		getenv(sessioningest.ServerTokenEnvironment),
		ingestStore,
		func(record sessioningest.Record, result sessioningest.PutResult) {
			action := "accepted"
			if result == sessioningest.PutDuplicate {
				action = "deduplicated"
			}
			logger.Printf(
				"%s session %s outcome=%s exit=%d",
				action,
				record.SessionID,
				record.Outcome,
				record.ExitCode,
			)
			if exporter == nil {
				return
			}
			path, created, err := exporter.Export(record)
			if err != nil {
				logger.Printf(
					"BUILD_SESSION export unavailable for session %s: %v",
					record.SessionID,
					err,
				)
				return
			}
			exportAction := "retained"
			if created {
				exportAction = "published"
			}
			logger.Printf(
				"%s BUILD_SESSION %s as %s",
				exportAction,
				record.SessionID,
				filepath.Base(path),
			)
		},
	)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid session ingest configuration",
		)
		return exitConfiguration
	}
	handler := http.NewServeMux()
	if cacheHandler != nil {
		handler.Handle("/cache/", cacheHandler)
	}
	handler.Handle("/", ingestHandler)

	listener, err := net.Listen("tcp4", *listenAddress)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-server: cannot listen: %v\n", err)
		return 1
	}
	defer listener.Close()

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       15 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          log.New(stderr, "buildopt-server: ", 0),
	}
	serveDone := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveDone <- err
	}()
	logger.Printf("listening on http://%s", listener.Addr())
	if sharedStorage != nil {
		if cacheHandler == nil {
			logger.Printf(
				"single-node Shared storage initialized and reconciled with cache/control schema %d; cache routing disabled without local authority",
				sharedcache.SchemaVersion,
			)
		} else {
			logger.Printf(
				"single-node Shared storage initialized and reconciled with cache/control schema %d; authenticated cache routing enabled",
				sharedcache.SchemaVersion,
			)
		}
	}

	select {
	case err := <-serveDone:
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt-server: serve failed: %v\n", err)
			return 1
		}
		return 0
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			_ = server.Close()
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: shutdown incomplete: %v\n",
				err,
			)
			return 1
		}
		if err := <-serveDone; err != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt-server: serve failed: %v\n", err)
			return 1
		}
		return 0
	}
}

func runExport(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("buildopt-server export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	exportDirectory := flags.String(
		"export-dir",
		"",
		"existing private BUILD_SESSION export directory",
	)
	format := flags.String(
		"format",
		"jsonl",
		"stdout export format",
	)
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	if *exportDirectory == "" || *format != "jsonl" {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: export requires an existing directory and format jsonl",
		)
		return exitConfiguration
	}
	if _, err := os.Lstat(*exportDirectory); err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: BUILD_SESSION export directory is unavailable",
		)
		return exitConfiguration
	}
	exporter, err := buildsession.NewExporter(*exportDirectory)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid BUILD_SESSION export configuration",
		)
		return exitConfiguration
	}
	if err := exporter.WriteJSONL(stdout); err != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt-server: JSONL export unavailable: %v\n",
			err,
		)
		return exitConfiguration
	}
	return 0
}

func validateListenAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return errors.New("WS-005 requires canonical IPv4 loopback")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	return nil
}

func isHelp(argument string) bool {
	return argument == "help" || argument == "--help" || argument == "-h"
}
