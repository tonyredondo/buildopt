package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/edgecache"
)

const (
	edgeExitUsage         = 64
	edgeExitConfiguration = 78
	edgeUsage             = "usage: buildopt-edge serve --config ABSOLUTE_PATH\n       buildopt-edge status --config ABSOLUTE_PATH\n       buildopt-edge validate --config ABSOLUTE_PATH\n"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 1 && isEdgeHelp(args[0]) {
		_, _ = io.WriteString(stdout, edgeUsage)
		return 0
	}
	if len(args) == 2 &&
		(args[0] == "serve" || args[0] == "status" || args[0] == "validate") &&
		isEdgeHelp(args[1]) {
		_, _ = io.WriteString(stdout, edgeUsage)
		return 0
	}
	if len(args) == 0 ||
		(args[0] != "serve" && args[0] != "status" && args[0] != "validate") {
		_, _ = io.WriteString(stderr, edgeUsage)
		return edgeExitUsage
	}
	flags := flag.NewFlagSet("buildopt-edge "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "private Edge configuration")
	if err := flags.Parse(args[1:]); err != nil ||
		flags.NArg() != 0 || *configPath == "" {
		_, _ = io.WriteString(stderr, edgeUsage)
		return edgeExitUsage
	}
	config, err := edgecache.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: invalid configuration: %v\n", err)
		return edgeExitConfiguration
	}
	if args[0] == "status" {
		return runStatus(config, stdout, stderr)
	}
	if args[0] == "validate" {
		_, _ = io.WriteString(stdout, "buildopt-edge: configuration valid\n")
		return 0
	}
	return runServe(ctx, config, stdout, stderr)
}

func runStatus(
	config edgecache.Config,
	stdout io.Writer,
	stderr io.Writer,
) int {
	status, err := edgecache.LoadRuntimeStatus(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: status unavailable: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(status); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: write status: %v\n", err)
		return 1
	}
	if !edgecache.StatusReady(status, time.Now().UTC()) {
		return 1
	}
	return 0
}

func runServe(
	ctx context.Context,
	config edgecache.Config,
	stdout io.Writer,
	stderr io.Writer,
) (exitCode int) {
	runtime, err := edgecache.OpenRuntime(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: preflight failed: %v\n", err)
		return edgeExitConfiguration
	}
	defer func() {
		if closeErr := runtime.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt-edge: shutdown incomplete: %v\n", closeErr)
			if exitCode == 0 {
				exitCode = 1
			}
		}
	}()
	listener, err := net.Listen("tcp4", config.Server.Listen)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: cannot listen: %v\n", err)
		return 1
	}
	defer listener.Close()
	_, _ = fmt.Fprintf(stdout, "buildopt-edge: listening on http://%s\n", listener.Addr())
	if err := runtime.Serve(ctx, listener); err != nil &&
		!errors.Is(err, context.Canceled) {
		_, _ = fmt.Fprintf(stderr, "buildopt-edge: serve failed: %v\n", err)
		return 1
	}
	return 0
}

func isEdgeHelp(argument string) bool {
	return argument == "help" || argument == "--help" || argument == "-h"
}
