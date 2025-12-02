package storage

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/liangyou/govm/pkg/models"
)

func TestMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	cfg := models.Config{
		RootDir:     temp,
		VersionsDir: filepath.Join(temp, "versions"),
	}

	store := NewFileStorage(cfg)

	version := models.Version{
		Number:      "1.21.0",
		FullName:    "go1.21.0",
		DownloadURL: "https://go.dev/dl/go1.21.0.linux-amd64.tar.gz",
		FileName:    "go1.21.0.linux-amd64.tar.gz",
		Checksum:    "abc123",
		OS:          "linux",
		Arch:        "amd64",
		InstallPath: store.GetInstallPath("1.21.0"),
		IsCurrent:   true,
		InstalledAt: time.Date(2024, time.January, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := store.SaveMetadata(version); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	loaded, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 version, got %d", len(loaded))
	}

	if !reflect.DeepEqual(version, loaded[0]) {
		t.Fatalf("round trip mismatch\nexpected: %#v\nactual: %#v", version, loaded[0])
	}
}

func TestCurrentVersionMarker(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	store := NewFileStorage(models.Config{RootDir: temp})

	version := "1.20.3"
	if err := store.SetCurrentVersionMarker(version); err != nil {
		t.Fatalf("SetCurrentVersionMarker failed: %v", err)
	}

	got, err := store.GetCurrentVersionMarker()
	if err != nil {
		t.Fatalf("GetCurrentVersionMarker failed: %v", err)
	}

	if got != version {
		t.Fatalf("expected marker %s, got %s", version, got)
	}
}

func TestDeleteMetadata(t *testing.T) {
	t.Parallel()

	temp := t.TempDir()
	store := NewFileStorage(models.Config{RootDir: temp})

	v1 := models.Version{Number: "1.19.0"}
	v2 := models.Version{Number: "1.20.0"}

	if err := store.SaveMetadata(v1); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}
	if err := store.SaveMetadata(v2); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	if err := store.DeleteMetadata("1.19.0"); err != nil {
		t.Fatalf("DeleteMetadata failed: %v", err)
	}

	loaded, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Number != "1.20.0" {
		t.Fatalf("unexpected metadata after delete: %#v", loaded)
	}
}
