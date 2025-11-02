package github

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/devcontainer-community/nanolayer-go/internal/linuxsystem"
)

func TestGetGitHubReleases_AddsPerPageParameter(t *testing.T) {
	var capturedQuery string

	transport := newMockTransport(transportRoute{
		match: func(req *http.Request) bool {
			return req.Method == http.MethodGet && req.URL.Host == "api.github.com" && req.URL.Path == "/repos/dev/repo/releases"
		},
		respond: func(req *http.Request) (*http.Response, error) {
			capturedQuery = req.URL.RawQuery
			resp := jsonResponse(http.StatusOK, `[{"tag_name":"v1.2.3","prerelease":false}]`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	})

	setDefaultTransport(t, transport)

	releases, err := GetGitHubReleases("dev/repo", false)
	if err != nil {
		t.Fatalf("GetGitHubReleases returned error: %v", err)
	}
	if capturedQuery != "per_page=30" {
		t.Fatalf("expected per_page query param, got %q", capturedQuery)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	if releases[0].TagName != "1.2.3" {
		t.Fatalf("expected tag name to be trimmed, got %q", releases[0].TagName)
	}
}

func TestGetGitHubReleases_AllPagesOmitsPerPage(t *testing.T) {
	var capturedQuery string

	transport := newMockTransport(transportRoute{
		match: func(req *http.Request) bool {
			return req.Method == http.MethodGet && req.URL.Host == "api.github.com" && req.URL.Path == "/repos/dev/repo/releases"
		},
		respond: func(req *http.Request) (*http.Response, error) {
			capturedQuery = req.URL.RawQuery
			resp := jsonResponse(http.StatusOK, `[{"tag_name":"v1.0.0","prerelease":false}]`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	})

	setDefaultTransport(t, transport)

	_, err := GetGitHubReleases("dev/repo", true)
	if err != nil {
		t.Fatalf("GetGitHubReleases returned error: %v", err)
	}
	if capturedQuery != "" {
		t.Fatalf("expected no query params when allPages=true, got %q", capturedQuery)
	}
}

func TestGetGitHubReleases_Non200Status(t *testing.T) {
	transport := newMockTransport(transportRoute{
		match: func(req *http.Request) bool {
			return req.Method == http.MethodGet && req.URL.Host == "api.github.com" && req.URL.Path == "/repos/dev/repo/releases"
		},
		respond: func(req *http.Request) (*http.Response, error) {
			resp := jsonResponse(http.StatusTooManyRequests, "rate limited")
			return resp, nil
		},
	})

	setDefaultTransport(t, transport)

	_, err := GetGitHubReleases("dev/repo", false)
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestGetLatestRelease_SkipsPreReleases(t *testing.T) {
	transport := newMockTransport(transportRoute{
		match: func(req *http.Request) bool {
			return req.Method == http.MethodGet && req.URL.Host == "api.github.com" && req.URL.Path == "/repos/dev/repo/releases"
		},
		respond: func(req *http.Request) (*http.Response, error) {
			payload := `[{"tag_name":"v2.0.0","prerelease":true},{"tag_name":"v1.5.0","prerelease":false}]`
			resp := jsonResponse(http.StatusOK, payload)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		},
	})

	setDefaultTransport(t, transport)

	release, err := GetLatestRelease("dev/repo", false)
	if err != nil {
		t.Fatalf("GetLatestRelease returned error: %v", err)
	}
	if release.TagName != "1.5.0" {
		t.Fatalf("expected TagName '1.5.0', got %q", release.TagName)
	}
}

func TestGetGitHubReleaseAsset_ResolvesLatest(t *testing.T) {
	expectedAssetURL := "https://downloads/dev/repo/1.2.3/linux/tool.tar.gz"
	var headRequested string

	transport := newMockTransport(
		transportRoute{
			match: func(req *http.Request) bool {
				return req.Method == http.MethodGet && req.URL.Host == "api.github.com" && req.URL.Path == "/repos/dev/repo/releases"
			},
			respond: func(req *http.Request) (*http.Response, error) {
				payload := `[{"tag_name":"v1.2.3","prerelease":false}]`
				resp := jsonResponse(http.StatusOK, payload)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			},
		},
		transportRoute{
			match: func(req *http.Request) bool {
				return req.Method == http.MethodHead && req.URL.String() == expectedAssetURL
			},
			respond: func(req *http.Request) (*http.Response, error) {
				headRequested = req.URL.String()
				return jsonResponse(http.StatusOK, ""), nil
			},
		},
	)

	setDefaultTransport(t, transport)

	url, err := GetGitHubReleaseAsset(
		"dev/repo",
		"latest",
		"https://downloads/${Repo}/${Version}/${Architecture}/${AssetName}",
		map[string]string{
			"Repo":         "dev/repo",
			"Architecture": "linux",
			"AssetName":    "tool.tar.gz",
		},
	)
	if err != nil {
		t.Fatalf("GetGitHubReleaseAsset returned error: %v", err)
	}
	if url != expectedAssetURL {
		t.Fatalf("expected asset URL %q, got %q", expectedAssetURL, url)
	}
	if headRequested != expectedAssetURL {
		t.Fatalf("expected HEAD against %q", expectedAssetURL)
	}
}

func TestDownloadAndInstallFromAssetUrl(t *testing.T) {
	entries := []archiveEntry{
		{name: "bin/", isDir: true},
		{name: "bin/tool", body: []byte("payload")},
	}

	tarData := createTarArchive(t, entries)
	gzData := compressGzipData(t, tarData)

	arch := string(linuxsystem.GetArchitecture())
	replacedArch := arch + "-portable"

	expectedAssetURL := fmt.Sprintf("https://downloads/1.0.0/%s/tool.tar.gz", replacedArch)

	tempDir := t.TempDir()
	destFile := filepath.Join(tempDir, "tool")

	transport := newMockTransport(
		transportRoute{
			match: func(req *http.Request) bool {
				return req.Method == http.MethodHead && req.URL.String() == expectedAssetURL
			},
			respond: func(req *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, ""), nil
			},
		},
		transportRoute{
			match: func(req *http.Request) bool {
				return req.Method == http.MethodGet && req.URL.String() == expectedAssetURL
			},
			respond: func(req *http.Request) (*http.Response, error) {
				resp := binaryResponse(http.StatusOK, gzData)
				return resp, nil
			},
		},
	)

	setDefaultTransport(t, transport)

	err := DownloadAndInstallFromAssetUrl(
		"dev/repo",
		"1.0.0",
		"tool.tar.gz",
		"https://downloads/${Version}/${Architecture}/${AssetName}",
		map[string]string{arch: replacedArch},
		map[string]string{"bin/tool": destFile},
	)
	if err != nil {
		t.Fatalf("DownloadAndInstallFromAssetUrl returned error: %v", err)
	}

	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

type transportRoute struct {
	match   func(*http.Request) bool
	respond func(*http.Request) (*http.Response, error)
}

type mockTransport struct {
	routes []transportRoute
}

func newMockTransport(routes ...transportRoute) *mockTransport {
	return &mockTransport{routes: routes}
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, route := range m.routes {
		if route.match(req) {
			return route.respond(req)
		}
	}
	return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
}

func setDefaultTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = previous
	})
}

func jsonResponse(status int, body string) *http.Response {
	return binaryResponse(status, []byte(body))
}

func binaryResponse(status int, body []byte) *http.Response {
	if body == nil {
		body = []byte{}
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Body:          io.NopCloser(bytes.NewReader(body)),
		Header:        make(http.Header),
		ContentLength: int64(len(body)),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
}
