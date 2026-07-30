//go:build !linux

package sharedcache

import (
	"errors"
	"os"
)

var errUnsupportedStoragePlatform = errors.New(
	"single-node Shared storage currently requires Linux",
)

func preparePrivateDirectory(string) error {
	return errUnsupportedStoragePlatform
}

func openPrivateLock(string) (*os.File, error) {
	return nil, errUnsupportedStoragePlatform
}

func preparePrivateDatabase(string) error {
	return errUnsupportedStoragePlatform
}

func openPrivateBlob(string) (*os.File, error) {
	return nil, errUnsupportedStoragePlatform
}

func validatePrivateSidecar(string) error {
	return errUnsupportedStoragePlatform
}

func acquireExclusiveLock(*os.File) error {
	return errUnsupportedStoragePlatform
}

func releaseExclusiveLock(*os.File) error {
	return errUnsupportedStoragePlatform
}

func isLockBusy(error) bool {
	return false
}

func validateLocalStorageFilesystem(...string) error {
	return errUnsupportedStoragePlatform
}

func isSupportedLocalFilesystemType(int64) bool {
	return false
}

func syncDirectory(string) error {
	return errUnsupportedStoragePlatform
}
