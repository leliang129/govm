package remote

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/liangyou/govm/pkg/models"
)

const (
	defaultBaseURL   = "https://go.dev/dl/?mode=json"
	defaultCacheTTL  = 5 * time.Minute
	downloadBasePath = "https://go.dev/dl/"
)

var supportedArch = map[string]struct{}{
	"amd64": {},
	"arm64": {},
	"386":   {},
}

// RemoteClient 定义远程版本源应具备的能力。
type RemoteClient interface {
	FetchVersions() ([]models.Version, error)
}

// HTTPClient 描述最小化的 HTTP 客户端接口，方便测试时替换。
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option 用于配置 Client。
type Option func(*Client)

// WithBaseURL 设置自定义远程源地址。
func WithBaseURL(base string) Option {
	return func(c *Client) {
		if base != "" {
			c.baseURL = base
		}
	}
}

// WithHTTPClient 设置 HTTP 客户端。
func WithHTTPClient(h HTTPClient) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithCacheTTL 设置远程缓存时间。
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Client) {
		if ttl > 0 {
			c.cacheTTL = ttl
		}
	}
}

// Client 实现 RemoteClient 接口。
type Client struct {
	baseURL    string
	httpClient HTTPClient
	cacheTTL   time.Duration

	mu       sync.Mutex
	cached   []models.Version
	cachedAt time.Time
}

// NewClient 创建远程版本源客户端。
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
		cacheTTL:   defaultCacheTTL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FetchVersions 获取远程可用版本并进行过滤与排序。
func (c *Client) FetchVersions() ([]models.Version, error) {
	if versions, ok := c.getCached(); ok {
		return versions, nil
	}

	req, err := http.NewRequest(http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("remote: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("remote: read body: %w", err)
	}

	versions, err := c.parseVersions(body)
	if err != nil {
		return nil, err
	}

	c.setCache(versions)
	return versions, nil
}

func (c *Client) parseVersions(data []byte) ([]models.Version, error) {
	var releases []release
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("remote: decode response: %w", err)
	}

	var versions []models.Version
	for _, rel := range releases {
		for _, file := range rel.Files {
			if !shouldInclude(file) {
				continue
			}
			versions = append(versions, models.Version{
				Number:      strings.TrimPrefix(rel.Version, "go"),
				FullName:    rel.Version,
				DownloadURL: downloadBasePath + file.Filename,
				FileName:    file.Filename,
				Checksum:    file.Checksum,
				OS:          file.OS,
				Arch:        file.Arch,
			})
		}
	}

	sort.SliceStable(versions, func(i, j int) bool {
		cmp := compareVersionStrings(versions[i].FullName, versions[j].FullName)
		if cmp == 0 {
			return versions[i].Arch < versions[j].Arch
		}
		return cmp > 0
	})

	return versions, nil
}

func shouldInclude(f releaseFile) bool {
	if f.OS != "linux" || f.Kind != "archive" {
		return false
	}
	_, ok := supportedArch[f.Arch]
	return ok
}

func (c *Client) getCached() ([]models.Version, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cached) == 0 {
		return nil, false
	}
	if c.cacheTTL > 0 && time.Since(c.cachedAt) > c.cacheTTL {
		c.cached = nil
		return nil, false
	}
	clone := make([]models.Version, len(c.cached))
	copy(clone, c.cached)
	return clone, true
}

func (c *Client) setCache(versions []models.Version) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cached = make([]models.Version, len(versions))
	copy(c.cached, versions)
	c.cachedAt = time.Now()
}

// release 表示 Go 官方 API 中的版本记录。
type release struct {
	Version string        `json:"version"`
	Files   []releaseFile `json:"files"`
}

// releaseFile 表示 release 下的文件条目。
type releaseFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Checksum string `json:"sha256"`
	Kind     string `json:"kind"`
}

// compareVersionStrings 比较两个 go 版本号，返回 1 表示 a>b。
func compareVersionStrings(a, b string) int {
	pa := normalizeVersion(a)
	pb := normalizeVersion(b)

	if pa.major != pb.major {
		return cmpInt(pa.major, pb.major)
	}
	if pa.minor != pb.minor {
		return cmpInt(pa.minor, pb.minor)
	}
	if pa.patch != pb.patch {
		return cmpInt(pa.patch, pb.patch)
	}

	return comparePrerelease(pa, pb)
}

func cmpInt(a, b int) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

var prereleaseRank = map[string]int{
	"":     3,
	"rc":   2,
	"beta": 1,
}

func comparePrerelease(a, b versionParts) int {
	if a.prerelease == b.prerelease {
		return cmpInt(a.prereleaseNum, b.prereleaseNum)
	}

	ra := prereleaseRank[a.prerelease]
	rb := prereleaseRank[b.prerelease]

	return cmpInt(ra, rb)
}

func normalizeVersion(v string) versionParts {
	trimmed := strings.TrimPrefix(v, "go")
	parts := strings.Split(trimmed, ".")

	result := versionParts{prerelease: "", prereleaseNum: 0}

	if len(parts) > 0 {
		result.major = parseInt(parts[0])
	}
	if len(parts) > 1 {
		minor, suffix := parseNumericPrefix(parts[1])
		result.minor = minor
		if suffix != "" {
			setPrerelease(&result, suffix)
			return result
		}
	}
	if len(parts) > 2 {
		patch, suffix := parseNumericPrefix(parts[2])
		result.patch = patch
		if suffix != "" {
			setPrerelease(&result, suffix)
		}
	}

	return result
}

func parseInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func parseNumericPrefix(input string) (int, string) {
	idx := 0
	for idx < len(input) && input[idx] >= '0' && input[idx] <= '9' {
		idx++
	}
	if idx == 0 {
		return 0, input
	}
	value := parseInt(input[:idx])
	return value, input[idx:]
}

func setPrerelease(parts *versionParts, suffix string) {
	idx := 0
	for idx < len(suffix) && (suffix[idx] < '0' || suffix[idx] > '9') {
		idx++
	}
	label := suffix[:idx]
	num := 0
	if idx < len(suffix) {
		num = parseInt(suffix[idx:])
	}

	parts.prerelease = label
	parts.prereleaseNum = num
}

type versionParts struct {
	major         int
	minor         int
	patch         int
	prerelease    string
	prereleaseNum int
}
