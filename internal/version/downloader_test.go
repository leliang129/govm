package version

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/liangyou/govm/pkg/models"
)

func TestDownloaderDownloadSuccess(t *testing.T) {
	t.Parallel()

	payload := []byte("hello govm")
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cfg := models.Config{RootDir: t.TempDir()}
	downloadsDir := filepath.Join(cfg.RootDir, "downloads")
	var lastProgress int64
	dl := NewDownloader(
		cfg,
		WithHTTPClient(server.Client()),
		WithDownloadsDir(downloadsDir),
		WithProgressFunc(func(done, total int64) {
			atomic.StoreInt64(&lastProgress, done)
		}),
	)

	version := models.Version{
		DownloadURL: server.URL,
		FileName:    "go1.21.0.linux-amd64.tar.gz",
		Checksum:    checksum,
	}

	path, err := dl.Download(version)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if path != filepath.Join(downloadsDir, version.FileName) {
		t.Fatalf("unexpected path: %s", path)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}

	if got := atomic.LoadInt64(&lastProgress); got != int64(len(payload)) {
		t.Fatalf("unexpected progress: %d", got)
	}
}

func TestDownloaderChecksumMismatch(t *testing.T) {
	t.Parallel()

	payload := []byte("bad sum")
	wrongChecksum := "0000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cfg := models.Config{RootDir: t.TempDir()}
	dl := NewDownloader(cfg, WithHTTPClient(server.Client()))

	version := models.Version{
		DownloadURL: server.URL,
		FileName:    "go1.20.linux-amd64.tar.gz",
		Checksum:    wrongChecksum,
	}

	if _, err := dl.Download(version); err == nil {
		t.Fatal("expected checksum mismatch error")
	}

	finalPath := filepath.Join(cfg.RootDir, "downloads", version.FileName)
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatalf("expected no file at %s", finalPath)
	}
}

func TestDownloaderHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := models.Config{RootDir: t.TempDir()}
	dl := NewDownloader(cfg, WithHTTPClient(server.Client()))

	version := models.Version{DownloadURL: server.URL, FileName: "go.tgz", Checksum: "abcd"}

	if _, err := dl.Download(version); err == nil {
		t.Fatal("expected http error")
	}
}
