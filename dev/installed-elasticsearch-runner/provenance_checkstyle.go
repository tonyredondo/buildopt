package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

const checkstyleOutputPath = "server/build/reports/checkstyle/main.xml"
const checkstyleOwner = ":server:checkstyleMain"
const checkstyleClass = "org.gradle.api.plugins.quality.Checkstyle_Decorated"
const seededCheckstyleSHA = "5f28cf2fbd11267698ed81dd994fe463c51a8b3102bb51b6b8911b0506366ef4"

// Preserve lexical XML bytes, including every finding and its attribute order.
// Only the producer prefix inside direct file-name attributes is replaceable.
func projectCheckstyle(raw []byte, producer string) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 16<<20 || !filepath.IsAbs(producer) || filepath.Clean(producer) != producer || producer == "/" || strings.ContainsAny(producer, "\\\r\n\"'<>&") {
		return nil, errors.New("bounded Checkstyle report and canonical producer required")
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	depth, roots, files := 0, 0, 0
	seen := map[string]bool{}
	var replacements []ratReplacement
	for {
		start := int(d.InputOffset())
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		end := int(d.InputOffset())
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space != "" {
				return nil, errors.New("unexpected Checkstyle namespace")
			}
			attrs := map[string]string{}
			for _, attr := range t.Attr {
				if _, found := attrs[attr.Name.Local]; found || attr.Name.Space != "" {
					return nil, errors.New("duplicate or namespaced Checkstyle attribute")
				}
				attrs[attr.Name.Local] = attr.Value
			}
			if depth == 0 {
				roots++
				if roots != 1 || t.Name.Local != "checkstyle" || attrs["version"] == "" {
					return nil, errors.New("Checkstyle root drift")
				}
			} else if t.Name.Local == "checkstyle" {
				return nil, errors.New("nested Checkstyle root")
			}
			if t.Name.Local == "file" {
				if depth != 1 {
					return nil, errors.New("unexpected Checkstyle file nesting")
				}
				name := attrs["name"]
				rel, ok := strings.CutPrefix(name, producer+"/")
				if !ok || !filepath.IsLocal(rel) || filepath.Clean(rel) != rel || strings.ContainsAny(rel, "\\\r\n\t") || seen[name] {
					return nil, errors.New("Checkstyle producer, file path or identity drift")
				}
				seen[name] = true
				files++
				if files > 100000 {
					return nil, errors.New("Checkstyle file limit")
				}
				a, b, err := ratAttributeSpan(raw, start, end, "name")
				if err != nil {
					return nil, err
				}
				if !bytes.HasPrefix(raw[a:b], []byte(producer+"/")) {
					return nil, errors.New("unexpected Checkstyle producer encoding")
				}
				replacements = append(replacements, ratReplacement{a, a + len(producer), "@BUILDOPT_WORKSPACE@"})
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.Directive:
			return nil, errors.New("Checkstyle directives unsupported")
		case xml.ProcInst:
			if t.Target != "xml" || roots != 0 {
				return nil, errors.New("unexpected Checkstyle processing instruction")
			}
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) != 0 {
				return nil, errors.New("text outside Checkstyle report")
			}
		}
	}
	if roots != 1 || depth != 0 || files == 0 {
		return nil, errors.New("incomplete Checkstyle report")
	}
	var out bytes.Buffer
	previous := 0
	for _, r := range replacements {
		if r.start < previous || r.end > len(raw) {
			return nil, errors.New("overlapping Checkstyle projection")
		}
		out.Write(raw[previous:r.start])
		out.WriteString(r.value)
		previous = r.end
	}
	out.Write(raw[previous:])
	return out.Bytes(), nil
}
