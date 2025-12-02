package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liangyou/govm/pkg/models"
)

func TestCheckerValidateSupportedPlatform(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	cfg := models.Config{RootDir: filepath.Join(temp, "govm")}

	checker := NewChecker(cfg)
	checker.goos = func() string { return "linux" }
	checker.goarch = func() string { return "amd64" }

	if err := checker.Validate(); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestCheckerUnsupportedOS(t *testing.T) {
	t.Parallel()

	checker := NewChecker(models.Config{})
	checker.goos = func() string { return "darwin" }

	if err := checker.Validate(); err == nil {
		t.Fatal("expected error for unsupported os")
	}
}

func TestCheckerUnsupportedArch(t *testing.T) {
	t.Parallel()

	checker := NewChecker(models.Config{})
	checker.goos = func() string { return "linux" }
	checker.goarch = func() string { return "sparc" }

	if err := checker.Validate(); err == nil {
		t.Fatal("expected error for unsupported arch")
	}
}

func TestCheckerPermissionError(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	filePath := filepath.Join(temp, "file")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	checker := NewChecker(models.Config{RootDir: filePath})
	checker.goos = func() string { return "linux" }
	checker.goarch = func() string { return "amd64" }

	if err := checker.Validate(); err == nil {
		t.Fatal("expected error due to invalid directory")
	}
}
