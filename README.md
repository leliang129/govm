# govm

govm 是一个面向 Linux 平台的 Go 版本管理器，使用 Go 语言实现，支持查询官方版本、下载安装包、切换当前版本、卸载等常见操作，同时自动维护 GOROOT/GOPATH/PATH 等环境变量。

## 功能特性

- 从 Go 官方源获取可用版本并按版本号降序展示
- 下载指定版本并校验 SHA256，完成解压与元数据落盘
- 自动配置 GOROOT/GOPATH/PATH，支持 bash 与 zsh
- 切换、查看、卸载本地版本，并保留当前版本标记
- 内置 `govm -help` / `govm -version` 等 CLI 支持

## 系统要求

- 操作系统：Linux（已验证 `amd64`、`arm64`、`386`）
- Go 1.21+（用于编译 govm）
- 可以访问 `https://go.dev/dl/` 的网络环境

## 安装与构建

```bash
git clone https://github.com/your-org/govm.git
cd govm
go build ./cmd/govm
./govm -help
```

构建过程中会自动检测当前操作系统与架构，如不受支持会给出清晰提示。

## 使用示例

```bash
# 查看远程版本列表
govm -remote

# 安装 Go 1.22.0（可省略 go 前缀）
govm install 1.22.0

# 查看本地版本并切换
govm -list
govm use 1.22.0

# 查看当前生效版本
govm current

# 卸载版本（如当前正在使用需加 --force）
govm uninstall 1.21.0 --force
```

## 故障排除

- **网络错误**：确认可以访问 `https://go.dev/dl/`，或配置代理后重试。
- **权限不足**：govm 默认安装到 `~/.govm`，请确保对该目录有写权限。
- **shell 配置未生效**：执行 `source ~/.bashrc` 或 `source ~/.zshrc` 重新加载配置。

## 开发与测试

```bash
go test ./...
```

integration 测试会创建临时目录，不会修改真实环境。提交前请确保 `go test` 全部通过，并更新 `todo.md` 进度。

## 构建发布产物

使用仓库提供的脚本可以一次性构建 Linux 各架构的二进制并生成 tar 包：

```bash
./scripts/build.sh v0.1.0
ls dist/
```

`dist/` 目录会包含 `govm-v0.1.0-linux-*.tar.gz`，每个包内含可执行文件及对应的 SHA256 校验文件，可直接上传到发布页面。

## 许可证

本项目尚未声明开源许可证，若需商用请先与作者联系。
