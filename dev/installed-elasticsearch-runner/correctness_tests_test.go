package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnerJUnitRequiresEveryExpectedTestAndNoHiddenFailure(t *testing.T) {
	definition := ownerTestDefinition{Class: "org.example.OwnerTests", Methods: []string{"first", "second"}}
	raw := `<testsuite name="org.example.OwnerTests" tests="2" failures="0" errors="0" skipped="0"><testcase name="first" classname="org.example.OwnerTests"/><testcase name="second" classname="org.example.OwnerTests"/></testsuite>`
	for _, item := range []struct {
		name, xml string
		valid     bool
	}{{"valid", raw, true}, {"skip", strings.Replace(raw, `skipped="0"`, `skipped="1"`, 1), false}, {"missing", strings.Replace(raw, `<testcase name="second" classname="org.example.OwnerTests"/>`, "", 1), false}, {"duplicate", strings.Replace(raw, `name="second"`, `name="first"`, 1), false}, {"other-class", strings.Replace(raw, `classname="org.example.OwnerTests"`, `classname="other"`, 1), false}, {"hidden-failure", strings.Replace(raw, `<testcase name="first" classname="org.example.OwnerTests"/>`, `<testcase name="first" classname="org.example.OwnerTests"><failure>broken</failure></testcase>`, 1), false}, {"trailing", raw + raw, false}} {
		t.Run(item.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.xml")
			if err := os.WriteFile(path, []byte(item.xml), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := checkOwnerJUnit(path, definition)
			if (err == nil) != item.valid {
				t.Fatalf("accepted=%t error=%v", err == nil, err)
			}
		})
	}
}

func TestOwnerTestDefinitionsComeFromPinnedClasses(t *testing.T) {
	java := ""
	for i := 0; i < 20; i++ {
		java += fmt.Sprintf("@Test\npublic void test%d() throws Exception {}\n", i)
	}
	groovy := ""
	for i := 0; i < 6; i++ {
		groovy += fmt.Sprintf("def \"native test %d\"() {}\n", i)
	}
	for workload, source := range map[string]string{"unit": java, "functional": groovy} {
		definition, err := parseOwnerTestDefinition(workload, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		want := 20
		if workload == "functional" {
			want = 6
		}
		if len(definition.Methods) != want {
			t.Fatal(definition)
		}
	}
	if _, err := parseOwnerTestDefinition("functional", []byte(groovy+`def "extra build"() {}`)); err == nil {
		t.Fatal("changed nested allocation accepted")
	}
}
