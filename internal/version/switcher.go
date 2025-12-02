package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liangyou/govm/internal/env"
	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

// Switcher 负责切换当前使用的 Go 版本。
type Switcher struct {
	storage storage.LocalStorage
	env     env.EnvManager
}

// NewSwitcher 创建 Switcher。
func NewSwitcher(store storage.LocalStorage, envManager env.EnvManager) *Switcher {
	return &Switcher{storage: store, env: envManager}
}

// UseVersion 将指定版本设置为当前版本。
func (s *Switcher) UseVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("switcher: version is required")
	}
	if s.storage == nil || s.env == nil {
		return fmt.Errorf("switcher: missing dependencies")
	}

	versions, err := s.storage.LoadMetadata()
	if err != nil {
		return fmt.Errorf("switcher: load metadata: %w", err)
	}

	var target *models.Version
	for i := range versions {
		if versions[i].Number == version {
			target = &versions[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("switcher: version %s not installed", version)
	}
	if target.InstallPath == "" {
		return fmt.Errorf("switcher: version %s missing install path", version)
	}

	if err := s.ensureExecutable(target.InstallPath); err != nil {
		return err
	}

	if err := s.env.ConfigureEnvironment(target.InstallPath); err != nil {
		return fmt.Errorf("switcher: configure environment: %w", err)
	}

	if err := s.env.SetCurrentVersion(target.Number); err != nil {
		return fmt.Errorf("switcher: set current version: %w", err)
	}

	for _, ver := range versions {
		ver.IsCurrent = ver.Number == target.Number
		if err := s.storage.SaveMetadata(ver); err != nil {
			return fmt.Errorf("switcher: update metadata: %w", err)
		}
	}

	return nil
}

func (s *Switcher) ensureExecutable(goRoot string) error {
	goBin := filepath.Join(goRoot, "bin", "go")
	info, err := os.Stat(goBin)
	if err != nil {
		return fmt.Errorf("switcher: go binary missing: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("switcher: go binary path is directory: %s", goBin)
	}
	return nil
}
