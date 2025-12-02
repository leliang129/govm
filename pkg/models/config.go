package models

// Config 保存 govm 的全局配置，与用户主目录下的资源保持一致。
type Config struct {
	RootDir        string // govm 安装根目录，默认 ~/.govm
	VersionsDir    string // 各版本安装目录，默认 ~/.govm/versions
	CurrentVersion string // 当前激活的纯版本号
	GoPath         string // GOPATH 配置
}
