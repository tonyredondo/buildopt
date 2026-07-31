// Package datalifecycle owns the isolated private-beta export and managed-data
// lifecycle boundary.
package datalifecycle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	// RedactionKeyBytes is the required entropy for deployment-local HMAC
	// tokenization keys.
	RedactionKeyBytes = 32
	// DiagnosticMaximumLifetime is the longest private-beta diagnostic opt-in.
	DiagnosticMaximumLifetime = 7 * 24 * time.Hour
)

// ExportProfile is the monotonic private-beta export vocabulary.
type ExportProfile string

const (
	ExportSummary    ExportProfile = "SUMMARY"
	ExportTasks      ExportProfile = "TASKS"
	ExportEvidence   ExportProfile = "EVIDENCE"
	ExportDiagnostic ExportProfile = "DIAGNOSTIC"
)

// ExportPolicy is one explicitly authorized export ceiling. Summary is the
// default. Every wider profile must be explicitly authorized, and diagnostic
// access additionally carries a bounded expiry.
type ExportPolicy struct {
	Profile              ExportProfile
	ExplicitlyAuthorized bool
	DiagnosticUntil      time.Time
}

// ValidateExportPolicy rejects implicit profile expansion and unbounded
// diagnostic collection.
func ValidateExportPolicy(policy ExportPolicy, now time.Time) error {
	if now.IsZero() {
		return errors.New("validate export policy: zero current time")
	}
	switch policy.Profile {
	case "", ExportSummary:
		if !policy.DiagnosticUntil.IsZero() {
			return errors.New("validate export policy: summary cannot authorize diagnostics")
		}
		return nil
	case ExportTasks, ExportEvidence:
		if !policy.ExplicitlyAuthorized || !policy.DiagnosticUntil.IsZero() {
			return errors.New("validate export policy: expanded profile requires exact authorization")
		}
		return nil
	case ExportDiagnostic:
		until := policy.DiagnosticUntil.UTC()
		if !policy.ExplicitlyAuthorized || !until.After(now.UTC()) ||
			until.After(now.UTC().Add(DiagnosticMaximumLifetime)) {
			return errors.New("validate export policy: diagnostic opt-in must expire within seven days")
		}
		return nil
	default:
		return errors.New("validate export policy: unknown profile")
	}
}

// Redactor applies deployment-keyed tokenization before managed persistence.
type Redactor struct {
	key     [RedactionKeyBytes]byte
	version string
}

// NewRedactor copies and validates one deployment-local key and its public
// rotation identifier.
func NewRedactor(key []byte, version string) (*Redactor, error) {
	if len(key) != RedactionKeyBytes || !validIdentifier(version) {
		return nil, errors.New("create data redactor: invalid key or version")
	}
	redactor := &Redactor{version: version}
	copy(redactor.key[:], key)
	return redactor, nil
}

// Token returns a keyed, domain-separated identifier suitable for persistence.
func (redactor *Redactor) Token(domain string, value string) string {
	if redactor == nil || domain == "" || value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, redactor.key[:])
	_, _ = mac.Write([]byte("buildopt-redaction-v1\x00"))
	_, _ = mac.Write([]byte(domain))
	_, _ = mac.Write([]byte{'\x00'})
	_, _ = mac.Write([]byte(value))
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

// Version is the non-secret key rotation identifier persisted with exports.
func (redactor *Redactor) Version() string {
	if redactor == nil {
		return ""
	}
	return redactor.version
}

func validIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for index := range len(value) {
		character := value[index]
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}
