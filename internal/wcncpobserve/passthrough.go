package wcncpobserve

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"
)

// PassthroughResult is the authoritative child outcome with preserved stdio
// behavior. The caller owns stdio streams; this helper never buffers,
// reorders, or alters child bytes.
type PassthroughResult struct {
	Child    ChildResult
	Duration time.Duration
}

// RunNativePassthrough executes the repository native Gradle Wrapper child
// with the original ordered arguments. It performs no pre-child network,
// starts no recorder that could alter task selection, and returns the child
// exit code or signal unchanged. Observation or upload failure cannot replace
// it; callers must prefer this result over any BuildOpt error.
func RunNativePassthrough(ctx context.Context, wrapperPath string, args []string, cwd string, stdin io.Reader, stdout, stderr io.Writer) PassthroughResult {
	start := time.Now()
	command := exec.CommandContext(ctx, wrapperPath, args...)
	command.Dir = cwd
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	// Inherit only the process environment minus BuildOpt internals? No:
	// native behavior requires the exact environment. Bypass and observation
	// toggles are read by the parent, never stripped from the child.
	err := command.Run()
	duration := time.Since(start)
	result := PassthroughResult{Duration: duration}
	if err == nil {
		code := 0
		result.Child = ChildResult{Outcome: "SUCCESS", ExitCode: &code}
		return result
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		code := exitError.ExitCode()
		if code >= 0 {
			result.Child = ChildResult{Outcome: "FAILED", ExitCode: &code}
			return result
		}
		if signal, ok := exitErrorSignal(exitError); ok {
			result.Child = ChildResult{Outcome: "SIGNALED", Signal: &signal}
			return result
		}
	}
	// Signal or cancellation: classify without inventing an exit code.
	if ctx.Err() != nil {
		result.Child = ChildResult{Outcome: "SIGNALED"}
		return result
	}
	code := 1
	result.Child = ChildResult{Outcome: "FAILED", ExitCode: &code}
	return result
}

// BypassRequested reports whether BUILDOPT_BYPASS=1 requests a direct native
// run without observation. Bypass still preserves native behavior exactly;
// it only skips recording.
func BypassRequested(environment map[string]string) bool {
	if environment == nil {
		return os.Getenv("BUILDOPT_BYPASS") == "1"
	}
	return environment["BUILDOPT_BYPASS"] == "1"
}

// CaptureForTest runs a child and captures stdout/stderr for equivalence
// tests. Production uses RunNativePassthrough with live streams.
func CaptureForTest(ctx context.Context, wrapperPath string, args []string, cwd string, stdin string) (stdout, stderr string, result PassthroughResult) {
	var stdoutBuffer, stderrBuffer bytes.Buffer
	result = RunNativePassthrough(ctx, wrapperPath, args, cwd, bytes.NewBufferString(stdin), &stdoutBuffer, &stderrBuffer)
	return stdoutBuffer.String(), stderrBuffer.String(), result
}
