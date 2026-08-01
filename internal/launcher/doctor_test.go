package launcher

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDoctorReportsCompleteRuntimeSurface(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exitCode := runDoctor(&stdout, &stderr); exitCode != 0 {
		t.Fatalf("doctor exit = %d, stderr = %q", exitCode, stderr.String())
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != "buildopt.doctor/v1" || report.OperatingSystem == "" ||
		!report.PersistentGateway || !report.ManagedL1 || !report.ManagedGradleBootstrap ||
		!report.Server || !report.Edge || report.ProcessIsolation == "" ||
		report.ResourceIsolation == "" || report.StoragePolicy == "" ||
		report.BackgroundService == "" {
		t.Fatalf("doctor report = %+v", report)
	}
}
