package main

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"regexp"
	"sort"
)

type ownerTestDefinition struct {
	Class   string   `json:"class"`
	Source  string   `json:"source"`
	SHA256  string   `json:"sha256"`
	Methods []string `json:"methods"`
}

func parseOwnerTestDefinition(workload string, source []byte) (ownerTestDefinition, error) {
	definition := ownerTestDefinition{SHA256: digest(source), Methods: []string{}}
	var pattern string
	want := 0
	switch workload {
	case "unit":
		definition.Class = "org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsTaskTests"
		definition.Source = "build-tools-internal/src/test/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTaskTests.java"
		pattern = `@Test(?:\([^)]*\))?\s+public void (\w+)\(`
		want = 20
	case "functional":
		definition.Class = "org.elasticsearch.gradle.internal.precommit.ForbiddenPatternsPrecommitPluginFuncTest"
		definition.Source = "build-tools-internal/src/integTest/groovy/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsPrecommitPluginFuncTest.groovy"
		pattern = `def "([^"]+)"\(\)`
		want = 6
	default:
		return definition, errors.New("unallocated owner test class")
	}
	for _, match := range regexp.MustCompile(pattern).FindAllSubmatch(source, -1) {
		definition.Methods = append(definition.Methods, string(match[1]))
	}
	sort.Strings(definition.Methods)
	if len(definition.Methods) != want {
		return definition, errors.New("upstream test count changed")
	}
	for i := 1; i < len(definition.Methods); i++ {
		if definition.Methods[i-1] == definition.Methods[i] {
			return definition, errors.New("duplicate upstream test method")
		}
	}
	return definition, nil
}

func checkOwnerJUnit(path string, definition ownerTestDefinition) (int, error) {
	var suite struct {
		XMLName  xml.Name `xml:"testsuite"`
		Name     string   `xml:"name,attr"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Errors   int      `xml:"errors,attr"`
		Skipped  int      `xml:"skipped,attr"`
		Cases    []struct {
			Name     string     `xml:"name,attr"`
			Class    string     `xml:"classname,attr"`
			Failures []struct{} `xml:"failure"`
			Errors   []struct{} `xml:"error"`
			Skipped  []struct{} `xml:"skipped"`
		} `xml:"testcase"`
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() > 4<<20 {
		return 0, errors.New("bounded owner XML required")
	}
	decoder := xml.NewDecoder(f)
	if err = decoder.Decode(&suite); err != nil {
		return 0, err
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		switch value := token.(type) {
		case xml.CharData:
			for _, b := range value {
				if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
					return 0, errors.New("trailing XML content")
				}
			}
		default:
			return 0, errors.New("trailing XML document")
		}
	}
	if suite.Name != definition.Class || suite.Tests != len(definition.Methods) || len(suite.Cases) != suite.Tests || suite.Failures != 0 || suite.Errors != 0 || suite.Skipped != 0 {
		return 0, errors.New("owner suite missing tests or contains failure, error or skip")
	}
	expected := map[string]bool{}
	for _, name := range definition.Methods {
		expected[name] = true
	}
	for _, test := range suite.Cases {
		if test.Class != definition.Class || !expected[test.Name] || len(test.Failures)+len(test.Errors)+len(test.Skipped) != 0 {
			return 0, errors.New("owner testcase identity or outcome mismatch")
		}
		delete(expected, test.Name)
	}
	if len(expected) != 0 {
		return 0, errors.New("missing owner test methods")
	}
	return suite.Tests, nil
}
