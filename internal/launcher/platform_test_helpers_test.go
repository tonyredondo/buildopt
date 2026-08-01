package launcher

import (
	"strings"
	"testing"
)

func clearManagedGatewayTestEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv(managedGatewayStateRootEnvironment, "")
	t.Setenv(managedRunnerSlotEnvironment, "")
	t.Setenv(managedGatewayIdleTimeoutEnvironment, "")
}

func environmentValue(environment []string, key string) string {
	prefix := key + "="
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func environmentKeyCount(environment []string, key string) int {
	prefix := key + "="
	count := 0
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			count++
		}
	}
	return count
}
