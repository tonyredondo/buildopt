package main

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"
)

const firstManifest = "Manifest-Version: 1.0\r\nBuild-Date: 2026-09-07T12:00:00Z\r\nBuild-Date-UTC: 2026-09-07T12:00:01.123456789Z\r\nVersion: 1\r\n\r\n"

var manifestWindow = nativeTimeWindow{1788782399000, 1788782405000}

func secondManifest() string {
	return strings.ReplaceAll(strings.ReplaceAll(firstManifest, "12:00:00Z", "12:00:02Z"), "12:00:01.123456789Z", "12:00:03Z")
}

func TestNativeDateManifestExceptionIsNarrow(t *testing.T) {
	a, err := manifestProjection([]byte(firstManifest), manifestWindow)
	if err != nil {
		t.Fatal(err)
	}
	b, err := manifestProjection([]byte(secondManifest()), manifestWindow)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("native dates: %v", err)
	}
	c, err := manifestProjection([]byte(strings.Replace(secondManifest(), "Version: 1\r", "Version: 2\r", 1)), manifestWindow)
	if err != nil || bytes.Equal(a, c) {
		t.Fatalf("other attributes hidden: %v", err)
	}
	for _, bad := range []string{
		strings.Replace(firstManifest, "Build-Date: ", "Other-Date: ", 1),
		strings.Replace(firstManifest, "2026-09-07", "2026-02-30", 1),
		strings.Replace(firstManifest, "12:00:00Z", "13:00:00Z", 1),
		strings.Replace(firstManifest, "12:00:00Z", "12:00:00+00:00", 1),
		strings.Replace(firstManifest, "Version: 1\r\n", "Version: 1\r\nbuild-date: 2026-09-07T12:00:00Z\r\n", 1),
		strings.Replace(firstManifest, "12:00:00Z", "12:00:\r\n 00Z", 1),
		strings.ReplaceAll(firstManifest, "\r\n", "\n"), firstManifest + "\xff",
	} {
		if _, err := manifestProjection([]byte(bad), manifestWindow); err == nil {
			t.Errorf("invalid manifest accepted: %q", bad)
		}
	}
}

func projectionJar(t *testing.T, manifest, name, payload string, mode uint32) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, e := range []struct{ name, data string }{{"META-INF/MANIFEST.MF", manifest}, {name, payload}} {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate, ModifiedDate: 33, CreatorVersion: 3<<8 | 20, ExternalAttrs: mode << 16}
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte(e.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestNativeDateArchivePreservesAllOtherBytesAndMetadata(t *testing.T) {
	original := projectionJar(t, firstManifest, "x/A.class", "class bytes", 0644)
	project := func(b []byte) string {
		t.Helper()
		v, e := archiveProjection(b, []string{"META-INF/MANIFEST.MF"}, manifestWindow)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	a := project(original)
	if a != project(projectionJar(t, secondManifest(), "x/A.class", "class bytes", 0644)) {
		t.Fatal("equivalent native dates differ")
	}
	for _, b := range [][]byte{
		projectionJar(t, firstManifest, "x/A.class", "changed class", 0644), projectionJar(t, firstManifest, "x/B.class", "class bytes", 0644),
		projectionJar(t, firstManifest, "x/A.class", "class bytes", 0755), projectionJar(t, strings.Replace(firstManifest, "Version: 1\r", "Version: 2\r", 1), "x/A.class", "class bytes", 0644),
	} {
		if a == project(b) {
			t.Fatal("non-date difference hidden")
		}
	}
	corrupt := append([]byte{}, original...)
	corrupt[14] ^= 255
	for _, b := range [][]byte{append([]byte("prefix"), original...), append(append([]byte{}, original...), []byte("trailer")...), corrupt, projectionJar(t, firstManifest, "../A.class", "x", 0644), projectionJar(t, firstManifest, "META-INF/MANIFEST.MF", firstManifest, 0644)} {
		if _, err := archiveProjection(b, []string{"META-INF/MANIFEST.MF"}, manifestWindow); err == nil {
			t.Fatal("unsafe archive accepted")
		}
	}
	for n := 0; n < len(original); n++ {
		if _, err := archiveProjection(original[:n], []string{"META-INF/MANIFEST.MF"}, manifestWindow); err == nil {
			t.Fatalf("truncation accepted at %d", n)
		}
	}
	if _, err := archiveProjection(original, []string{"missing/MANIFEST.MF"}, manifestWindow); err == nil {
		t.Fatal("missing approved member accepted")
	}
}

func TestNativeDateWindowRetainsNanosecondBoundary(t *testing.T) {
	w := nativeTimeWindow{1788782401123, 1788782401123}
	raw := strings.ReplaceAll(strings.ReplaceAll(firstManifest, "12:00:00Z", "12:00:01.123000000Z"), "12:00:01.123456789Z", "12:00:01.123999999Z")
	if _, e := manifestProjection([]byte(raw), w); e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"12:00:01.122999999Z", "12:00:01.124000000Z"} {
		if _, e := manifestProjection([]byte(strings.Replace(raw, "12:00:01.123000000Z", s, 1)), w); e == nil {
			t.Fatal("nanosecond outside window accepted")
		}
	}
	if _, e := manifestProjection([]byte(firstManifest), nativeTimeWindow{time.Now().UnixMilli(), 0}); e == nil {
		t.Fatal("reversed window accepted")
	}
}
