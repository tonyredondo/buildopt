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
	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	exitUsage         = 64
	exitConfiguration = 78
	serverUsage       = "usage: buildopt-server serve [--listen 127.0.0.1:8042] [--export-dir PATH]\n"
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
) int {
	if len(args) == 1 && isHelp(args[0]) {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if len(args) == 2 && args[0] == "serve" && isHelp(args[1]) {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if len(args) == 0 || args[0] != "serve" {
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

	store := sessioningest.NewStore()
	logger := log.New(stdout, "buildopt-server: ", 0)
	handler, err := sessioningest.NewHandler(
		getenv(sessioningest.ServerTokenEnvironment),
		store,
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
