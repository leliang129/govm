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

type integrationDownloader struct {
	path string
}

func (d *integrationDownloader) Download(models.Version) (string, error) {
	return d.path, nil
}

type integrationEnvManager struct {
	versions []string
	roots    []string
}

func (e *integrationEnvManager) SetCurrentVersion(version string) error {
	e.versions = append(e.versions, version)
	return nil
}

func (e *integrationEnvManager) ConfigureEnvironment(goRoot string) error {
	e.roots = append(e.roots, goRoot)
	return nil
}

func (e *integrationEnvManager) DetectShell() (string, error) {
	return "bash", nil
}

func (e *integrationEnvManager) UpdateShellConfig(shellType, goRoot string) error {
	return nil
}

func TestIntegrationInstallUseUninstall(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	cfg := models.Config{RootDir: temp, VersionsDir: filepath.Join(temp, "versions")}
	store := storage.NewFileStorage(cfg)

	archive := createIntegrationArchive(t, map[string]string{
		"bin/go":    "binary",
		"bin/gofmt": "fmt",
	})

	downloader := &integrationDownloader{path: archive}
	installer := NewInstaller(store, downloader)

	version := models.Version{
		Number:   "1.22.0",
		FullName: "go1.22.0",
		FileName: "go1.22.0.linux-amd64.tar.gz",
	}

	if err := installer.Install(version); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	envMgr := &integrationEnvManager{}
	switcher := NewSwitcher(store, envMgr)
	if err := switcher.UseVersion("1.22.0"); err != nil {
		t.Fatalf("UseVersion failed: %v", err)
	}

	if len(envMgr.versions) != 1 || envMgr.versions[0] != "1.22.0" {
		t.Fatalf("current version not recorded: %#v", envMgr.versions)
	}

	uninstaller := NewUninstaller(store)
	if _, err := uninstaller.Uninstall("1.22.0", true); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	installPath := store.GetInstallPath("1.22.0")
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		t.Fatalf("install path still exists: %v", err)
	}

	remaining, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no metadata, got %#v", remaining)
	}
}

func createIntegrationArchive(t *testing.T, files map[string]string) string {
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
		ensureIntegrationDirs(t, tw, rel, dirs)
		writeIntegrationFile(t, tw, rel, content)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return pathOnDisk
}

func ensureIntegrationDirs(t *testing.T, tw *tar.Writer, rel string, seen map[string]struct{}) {
	t.Helper()
	parent := path.Dir(rel)
	if parent == "." || parent == "" {
		parent = ""
	}
	if parent == "" {
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
		writeIntegrationDir(t, tw, prefix)
	}
}

func writeIntegrationDir(t *testing.T, tw *tar.Writer, dir string) {
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

func writeIntegrationFile(t *testing.T, tw *tar.Writer, rel, content string) {
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
