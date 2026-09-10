package main

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type nativeTimeWindow struct{ StartMs, EndMs int64 }

var nativeDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$`)
var manifestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,69}$`)

// Project only the two accepted native clock values. Preserve attribute order,
// spelling, continuation bytes, line endings and every other manifest value.
func manifestProjection(raw []byte, window nativeTimeWindow) ([]byte, error) {
	if len(raw) > 1<<20 || !utf8.Valid(raw) || !bytes.HasSuffix(raw, []byte("\r\n\r\n")) {
		return nil, errors.New("invalid or oversized native manifest")
	}
	if window.StartMs <= 0 || window.EndMs < window.StartMs || window.EndMs-window.StartMs > 3600000 {
		return nil, errors.New("invalid native invocation window")
	}
	if bytes.ContainsAny(bytes.ReplaceAll(raw, []byte("\r\n"), nil), "\r\n") {
		return nil, errors.New("manifest requires CRLF")
	}
	start, end := time.UnixMilli(window.StartMs), time.UnixMilli(window.EndMs).Add(time.Millisecond-time.Nanosecond)
	lines := strings.Split(string(raw), "\r\n")
	seen := map[string]bool{}
	dates := map[string]bool{}
	section := 0
	prior := ""
	for i, line := range lines[:len(lines)-1] {
		if line == "" {
			section++
			seen = map[string]bool{}
			prior = ""
			continue
		}
		if strings.HasPrefix(line, " ") {
			if prior == "" || prior == "Build-Date" || prior == "Build-Date-UTC" {
				return nil, errors.New("invalid manifest continuation")
			}
			continue
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok || !manifestKeyPattern.MatchString(key) || seen[strings.ToLower(key)] {
			return nil, errors.New("invalid or duplicate manifest attribute")
		}
		seen[strings.ToLower(key)] = true
		prior = key
		if key != "Build-Date" && key != "Build-Date-UTC" {
			continue
		}
		if section != 0 || !nativeDatePattern.MatchString(value) {
			return nil, errors.New("native date must be canonical UTC in main section")
		}
		instant, err := time.Parse(time.RFC3339Nano, value)
		if err != nil || instant.Before(start) || instant.After(end) {
			return nil, errors.New("native date outside invocation window")
		}
		dates[key] = true
		lines[i] = key + ": <NATIVE-EXECUTION-DATE>"
	}
	if len(dates) != 2 {
		return nil, errors.New("both exact native date attributes required")
	}
	return []byte(strings.Join(lines, "\r\n")), nil
}
