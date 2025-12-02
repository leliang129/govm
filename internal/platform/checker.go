package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/liangyou/govm/pkg/models"
)

var supportedArch = map[string]struct{}{
	"amd64": {},
	"arm64": {},
	"386":   {},
}

// Checker 校验当前系统是否满足 govm 的运行要求。
type Checker struct {
	cfg    models.Config
	goos   func() string
	goarch func() string
}

// NewChecker 创建平台检测器。
func NewChecker(cfg models.Config) *Checker {
	return &Checker{
		cfg:    cfg,
		goos:   func() string { return runtime.GOOS },
		goarch: func() string { return runtime.GOARCH },
	}
}

// Validate 校验当前平台与安装目录权限。
func (c *Checker) Validate() error {
	if c.goos() != "linux" {
		return fmt.Errorf("platform: unsupported operating system %s", c.goos())
	}
	if _, ok := supportedArch[c.goarch()]; !ok {
		return fmt.Errorf("platform: unsupported architecture %s", c.goarch())
	}

	root := c.resolveRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("platform: cannot access install directory %s: %w", root, err)
	}
	return nil
}

func (c *Checker) resolveRoot() string {
	if c.cfg.RootDir != "" {
		return c.cfg.RootDir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".govm")
	}
	return filepath.Join(os.TempDir(), "govm")
}
