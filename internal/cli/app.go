package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/liangyou/govm/internal/version"
	"github.com/liangyou/govm/pkg/models"
)

// ListService 描述版本查询能力。
type ListService interface {
	RemoteVersions() ([]models.Version, error)
	LocalVersions() ([]models.Version, error)
	CurrentVersion() (*models.Version, error)
}

// InstallService 描述安装能力。
type InstallService interface {
	Install(models.Version) error
}

// SwitchService 描述版本切换能力。
type SwitchService interface {
	UseVersion(string) error
}

// UninstallService 描述卸载能力。
type UninstallService interface {
	Uninstall(version string, force bool) ([]models.Version, error)
}

// App 负责 CLI 命令解析与分发。
type App struct {
	out         io.Writer
	version     string
	lister      ListService
	installer   InstallService
	switcher    SwitchService
	uninstaller UninstallService
}

// NewApp 创建 CLI 应用实例。
func NewApp(out io.Writer, lister ListService, installer InstallService, switcher SwitchService, uninstaller UninstallService, version string) *App {
	if out == nil {
		out = os.Stdout
	}
	return &App{
		out:         out,
		version:     version,
		lister:      lister,
		installer:   installer,
		switcher:    switcher,
		uninstaller: uninstaller,
	}
}

// Run 解析参数并执行命令。
func (a *App) Run(args []string) error {
	fs := flag.NewFlagSet("govm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	remoteFlg := fs.Bool("remote", false, "list remote versions")
	listFlg := fs.Bool("list", false, "list local versions")
	helpFlg := fs.Bool("help", false, "show help")
	versionFlg := fs.Bool("version", false, "show version")

	if err := fs.Parse(args); err != nil {
		return err
	}

	switch {
	case *helpFlg:
		a.printHelp()
		return nil
	case *versionFlg:
		fmt.Fprintf(a.out, "govm version %s\n", a.version)
		return nil
	case *remoteFlg:
		return a.handleRemote()
	case *listFlg:
		return a.handleList()
	}

	rest := fs.Args()
	if len(rest) == 0 {
		a.printHelp()
		return nil
	}

	switch rest[0] {
	case "install":
		if len(rest) < 2 {
			return errors.New("install command requires a version")
		}
		return a.handleInstall(rest[1])
	case "use":
		if len(rest) < 2 {
			return errors.New("use command requires a version")
		}
		return a.handleUse(rest[1])
	case "current":
		return a.handleCurrent()
	case "uninstall":
		if len(rest) < 2 {
			return errors.New("uninstall command requires a version")
		}
		force := len(rest) > 2 && rest[2] == "--force"
		return a.handleUninstall(rest[1], force)
	default:
		return fmt.Errorf("unknown command: %s", rest[0])
	}
}

func (a *App) handleRemote() error {
	if a.lister == nil {
		return errors.New("remote listing is unavailable")
	}
	versions, err := a.lister.RemoteVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		fmt.Fprintln(a.out, "No remote versions available.")
		return nil
	}
	fmt.Fprintln(a.out, "Remote versions:")
	for _, v := range versions {
		fmt.Fprintf(a.out, "  %s\n", version.FormatRemoteVersion(v))
	}
	return nil
}

func (a *App) handleList() error {
	if a.lister == nil {
		return errors.New("local listing is unavailable")
	}
	versions, err := a.lister.LocalVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		fmt.Fprintln(a.out, "No versions installed.")
		return nil
	}
	fmt.Fprintln(a.out, "Installed versions:")
	for _, v := range versions {
		fmt.Fprintf(a.out, "  %s\n", version.FormatLocalVersion(v))
	}
	return nil
}

func (a *App) handleCurrent() error {
	if a.lister == nil {
		return errors.New("current version query is unavailable")
	}
	current, err := a.lister.CurrentVersion()
	if err != nil {
		return err
	}
	if current == nil {
		fmt.Fprintln(a.out, "No active Go version.")
		return nil
	}
	fmt.Fprintf(a.out, "Current version: %s\n", version.FormatLocalVersion(*current))
	return nil
}

func (a *App) handleInstall(input string) error {
	if a.installer == nil || a.lister == nil {
		return errors.New("install command is unavailable")
	}
	normalized := normalizeVersion(input)
	versions, err := a.lister.RemoteVersions()
	if err != nil {
		return err
	}
	target, err := findVersion(versions, normalized)
	if err != nil {
		return err
	}
	if err := a.installer.Install(*target); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "Installed %s\n", target.FullName)
	return nil
}

func (a *App) handleUse(ver string) error {
	if a.switcher == nil {
		return errors.New("use command is unavailable")
	}
	normalized := normalizeVersion(ver)
	if err := a.switcher.UseVersion(normalized); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "Now using go%s\n", normalized)
	return nil
}

func (a *App) handleUninstall(ver string, force bool) error {
	if a.uninstaller == nil || a.lister == nil {
		return errors.New("uninstall command is unavailable")
	}
	normalized := normalizeVersion(ver)
	if _, err := a.uninstaller.Uninstall(normalized, force); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "Uninstalled go%s\n", normalized)
	versions, err := a.lister.LocalVersions()
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, "Remaining versions:")
	if len(versions) == 0 {
		fmt.Fprintln(a.out, "  (none)")
		return nil
	}
	for _, v := range versions {
		fmt.Fprintf(a.out, "  %s\n", version.FormatLocalVersion(v))
	}
	return nil
}

func (a *App) printHelp() {
	fmt.Fprintln(a.out, `govm - Go version manager

Commands:
  govm -remote              List remote versions
  govm -list                List installed versions
  govm install <version>    Install a specific version
  govm use <version>        Switch to an installed version
  govm current              Show the active version
  govm uninstall <version> [--force]  Remove an installed version
  govm -help                Show this message
  govm -version             Show govm version`)
}

func normalizeVersion(input string) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.TrimPrefix(cleaned, "go")
	return cleaned
}

func findVersion(versions []models.Version, number string) (*models.Version, error) {
	for i := range versions {
		if versions[i].Number == number {
			return &versions[i], nil
		}
	}
	return nil, fmt.Errorf("version %s not found in remote list", number)
}
