package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/liangyou/govm/pkg/models"
)

// LocalStorage 定义本地元数据与当前版本标记的读写接口。
type LocalStorage interface {
	SaveMetadata(version models.Version) error
	LoadMetadata() ([]models.Version, error)
	DeleteMetadata(version string) error
	GetInstallPath(version string) string
	GetCurrentVersionMarker() (string, error)
	SetCurrentVersionMarker(version string) error
}

// FileStorage 通过文件系统持久化版本信息。
type FileStorage struct {
	cfg          models.Config
	metadataPath string
	currentPath  string
	versionsDir  string
	mu           sync.Mutex
}

// MetadataFile 表示 metadata.json 的结构。
type MetadataFile struct {
	Versions []models.Version `json:"versions"`
}

// NewFileStorage 构造一个文件系统存储实例。
func NewFileStorage(cfg models.Config) *FileStorage {
	root := cfg.RootDir
	if root == "" {
		if home, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(home, ".govm")
		}
	}
	versionsDir := cfg.VersionsDir
	if versionsDir == "" {
		if root != "" {
			versionsDir = filepath.Join(root, "versions")
		}
	}
	cfg.RootDir = root
	cfg.VersionsDir = versionsDir
	return &FileStorage{
		cfg:          cfg,
		metadataPath: filepath.Join(root, "metadata.json"),
		currentPath:  filepath.Join(root, "current"),
		versionsDir:  versionsDir,
	}
}

// SaveMetadata 保存或更新版本元数据。
func (s *FileStorage) SaveMetadata(version models.Version) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureRoot(); err != nil {
		return err
	}

	versions, err := s.readMetadataLocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			versions = []models.Version{}
		} else {
			return err
		}
	}

	updated := false
	for i := range versions {
		if versions[i].Number == version.Number {
			versions[i] = version
			updated = true
			break
		}
	}
	if !updated {
		versions = append(versions, version)
	}

	return s.writeMetadataLocked(versions)
}

// LoadMetadata 读取所有本地元数据。
func (s *FileStorage) LoadMetadata() ([]models.Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.readMetadataLocked()
	if errors.Is(err, os.ErrNotExist) {
		return []models.Version{}, nil
	}
	return versions, err
}

// DeleteMetadata 移除指定版本的记录。
func (s *FileStorage) DeleteMetadata(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versions, err := s.readMetadataLocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	filtered := versions[:0]
	for _, v := range versions {
		if v.Number != version {
			filtered = append(filtered, v)
		}
	}

	return s.writeMetadataLocked(filtered)
}

// GetInstallPath 返回指定版本的安装目录。
func (s *FileStorage) GetInstallPath(version string) string {
	dir := s.versionsDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "govm", "versions")
	}
	return filepath.Join(dir, fmt.Sprintf("go%s", version))
}

// GetCurrentVersionMarker 读取当前版本标记。
func (s *FileStorage) GetCurrentVersionMarker() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.currentPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetCurrentVersionMarker 写入当前版本标记。
func (s *FileStorage) SetCurrentVersionMarker(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureRoot(); err != nil {
		return err
	}

	return os.WriteFile(s.currentPath, []byte(strings.TrimSpace(version)), 0o644)
}

func (s *FileStorage) ensureRoot() error {
	if s.metadataPath == "" {
		return errors.New("metadata path is not configured")
	}
	return os.MkdirAll(filepath.Dir(s.metadataPath), 0o755)
}

func (s *FileStorage) readMetadataLocked() ([]models.Version, error) {
	if s.metadataPath == "" {
		return nil, errors.New("metadata path is not configured")
	}

	file, err := os.Open(s.metadataPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []models.Version{}, os.ErrNotExist
		}
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(bytes) == 0 {
		return []models.Version{}, nil
	}

	var metadata MetadataFile
	if err := json.Unmarshal(bytes, &metadata); err != nil {
		return nil, err
	}
	if metadata.Versions == nil {
		metadata.Versions = []models.Version{}
	}
	return metadata.Versions, nil
}

func (s *FileStorage) writeMetadataLocked(versions []models.Version) error {
	if s.metadataPath == "" {
		return errors.New("metadata path is not configured")
	}

	metadata := MetadataFile{Versions: versions}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.metadataPath, data, 0o644)
}
