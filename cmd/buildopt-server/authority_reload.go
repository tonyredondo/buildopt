package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tonyredondo/buildopt/internal/localauthority"
	"github.com/tonyredondo/buildopt/internal/sharedcache"
)

const (
	authorityReloadInterval  = time.Second
	maximumAuthorityFileSize = 1 << 20
)

type authorityPaths struct {
	document   string
	trustRoot  string
	credential string
}

type loadedAuthority struct {
	handler     http.Handler
	digest      string
	expiresAt   time.Time
	fingerprint [sha256.Size]byte
}

func loadServerAuthority(
	ctx context.Context,
	storage *sharedcache.Storage,
	paths authorityPaths,
	now time.Time,
) (loadedAuthority, error) {
	before, err := authorityFilesFingerprint(paths)
	if err != nil {
		return loadedAuthority{}, err
	}
	verified, _, credential, err := localauthority.LoadFiles(
		ctx,
		paths.document,
		paths.trustRoot,
		paths.credential,
		now.UTC(),
	)
	if err != nil {
		return loadedAuthority{}, err
	}
	defer clearCredential(credential)
	binding, _, err := storage.InstallLocalAuthority(
		ctx,
		verified,
		credential,
		now.UTC(),
	)
	if err != nil {
		return loadedAuthority{}, err
	}
	handler, err := sharedcache.NewLocalAuthorityHTTPHandler(
		storage,
		binding,
		credential,
	)
	if err != nil {
		return loadedAuthority{}, err
	}
	after, err := authorityFilesFingerprint(paths)
	if err != nil {
		return loadedAuthority{}, err
	}
	if before != after {
		return loadedAuthority{}, fmt.Errorf(
			"local authority files changed during verification",
		)
	}
	return loadedAuthority{
		handler:     handler,
		digest:      binding.AuthorityDigest,
		expiresAt:   binding.ExpiresAt,
		fingerprint: after,
	}, nil
}

func watchServerAuthority(
	ctx context.Context,
	storage *sharedcache.Storage,
	paths authorityPaths,
	initial loadedAuthority,
	cache *switchableHandler,
	operational *operationalRouter,
	application http.Handler,
	logger *log.Logger,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	active := initial
	lastError := ""
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			fingerprint, err := authorityFilesFingerprint(paths)
			if err == nil &&
				fingerprint == active.fingerprint &&
				now.UTC().Before(active.expiresAt) {
				continue
			}

			operational.deactivate()
			cache.set(nil)
			next, loadErr := loadServerAuthority(
				ctx,
				storage,
				paths,
				now.UTC(),
			)
			if loadErr != nil {
				message := loadErr.Error()
				if message != lastError {
					logger.Printf(
						"local cache authority reload unavailable; readiness disabled: %v",
						loadErr,
					)
					lastError = message
				}
				continue
			}
			cache.set(next.handler)
			operational.activate(application)
			if next.digest != active.digest {
				logger.Printf(
					"local cache authority reloaded: %s",
					next.digest,
				)
			}
			active = next
			lastError = ""
		}
	}
}

func authorityFilesFingerprint(
	paths authorityPaths,
) ([sha256.Size]byte, error) {
	hash := sha256.New()
	for _, path := range []string{
		paths.document,
		paths.trustRoot,
		paths.credential,
	} {
		info, err := os.Lstat(path)
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		if !info.Mode().IsRegular() ||
			info.Mode().Perm() != 0o600 ||
			info.Size() < 1 ||
			info.Size() > maximumAuthorityFileSize {
			return [sha256.Size]byte{}, fmt.Errorf(
				"unsafe local authority source",
			)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(content)
		_, _ = hash.Write([]byte{0})
	}
	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint, nil
}

func clearCredential(credential []byte) {
	for index := range credential {
		credential[index] = 0
	}
}
