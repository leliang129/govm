package models

import "time"

// Version 描述远程或本地 Go 版本的核心元数据。
type Version struct {
	Number      string    // 纯版本号，例如 1.21.0
	FullName    string    // 完整版本字符串，例如 go1.21.0
	DownloadURL string    // 可下载的 URL
	FileName    string    // 下载安装包的文件名
	Checksum    string    // 官方提供的 SHA256 校验值
	OS          string    // 操作系统标识
	Arch        string    // 架构标识
	InstallPath string    // 本地安装路径（如果已安装）
	IsCurrent   bool      // 是否为当前激活版本
	InstalledAt time.Time // 安装时间
}
