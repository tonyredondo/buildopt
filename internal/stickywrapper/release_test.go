package stickywrapper

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGitHubResolverUsesImmutableAssetDigests(t *testing.T) {
	const body = `{
  "tag_name":"v3.4.5",
  "draft":false,
  "prerelease":false,
  "ignored":"forward-compatible",
  "assets":[
    {"name":"buildopt-3.4.5-linux-amd64.tar.gz","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","browser_download_url":"https://github.example/releases/download/v3.4.5/buildopt-3.4.5-linux-amd64.tar.gz"},
    {"name":"buildopt-3.4.5-darwin-amd64.tar.gz","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","browser_download_url":"https://github.example/releases/download/v3.4.5/buildopt-3.4.5-darwin-amd64.tar.gz"},
    {"name":"buildopt-3.4.5-darwin-arm64.tar.gz","digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","browser_download_url":"https://github.example/releases/download/v3.4.5/buildopt-3.4.5-darwin-arm64.tar.gz"},
    {"name":"buildopt-3.4.5-windows-amd64.zip","digest":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","browser_download_url":"https://github.example/releases/download/v3.4.5/buildopt-3.4.5-windows-amd64.zip"}
  ]
}`
	requests := 0
	resolver := &GitHubResolver{
		baseURL: "https://api.example.invalid/repository",
		token:   "test-authorization-value",
		client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requests++
			if request.URL.String() != "https://api.example.invalid/repository/releases/latest" {
				t.Fatalf("request URL = %s", request.URL)
			}
			if request.Header.Get("Authorization") != "Bearer test-authorization-value" {
				t.Fatal("test authorization value was not used only as request authorization")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    request,
			}, nil
		})},
	}
	release, err := resolver.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || release.Version != "3.4.5" ||
		release.Distributions["linux-amd64"].SHA256 != strings.Repeat("a", 64) ||
		release.Distributions["windows-amd64"].SHA256 != strings.Repeat("d", 64) {
		t.Fatalf("unexpected resolved release: %#v", release)
	}
}

func TestGitHubResolverRejectsIncompleteOrRedirectedMetadata(t *testing.T) {
	testCases := []struct {
		name string
		body string
		err  error
	}{
		{name: "redirect", err: errors.New("redirect rejected")},
		{name: "draft", body: `{"tag_name":"v1.0.0","draft":true,"prerelease":false,"assets":[]}`},
		{name: "missing assets", body: `{"tag_name":"v1.0.0","draft":false,"prerelease":false,"assets":[]}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolver := &GitHubResolver{
				baseURL: "https://api.example.invalid/repository",
				client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if testCase.err != nil {
						return nil, testCase.err
					}
					return &http.Response{
						StatusCode: http.StatusOK, Status: "200 OK",
						Header: make(http.Header), Body: io.NopCloser(strings.NewReader(testCase.body)), Request: request,
					}, nil
				})},
			}
			if _, err := resolver.Latest(context.Background()); err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
}
