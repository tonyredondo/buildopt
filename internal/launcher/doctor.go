package launcher

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

type doctorReport struct {
	SchemaVersion          string `json:"schemaVersion"`
	OperatingSystem        string `json:"operatingSystem"`
	Architecture           string `json:"architecture"`
	PersistentGateway      bool   `json:"persistentGateway"`
	ManagedL1              bool   `json:"managedL1"`
	ManagedGradleBootstrap bool   `json:"managedGradleBootstrap"`
	Server                 bool   `json:"server"`
	Edge                   bool   `json:"edge"`
	ProcessIsolation       string `json:"processIsolation"`
	ResourceIsolation      string `json:"resourceIsolation"`
	StoragePolicy          string `json:"storagePolicy"`
	BackgroundService      string `json:"backgroundService"`
}

func runDoctor(stdout, stderr io.Writer) int {
	report := doctorReport{
		SchemaVersion:          "buildopt.doctor/v1",
		OperatingSystem:        runtime.GOOS,
		Architecture:           runtime.GOARCH,
		PersistentGateway:      true,
		ManagedL1:              true,
		ManagedGradleBootstrap: true,
		Server:                 true,
		Edge:                   true,
		ProcessIsolation:       platformProcessIsolation(),
		ResourceIsolation:      platformResourceIsolation(),
		StoragePolicy:          platformStoragePolicy(),
		BackgroundService:      platformBackgroundService(),
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		_, _ = fmt.Fprintln(stderr, "buildopt: encode doctor report")
		return 1
	}
	return 0
}
