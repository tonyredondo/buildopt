package main

import (
	"bytes"
	"context"
	"crypto/tls"
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
	"strings"
	"syscall"
	"time"

	"github.com/tonyredondo/buildopt/internal/buildhistory"
	"github.com/tonyredondo/buildopt/internal/buildsession"
	"github.com/tonyredondo/buildopt/internal/datalifecycle"
	"github.com/tonyredondo/buildopt/internal/githubqueue"
	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/selfhosted"
	"github.com/tonyredondo/buildopt/internal/sessioningest"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	exitUsage         = 64
	exitConfiguration = 78
	serverUsage       = "usage: buildopt-server serve [--self-hosted-config ABSOLUTE_PATH] [--listen 127.0.0.1:8042] [--tls-cert ABSOLUTE_PATH --tls-key ABSOLUTE_PATH --central-auth] [--export-dir PATH] [--export-profile summary|tasks|evidence|diagnostic --authorize-expanded-export [--diagnostic-until RFC3339]] [--state-dir ABSOLUTE_PATH] [--cache-authority ABSOLUTE_PATH --cache-trust-root ABSOLUTE_PATH --cache-credential ABSOLUTE_PATH] [--cache-token-auth] [--github-webhook-secret ABSOLUTE_PATH]\n       buildopt-server export --export-dir PATH [--format jsonl] [--export-profile summary|tasks|evidence|diagnostic --authorize-expanded-export [--diagnostic-until RFC3339]]\n       buildopt-server data delete --data-root ABSOLUTE_PATH --deletion-id ID --tenant ID --repository ID --trust-domain ID --next-namespace-generation N --next-l1-security-generation N --token-key ABSOLUTE_PATH --token-key-version ID [--external-destination ID]\n       buildopt-server token issue --state-dir ABSOLUTE_PATH --tenant ID --repository ID --trust-domain ID --namespace ID --namespace-generation N --plane stable|quarantine|control --access read|read-write --expires-at RFC3339\n       buildopt-server token revoke --state-dir ABSOLUTE_PATH --token-id ID\n       buildopt-server central-token issue --state-dir ABSOLUTE_PATH --repository-scope-sha256 SHA256 --tenant ID --repository ID --trust-domain ID --namespace ID --namespace-generation N --capabilities LIST --expires-at RFC3339\n       buildopt-server central-token revoke --state-dir ABSOLUTE_PATH --token-id ID\n       buildopt-server authority inspect --authority ABSOLUTE_PATH --trust-root ABSOLUTE_PATH --credential ABSOLUTE_PATH\n"
)

