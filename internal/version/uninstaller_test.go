package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

func TestUninstallRemovesFilesAndMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := models.Config{RootDir: root, VersionsDir: filepath.Join(root, "versions")}
	store := storage.NewFileStorage(cfg)

	version := models.Version{Number: "1.21.0", InstallPath: store.GetInstallPath("1.21.0")}
	if err := os.MkdirAll(version.InstallPath, 0o755); err != nil {
		t.Fatalf("mkdir install path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(version.InstallPath, "bin.go"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := store.SaveMetadata(version); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}

	u := NewUninstaller(store)
	remaining, err := u.Uninstall("1.21.0", false)
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty metadata, got %#v", remaining)
	}
	if _, err := os.Stat(version.InstallPath); !os.IsNotExist(err) {
		t.Fatalf("install path still exists: %v", err)
	}
}

func TestUninstallCurrentVersionNeedsForce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := storage.NewFileStorage(models.Config{RootDir: root})

	version := models.Version{Number: "1.20.0", InstallPath: store.GetInstallPath("1.20.0")}
	if err := os.MkdirAll(version.InstallPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := store.SaveMetadata(version); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if err := store.SetCurrentVersionMarker("1.20.0"); err != nil {
		t.Fatalf("set marker: %v", err)
	}

	u := NewUninstaller(store)
	if _, err := u.Uninstall("1.20.0", false); err == nil {
		t.Fatal("expected error when uninstalling current version without force")
	}

	remaining, err := u.Uninstall("1.20.0", true)
	if err != nil {
		t.Fatalf("forced uninstall failed: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no remaining versions, got %#v", remaining)
	}
}

func TestUninstallNonexistentVersion(t *testing.T) {
	t.Parallel()

	store := storage.NewFileStorage(models.Config{RootDir: t.TempDir()})
	u := NewUninstaller(store)

	if _, err := u.Uninstall("1.99.0", false); err == nil {
		t.Fatal("expected error for missing version")
	}
}
