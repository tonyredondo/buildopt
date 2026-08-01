//go:build linux

package launcher

func platformProcessIsolation() string  { return "POSIX_PROCESS_GROUP" }
func platformResourceIsolation() string { return "CGROUP_V2_WHEN_CONFIGURED" }
func platformStoragePolicy() string     { return "ALLOWLIST_PROVEN_LOCAL_FILESYSTEM" }
func platformBackgroundService() string { return "SYSTEMD" }
