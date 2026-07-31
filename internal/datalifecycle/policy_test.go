package datalifecycle

import (
	"strings"
	"testing"
	"time"
)

func TestValidateExportPolicyRequiresBoundedExplicitExpansion(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	valid := []ExportPolicy{
		{Profile: ExportSummary},
		{Profile: ExportTasks, ExplicitlyAuthorized: true},
		{Profile: ExportEvidence, ExplicitlyAuthorized: true},
		{
			Profile:              ExportDiagnostic,
			ExplicitlyAuthorized: true,
			DiagnosticUntil:      now.Add(DiagnosticMaximumLifetime),
		},
	}
	for _, policy := range valid {
		if err := ValidateExportPolicy(policy, now); err != nil {
			t.Fatalf("valid policy %+v: %v", policy, err)
		}
	}
	invalid := []ExportPolicy{
		{Profile: ExportTasks},
		{Profile: ExportEvidence},
		{Profile: ExportDiagnostic, ExplicitlyAuthorized: true},
		{
			Profile:              ExportDiagnostic,
			ExplicitlyAuthorized: true,
			DiagnosticUntil:      now.Add(DiagnosticMaximumLifetime + time.Second),
		},
		{Profile: ExportSummary, DiagnosticUntil: now.Add(time.Hour)},
		{Profile: "UNKNOWN", ExplicitlyAuthorized: true},
	}
	for _, policy := range invalid {
		if err := ValidateExportPolicy(policy, now); err == nil {
			t.Fatalf("accepted invalid policy %+v", policy)
		}
	}
}

func TestRedactorIsKeyedDomainSeparatedAndSecretFree(t *testing.T) {
	key := make([]byte, RedactionKeyBytes)
	for index := range key {
		key[index] = byte(index + 1)
	}
	redactor, err := NewRedactor(key, "deployment-key-v1")
	if err != nil {
		t.Fatalf("create redactor: %v", err)
	}
	first := redactor.Token("repository", "sensitive-repository")
	replay := redactor.Token("repository", "sensitive-repository")
	otherDomain := redactor.Token("tenant", "sensitive-repository")
	if first != replay || first == otherDomain ||
		!strings.HasPrefix(first, "hmac-sha256:") ||
		strings.Contains(first, "sensitive-repository") {
		t.Fatalf("unexpected redaction tokens %q/%q/%q", first, replay, otherDomain)
	}
	if redactor.Version() != "deployment-key-v1" {
		t.Fatalf("key version = %q", redactor.Version())
	}
}
