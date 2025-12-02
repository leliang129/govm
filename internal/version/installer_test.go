package version

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

type stubDownloader struct {
	path  string
	calls int
	fail  error
}

func (s *stubDownloader) Download(models.Version) (string, error) {
	s.calls++
	if s.fail != nil {
		return "", s.fail
	}
	return s.path, nil
}

func TestInstallerInstallAndIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := models.Config{RootDir: root, VersionsDir: filepath.Join(root, "versions")}
	store := storage.NewFileStorage(cfg)

	tarPath := createGoArchive(t, map[string]string{
		"bin/go":    "binary",
		"bin/gofmt": "fmt",
	})

	down := &stubDownloader{path: tarPath}
	installer := NewInstaller(store, down)

	version := models.Version{
		Number:      "1.21.0",
		FullName:    "go1.21.0",
		DownloadURL: "https://example/go1.21.0.tar.gz",
		FileName:    "go1.21.0.tar.gz",
		Checksum:    "checksum",
	}

	if err := installer.Install(version); err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	installPath := store.GetInstallPath("1.21.0")
	if _, err := os.Stat(filepath.Join(installPath, "bin/go")); err != nil {
		t.Fatalf("expected bin/go in %s: %v", installPath, err)
	}

	meta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if len(meta) != 1 || meta[0].Number != "1.21.0" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	if err := installer.Install(version); err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	if down.calls != 1 {
		t.Fatalf("expected downloader called once, got %d", down.calls)
	}
}

func TestInstallerFailureCleansUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := models.Config{RootDir: root, VersionsDir: filepath.Join(root, "versions")}
	store := storage.NewFileStorage(cfg)

	badArchive := createInvalidArchive(t)
	down := &stubDownloader{path: badArchive}
	installer := NewInstaller(store, down)

	version := models.Version{
		Number:   "1.20.0",
		FullName: "go1.20.0",
		FileName: "go1.20.0.tar.gz",
	}

	if err := installer.Install(version); err == nil {
		t.Fatal("expected install to fail for invalid archive")
	}

	installPath := store.GetInstallPath("1.20.0")
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		t.Fatalf("expected no install path, got err=%v", err)
	}

	meta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if len(meta) != 0 {
		t.Fatalf("expected no metadata after failure, got %#v", meta)
	}
}

func createGoArchive(t *testing.T, files map[string]string) string {
	t.Helper()

	pathOnDisk := filepath.Join(t.TempDir(), "go.tar.gz")
	file, err := os.Create(pathOnDisk)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)

	dirs := map[string]struct{}{}

	for rel, content := range files {
		ensureDirs(t, tw, rel, dirs)
		writeFile(t, tw, rel, content)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return pathOnDisk
}

func writeDir(t *testing.T, tw *tar.Writer, dir string) {
	t.Helper()
	hdr := &tar.Header{
		Name:     path.Join("go", dir) + "/",
		Mode:     0o755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write dir header: %v", err)
	}
}

func ensureDirs(t *testing.T, tw *tar.Writer, rel string, seen map[string]struct{}) {
	if rel == "" {
		return
	}
	parent := path.Dir(rel)
	if parent == "." || parent == "" {
		return
	}
	parts := strings.Split(parent, "/")
	var prefix string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if prefix == "" {
			prefix = part
		} else {
			prefix = path.Join(prefix, part)
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		writeDir(t, tw, prefix)
	}
}

func writeFile(t *testing.T, tw *tar.Writer, rel, content string) {
	t.Helper()
	hdr := &tar.Header{
		Name:     path.Join("go", rel),
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write file header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("write file content: %v", err)
	}
}

func createInvalidArchive(t *testing.T) string {
	t.Helper()
	pathOnDisk := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(pathOnDisk, []byte("invalid"), 0o644); err != nil {
		t.Fatalf("write invalid archive: %v", err)
	}
	return pathOnDisk
}
