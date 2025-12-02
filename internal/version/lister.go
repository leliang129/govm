package version

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/liangyou/govm/internal/remote"
	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/pkg/models"
)

// Lister 聚合远程与本地版本信息。
type Lister struct {
	remote  remote.RemoteClient
	storage storage.LocalStorage
}

// NewLister 创建版本列表服务。
func NewLister(remoteClient remote.RemoteClient, store storage.LocalStorage) *Lister {
	return &Lister{remote: remoteClient, storage: store}
}

// RemoteVersions 返回远程版本并格式化。
func (l *Lister) RemoteVersions() ([]models.Version, error) {
	if l.remote == nil {
		return nil, fmt.Errorf("lister: remote client is required")
	}
	versions, err := l.remote.FetchVersions()
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// LocalVersions 返回本地安装版本，标记当前版本。
func (l *Lister) LocalVersions() ([]models.Version, error) {
	if l.storage == nil {
		return nil, fmt.Errorf("lister: storage is required")
	}
	versions, err := l.storage.LoadMetadata()
	if err != nil {
		return nil, fmt.Errorf("lister: load metadata: %w", err)
	}

	current, err := l.storage.GetCurrentVersionMarker()
	if err != nil {
		return nil, fmt.Errorf("lister: current marker: %w", err)
	}

	for i := range versions {
		versions[i].IsCurrent = versions[i].Number == current
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return compareLocalVersions(versions[i].Number, versions[j].Number) > 0
	})

	return versions, nil
}

// CurrentVersion 返回当前激活版本。
func (l *Lister) CurrentVersion() (*models.Version, error) {
	versions, err := l.LocalVersions()
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.IsCurrent {
			if err := validateExecutable(v.InstallPath); err != nil {
				return nil, err
			}
			return &v, nil
		}
	}
	return nil, nil
}

func validateExecutable(goRoot string) error {
	root := strings.TrimSpace(goRoot)
	if root == "" {
		return fmt.Errorf("lister: current version missing install path")
	}
	goBin := filepath.Join(root, "bin", "go")
	info, err := os.Stat(goBin)
	if err != nil {
		return fmt.Errorf("lister: go binary missing: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("lister: go binary path is directory: %s", goBin)
	}
	return nil
}

// FormatRemoteVersion 格式化远程版本输出，包含版本号与架构信息。
func FormatRemoteVersion(v models.Version) string {
	name := v.FullName
	if name == "" {
		name = "go" + v.Number
	}
	return fmt.Sprintf("%s (%s/%s)", name, v.OS, v.Arch)
}

// FormatLocalVersion 格式化本地版本输出，包含安装路径并标记当前版本。
func FormatLocalVersion(v models.Version) string {
	marker := " "
	if v.IsCurrent {
		marker = "*"
	}
	pathInfo := v.InstallPath
	if pathInfo == "" {
		pathInfo = "(unknown path)"
	}
	name := v.FullName
	if name == "" {
		name = "go" + v.Number
	}
	return fmt.Sprintf("%s %s - %s", marker, name, pathInfo)
}

func compareLocalVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	max := len(ap)
	if len(bp) > max {
		max = len(bp)
	}
	for i := 0; i < max; i++ {
		ai := 0
		if i < len(ap) {
			ai = parseInt(ap[i])
		}
		bi := 0
		if i < len(bp) {
			bi = parseInt(bp[i])
		}
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}
	return 0
}

func parseInt(value string) int {
	var n int
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
