//go:build darwin

package launcher

func platformProcessIsolation() string  { return "POSIX_PROCESS_GROUP" }
func platformResourceIsolation() string { return "PROCESS_GROUP_NO_HARD_LIMIT" }
func platformStoragePolicy() string     { return "MNT_LOCAL_SAME_DEVICE" }
func platformBackgroundService() string { return "LAUNCHD" }
