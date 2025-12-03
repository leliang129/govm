package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchVersionsFiltersAndSorts(t *testing.T) {
	t.Parallel()

	releases := []release{
		{
			Version: "go1.20.1",
			Files: []releaseFile{
				{Filename: "go1.20.1.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Checksum: "amd64", Kind: "archive"},
				{Filename: "go1.20.1.windows-amd64.zip", OS: "windows", Arch: "amd64", Checksum: "win", Kind: "archive"},
				{Filename: "go1.20.1.linux-armv6l.tar.gz", OS: "linux", Arch: "armv6l", Checksum: "armv6l", Kind: "archive"},
			},
		},
		{
			Version: "go1.21rc1",
			Files: []releaseFile{
				{Filename: "go1.21rc1.linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Checksum: "arm64", Kind: "archive"},
				{Filename: "go1.21rc1.linux-386.tar.gz", OS: "linux", Arch: "386", Checksum: "386", Kind: "archive"},
				{Filename: "go1.21rc1.darwin-amd64.tar.gz", OS: "darwin", Arch: "amd64", Checksum: "darwin", Kind: "archive"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Fatalf("encode test data failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithCacheTTL(time.Minute),
	)

	versions, err := client.FetchVersions()
	if err != nil {
		t.Fatalf("FetchVersions error: %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("expected 3 linux versions, got %d", len(versions))
	}

	wantOrder := []string{"1.21rc1", "1.21rc1", "1.20.1"}
	for i, ver := range versions {
		if ver.Number != wantOrder[i] {
			t.Fatalf("unexpected order at %d: got %s want %s", i, ver.Number, wantOrder[i])
		}
		if ver.OS != "linux" {
			t.Fatalf("non-linux entry returned: %#v", ver)
		}
		if _, ok := supportedArch[ver.Arch]; !ok {
			t.Fatalf("unsupported arch returned: %s", ver.Arch)
		}
	}
}

func TestFetchVersionsHandlesHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithCacheTTL(time.Minute),
	)

	if _, err := client.FetchVersions(); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestFetchVersionsUsesCache(t *testing.T) {
	t.Parallel()

	hitCount := 0
	releases := []release{{Version: "go1.20", Files: []releaseFile{{Filename: "go1.20.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Checksum: "x", Kind: "archive"}}}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Fatalf("encode test data failed: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithCacheTTL(time.Hour),
	)

	for i := 0; i < 2; i++ {
		versions, err := client.FetchVersions()
		if err != nil {
			t.Fatalf("FetchVersions error: %v", err)
		}
		if len(versions) != 1 {
			t.Fatalf("unexpected length: %d", len(versions))
		}
	}

	if hitCount != 1 {
		t.Fatalf("expected single upstream hit, got %d", hitCount)
	}
}

func TestCompareVersionStrings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want int
	}{
		{"go1.21.1", "go1.21", 1},
		{"go1.21", "go1.21rc1", 1},
		{"go1.21rc2", "go1.21rc1", 1},
		{"go1.21beta1", "go1.21rc1", -1},
		{"go1.20.5", "go1.21beta1", -1},
		{"go1.20", "go1.20", 0},
	}

	for _, tc := range cases {
		if got := compareVersionStrings(tc.a, tc.b); got != tc.want {
			t.Fatalf("compareVersionStrings(%s,%s)=%d want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFetchVersionsUsesCustomDownloadBase(t *testing.T) {
	t.Parallel()

	releases := []release{
		{
			Version: "go1.21.0",
			Files: []releaseFile{
				{Filename: "go1.21.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Checksum: "sum", Kind: "archive"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Fatalf("encode test data failed: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithDownloadBase("https://mirror.example.com/go"),
	)

	versions, err := client.FetchVersions()
	if err != nil {
		t.Fatalf("FetchVersions error: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("unexpected length: %d", len(versions))
	}

	want := "https://mirror.example.com/go/go1.21.0.linux-amd64.tar.gz"
	if versions[0].DownloadURL != want {
		t.Fatalf("unexpected download url: got %s want %s", versions[0].DownloadURL, want)
	}
}

// compile-time检查，确保 Client 满足 RemoteClient 接口
var _ RemoteClient = (*Client)(nil)
