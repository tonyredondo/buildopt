package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestBuildoptEdgeUsageAndConfiguration(t *testing.T) {
	for _, test := range []struct {
		args []string
		want int
	}{
		{args: nil, want: edgeExitUsage},
		{args: []string{"unknown"}, want: edgeExitUsage},
		{args: []string{"serve"}, want: edgeExitUsage},
		{args: []string{"status", "--config", "/missing/edge.json"}, want: edgeExitConfiguration},
	} {
		var stdout, stderr bytes.Buffer
		if actual := run(context.Background(), test.args, &stdout, &stderr); actual != test.want {
			t.Fatalf("run(%q) = %d, want %d", test.args, actual, test.want)
		}
	}
	var stdout, stderr bytes.Buffer
	if actual := run(context.Background(), []string{"--help"}, &stdout, &stderr); actual != 0 {
		t.Fatalf("help exit = %d", actual)
	}
	if stdout.String() != edgeUsage || stderr.Len() != 0 ||
		!strings.Contains(stdout.String(), "buildopt-edge status") {
		t.Fatalf("unexpected help output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
