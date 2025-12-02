package version

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

// Uninstaller 删除本地已安装的 Go 版本。
type Uninstaller struct {
	storage storage.LocalStorage
}

// NewUninstaller 创建卸载器。
func NewUninstaller(store storage.LocalStorage) *Uninstaller {
	return &Uninstaller{storage: store}
}

// Uninstall 删除指定版本。当 force=true 时允许卸载当前版本。
func (u *Uninstaller) Uninstall(version string, force bool) ([]models.Version, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, errors.New("uninstaller: version is required")
	}
	if u.storage == nil {
		return nil, errors.New("uninstaller: storage is required")
	}

	versions, err := u.storage.LoadMetadata()
	if err != nil {
		return nil, fmt.Errorf("uninstaller: load metadata: %w", err)
	}

	var target *models.Version
	for i := range versions {
		if versions[i].Number == version {
			target = &versions[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("uninstaller: version %s not installed", version)
	}

	current, err := u.storage.GetCurrentVersionMarker()
	if err != nil {
		return nil, fmt.Errorf("uninstaller: read current marker: %w", err)
	}
	if current == target.Number && !force {
		return nil, fmt.Errorf("uninstaller: version %s is active, pass force to remove", version)
	}

	if target.InstallPath != "" {
		if err := os.RemoveAll(target.InstallPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("uninstaller: remove dir: %w", err)
		}
	}

	if err := u.storage.DeleteMetadata(target.Number); err != nil {
		return nil, fmt.Errorf("uninstaller: delete metadata: %w", err)
	}

	if current == target.Number {
		if err := u.storage.SetCurrentVersionMarker(""); err != nil {
			return nil, fmt.Errorf("uninstaller: clear current marker: %w", err)
		}
	}

	remaining, err := u.storage.LoadMetadata()
	if err != nil {
		return nil, fmt.Errorf("uninstaller: reload metadata: %w", err)
	}

	return remaining, nil
}
