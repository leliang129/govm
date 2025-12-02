package version

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/liangyou/govm/pkg/models"
)

// ProgressFunc 在下载过程中回调当前已完成的字节数以及总字节数。
type ProgressFunc func(downloaded, total int64)

// Downloader 负责下载版本压缩包并进行校验。
type Downloader struct {
	httpClient   HTTPClient
	downloadsDir string
	progressFunc ProgressFunc
}

// HTTPClient 定义 Downloader 所需的 HTTP 客户端能力。
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DownloaderOption 配置 Downloader。
type DownloaderOption func(*Downloader)

// WithHTTPClient 指定自定义 HTTP 客户端。
func WithHTTPClient(client HTTPClient) DownloaderOption {
	return func(d *Downloader) {
		if client != nil {
			d.httpClient = client
		}
	}
}

// WithDownloadsDir 指定下载目录。
func WithDownloadsDir(dir string) DownloaderOption {
	return func(d *Downloader) {
		if dir != "" {
			d.downloadsDir = dir
		}
	}
}

// WithProgressFunc 指定进度回调。
func WithProgressFunc(fn ProgressFunc) DownloaderOption {
	return func(d *Downloader) {
		d.progressFunc = fn
	}
}

// NewDownloader 创建 Downloader。
func NewDownloader(cfg models.Config, opts ...DownloaderOption) *Downloader {
	dir := cfg.RootDir
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".govm")
		}
	}
	downloads := filepath.Join(dir, "downloads")
	d := &Downloader{
		httpClient:   http.DefaultClient,
		downloadsDir: downloads,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Download 获取指定版本的压缩包并校验 SHA256，返回本地文件路径。
func (d *Downloader) Download(version models.Version) (string, error) {
	if err := os.MkdirAll(d.downloadsDir, 0o755); err != nil {
		return "", fmt.Errorf("downloader: create dir: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, version.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("downloader: build request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloader: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloader: unexpected status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp(d.downloadsDir, "download-*.tmp")
	if err != nil {
		return "", fmt.Errorf("downloader: temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	total := resp.ContentLength
	reader := d.wrapProgress(resp.Body, total)

	if _, err := io.Copy(tempFile, reader); err != nil {
		return "", fmt.Errorf("downloader: write file: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return "", fmt.Errorf("downloader: sync file: %w", err)
	}

	if err := d.verifyChecksum(tempPath, version.Checksum); err != nil {
		return "", err
	}

	finalPath := filepath.Join(d.downloadsDir, version.FileName)
	if err := os.Remove(finalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("downloader: remove existing: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", fmt.Errorf("downloader: finalize file: %w", err)
	}

	return finalPath, nil
}

func (d *Downloader) wrapProgress(reader io.Reader, total int64) io.Reader {
	if d.progressFunc == nil {
		return reader
	}

	pr := &progressReader{r: reader, total: total, report: d.progressFunc}
	return pr
}

func (d *Downloader) verifyChecksum(path, expected string) error {
	if expected == "" {
		return fmt.Errorf("downloader: empty checksum for %s", filepath.Base(path))
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("downloader: open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("downloader: hash file: %w", err)
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("downloader: checksum mismatch, got %s want %s", actual, expected)
	}
	return nil
}

type progressReader struct {
	r      io.Reader
	total  int64
	read   int64
	report ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		p.report(p.read, p.total)
	}
	return n, err
}
