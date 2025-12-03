# 需求文档

## 介绍

Go 版本管理器（govm）是一个命令行工具，用于管理系统中的多个 Go 语言版本。该工具允许用户从官方源获取、安装、切换和管理不同版本的 Go，并自动配置相关环境变量。该工具专为 Linux 发行版设计，包括但不限于 CentOS 和 Ubuntu。

## 术语表

- **govm**: Go 版本管理器，本系统的名称
- **Go 官方源**: Go 语言官方发布版本的下载源（golang.org/dl）
- **已安装版本**: 已下载并解压到本地系统的 Go 版本
- **可用版本**: 可从官方源下载的所有 Go 版本
- **当前使用版本**: 系统当前激活并在 PATH 中的 Go 版本
- **GOROOT**: Go 安装目录的环境变量
- **GOPATH**: Go 工作空间目录的环境变量
- **版本标识符**: Go 版本的唯一标识，格式如 "1.21.0"

## 需求

### 需求 1

**用户故事:** 作为开发者，我想查看所有可从官方源下载的 Go 版本，以便我可以选择需要安装的版本

#### 验收标准

1. WHEN 用户执行 `govm -remote` 命令，THEN govm SHALL 从 Go 官方源获取所有可用版本列表（不限于最近两个版本）
2. WHEN govm 获取到版本列表，THEN govm SHALL 按版本号降序显示所有可用版本
3. WHEN 官方源无法访问，THEN govm SHALL 显示清晰的错误信息并返回非零退出码
4. WHEN 显示版本列表，THEN govm SHALL 包含版本号和对应的 Linux 平台架构信息
5. WHEN 版本列表为空，THEN govm SHALL 通知用户未找到可用版本

### 需求 2

**用户故事:** 作为开发者，我想查看本地已安装的 Go 版本，以便我了解系统中有哪些版本可用

#### 验收标准

1. WHEN 用户执行 `govm -list` 命令，THEN govm SHALL 显示所有已安装到本地的 Go 版本
2. WHEN 显示已安装版本，THEN govm SHALL 标记当前正在使用的版本
3. WHEN 本地没有已安装版本，THEN govm SHALL 显示提示信息告知用户尚未安装任何版本
4. WHEN 显示已安装版本，THEN govm SHALL 包含每个版本的安装路径信息

### 需求 3

**用户故事:** 作为开发者，我想安装指定版本的 Go，以便我可以在项目中使用该版本

#### 验收标准

1. WHEN 用户执行 `govm install <版本标识符>` 命令，THEN govm SHALL 从官方源下载指定版本的 Go 安装包
2. WHEN 下载完成，THEN govm SHALL 验证下载文件的完整性
3. WHEN 验证通过，THEN govm SHALL 解压安装包到指定的安装目录
4. WHEN 安装完成，THEN govm SHALL 在本地记录该版本的安装信息
5. WHEN 指定的版本已经安装，THEN govm SHALL 跳过下载并通知用户该版本已存在
6. WHEN 指定的版本不存在于官方源，THEN govm SHALL 显示错误信息并返回非零退出码
7. WHEN 下载或安装过程失败，THEN govm SHALL 清理不完整的文件并显示错误信息
8. WHEN 安装完成，THEN govm SHALL 输出包含 go 版本、GOROOT、GOPATH 以及 `source` 命令提示的摘要信息，并使用醒目的样式提示用户下一步操作

### 需求 4

**用户故事:** 作为开发者，我想切换到不同的 Go 版本，以便我可以在不同项目中使用不同版本的 Go

#### 验收标准

1. WHEN 用户执行 `govm use <版本标识符>` 命令，THEN govm SHALL 将指定版本设置为当前使用版本
2. WHEN 切换版本，THEN govm SHALL 自动更新 GOROOT 环境变量指向新版本的安装目录
3. WHEN 切换版本，THEN govm SHALL 自动更新 PATH 环境变量以包含新版本的 bin 目录
4. WHEN 切换版本，THEN govm SHALL 自动配置 GOPATH 环境变量（如果尚未配置）
5. WHEN 指定的版本未安装，THEN govm SHALL 显示错误信息并提示用户先安装该版本
6. WHEN 环境变量更新完成，THEN govm SHALL 持久化配置到用户的 shell 配置文件（如 .bashrc 或 .zshrc）

