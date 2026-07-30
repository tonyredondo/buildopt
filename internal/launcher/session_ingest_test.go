package launcher

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const launcherIngestTestToken = "launcher-ingest-test-token-0123456789abcdef"

func TestLauncherDeliversSessionWithoutExposingServerCredential(t *testing.T) {
	clearManagedGatewayTestEnvironment(t)
	testCases := []struct {
		name        string
		childExit   int
		wantOutcome string
	}{
		{
			name:        "success",
			childExit:   0,
			wantOutcome: sessioningest.OutcomeSuccess,
		},
		{
			name:        "build failure",
			childExit:   37,
			wantOutcome: sessioningest.OutcomeBuildFailure,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := sessioningest.NewStore()
			handler, err := sessioningest.NewHandler(
				launcherIngestTestToken,
				store,
				nil,
			)
			if err != nil {
				t.Fatalf("create session ingest handler: %v", err)
			}
			server := httptest.NewServer(handler)
			defer server.Close()

			t.Setenv(serverURLEnvironment, server.URL)
			t.Setenv(serverTokenEnvironment, launcherIngestTestToken)
			script := `
test -z "${BUILDOPT_SERVER_URL+x}" || exit 97
test -z "${BUILDOPT_SERVER_INGEST_TOKEN+x}" || exit 98
exit "$1"
`
			var stderr bytes.Buffer
			exitCode := Run(
				[]string{
					"run",
					"--",
					"/bin/sh",
					"-c",
					script,
					"buildopt-session-test",
					strconv.Itoa(testCase.childExit),
				},
				strings.NewReader(""),
				&bytes.Buffer{},
				&stderr,
			)
			if exitCode != testCase.childExit {
				t.Fatalf(
					"launcher exit = %d, want %d; stderr=%q",
					exitCode,
					testCase.childExit,
					stderr.String(),
				)
			}

			sessions := store.Snapshot()
			if len(sessions) != 1 {
				t.Fatalf("accepted sessions = %d, want 1", len(sessions))
			}
			record := sessions[0]
			if record.Outcome != testCase.wantOutcome ||
				record.ExitCode != testCase.childExit ||
				record.GatewayConnectionGeneration == "" ||
				record.SessionID == "" {
				t.Fatalf("unexpected session record: %+v", record)
			}
			if _, err := time.Parse(time.RFC3339Nano, record.StartedAt); err != nil {
				t.Errorf("invalid startedAt: %v", err)
			}
			if _, err := time.Parse(time.RFC3339Nano, record.CompletedAt); err != nil {
				t.Errorf("invalid completedAt: %v", err)
			}
			if !strings.Contains(
				stderr.String(),
				"buildopt-server accepted session "+record.SessionID,
			) {
				t.Fatalf("missing ingest acknowledgement: %q", stderr.String())
			}
			if strings.Contains(stderr.String(), launcherIngestTestToken) {
				t.Fatal("launcher diagnostic exposed the server ingest token")
			}
		})
	}
}

func TestLauncherPreservesChildExitWhenSessionIngestFails(t *testing.T) {
	clearManagedGatewayTestEnvironment(t)
	t.Setenv(serverURLEnvironment, "http://127.0.0.1:1")
	t.Setenv(serverTokenEnvironment, launcherIngestTestToken)

	var stderr bytes.Buffer
	exitCode := Run(
		[]string{"run", "--", "/bin/sh", "-c", "exit 23"},
		strings.NewReader(""),
		&bytes.Buffer{},
		&stderr,
	)
	if exitCode != 23 {
		t.Fatalf("launcher exit = %d, want 23", exitCode)
	}
	if !strings.Contains(
		stderr.String(),
		"buildopt-server session ingest unavailable",
	) {
		t.Fatalf("missing fail-open diagnostic: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), launcherIngestTestToken) {
		t.Fatal("fail-open diagnostic exposed the server ingest token")
	}
}

func TestGatewayRejectsSessionFromAnotherGeneration(t *testing.T) {
	gateway, err := startLocalGateway()
	if err != nil {
		t.Fatalf("start local gateway: %v", err)
	}
	defer func() {
		if err := gateway.close(); err != nil {
			t.Errorf("close local gateway: %v", err)
		}
	}()

	client, err := sessioningest.NewClient(
		"http://127.0.0.1:1",
		launcherIngestTestToken,
	)
	if err != nil {
		t.Fatalf("create ingest client: %v", err)
	}
	startedAt := time.Now()
	record := sessioningest.NewRecord(
		"wrong-generation-session",
		"another-generation",
		startedAt,
		startedAt.Add(time.Second),
		sessioningest.OutcomeSuccess,
		0,
	)
	if _, err := gateway.deliverSession(
		context.Background(),
		client,
		record,
	); err == nil || !strings.Contains(err.Error(), "active gateway generation") {
		t.Fatalf("generation mismatch error = %v", err)
	}
}
