package stickywrapper

import (
	"bytes"
	"strings"
	"testing"
)

func TestBootstrapTemplatesStayWithinPortableFileContract(t *testing.T) {
	for name, raw := range map[string][]byte{posixPath: posixWrapper, windowsPath: windowsWrapper} {
		if len(raw) == 0 || int64(len(raw)) > maximumBytes[name] {
			t.Fatalf("%s size = %d", name, len(raw))
		}
		if raw[len(raw)-1] != '\n' || bytes.ContainsRune(raw, '\r') || bytes.HasPrefix(raw, []byte("\xef\xbb\xbf")) {
			t.Fatalf("%s has non-portable text encoding", name)
		}
	}
	if !bytes.HasPrefix(posixWrapper, []byte("#!/bin/sh\nset -eu\n")) ||
		!bytes.HasPrefix(windowsWrapper, []byte("@echo off\n")) {
		t.Fatal("bootstrap template entrypoint changed")
	}
	for _, required := range []string{
		"distribution checksum mismatch",
		"distribution archive contains a link or unsafe entry",
		"cached distribution failed verification",
		"verified bootstrap unavailable; running Gradle directly",
		"run --",
	} {
		if !strings.Contains(string(posixWrapper), required) {
			t.Fatalf("POSIX template is missing %q", required)
		}
	}
	for _, required := range []string{
		"AllowAutoRedirect = $false",
		"[Environment]::GetCommandLineArgs()",
		"$seen -and $item -ceq '--buildopt-wrapper-arguments'",
		"[Array]::IndexOf($WrapperArgs, '--buildopt-wrapper-arguments')",
		"BUILDOPT_WRAPPER_MANAGEMENT=version-json",
		"$env:BUILDOPT_WRAPPER_MANAGEMENT -ceq 'version-json'",
		"Security.Cryptography.SHA256",
		"distribution archive contains an unsafe entry",
		"verified bootstrap unavailable; running Gradle directly",
		"run --",
	} {
		if !strings.Contains(string(windowsWrapper), required) {
			t.Fatalf("Windows template is missing %q", required)
		}
	}
}