### 需求 5

**用户故事:** 作为开发者，我想查看当前正在使用的 Go 版本，以便我确认当前环境配置

#### 验收标准

1. WHEN 用户执行 `govm current` 命令，THEN govm SHALL 显示当前激活的 Go 版本号
2. WHEN 显示当前版本，THEN govm SHALL 包含该版本的 GOROOT 路径信息
3. WHEN 没有激活的版本，THEN govm SHALL 显示提示信息告知用户尚未设置任何版本
4. WHEN 显示当前版本，THEN govm SHALL 验证该版本的可执行文件是否存在且可用

### 需求 6

**用户故事:** 作为开发者，我想卸载不再需要的 Go 版本，以便释放磁盘空间

#### 验收标准

1. WHEN 用户执行 `govm uninstall <版本标识符>` 命令，THEN govm SHALL 删除指定版本的所有文件
2. WHEN 卸载版本，THEN govm SHALL 从本地记录中移除该版本的安装信息
3. WHEN 尝试卸载当前正在使用的版本，THEN govm SHALL 显示警告信息并要求用户确认
4. WHEN 指定的版本未安装，THEN govm SHALL 显示错误信息并返回非零退出码
5. WHEN 卸载完成，THEN govm SHALL 显示成功信息并列出剩余已安装版本

### 需求 7

**用户故事:** 作为开发者，我想在不同的 Linux 发行版上使用 govm，以便我可以在各种环境中管理 Go 版本

#### 验收标准

1. WHEN govm 在 CentOS 系统上运行，THEN govm SHALL 正确识别系统架构并下载对应的 Go 安装包
2. WHEN govm 在 Ubuntu 系统上运行，THEN govm SHALL 正确识别系统架构并下载对应的 Go 安装包
3. WHEN govm 检测到不支持的操作系统，THEN govm SHALL 显示错误信息并拒绝执行
4. WHEN govm 配置环境变量，THEN govm SHALL 根据用户的默认 shell 类型选择正确的配置文件
5. WHEN govm 需要系统权限，THEN govm SHALL 提示用户使用适当的权限执行命令

### 需求 8

**用户故事:** 作为开发者，我想查看 govm 的帮助信息，以便我了解如何使用各种命令

#### 验收标准

1. WHEN 用户执行 `govm -help` 或 `govm --help` 命令，THEN govm SHALL 显示所有可用命令的列表和说明
2. WHEN 显示帮助信息，THEN govm SHALL 包含每个命令的用法示例及等价 flag（例如 `govm uninstall <version>` 与 `govm -uninstall <version>`）
3. WHEN 用户执行 `govm -version` 命令，THEN govm SHALL 显示 govm 自身的版本号
4. WHEN 用户执行不存在的命令，THEN govm SHALL 显示错误信息并提示查看帮助文档

### 需求 9

**用户故事:** 作为位于中国大陆的开发者，我希望 govm 自动切换到国内镜像源，以便能够顺畅地拉取 Go 版本列表并下载安装包

#### 验收标准

1. WHEN govm 启动需要访问远程版本源时，THEN govm SHALL 探测当前公网 IP 所在国家，并缓存探测结果
2. WHEN 探测结果显示为中国（country code = `CN`），THEN govm SHALL 使用 `https://golang.google.cn/dl/?mode=json` 作为远程版本列表 API
3. WHEN 探测结果显示为中国（country code = `CN`），THEN govm SHALL 使用 `https://studygolang.com/dl/golang/` 作为下载基础地址构造安装包 URL
4. WHEN IP 探测失败或返回的国家不是中国，THEN govm SHALL 回退到默认的 `https://go.dev/dl/` 远程源和下载地址
5. WHEN govm 切换镜像源，THEN govm SHALL 保持与默认源一致的返回格式和功能行为，不影响其他功能
