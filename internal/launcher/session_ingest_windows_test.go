//go:build windows

package launcher

import (
	"fmt"
	"strconv"
)

func sessionIngestChildCommand(exitCode int) []string {
	script := fmt.Sprintf(
		"if defined BUILDOPT_SERVER_URL (exit /b 97) else if defined BUILDOPT_SERVER_INGEST_TOKEN (exit /b 98) else exit /b %s",
		strconv.Itoa(exitCode),
	)
	return []string{"run", "--", "cmd.exe", "/d", "/s", "/c", script}
}
