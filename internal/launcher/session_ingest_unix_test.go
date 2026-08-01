//go:build !windows

package launcher

import "strconv"

func sessionIngestChildCommand(exitCode int) []string {
	return []string{
		"run", "--", "/bin/sh", "-c",
		`test -z "${BUILDOPT_SERVER_URL+x}" || exit 97
test -z "${BUILDOPT_SERVER_INGEST_TOKEN+x}" || exit 98
exit "$1"`,
		"buildopt-session-test", strconv.Itoa(exitCode),
	}
}
