package env

import (
	"os"
	"strings"
	"testing"

	"github.com/liangyou/govm/pkg/models"
)

type stubStorage struct {
	version string
}

func (s *stubStorage) SaveMetadata(models.Version) error            { return nil }
func (s *stubStorage) LoadMetadata() ([]models.Version, error)      { return nil, nil }
func (s *stubStorage) DeleteMetadata(string) error                  { return nil }
func (s *stubStorage) GetInstallPath(string) string                 { return "" }
func (s *stubStorage) GetCurrentVersionMarker() (string, error)     { return s.version, nil }
func (s *stubStorage) SetCurrentVersionMarker(version string) error { s.version = version; return nil }

func TestConfigFileSelection(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	mgr := NewManager(&stubStorage{}, models.Config{})
	mgr.homeFn = func() (string, error) { return temp, nil }

	bashFile, err := mgr.configFileForShell("bash")
	if err != nil {
		t.Fatalf("configFileForShell bash err: %v", err)
	}
	if !strings.HasSuffix(bashFile, ".bashrc") && !strings.HasSuffix(bashFile, ".bash_profile") {
		t.Fatalf("bash config file invalid: %s", bashFile)
	}

	zshFile, err := mgr.configFileForShell("zsh")
	if err != nil {
		t.Fatalf("configFileForShell zsh err: %v", err)
	}
	if !strings.HasSuffix(zshFile, ".zshrc") {
		t.Fatalf("zsh config file invalid: %s", zshFile)
	}
}

func TestUpdateShellConfigCreatesAndReplacesBlock(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	mgr := NewManager(&stubStorage{}, models.Config{})
	mgr.homeFn = func() (string, error) { return temp, nil }

	configPath, err := mgr.configFileForShell("bash")
	if err != nil {
		t.Fatalf("configFileForShell err: %v", err)
	}

	goRoot := "/tmp/go"
	if err := mgr.UpdateShellConfig("bash", goRoot); err != nil {
		t.Fatalf("UpdateShellConfig failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read bashrc: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, goRoot) {
		t.Fatalf("config block missing GOROOT: %s", content)
	}

	newRoot := "/opt/go"
	if err := mgr.UpdateShellConfig("bash", newRoot); err != nil {
		t.Fatalf("UpdateShellConfig second run failed: %v", err)
	}

	data, _ = os.ReadFile(configPath)
	if strings.Count(string(data), blockStart) != 1 {
		t.Fatalf("expected single config block, got %d", strings.Count(string(data), blockStart))
	}
	if !strings.Contains(string(data), newRoot) {
		t.Fatalf("config not updated: %s", string(data))
	}
}

func TestDetectShell(t *testing.T) {
	t.Parallel()

	mgr := NewManager(&stubStorage{}, models.Config{})
	mgr.envFn = func(key string) string {
		if key == "SHELL" {
			return "/bin/zsh"
		}
		return ""
	}

	shell, err := mgr.DetectShell()
	if err != nil {
		t.Fatalf("DetectShell error: %v", err)
	}
	if shell != "zsh" {
		t.Fatalf("expected zsh, got %s", shell)
	}
}