var (
	openSharedStorage     = sharedcache.Open
	openSelfHostedStorage = sharedcache.OpenSelfHosted
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
	if args[0] == "token" {
		return runBetaToken(ctx, args[1:], stdout, stderr)
	}
	if args[0] == "central-token" {
		return runCentralToken(ctx, args[1:], stdout, stderr)
	}
	if args[0] == "data" {
		return runDataLifecycle(ctx, args[1:], stdout, stderr)
	}
	if args[0] == "authority" {
		return runAuthorityInspect(ctx, args[1:], stdout, stderr)
	}
	if args[0] != "serve" {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}

	flags := flag.NewFlagSet("buildopt-server serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	selfHostedConfigPath := flags.String(
		"self-hosted-config",
		"",
		"private declarative isolated single-node configuration",
	)
	listenAddress := flags.String(
		"listen",
		"127.0.0.1:8042",
		"loopback address for the WS-005 ingest server",
	)
	tlsCertificatePath := flags.String(
		"tls-cert",
		"",
		"PEM certificate chain for the external HTTPS listener",
	)
	tlsKeyPath := flags.String(
		"tls-key",
		"",
		"private PEM key for the external HTTPS listener",
	)
	centralAuthentication := flags.Bool(
		"central-auth",
		false,
		"enable scoped central cache/state routes over HTTPS",
	)
	exportDirectory := flags.String(
		"export-dir",
		"",
		"local directory for atomic BUILD_SESSION v1 JSON exports",
	)
	exportProfile := flags.String(
		"export-profile",
		"summary",
		"maximum authorized export profile",
	)
	authorizeExpandedExport := flags.Bool(
		"authorize-expanded-export",
		false,
		"explicitly authorize a non-summary export profile",
	)
	diagnosticUntil := flags.String(
		"diagnostic-until",
		"",
		"UTC RFC3339 expiry for diagnostic opt-in",
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
	cacheTokenAuth := flags.Bool(
		"cache-token-auth",
		false,
		"authenticate remote cache requests with scoped beta tokens",
	)
	githubWebhookSecretPath := flags.String(
		"github-webhook-secret",
		"",
		"private GitHub workflow_job webhook secret",
	)
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		_, _ = io.WriteString(stderr, serverUsage)
		return exitUsage
	}
	selfHostedMode := *selfHostedConfigPath != ""
	if selfHostedMode {
		visited := 0
		flags.Visit(func(*flag.Flag) { visited++ })
		if visited != 1 {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: self-hosted configuration cannot be combined with serve flags")
			return exitConfiguration
		}
		deployment, configErr := selfhosted.Load(*selfHostedConfigPath)
		if configErr != nil {
			_, _ = fmt.Fprintf(stderr, "buildopt-server: invalid self-hosted configuration: %v\n", configErr)
			return exitConfiguration
		}
		*listenAddress = deployment.Server.Listen
		*exportDirectory = deployment.Export.Directory
		*exportProfile = deployment.Export.Profile
		*stateDirectory = deployment.Storage.StateDirectory
		*cacheAuthorityPath = deployment.Cache.AuthorityPath
		*cacheTrustRootPath = deployment.Cache.TrustRootPath
		*cacheCredentialPath = deployment.Cache.CredentialPath
		*cacheTokenAuth = deployment.Cache.BetaTokenAuthentication
		if deployment.GitHubQueue != nil {
			*githubWebhookSecretPath = deployment.GitHubQueue.WebhookSecretPath
		}
	}
	tlsConfigured := *tlsCertificatePath != "" || *tlsKeyPath != ""
	if (*tlsCertificatePath == "") != (*tlsKeyPath == "") {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: TLS requires both certificate and private key")
		return exitConfiguration
	}
	if err := validateListenAddress(*listenAddress, tlsConfigured); err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-server: invalid listen address: %v\n", err)
		return exitConfiguration
	}
	var tlsConfiguration *tls.Config
	if tlsConfigured {
		configuredTLS, tlsErr := loadServerTLS(*tlsCertificatePath, *tlsKeyPath)
		if tlsErr != nil {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid TLS configuration")
			return exitConfiguration
		}
		tlsConfiguration = configuredTLS
	}

	var exporter *buildsession.Exporter
	if *exportDirectory == "" &&
		(*exportProfile != "summary" ||
			*authorizeExpandedExport ||
			*diagnosticUntil != "") {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: export policy requires an export directory",
		)
		return exitConfiguration
	}
	if *exportDirectory != "" {
		exportPolicy, policyErr := parseExportPolicy(
			*exportProfile,
			*authorizeExpandedExport,
			*diagnosticUntil,
			time.Now().UTC(),
		)
		if policyErr != nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: invalid BUILD_SESSION export policy",
			)
			return exitConfiguration
		}
		configuredExporter, exportErr := buildsession.NewExporterWithPolicy(
			*exportDirectory,
			exportPolicy,
		)
		if exportErr != nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: invalid BUILD_SESSION export configuration",
			)
			return exitConfiguration
		}
		exporter = configuredExporter
		defer exporter.Close()
	}

	historyToken := getenv(buildhistory.TokenEnvironment)
	var historyHandler http.Handler
	if historyToken != "" {
		if exporter == nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: history API requires an export directory",
			)
			return exitConfiguration
		}
		configuredHistory, historyErr := buildhistory.NewHandler(
			historyToken,
			exporter.Directory(),
		)
		if historyErr != nil {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid history API configuration")
			return exitConfiguration
		}
		historyHandler = configuredHistory
	}

	if *stateDirectory != "" && !filepath.IsAbs(*stateDirectory) {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid Shared storage configuration: state directory must be absolute",
		)
		return exitConfiguration
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
	if cacheConfigured && (!cacheComplete || *stateDirectory == "") {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: authenticated Shared cache requires state directory, authority, trust root, and credential",
		)
		return exitConfiguration
	}
	if *cacheTokenAuth && !cacheConfigured {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: beta token authentication requires authenticated Shared cache configuration",
		)
		return exitConfiguration
	}
	if *centralAuthentication &&
		(!tlsConfigured || *stateDirectory == "" || cacheConfigured || selfHostedMode) {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: central authentication requires TLS, state directory, direct serve flags, and no local cache authority",
		)
		return exitConfiguration
	}
	if *githubWebhookSecretPath != "" && (*stateDirectory == "" || !filepath.IsAbs(*githubWebhookSecretPath)) {
		_, _ = fmt.Fprintln(stderr, "buildopt-server: GitHub queue adapter requires state directory and absolute webhook secret path")
		return exitConfiguration
	}
	var githubWebhookSecret []byte
	if *githubWebhookSecretPath != "" {
		secretBytes, secretErr := localauthority.ReadPrivateFile(*githubWebhookSecretPath, 4096)
		if secretErr != nil {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid GitHub webhook secret")
			return exitConfiguration
		}
		githubWebhookSecret = bytes.TrimSpace(secretBytes)
		if len(githubWebhookSecret) < 32 {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid GitHub webhook secret")
			return exitConfiguration
		}
	}

	ingestStore := sessioningest.NewStore()
	logger := log.New(stdout, "buildopt-server: ", 0)
	alerts := newOperationalAlertMonitor(time.Now)
	var ingestHandler http.Handler
	sessionToken := getenv(sessioningest.ServerTokenEnvironment)
	if sessionToken != "" || !*centralAuthentication {
		rawIngestHandler, ingestErr := sessioningest.NewHandler(
			sessionToken,
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
				alerts.exportStarted()
				path, created, exportErr := exporter.Export(record)
				alerts.exportFinished(exportErr)
				if exportErr != nil {
					logger.Printf(
						"BUILD_SESSION export unavailable for session %s: %v",
						record.SessionID,
						exportErr,
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
		if ingestErr != nil {
			_, _ = fmt.Fprintln(
				stderr,
				"buildopt-server: invalid session ingest configuration",
			)
			return exitConfiguration
		}
		ingestHandler = alerts.instrumentAcceptance(rawIngestHandler)
	}

	var sharedStorage *sharedcache.Storage
	openConfiguredStorage := func(openStorage func(context.Context, string) (*sharedcache.Storage, error)) error {
		configuredStorage, storageErr := openStorage(ctx, filepath.Clean(*stateDirectory))
		if storageErr != nil {
			return storageErr
		}
		sharedStorage = configuredStorage
		return nil
	}
	defer func() {
		if sharedStorage == nil {
			return
		}
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
	if selfHostedMode {
		if storageErr := openConfiguredStorage(openSelfHostedStorage); storageErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: invalid Shared storage configuration: %v\n",
				storageErr,
			)
			return exitConfiguration
		}
	}

	listener, err := net.Listen("tcp4", *listenAddress)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "buildopt-server: cannot listen: %v\n", err)
		return 1
	}
	defer listener.Close()

	operational := &operationalRouter{alerts: alerts}
	server := &http.Server{
		Handler:           operational,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       15 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          log.New(stderr, "buildopt-server: ", 0),
		TLSConfig:         tlsConfiguration,
	}
	defer server.Close()
	serveDone := make(chan error, 1)
	go func() {
		serveListener := net.Listener(listener)
		if tlsConfiguration != nil {
			serveListener = tls.NewListener(listener, tlsConfiguration)
		}
		serveErr := server.Serve(serveListener)
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		serveDone <- serveErr
	}()
	scheme := "http"
	if tlsConfiguration != nil {
		scheme = "https"
	}
	logger.Printf("listening on %s://%s", scheme, listener.Addr())

	if *stateDirectory != "" && sharedStorage == nil {
		if storageErr := openConfiguredStorage(openSharedStorage); storageErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: invalid Shared storage configuration: %v\n",
				storageErr,
			)
			return exitConfiguration
		}
	}

	var githubQueueHandler http.Handler
	if len(githubWebhookSecret) != 0 {
		configuredQueueHandler, queueErr := githubqueue.NewHandler(githubWebhookSecret, sharedStorage)
		if queueErr != nil {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid GitHub queue adapter configuration")
			return exitConfiguration
		}
		githubQueueHandler = configuredQueueHandler
	}

	var cacheHandler http.Handler
	var cacheSwitch *switchableHandler
	var cacheAuthority loadedAuthority
	var cachePaths authorityPaths
	if cacheConfigured {
		cachePaths = authorityPaths{
			document:   *cacheAuthorityPath,
			trustRoot:  *cacheTrustRootPath,
			credential: *cacheCredentialPath,
			betaTokens: *cacheTokenAuth,
		}
		loaded, authorityErr := loadServerAuthority(
			ctx,
			sharedStorage,
			cachePaths,
			time.Now().UTC(),
		)
		if authorityErr != nil {
			_, _ = fmt.Fprintf(
				stderr,
				"buildopt-server: invalid local cache authority: %v\n",
				authorityErr,
			)
			return exitConfiguration
		}
		cacheAuthority = loaded
		alerts.authorityLoaded(loaded.expiresAt)
		cacheSwitch = &switchableHandler{}
		cacheSwitch.set(loaded.handler)
		cacheHandler = cacheSwitch
	}
	var centralHandler http.Handler
	if *centralAuthentication {
		configuredCentral, centralErr := sharedcache.NewCentralHTTPSHandler(sharedStorage)
		if centralErr != nil {
			_, _ = fmt.Fprintln(stderr, "buildopt-server: invalid central HTTPS configuration")
			return exitConfiguration
		}
		centralHandler = configuredCentral
	}

	handler := http.NewServeMux()
	if githubQueueHandler != nil {
		handler.Handle(githubqueue.WebhookPath, githubQueueHandler)
	}
	if historyHandler != nil {
		handler.Handle(buildhistory.ListPath, historyHandler)
		handler.Handle(buildhistory.DetailPath, historyHandler)
		handler.Handle(buildhistory.DashboardPath, buildhistory.NewDashboardHandler())
	}
	if cacheHandler != nil {
		handler.Handle("/cache/", cacheHandler)
	}
	if centralHandler != nil {
		handler.Handle("/cache/", centralHandler)
		handler.Handle("/api/v1/repositories/", centralHandler)
	}
	if ingestHandler != nil {
		handler.Handle("/", ingestHandler)
	} else {
		handler.Handle("/", http.NotFoundHandler())
	}
	operational.activate(handler)
	reloadContext, cancelReload := context.WithCancel(ctx)
	defer cancelReload()
	if sharedStorage != nil {
		go watchStorageAlerts(
			reloadContext,
			sharedStorage,
			alerts,
			storageAlertInterval,
		)
	}
	if cacheSwitch != nil {
		go watchServerAuthority(
			reloadContext,
			sharedStorage,
			cachePaths,
			cacheAuthority,
			cacheSwitch,
			operational,
			handler,
			logger,
			alerts,
			authorityReloadInterval,
		)
	}
	if sharedStorage != nil {
		switch {
		case centralHandler != nil:
			logger.Printf(
				"single-node Shared storage initialized and reconciled with cache/control schema %d; central HTTPS cache/state routing enabled",
				sharedcache.SchemaVersion,
			)
		case cacheHandler == nil:
			logger.Printf(
				"single-node Shared storage initialized and reconciled with cache/control schema %d; cache routing disabled without local authority",
				sharedcache.SchemaVersion,
			)
		default:
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
		operational.deactivate()
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
	exportProfile := flags.String(
		"export-profile",
		"summary",
		"maximum authorized export profile",
	)
	authorizeExpandedExport := flags.Bool(
		"authorize-expanded-export",
		false,
		"explicitly authorize a non-summary export profile",
	)
	diagnosticUntil := flags.String(
		"diagnostic-until",
		"",
		"UTC RFC3339 expiry for diagnostic opt-in",
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
	exportPolicy, err := parseExportPolicy(
		*exportProfile,
		*authorizeExpandedExport,
		*diagnosticUntil,
		time.Now().UTC(),
	)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid BUILD_SESSION export policy",
		)
		return exitConfiguration
	}
	exporter, err := buildsession.NewExporterWithPolicy(
		*exportDirectory,
		exportPolicy,
	)
	if err != nil {
		_, _ = fmt.Fprintln(
			stderr,
			"buildopt-server: invalid BUILD_SESSION export configuration",
		)
		return exitConfiguration
	}
	defer exporter.Close()
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

func parseExportPolicy(
	profile string,
	authorized bool,
	diagnosticUntil string,
	now time.Time,
) (datalifecycle.ExportPolicy, error) {
	policy := datalifecycle.ExportPolicy{
		Profile:              datalifecycle.ExportProfile(strings.ToUpper(profile)),
		ExplicitlyAuthorized: authorized,
	}
	if diagnosticUntil != "" {
		until, err := time.Parse(time.RFC3339, diagnosticUntil)
		if err != nil || until.Location() != time.UTC {
			return datalifecycle.ExportPolicy{}, errors.New(
				"diagnostic expiry must be UTC RFC3339",
			)
		}
		policy.DiagnosticUntil = until
	}
	if err := datalifecycle.ValidateExportPolicy(policy, now); err != nil {
		return datalifecycle.ExportPolicy{}, err
	}
	return policy, nil
}

func validateListenAddress(address string, tlsConfigured bool) error {
	host, portText, err := net.SplitHostPort(address)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || ip.To4() == nil || ip.String() != host {
		return errors.New("listener requires a canonical IPv4 address")
	}
	if !tlsConfigured && host != "127.0.0.1" {
		return errors.New("plaintext listener requires canonical IPv4 loopback")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	return nil
}

func loadServerTLS(certificatePath, keyPath string) (*tls.Config, error) {
	if !filepath.IsAbs(certificatePath) || filepath.Clean(certificatePath) != certificatePath ||
		!filepath.IsAbs(keyPath) || filepath.Clean(keyPath) != keyPath {
		return nil, errors.New("TLS paths must be absolute and canonical")
	}
	certificateInfo, err := os.Lstat(certificatePath)
	if err != nil || !certificateInfo.Mode().IsRegular() || certificateInfo.Size() < 1 ||
		certificateInfo.Size() > 1<<20 {
		return nil, errors.New("TLS certificate must be a bounded regular file")
	}
	certificatePEM, err := os.ReadFile(certificatePath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := localauthority.ReadPrivateFile(keyPath, 1<<20)
	if err != nil {
		return nil, err
	}
	defer clear(keyPEM)
	pair, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{pair},
		NextProtos:   []string{"http/1.1"},
	}, nil
}

func isHelp(argument string) bool {
	return argument == "help" || argument == "--help" || argument == "-h"
}
