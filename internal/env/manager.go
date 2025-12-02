package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

const (
	blockStart = "# >>> govm initialize >>>"
	blockEnd   = "# <<< govm initialize <<<"
)

// EnvManager 暴露环境配置能力。
type EnvManager interface {
	SetCurrentVersion(version string) error
	ConfigureEnvironment(goRoot string) error
	DetectShell() (string, error)
	UpdateShellConfig(shellType, goRoot string) error
}

// Manager 实现 EnvManager。
type Manager struct {
	storage storage.LocalStorage
	cfg     models.Config

	homeFn func() (string, error)
	envFn  func(string) string
}

// NewManager 构造环境配置服务。
func NewManager(store storage.LocalStorage, cfg models.Config) *Manager {
	return &Manager{
		storage: store,
		cfg:     cfg,
		homeFn:  os.UserHomeDir,
		envFn:   os.Getenv,
	}
}

// SetCurrentVersion 将版本写入存储标记。
func (m *Manager) SetCurrentVersion(version string) error {
	if m.storage == nil {
		return errors.New("env: storage is required")
	}
	return m.storage.SetCurrentVersionMarker(strings.TrimSpace(version))
}

// ConfigureEnvironment 根据 goRoot 自动检测 shell 并更新配置。
func (m *Manager) ConfigureEnvironment(goRoot string) error {
	shell, err := m.DetectShell()
	if err != nil {
		return err
	}
	return m.UpdateShellConfig(shell, goRoot)
}

// DetectShell 根据 SHELL 环境变量推断当前 shell。
func (m *Manager) DetectShell() (string, error) {
	shellPath := m.envFn("SHELL")
	if shellPath == "" {
		shellPath = "bash"
	}
	shell := filepath.Base(shellPath)
	switch shell {
	case "bash", "zsh":
		return shell, nil
	default:
		return "", fmt.Errorf("env: unsupported shell %q", shell)
	}
}

// UpdateShellConfig 对指定 shell 写入配置块。
func (m *Manager) UpdateShellConfig(shellType, goRoot string) error {
	if goRoot == "" {
		return errors.New("env: goRoot is required")
	}

	configPath, err := m.configFileForShell(shellType)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("env: ensure config dir: %w", err)
	}

	var existing []byte
	if data, err := os.ReadFile(configPath); err == nil {
		existing = data
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("env: read config: %w", err)
	}

	block := m.buildConfigBlock(goRoot)
	merged := mergeConfig(string(existing), block)

	return os.WriteFile(configPath, []byte(merged), 0o644)
}

func (m *Manager) configFileForShell(shellType string) (string, error) {
	home, err := m.homeFn()
	if err != nil {
		return "", fmt.Errorf("env: home dir: %w", err)
	}

	switch shellType {
	case "bash":
		path := filepath.Join(home, ".bashrc")
		if fileExists(path) {
			return path, nil
		}
		profile := filepath.Join(home, ".bash_profile")
		return profile, nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("env: unsupported shell %q", shellType)
	}
}

func (m *Manager) buildConfigBlock(goRoot string) string {
	defaultGopath := m.cfg.GoPath
	if defaultGopath == "" {
		defaultGopath = "$HOME/go"
	}
	lines := []string{
		blockStart,
		fmt.Sprintf("export GOROOT=\"%s\"", goRoot),
		fmt.Sprintf("export GOPATH=\"${GOPATH:-%s}\"", defaultGopath),
		"export PATH=\"$GOROOT/bin:$PATH\"",
		blockEnd,
	}
	return strings.Join(lines, "\n")
}

func mergeConfig(existing, block string) string {
	cleaned := removeExistingBlock(existing)
	cleaned = strings.TrimRight(cleaned, "\n")
	if strings.TrimSpace(cleaned) == "" {
		return block + "\n"
	}
	return cleaned + "\n\n" + block + "\n"
}

func removeExistingBlock(content string) string {
	var builder strings.Builder
	lines := strings.Split(content, "\n")
	skipping := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == blockStart {
			skipping = true
			continue
		}
		if trimmed == blockEnd {
			skipping = false
			continue
		}
		if skipping {
			continue
		}
		if line == "" && builder.Len() == 0 {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line)
	}
	result := builder.String()
	return strings.Trim(result, "\n")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
