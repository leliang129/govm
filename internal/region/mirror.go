package region

import "strings"

// MirrorConfig 描述远程 API 与下载地址基础配置。
type MirrorConfig struct {
	APIBase      string
	DownloadBase string
}

var (
	// GoDevMirror 表示默认官方源。
	GoDevMirror = MirrorConfig{
		APIBase:      "https://go.dev/dl/?mode=json&include=all",
		DownloadBase: "https://go.dev/dl/",
	}
	// StudyGolangMirror 表示国内镜像源。
	StudyGolangMirror = MirrorConfig{
		APIBase:      "https://golang.google.cn/dl/?mode=json&include=all",
		DownloadBase: "https://studygolang.com/dl/golang/",
	}
)

// SelectMirror 根据国家代码返回镜像配置。
func SelectMirror(countryCode string) MirrorConfig {
	if strings.EqualFold(strings.TrimSpace(countryCode), "CN") {
		return StudyGolangMirror
	}
	return GoDevMirror
}
