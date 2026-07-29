package main

import (
	"bytes"
	"testing"
)

func TestUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := run(nil, &stdout, &stderr); exitCode != 64 {
		t.Fatalf("usage exit = %d, want 64", exitCode)
	}
	if stdout.Len() != 0 || stderr.String() != usage {
		t.Fatalf(
			"usage output = %q/%q",
			stdout.String(),
			stderr.String(),
		)
	}
}
