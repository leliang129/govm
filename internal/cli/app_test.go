package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/liangyou/govm/pkg/models"
)

type fakeLister struct {
	remote     []models.Version
	local      []models.Version
	current    *models.Version
	remoteErr  error
	localErr   error
	currentErr error
}

func (f *fakeLister) RemoteVersions() ([]models.Version, error) {
	return f.remote, f.remoteErr
}

func (f *fakeLister) LocalVersions() ([]models.Version, error) {
	return f.local, f.localErr
}

func (f *fakeLister) CurrentVersion() (*models.Version, error) {
	return f.current, f.currentErr
}

type fakeInstaller struct {
	installed []models.Version
	err       error
}

func (f *fakeInstaller) Install(v models.Version) error {
	if f.err != nil {
		return f.err
	}
	f.installed = append(f.installed, v)
	return nil
}

type fakeSwitcher struct {
	used []string
	err  error
}

func (f *fakeSwitcher) UseVersion(version string) error {
	if f.err != nil {
		return f.err
	}
	f.used = append(f.used, version)
	return nil
}

type fakeUninstaller struct {
	removed []string
	forced  []bool
	err     error
}

func (f *fakeUninstaller) Uninstall(version string, force bool) ([]models.Version, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.removed = append(f.removed, version)
	f.forced = append(f.forced, force)
	return []models.Version{}, nil
}

func TestAppRemoteList(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	lister := &fakeLister{remote: []models.Version{{FullName: "go1.21.0", Number: "1.21.0", OS: "linux", Arch: "amd64"}}}
	app := NewApp(buf, lister, &fakeInstaller{}, &fakeSwitcher{}, &fakeUninstaller{}, "test")

	if err := app.Run([]string{"-remote"}); err != nil {
		t.Fatalf("run -remote: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Remote versions:") || !strings.Contains(output, "go1.21.0") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestAppInstallUsesRemoteVersion(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	installs := &fakeInstaller{}
	lister := &fakeLister{remote: []models.Version{{Number: "1.20.3", FullName: "go1.20.3"}}}
	app := NewApp(buf, lister, installs, &fakeSwitcher{}, &fakeUninstaller{}, "test")

	if err := app.Run([]string{"install", "1.20.3"}); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	if len(installs.installed) != 1 || installs.installed[0].Number != "1.20.3" {
		t.Fatalf("installer not invoked properly: %#v", installs.installed)
	}
}

func TestAppUninstallRequiresForce(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	u := &fakeUninstaller{}
	lister := &fakeLister{local: []models.Version{}}
	app := NewApp(buf, lister, &fakeInstaller{}, &fakeSwitcher{}, u, "test")

	if err := app.Run([]string{"uninstall", "1.18", "--force"}); err != nil {
		t.Fatalf("uninstall with force failed: %v", err)
	}

	if len(u.removed) != 1 || u.removed[0] != "1.18" || !u.forced[0] {
		t.Fatalf("uninstaller not invoked with force: %#v %#v", u.removed, u.forced)
	}
}

func TestAppUninstallFlag(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	u := &fakeUninstaller{}
	lister := &fakeLister{local: []models.Version{}}
	app := NewApp(buf, lister, &fakeInstaller{}, &fakeSwitcher{}, u, "test")

	if err := app.Run([]string{"-uninstall", "1.19"}); err != nil {
		t.Fatalf("flag uninstall failed: %v", err)
	}
	if len(u.removed) != 1 || u.removed[0] != "1.19" || len(u.forced) == 0 || u.forced[0] {
		t.Fatalf("flag uninstall not recorded: removed=%v forced=%v", u.removed, u.forced)
	}

	if err := app.Run([]string{"-uninstall", "1.20", "-force"}); err != nil {
		t.Fatalf("flag uninstall with force failed: %v", err)
	}
	if len(u.removed) != 2 || u.removed[1] != "1.20" || !u.forced[1] {
		t.Fatalf("flag uninstall force not recorded: removed=%v forced=%v", u.removed, u.forced)
	}
}
