package region

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultEndpoint        = "https://ipinfo.io/country"
	defaultFallback        = "https://ipapi.co/json"
	defaultTimeout         = 3 * time.Second
	errEmptyCountryMessage = "region: empty country code"
)

type responseParser func([]byte) (string, error)

// HTTPClient 最小化 HTTP 客户端接口，便于测试替换。
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Detector 实现 RegionDetector，负责探测公网 IP 所在国家。
type Detector struct {
	endpoint         string
	fallbackEndpoint string
	client           HTTPClient
	timeout          time.Duration

	parsePrimary  responseParser
	parseFallback responseParser

	mu    sync.Mutex
	cache string
}

// Option 用于配置 Detector。
type Option func(*Detector)

// WithEndpoint 设置自定义探测接口地址。
func WithEndpoint(endpoint string) Option {
	return func(d *Detector) {
		if endpoint != "" {
			d.endpoint = endpoint
		}
	}
}

// WithFallbackEndpoint 设置自定义备选接口地址。
func WithFallbackEndpoint(endpoint string) Option {
	return func(d *Detector) {
		d.fallbackEndpoint = endpoint
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端。
func WithHTTPClient(client HTTPClient) Option {
	return func(d *Detector) {
		if client != nil {
			d.client = client
		}
	}
}

// WithTimeout 设置探测请求超时时间。
func WithTimeout(timeout time.Duration) Option {
	return func(d *Detector) {
		if timeout > 0 {
			d.timeout = timeout
		}
	}
}

// NewDetector 创建 Detector 实例。
func NewDetector(opts ...Option) *Detector {
	detector := &Detector{
		endpoint:         defaultEndpoint,
		fallbackEndpoint: defaultFallback,
		client:           http.DefaultClient,
		timeout:          defaultTimeout,
		parsePrimary:     parsePlainCountry,
		parseFallback:    parseJSONCountry,
	}
	for _, opt := range opts {
		opt(detector)
	}
	return detector
}

// CountryCode 返回 ISO 国家代码（如 CN、US）。探测成功后结果会缓存，后续调用不再触发 HTTP 请求。
func (d *Detector) CountryCode(ctx context.Context) (string, error) {
	d.mu.Lock()
	if d.cache != "" {
		code := d.cache
		d.mu.Unlock()
		return code, nil
	}
	d.mu.Unlock()

	code, err := d.lookup(ctx)
	if err != nil {
		return "", err
	}

	d.mu.Lock()
	if d.cache == "" {
		d.cache = code
	}
	d.mu.Unlock()
	return code, nil
}

func (d *Detector) lookup(ctx context.Context) (string, error) {
	if d.client == nil {
		return "", errors.New("region: http client is nil")
	}

	code, err := d.fetchCountry(ctx, d.endpoint, d.parsePrimary)
	if err == nil {
		return code, nil
	}

	if d.fallbackEndpoint != "" {
		if fallbackCode, fbErr := d.fetchCountry(ctx, d.fallbackEndpoint, d.parseFallback); fbErr == nil {
			return fallbackCode, nil
		} else {
			return "", fmt.Errorf("%v (fallback: %v)", err, fbErr)
		}
	}

	return "", err
}

func (d *Detector) fetchCountry(ctx context.Context, endpoint string, parser responseParser) (string, error) {
	if parser == nil {
		return "", errors.New("region: response parser is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("region: build request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("region: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("region: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("region: read body: %w", err)
	}

	code, err := parser(data)
	if err != nil {
		return "", err
	}
	return code, nil
}

func parsePlainCountry(data []byte) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(string(data)))
	if code == "" {
		return "", errors.New(errEmptyCountryMessage)
	}
	return code, nil
}

func parseJSONCountry(data []byte) (string, error) {
	var payload struct {
		CountryCode string `json:"country_code"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("region: decode response: %w", err)
	}
	return parsePlainCountry([]byte(payload.CountryCode))
}
