package version

import (
	"strings"
	"testing"

	"github.com/liangyou/govm/internal/remote"
	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

type fakeRemoteClient struct {
	versions []models.Version
	err      error
}

func (f *fakeRemoteClient) FetchVersions() ([]models.Version, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.versions, nil
}

func TestRemoteVersionsPassThrough(t *testing.T) {
	t.Parallel()

	rc := &fakeRemoteClient{versions: []models.Version{{Number: "1.21.0"}}}
	lister := NewLister(rc, nil)

	versions, err := lister.RemoteVersions()
	if err != nil {
		t.Fatalf("RemoteVersions err: %v", err)
	}
	if len(versions) != 1 || versions[0].Number != "1.21.0" {
		t.Fatalf("unexpected versions: %#v", versions)
	}
}

type fakeStorage struct {
	versions []models.Version
	current  string
	err      error
}

func (f *fakeStorage) SaveMetadata(models.Version) error        { return nil }
func (f *fakeStorage) LoadMetadata() ([]models.Version, error)  { return f.versions, f.err }
func (f *fakeStorage) DeleteMetadata(string) error              { return nil }
func (f *fakeStorage) GetInstallPath(version string) string     { return "/opt/go" + version }
func (f *fakeStorage) GetCurrentVersionMarker() (string, error) { return f.current, nil }
func (f *fakeStorage) SetCurrentVersionMarker(string) error     { return nil }

func TestLocalVersionsMarksCurrentAndSorts(t *testing.T) {
	t.Parallel()

	store := &fakeStorage{
		versions: []models.Version{
			{Number: "1.18.0", InstallPath: "/tmp/go1.18.0"},
			{Number: "1.20.2", InstallPath: "/tmp/go1.20.2"},
		},
		current: "1.18.0",
	}

	lister := NewLister(nil, store)
	versions, err := lister.LocalVersions()
	if err != nil {
		t.Fatalf("LocalVersions err: %v", err)
	}

	if versions[0].Number != "1.20.2" {
		t.Fatalf("expected descending order, got %#v", versions)
	}
	if !versions[1].IsCurrent {
		t.Fatalf("expected current flag on version 1.18.0: %#v", versions)
	}
}

func TestCurrentVersionValidatesExecutable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := &fakeStorage{
		versions: []models.Version{{Number: "1.21.0", InstallPath: root}},
		current:  "1.21.0",
	}

	lister := NewLister(&fakeRemoteClient{}, store)

	if _, err := lister.CurrentVersion(); err == nil {
		t.Fatal("expected error for missing go binary")
	}
}

func TestFormatRemoteVersion(t *testing.T) {
	v := models.Version{Number: "1.22.0", FullName: "go1.22.0", OS: "linux", Arch: "amd64"}
	out := FormatRemoteVersion(v)
	if !strings.Contains(out, "go1.22.0") || !strings.Contains(out, "linux/amd64") {
		t.Fatalf("remote format missing fields: %s", out)
	}
}

func TestFormatLocalVersion(t *testing.T) {
	v := models.Version{Number: "1.20.0", InstallPath: "/tmp/go1.20.0", IsCurrent: true}
	out := FormatLocalVersion(v)
	if !strings.Contains(out, "*") || !strings.Contains(out, "/tmp/go1.20.0") {
		t.Fatalf("local format missing info: %s", out)
	}
}

var _ remote.RemoteClient = (*fakeRemoteClient)(nil)
var _ storage.LocalStorage = (*fakeStorage)(nil)
