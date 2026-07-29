// Package contractcrypto implements the small, dependency-free cryptographic
// boundary shared by BuildOpt contract conformance tests.
package contractcrypto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

var (
	errDuplicateKey      = errors.New("duplicate-key")
	errInvalidUTF8       = errors.New("invalid-utf8")
	errInvalidJSON       = errors.New("invalid-json")
	errUnpairedSurrogate = errors.New("unpaired-surrogate")
)

// CanonicalizeJCS decodes one JSON value and returns its RFC 8785 canonical
// UTF-8 representation. JSON numbers use the ECMAScript-compatible IEEE-754
// rendering boundaries required by JCS.
func CanonicalizeJCS(input []byte) ([]byte, error) {
	if !utf8.Valid(input) {
		return nil, errInvalidUTF8
	}
	if err := validateJCSUnicodeEscapes(input); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	value, err := decodeJCSValue(decoder)
	if err != nil {
		return nil, err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("%w: trailing token: %v", errInvalidJSON, err)
		}
		return nil, fmt.Errorf("%w: trailing value %v", errInvalidJSON, token)
	}

	var canonical bytes.Buffer
	if err := appendJCSValue(&canonical, value); err != nil {
		return nil, err
	}
	return canonical.Bytes(), nil
}

func validateJCSUnicodeEscapes(input []byte) error {
	inString := false
	for index := 0; index < len(input); index++ {
		switch input[index] {
		case '"':
			inString = !inString
		case '\\':
			if !inString || index+1 >= len(input) {
				continue
			}
			index++
			if input[index] != 'u' {
				continue
			}
			first, ok := parseJCSHexCodeUnit(input, index+1)
			if !ok {
				continue
			}
			index += 4
			if first >= 0xd800 && first <= 0xdbff {
				if index+6 >= len(input) ||
					input[index+1] != '\\' ||
					input[index+2] != 'u' {
					return errUnpairedSurrogate
				}
				second, ok := parseJCSHexCodeUnit(input, index+3)
				if !ok || second < 0xdc00 || second > 0xdfff {
					return errUnpairedSurrogate
				}
				index += 6
			} else if first >= 0xdc00 && first <= 0xdfff {
				return errUnpairedSurrogate
			}
		}
	}
	return nil
}

func parseJCSHexCodeUnit(input []byte, start int) (uint16, bool) {
	if start+4 > len(input) {
		return 0, false
	}
	var value uint16
	for index := start; index < start+4; index++ {
		var digit byte
		switch {
		case input[index] >= '0' && input[index] <= '9':
			digit = input[index] - '0'
		case input[index] >= 'a' && input[index] <= 'f':
			digit = input[index] - 'a' + 10
		case input[index] >= 'A' && input[index] <= 'F':
			digit = input[index] - 'A' + 10
		default:
			return 0, false
		}
		value = value*16 + uint16(digit)
	}
	return value, true
}

// ValidUTCTimestamp reports whether value is an RFC 3339 timestamp with
// explicit seconds and the uppercase UTC Z suffix.
func ValidUTCTimestamp(value string) bool {
	if !strings.HasSuffix(value, "Z") {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func decodeJCSValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidJSON, err)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		switch value := token.(type) {
		case nil, bool, string, json.Number:
			return value, nil
		default:
			return nil, fmt.Errorf("%w: unsupported token %T", errInvalidJSON, token)
		}
	}

	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("%w: object key: %v", errInvalidJSON, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("%w: non-string object key", errInvalidJSON)
			}
			if _, duplicate := object[key]; duplicate {
				return nil, fmt.Errorf("%w: %q", errDuplicateKey, key)
			}
			value, err := decodeJCSValue(decoder)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, fmt.Errorf("%w: close object: %v", errInvalidJSON, err)
		}
		return object, nil
	case '[':
		var array []any
		for decoder.More() {
			value, err := decodeJCSValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, fmt.Errorf("%w: close array: %v", errInvalidJSON, err)
		}
		return array, nil
	default:
		return nil, fmt.Errorf("%w: unexpected delimiter %q", errInvalidJSON, delimiter)
	}
}

func appendJCSValue(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		output.WriteString(strconv.FormatBool(typed))
	case string:
		appendJCSString(output, typed)
	case json.Number:
		number, err := canonicalJCSNumber(typed.String())
		if err != nil {
			return err
		}
		output.WriteString(number)
	case []any:
		output.WriteByte('[')
		for index, element := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendJCSValue(output, element); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(first, second int) bool {
			return compareUTF16(keys[first], keys[second]) < 0
		})
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			appendJCSString(output, key)
			output.WriteByte(':')
			if err := appendJCSValue(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("%w: unsupported value %T", errInvalidJSON, value)
	}
	return nil
}

func appendJCSString(output *bytes.Buffer, value string) {
	output.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"', '\\':
			output.WriteByte('\\')
			output.WriteRune(character)
		case '\b':
			output.WriteString(`\b`)
		case '\t':
			output.WriteString(`\t`)
		case '\n':
			output.WriteString(`\n`)
		case '\f':
			output.WriteString(`\f`)
		case '\r':
			output.WriteString(`\r`)
		default:
			if character < 0x20 {
				fmt.Fprintf(output, `\u%04x`, character)
			} else {
				output.WriteRune(character)
			}
		}
	}
	output.WriteByte('"')
}

func canonicalJCSNumber(source string) (string, error) {
	value, err := strconv.ParseFloat(source, 64)
	if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		return "", fmt.Errorf("%w: non-finite number %q", errInvalidJSON, source)
	}
	if value == 0 {
		return "0", nil
	}
	absolute := math.Abs(value)
	if absolute >= 1e-6 && absolute < 1e21 {
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	}
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	exponentMarker := strings.LastIndexByte(scientific, 'e')
	if exponentMarker < 0 {
		return scientific, nil
	}
	mantissa := scientific[:exponentMarker]
	exponent := scientific[exponentMarker+1:]
	sign := ""
	if strings.HasPrefix(exponent, "+") || strings.HasPrefix(exponent, "-") {
		sign = exponent[:1]
		exponent = exponent[1:]
	}
	exponent = strings.TrimLeft(exponent, "0")
	if exponent == "" {
		exponent = "0"
	}
	return mantissa + "e" + sign + exponent, nil
}

func compareUTF16(first string, second string) int {
	firstUnits := utf16.Encode([]rune(first))
	secondUnits := utf16.Encode([]rune(second))
	limit := min(len(firstUnits), len(secondUnits))
	for index := 0; index < limit; index++ {
		if firstUnits[index] < secondUnits[index] {
			return -1
		}
		if firstUnits[index] > secondUnits[index] {
			return 1
		}
	}
	switch {
	case len(firstUnits) < len(secondUnits):
		return -1
	case len(firstUnits) > len(secondUnits):
		return 1
	default:
		return 0
	}
}
