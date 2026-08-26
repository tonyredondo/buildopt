package stickywrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const defaultGitHubAPI = "https://api.github.com/repos/tonyredondo/buildopt"

// GitHubResolver reads immutable release asset digests from the public GitHub
// API. It never downloads a distribution archive and rejects redirects.
type GitHubResolver struct {
	client  *http.Client
	baseURL string
	token   string
}

// NewGitHubResolver creates the production release-metadata resolver. token is
// optional and is used only as an HTTP Authorization value, never persisted.
func NewGitHubResolver(token string) *GitHubResolver {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &GitHubResolver{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:       http.ProxyFromEnvironment,
				DialContext: dialer.DialContext,
			},
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("redirects are rejected")
			},
		},
		baseURL: defaultGitHubAPI,
		token:   token,
	}
}

// Latest resolves the latest non-draft, non-prerelease public release.
func (resolver *GitHubResolver) Latest(ctx context.Context) (Release, error) {
	return resolver.resolve(ctx, "/releases/latest")
}

// Version resolves one exact public release tag.
func (resolver *GitHubResolver) Version(ctx context.Context, version string) (Release, error) {
	if !versionPattern.MatchString(version) {
		return Release{}, fmt.Errorf("invalid release version %q", version)
	}
	return resolver.resolve(ctx, "/releases/tags/v"+version)
}

func (resolver *GitHubResolver) resolve(ctx context.Context, path string) (Release, error) {
	if resolver == nil || resolver.client == nil || resolver.baseURL == "" {
		return Release{}, errors.New("GitHub release resolver is unavailable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolver.baseURL+path, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "buildopt-sticky-wrapper-poc")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if resolver.token != "" {
		request.Header.Set("Authorization", "Bearer "+resolver.token)
	}
	response, err := resolver.client.Do(request)
	if err != nil {
		return Release{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return Release{}, fmt.Errorf("GitHub release API returned %s", response.Status)
	}
	limited := io.LimitReader(response.Body, 1<<20)
	var document struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name               string `json:"name"`
			Digest             string `json:"digest"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(&document); err != nil {
		return Release{}, fmt.Errorf("decode GitHub release metadata: %w", err)
	}
	if document.Draft || document.Prerelease || !strings.HasPrefix(document.TagName, "v") {
		return Release{}, errors.New("release is not an immutable public stable release")
	}
	version := strings.TrimPrefix(document.TagName, "v")
	if !versionPattern.MatchString(version) {
		return Release{}, fmt.Errorf("release tag %q is not supported", document.TagName)
	}
	expectedNames := map[string]string{
		"linux-amd64":   "buildopt-" + version + "-linux-amd64.tar.gz",
		"macos-amd64":   "buildopt-" + version + "-darwin-amd64.tar.gz",
		"macos-arm64":   "buildopt-" + version + "-darwin-arm64.tar.gz",
		"windows-amd64": "buildopt-" + version + "-windows-amd64.zip",
	}
	release := Release{Version: version, Distributions: make(map[string]Distribution, len(expectedNames))}
	for platform, expected := range expectedNames {
		for _, asset := range document.Assets {
			if asset.Name != expected {
				continue
			}
			digest := strings.TrimPrefix(asset.Digest, "sha256:")
			if digest == asset.Digest || !sha256Pattern.MatchString(digest) {
				return Release{}, fmt.Errorf("asset %s has no SHA-256 digest", expected)
			}
			release.Distributions[platform] = Distribution{
				URL:    asset.BrowserDownloadURL,
				SHA256: digest,
			}
			break
		}
		if _, ok := release.Distributions[platform]; !ok {
			return Release{}, fmt.Errorf("release is missing asset %s", expected)
		}
	}
	return release, nil
}
