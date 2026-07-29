package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const serverTestToken = "server-test-ingest-token-0123456789abcdef"

func TestBuildoptServerUsageAndConfiguration(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		token      string
		wantExit   int
		wantOutput string
	}{
		{
			name:       "help",
			args:       []string{"--help"},
			wantExit:   0,
			wantOutput: serverUsage,
		},
		{
			name:       "missing command",
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "unknown command",
			args:       []string{"unknown"},
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "unknown argument",
			args:       []string{"serve", "extra"},
			wantExit:   exitUsage,
			wantOutput: serverUsage,
		},
		{
			name:       "non-loopback listener",
			args:       []string{"serve", "--listen", "0.0.0.0:8042"},
			token:      serverTestToken,
			wantExit:   exitConfiguration,
			wantOutput: "invalid listen address",
		},
		{
			name:       "missing token",
			args:       []string{"serve", "--listen", "127.0.0.1:0"},
			wantExit:   exitConfiguration,
			wantOutput: "invalid session ingest configuration",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return testCase.token
				}
				return ""
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(
				context.Background(),
				testCase.args,
				getenv,
				&stdout,
				&stderr,
			)
			if exitCode != testCase.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, testCase.wantExit)
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, testCase.wantOutput) {
				t.Fatalf(
					"output = %q, want %q",
					output,
					testCase.wantOutput,
				)
			}
		})
	}
}

func TestBuildoptServerReceivesAndStopsGracefully(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := newNotifyingWriter()
	var stderr bytes.Buffer
	exited := make(chan int, 1)
	go func() {
		exited <- run(
			ctx,
			[]string{"serve", "--listen", "127.0.0.1:0"},
			func(key string) string {
				if key == sessioningest.ServerTokenEnvironment {
					return serverTestToken
				}
				return ""
			},
			output,
			&stderr,
		)
	}()

	select {
	case <-output.written:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not report readiness")
	}
	line := output.String()
	const prefix = "buildopt-server: listening on "
	start := strings.Index(line, prefix)
	if start < 0 {
		t.Fatalf("missing listen line: %q", line)
	}
	endpoint := strings.TrimSpace(line[start+len(prefix):])

	client, err := sessioningest.NewClient(endpoint, serverTestToken)
	if err != nil {
		t.Fatalf("create ingest client: %v", err)
	}
	startedAt := time.Now()
	record := sessioningest.NewRecord(
		"server-main-test",
		"gateway-main-test",
		startedAt,
		startedAt.Add(time.Second),
		sessioningest.OutcomeSuccess,
		0,
	)
	if result, err := client.Deliver(context.Background(), record); err != nil ||
		result != sessioningest.PutCreated {
		t.Fatalf("deliver session = %d/%v", result, err)
	}

	cancel()
	select {
	case exitCode := <-exited:
		if exitCode != 0 {
			t.Fatalf(
				"server exit = %d, stderr = %q",
				exitCode,
				stderr.String(),
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after cancellation")
	}
	if !strings.Contains(
		output.String(),
		"accepted session server-main-test outcome=SUCCESS exit=0",
	) {
		t.Fatalf("missing acceptance log: %q", output.String())
	}
}

type notifyingWriter struct {
	mutex   sync.Mutex
	buffer  bytes.Buffer
	written chan struct{}
	once    sync.Once
}

func newNotifyingWriter() *notifyingWriter {
	return &notifyingWriter{written: make(chan struct{})}
}

func (writer *notifyingWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	count, err := writer.buffer.Write(data)
	writer.once.Do(func() {
		close(writer.written)
	})
	return count, err
}

func (writer *notifyingWriter) String() string {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.buffer.String()
}
