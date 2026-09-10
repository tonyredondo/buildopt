package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const ratOutputPath = "server/build/reports/licenseHeaders/rat.xml"
const ratOwner = ":server:licenseHeaders"

// Token offsets preserve the original XML. Only attribute value spans are
// replaced; re-encoding XML would silently ignore unrelated formatting changes.
var ratAttribute = regexp.MustCompile(`[A-Za-z_:][A-Za-z0-9_.:-]*\s*=\s*(?:'[^']*'|"[^"]*")`)

type ratReplacement struct {
	start, end int
	value      string
}

func ratAttributeSpan(raw []byte, start, end int, name string) (int, int, error) {
	found, begin, finish := 0, 0, 0
	for _, span := range ratAttribute.FindAllIndex(raw[start:end], -1) {
		text := raw[start+span[0] : start+span[1]]
		equals := bytes.IndexByte(text, '=')
		if strings.TrimSpace(string(text[:equals])) != name {
			continue
		}
		found++
		quote := bytes.IndexAny(text[equals+1:], "\"'") + equals + 1
		begin, finish = start+span[0]+quote+1, start+span[1]-1
	}
	if found != 1 {
		return 0, 0, errors.New("RAT attribute missing or duplicated")
	}
	return begin, finish, nil
}

func projectRAT(raw []byte, producer string, window nativeTimeWindow) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 16<<20 || !filepath.IsAbs(producer) || filepath.Clean(producer) != producer || producer == "/" || strings.ContainsAny(producer, "\\\r\n\"'<>&") || window.StartMs <= 0 || window.EndMs < window.StartMs {
		return nil, errors.New("bounded RAT input and producer window required")
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	depth, roots, resources := 0, 0, 0
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
				return nil, errors.New("unexpected RAT namespace")
			}
			attrs := map[string]string{}
			for _, attr := range t.Attr {
				if attr.Name.Space != "" {
					return nil, errors.New("unexpected RAT attribute namespace")
				}
				if _, ok := attrs[attr.Name.Local]; ok {
					return nil, errors.New("duplicate RAT attribute")
				}
				attrs[attr.Name.Local] = attr.Value
			}
			if depth == 0 {
				roots++
				if roots != 1 || t.Name.Local != "rat-report" {
					return nil, errors.New("RAT root drift")
				}
				stamp := attrs["timestamp"]
				parsed, err := time.Parse("2006-01-02T15:04:05+00:00", stamp)
				if err != nil || parsed.Format("2006-01-02T15:04:05+00:00") != stamp || parsed.Unix() < window.StartMs/1000 || parsed.Unix() > window.EndMs/1000 {
					return nil, errors.New("RAT timestamp outside producer execution")
				}
				a, b, err := ratAttributeSpan(raw, start, end, "timestamp")
				if err != nil {
					return nil, err
				}
				if string(raw[a:b]) != stamp {
					return nil, errors.New("unexpected RAT timestamp encoding")
				}
				replacements = append(replacements, ratReplacement{a, b, "@BUILDOPT_TIMESTAMP@"})
			} else if t.Name.Local == "rat-report" {
				return nil, errors.New("nested RAT root")
			}
			if t.Name.Local == "resource" {
				if depth != 1 {
					return nil, errors.New("unexpected RAT resource nesting")
				}
				name := attrs["name"]
				rel, ok := strings.CutPrefix(name, producer+"/")
				if !ok || !filepath.IsLocal(rel) || filepath.Clean(rel) != rel || strings.ContainsAny(rel, "\\\r\n") || seen[name] {
					return nil, errors.New("RAT resource producer, path or identity drift")
				}
				seen[name] = true
				resources++
				if resources > 100000 {
					return nil, errors.New("RAT resource limit")
				}
				a, b, err := ratAttributeSpan(raw, start, end, "name")
				if err != nil {
					return nil, err
				}
				if !bytes.HasPrefix(raw[a:b], []byte(producer+"/")) {
					return nil, errors.New("unexpected RAT producer encoding")
				}
				replacements = append(replacements, ratReplacement{a, a + len(producer), "@BUILDOPT_WORKSPACE@"})
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.Directive:
			return nil, errors.New("RAT directives unsupported")
		case xml.ProcInst:
			if t.Target != "xml" || roots != 0 {
				return nil, errors.New("unexpected RAT processing instruction")
			}
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) != 0 {
				return nil, errors.New("text outside RAT report")
			}
		}
	}
	if depth != 0 || roots != 1 || resources == 0 {
		return nil, errors.New("incomplete RAT report")
	}
	var out bytes.Buffer
	previous := 0
	for _, replacement := range replacements {
		if replacement.start < previous || replacement.end > len(raw) {
			return nil, fmt.Errorf("overlapping RAT projection")
		}
		out.Write(raw[previous:replacement.start])
		out.WriteString(replacement.value)
		previous = replacement.end
	}
	out.Write(raw[previous:])
	return out.Bytes(), nil
}
