package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

type fakeEnvManager struct {
	configuredRoots []string
	currentVersion  string
	configureErr    error
	setErr          error
}

func (f *fakeEnvManager) SetCurrentVersion(version string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.currentVersion = version
	return nil
}

func (f *fakeEnvManager) ConfigureEnvironment(goRoot string) error {
	if f.configureErr != nil {
		return f.configureErr
	}
	f.configuredRoots = append(f.configuredRoots, goRoot)
	return nil
}

func (f *fakeEnvManager) DetectShell() (string, error) {
	return "bash", nil
}

func (f *fakeEnvManager) UpdateShellConfig(shellType, goRoot string) error {
	return nil
}

func TestSwitcherUseVersionUpdatesEnvironmentAndMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := models.Config{RootDir: root, VersionsDir: filepath.Join(root, "versions")}
	store := storage.NewFileStorage(cfg)

	version := models.Version{
		Number:      "1.21.0",
		FullName:    "go1.21.0",
		InstallPath: store.GetInstallPath("1.21.0"),
	}

	if err := os.MkdirAll(filepath.Join(version.InstallPath, "bin"), 0o755); err != nil {
		t.Fatalf("create bin: %v", err)
	}
	goBinary := filepath.Join(version.InstallPath, "bin", "go")
	if err := os.WriteFile(goBinary, []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatalf("write go binary: %v", err)
	}

	if err := store.SaveMetadata(version); err != nil {
		t.Fatalf("SaveMetadata err: %v", err)
	}
	if err := store.SaveMetadata(models.Version{Number: "1.20.0", InstallPath: store.GetInstallPath("1.20.0"), IsCurrent: true}); err != nil {
		t.Fatalf("save other metadata: %v", err)
	}

	envManager := &fakeEnvManager{}
	switcher := NewSwitcher(store, envManager)

	if err := switcher.UseVersion("1.21.0"); err != nil {
		t.Fatalf("UseVersion failed: %v", err)
	}

	if envManager.currentVersion != "1.21.0" {
		t.Fatalf("current version mismatch: %s", envManager.currentVersion)
	}
	if len(envManager.configuredRoots) != 1 || envManager.configuredRoots[0] != version.InstallPath {
		t.Fatalf("env not configured properly: %#v", envManager.configuredRoots)
	}

	meta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata err: %v", err)
	}
	var currentCount int
	for _, v := range meta {
		if v.IsCurrent {
			currentCount++
			if v.Number != "1.21.0" {
				t.Fatalf("unexpected current version: %#v", v)
			}
		}
	}
	if currentCount != 1 {
		t.Fatalf("expected one current version, got %d", currentCount)
	}
}

func TestSwitcherFailsWhenVersionMissing(t *testing.T) {
	t.Parallel()

	store := storage.NewFileStorage(models.Config{RootDir: t.TempDir()})
	switcher := NewSwitcher(store, &fakeEnvManager{})

	if err := switcher.UseVersion("1.99.0"); err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestSwitcherFailsWhenGoBinaryMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := models.Config{RootDir: root}
	store := storage.NewFileStorage(cfg)

	version := models.Version{
		Number:      "1.18.0",
		InstallPath: store.GetInstallPath("1.18.0"),
	}
	if err := store.SaveMetadata(version); err != nil {
		t.Fatalf("SaveMetadata err: %v", err)
	}

	switcher := NewSwitcher(store, &fakeEnvManager{})
	if err := switcher.UseVersion("1.18.0"); err == nil {
		t.Fatal("expected missing binary error")
	}
}
