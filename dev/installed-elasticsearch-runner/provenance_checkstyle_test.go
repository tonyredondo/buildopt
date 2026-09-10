package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckstyleProjectionPreservesEveryOtherByte(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?><checkstyle version="13.11.0"><file name='/producer/A.java'><error line="7" column="2" severity="warning" message="bad &amp; worse" source="Rule"/></file><file name="/producer/B.java"/></checkstyle>`)
	want := bytes.ReplaceAll(raw, []byte("/producer/"), []byte("@BUILDOPT_WORKSPACE@/"))
	got, err := projectCheckstyle(raw, "/producer")
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("projection changed other bytes", err)
	}
	other, err := projectCheckstyle(bytes.ReplaceAll(raw, []byte("/producer/"), []byte("/other/")), "/other")
	if err != nil || !bytes.Equal(got, other) {
		t.Fatal("workspace-only pair differs", err)
	}
	for _, change := range [][2]string{{`line="7"`, `line="8"`}, {`column="2"`, `column="3"`}, {`severity="warning"`, `severity="error"`}, {`bad &amp; worse`, `different`}, {`source="Rule"`, `source="Other"`}, {`13.11.0`, `13.11.1`}, {`A.java`, `C.java`}, {`<file name='/producer/A.java'>`, `<file extra="1" name='/producer/A.java'>`}, {`<error line`, `<error  line`}} {
		changed := bytes.Replace(raw, []byte(change[0]), []byte(change[1]), 1)
		p, err := projectCheckstyle(changed, "/producer")
		if err != nil || bytes.Equal(p, got) {
			t.Fatalf("hidden content change %v: %v", change, err)
		}
	}
	a := []byte(`<checkstyle version="13.11.0"><file name="/producer/A"/><file name="/producer/B"/></checkstyle>`)
	b := []byte(`<checkstyle version="13.11.0"><file name="/producer/B"/><file name="/producer/A"/></checkstyle>`)
	pa, _ := projectCheckstyle(a, "/producer")
	pb, _ := projectCheckstyle(b, "/producer")
	if bytes.Equal(pa, pb) {
		t.Fatal("file order hidden")
	}
}

func TestCheckstyleProjectionRejectsMalformedAndUnprovenPaths(t *testing.T) {
	valid := `<checkstyle version="13.11.0"><file name="/producer/A"/></checkstyle>`
	for _, raw := range []string{
		strings.Replace(valid, "/producer/A", "/producer-other/A", 1),
		strings.Replace(valid, "/producer/A", "/producer/../A", 1),
		strings.Replace(valid, "/producer/A", "/producer//A", 1),
		strings.Replace(valid, "/producer/A", "/produc&#101;r/A", 1),
		strings.Replace(valid, `name="/producer/A"`, `name="/producer/A" name="/producer/A"`, 1),
		strings.Replace(valid, `<file name="/producer/A"/>`, `<file name="/producer/A"/><file name="/producer/A"/>`, 1),
		strings.Replace(valid, `<file name="/producer/A"/>`, `<x><file name="/producer/A"/></x>`, 1),
		valid + valid, valid + "garbage", strings.TrimSuffix(valid, "</checkstyle>"),
		`<!DOCTYPE checkstyle>` + valid, strings.Replace(valid, "<checkstyle ", `<checkstyle xmlns="unexpected" `, 1),
		`<checkstyle/>`, `<wrong><file name="/producer/A"/></wrong>`,
	} {
		if _, err := projectCheckstyle([]byte(raw), "/producer"); err == nil {
			t.Fatalf("accepted invalid report %q", raw)
		}
	}
}
