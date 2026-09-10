package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRATProjectionOnlyReplacesApprovedAttributes(t *testing.T) {
	a := []byte(`<rat-report timestamp='2026-09-07T11:22:48+00:00'><resource name='/producer/a/F.java'><license-approval name='true'/></resource></rat-report>`)
	b := bytes.ReplaceAll(a, []byte("/producer/a"), []byte("/producer/b"))
	b = bytes.Replace(b, []byte("11:22:48"), []byte("11:22:49"), 1)
	x, err := projectRAT(a, "/producer/a", nativeTimeWindow{StartMs: 1788780168123, EndMs: 1788780168999})
	if err != nil {
		t.Fatal(err)
	}
	y, err := projectRAT(b, "/producer/b", nativeTimeWindow{StartMs: 1788780169000, EndMs: 1788780169999})
	if err != nil || !bytes.Equal(x, y) {
		t.Fatalf("approved attributes: %s %v", y, err)
	}
	for _, change := range []struct{ before, after string }{
		{"name='true'", "name='false'"}, {"F.java", "G.java"}, {"/></resource>", "/> changed</resource>"},
		{"<resource name=", "<resource extra='x' name="},
	} {
		altered := strings.Replace(string(a), change.before, change.after, 1)
		z, err := projectRAT([]byte(altered), "/producer/a", nativeTimeWindow{StartMs: 1788780168000, EndMs: 1788780168999})
		if err == nil && bytes.Equal(z, x) {
			t.Fatalf("content ignored: %+v", change)
		}
	}
	for _, bad := range []string{
		strings.Replace(string(a), "11:22:48", "11:22:47", 1),
		strings.Replace(string(a), "11:22:48", "11:22:49", 1),
		strings.Replace(string(a), "/producer/a/F.java", "/producer/ab/F.java", 1),
		strings.Replace(string(a), "/producer/a/F.java", "/producer/a/../F.java", 1),
		strings.Replace(string(a), "timestamp=", "other=", 1),
		strings.Replace(string(a), "<rat-report ", "<rat-report timestamp='2026-09-07T11:22:48+00:00' ", 1),
		string(a[:len(a)-4]), string(a) + "<extra/>", "<!DOCTYPE rat-report>" + string(a),
	} {
		if _, err := projectRAT([]byte(bad), "/producer/a", nativeTimeWindow{StartMs: 1788780168000, EndMs: 1788780168999}); err == nil {
			t.Fatalf("invalid RAT accepted: %s", bad)
		}
	}
}

func TestRATProjectionPreservesResourceOrder(t *testing.T) {
	wrap := func(s string) []byte {
		return []byte(`<rat-report timestamp='2026-09-07T11:22:48+00:00'>` + s + `</rat-report>`)
	}
	a, b := `<resource name='/p/A'/>`, `<resource name='/p/B'/>`
	w := nativeTimeWindow{StartMs: 1788780168000, EndMs: 1788780168999}
	x, e := projectRAT(wrap(a+b), "/p", w)
	if e != nil {
		t.Fatal(e)
	}
	y, e := projectRAT(wrap(b+a), "/p", w)
	if e != nil || bytes.Equal(x, y) {
		t.Fatal("resource order ignored", e)
	}
	if _, e := projectRAT(wrap(a+a), "/p", w); e == nil {
		t.Fatal("duplicate resource accepted")
	}
}
