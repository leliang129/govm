package version

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

// ArtifactDownloader 用于获取远程 Go 发行版的压缩包。
type ArtifactDownloader interface {
	Download(models.Version) (string, error)
}

// Installer 负责将下载好的 Go 版本安装到本地。
type Installer struct {
	storage    storage.LocalStorage
	downloader ArtifactDownloader
	now        func() time.Time
}

// NewInstaller 创建 Installer。
func NewInstaller(store storage.LocalStorage, downloader ArtifactDownloader) *Installer {
	return &Installer{
		storage:    store,
		downloader: downloader,
		now:        time.Now,
	}
}

// Install 执行完整的安装流程，满足需求 3 的验收标准。
func (i *Installer) Install(version models.Version) error {
	if i.storage == nil || i.downloader == nil {
		return errors.New("installer: missing dependencies")
	}

	installed, err := i.isVersionInstalled(version.Number)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	installPath := i.storage.GetInstallPath(version.Number)
	if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
		return fmt.Errorf("installer: prepare parent dir: %w", err)
	}

	archivePath, err := i.downloader.Download(version)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(installPath), "install-*")
	if err != nil {
		return fmt.Errorf("installer: create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	destDir := filepath.Join(tempDir, "root")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("installer: prepare extract dir: %w", err)
	}

	if err := extractTarGz(archivePath, destDir); err != nil {
		return err
	}

	if err := os.RemoveAll(installPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("installer: cleanup previous install: %w", err)
	}

	if err := os.Rename(destDir, installPath); err != nil {
		return fmt.Errorf("installer: move install directory: %w", err)
	}

	version.InstallPath = installPath
	version.InstalledAt = i.now().UTC()

	if err := i.storage.SaveMetadata(version); err != nil {
		return fmt.Errorf("installer: save metadata: %w", err)
	}

	return nil
}

func (i *Installer) isVersionInstalled(version string) (bool, error) {
	versions, err := i.storage.LoadMetadata()
	if err != nil {
		return false, fmt.Errorf("installer: load metadata: %w", err)
	}

	for _, v := range versions {
		if v.Number != version {
			continue
		}
		if v.InstallPath == "" {
			return false, nil
		}
		if info, err := os.Stat(v.InstallPath); err == nil && info.IsDir() {
			return true, nil
		}
		return false, nil
	}

	return false, nil
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("installer: open archive: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("installer: gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("installer: read archive: %w", err)
		}

		relPath, skip := normalizeTarPath(header.Name)
		if skip {
			continue
		}

		target := filepath.Join(dest, relPath)
		if err := ensureWithinRoot(dest, target); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("installer: mkdir %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("installer: mkdir for file %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("installer: create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("installer: copy file %s: %w", target, err)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("installer: symlink %s: %w", target, err)
			}
		default:
			return fmt.Errorf("installer: unsupported tar entry %q", header.Name)
		}
	}

	return nil
}

func normalizeTarPath(name string) (string, bool) {
	clean := path.Clean(name)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "go" || clean == "." || clean == "" {
		return "", true
	}
	if strings.HasPrefix(clean, "go/") {
		clean = strings.TrimPrefix(clean, "go/")
	} else {
		return "", true
	}
	if clean == "" {
		return "", true
	}
	return clean, false
}

func ensureWithinRoot(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root {
		return nil
	}
	if !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return fmt.Errorf("installer: illegal path %s", target)
	}
	return nil
}
