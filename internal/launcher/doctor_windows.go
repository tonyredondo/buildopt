//go:build windows

package launcher

func platformProcessIsolation() string  { return "JOB_OBJECT_KILL_ON_CLOSE" }
func platformResourceIsolation() string { return "JOB_OBJECT" }
func platformStoragePolicy() string     { return "LOCAL_VOLUME_NO_REPARSE_POINTS" }
func platformBackgroundService() string { return "WINDOWS_SERVICE" }
