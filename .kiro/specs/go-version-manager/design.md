# Go 版本管理器设计文档

## 概述

Go 版本管理器（govm）是一个用 Go 语言编写的命令行工具，用于管理系统中的多个 Go 版本。该工具提供版本查询、安装、切换和卸载功能，并自动管理相关环境变量。

核心设计原则：
- 简单易用的命令行界面
- 最小化外部依赖
- 可靠的版本管理和环境配置
- 支持主流 Linux 发行版

## 架构

### 系统架构

```
┌─────────────────────────────────────────┐
│           CLI 命令层                     │
│  (命令解析、参数验证、输出格式化)         │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│         核心服务层                       │
│  ┌────────────┐  ┌──────────────┐       │
│  │版本管理服务│  │环境配置服务  │       │
│  └────────────┘  └──────────────┘       │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│         数据访问层                       │
│  ┌────────────┐  ┌──────────────┐       │
│  │远程版本源  │  │本地存储      │       │
│  └────────────┘  └──────────────┘       │
└─────────────────────────────────────────┘
```

### 目录结构

```
govm/
├── cmd/
│   └── govm/           # 主程序入口
│       └── main.go
├── internal/
│   ├── cli/            # CLI 命令处理
│   │   ├── commands.go
│   │   └── flags.go
│   ├── version/        # 版本管理服务
│   │   ├── manager.go
│   │   ├── installer.go
│   │   └── fetcher.go
│   ├── env/            # 环境配置服务
│   │   ├── config.go
│   │   └── shell.go
│   ├── storage/        # 本地存储
│   │   ├── local.go
│   │   └── metadata.go
│   └── remote/         # 远程版本源
│       ├── client.go
│       └── parser.go
├── pkg/
│   └── models/         # 数据模型
│       └── version.go
└── go.mod
```

## 组件和接口

### 1. CLI 命令层

**职责**: 解析用户命令、验证参数、调用核心服务、格式化输出

**主要命令**:
- `govm -remote`: 列出远程可用版本
- `govm -list`: 列出本地已安装版本
- `govm install <version>`: 安装指定版本
- `govm use <version>`: 切换到指定版本
- `govm current`: 显示当前使用版本
- `govm uninstall <version>`: 卸载指定版本
- `govm -help`: 显示帮助信息
- `govm -version`: 显示 govm 版本
- `govm -uninstall <version>`: flag 形式触发卸载（等价于 `govm uninstall <version>`）

**交互细节**:

- `install` 命令成功完成后，会调用 `Lister` 读取最新安装信息，并以彩色高亮的格式输出摘要，包括 `go version`、`GOROOT`、`GOPATH` 以及需要执行的 `source ~/.bashrc`/`source ~/.zshrc` 等下一步提示。

### 2. 版本管理服务

**接口定义**:

```go
type VersionManager interface {
    // 获取远程可用版本列表
    ListRemoteVersions() ([]Version, error)
    
    // 获取本地已安装版本列表
    ListLocalVersions() ([]Version, error)
    
    // 安装指定版本
    Install(version string) error
    
    // 卸载指定版本
    Uninstall(version string) error
    
    // 获取当前使用的版本
    GetCurrentVersion() (*Version, error)
}
```

### 3. 环境配置服务

**接口定义**:

```go
type EnvManager interface {
    // 设置指定版本为当前使用版本
    SetCurrentVersion(version string) error
    
    // 配置环境变量（GOROOT, GOPATH, PATH）
    ConfigureEnvironment(goRoot string) error
    
    // 检测用户的 shell 类型
    DetectShell() (string, error)
    
    // 更新 shell 配置文件
    UpdateShellConfig(shellType, goRoot string) error
}
```

### 4. 远程版本源客户端

**接口定义**:

```go
type RemoteClient interface {
    // 获取所有可用版本
    FetchVersions() ([]Version, error)
    
    // 下载指定版本的安装包
    DownloadVersion(version, destPath string) error
    
    // 验证下载文件的校验和
    VerifyChecksum(filePath, expectedChecksum string) error
}
```

### 5. 本地存储

**接口定义**:

```go
type LocalStorage interface {
    // 保存版本元数据
    SaveMetadata(version Version) error
    
    // 加载版本元数据
    LoadMetadata() ([]Version, error)
    
    // 删除版本元数据
    DeleteMetadata(version string) error
    
    // 获取安装目录路径
    GetInstallPath(version string) string
    
    // 获取当前版本标记
    GetCurrentVersionMarker() (string, error)
    
    // 设置当前版本标记
    SetCurrentVersionMarker(version string) error
}
```

### 6. 区域探测与镜像选择

**职责**: 根据公网 IP 探测当前所在国家，决定是否切换到国内镜像源，以避免中国大陆环境下访问 go.dev 网络不稳定。

**接口定义**:

```go
type RegionDetector interface {
    // CountryCode 返回探测到的 ISO 国家代码（如 CN、US），失败时返回空串和错误
    CountryCode(ctx context.Context) (string, error)
}

type MirrorSelector interface {
    // Select 根据国家代码返回远程 API 和下载地址
    Select(countryCode string) MirrorConfig
}

type MirrorConfig struct {
    APIBase      string
    DownloadBase string
}
```

**实现说明**:

- 默认 RegionDetector 调用 `https://ipapi.co/json` 获取 `country_code` 字段，并设置 3 秒超时；结果缓存到内存中，保证只探测一次。
- MirrorSelector 仅区分中国 (`CN`) 与其他国家：命中 `CN` 时返回 `https://golang.google.cn/dl/?mode=json&include=all` 与 `https://studygolang.com/dl/golang/`，否则回退到默认的 `https://go.dev/dl/?mode=json&include=all` 与 `https://go.dev/dl/`，确保远程版本列表包含历史版本。
- `internal/remote.Client` 在第一次请求远程版本列表前使用 MirrorSelector 调整自身 `baseURL` 与 `downloadBasePath`，之后的请求复用相同的配置，不额外探测。

## 数据模型

### Version 结构

```go
type Version struct {
    // 版本号（如 "1.21.0"）
    Number string
    
    // 完整版本字符串（如 "go1.21.0"）
    FullName string
    
    // 下载 URL
    DownloadURL string
    
    // 文件名
    FileName string
    
    // SHA256 校验和
    Checksum string
    
    // 操作系统
    OS string
    
    // 架构
    Arch string
    
    // 安装路径（本地版本）
    InstallPath string
    
    // 是否为当前使用版本
    IsCurrent bool
    
    // 安装时间
    InstalledAt time.Time
}
```

### 配置文件结构

```go
type Config struct {
    // govm 安装根目录（默认 ~/.govm）
    RootDir string
    
    // Go 版本安装目录（默认 ~/.govm/versions）
    VersionsDir string
    
    // 当前使用的版本
    CurrentVersion string
    
    // GOPATH 配置
    GoPath string
}
```

## 正确性属性

*属性是一个特征或行为，应该在系统的所有有效执行中保持为真——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### 属性 1: 版本列表排序不变性

**需求追溯**: 需求 1.2

**属性陈述**: 对于任意从远程源获取的版本列表 V，排序后的列表 V' 必须满足：对于所有相邻元素 v[i] 和 v[i+1]，版本号 v[i] >= v[i+1]（降序）

**形式化**:
```
∀ versions V, sorted_versions V' = sort_desc(V)
  ⇒ ∀ i ∈ [0, len(V')-2]: V'[i].Number >= V'[i+1].Number
```

**测试策略**: 基于属性的测试 - 生成随机版本列表，验证排序后满足降序规则

---

### 属性 2: 版本安装幂等性

**需求追溯**: 需求 3.5

**属性陈述**: 对于任意已安装的版本 v，重复执行安装操作应该跳过下载，不改变系统状态，且返回成功

**形式化**:
```
∀ version v, state S where v ∈ installed(S)
  ⇒ install(v, S) = (S, skip_message, success)
```

**测试策略**: 基于属性的测试 - 安装版本后再次安装，验证状态不变且返回跳过消息

---

### 属性 3: 元数据持久化往返属性

**需求追溯**: 需求 3.4

**属性陈述**: 对于任意版本元数据 m，保存后再加载应该得到相同的数据

**形式化**:
```
∀ metadata m
  ⇒ load(save(m)) = m
```

**测试策略**: 基于属性的测试 - 生成随机元数据，保存后加载，验证数据一致性

---

### 属性 4: 版本切换一致性

**需求追溯**: 需求 4.1, 4.2, 4.3

**属性陈述**: 对于任意已安装版本 v，执行 use(v) 后，当前版本应该是 v，且 GOROOT 指向 v 的安装目录，PATH 包含 v 的 bin 目录

**形式化**:
```
∀ version v, state S where v ∈ installed(S)
  ⇒ let S' = use(v, S) in
      current_version(S') = v ∧
      GOROOT(S') = install_path(v) ∧
      bin_path(v) ∈ PATH(S')
```

**测试策略**: 基于属性的测试 - 安装多个版本，切换到任意版本，验证环境变量正确配置

---

### 属性 5: 卸载完整性

**需求追溯**: 需求 6.1, 6.2

**属性陈述**: 对于任意已安装版本 v，执行 uninstall(v) 后，v 的安装目录不存在，且元数据中不包含 v

**形式化**:
```
∀ version v, state S where v ∈ installed(S)
  ⇒ let S' = uninstall(v, S) in
      ¬exists(install_path(v)) ∧
      v ∉ installed(S')
```

**测试策略**: 基于属性的测试 - 安装版本后卸载，验证文件和元数据都被清理

---

### 属性 6: 安装失败清理属性

**需求追溯**: 需求 3.7

**属性陈述**: 对于任意版本 v，如果安装过程失败，则不应该留下不完整的文件或元数据

**形式化**:
```
∀ version v, state S
  ⇒ if install(v, S) = (S', _, error) then
      ¬exists(partial_files(v)) ∧
      v ∉ installed(S')
```

**测试策略**: 基于属性的测试 - 模拟安装失败（网络错误、校验和错误等），验证清理完整

---

### 属性 7: 版本格式化输出完整性

**需求追溯**: 需求 1.4, 2.4

**属性陈述**: 对于任意版本 v，格式化输出必须包含版本号和架构信息（远程版本），或版本号和安装路径（本地版本）

**形式化**:
```
∀ version v
  ⇒ let output = format(v) in
      contains(output, v.Number) ∧
      (contains(output, v.Arch) ∨ contains(output, v.InstallPath))
```

**测试策略**: 基于属性的测试 - 生成随机版本对象，验证格式化输出包含必需字段

---

### 属性 8: Shell 配置文件选择正确性

**需求追溯**: 需求 7.4

**属性陈述**: 对于任意 shell 类型 s，选择的配置文件必须是该 shell 的标准配置文件

**形式化**:
```
∀ shell_type s
  ⇒ config_file(s) ∈ valid_config_files(s)
  where valid_config_files("bash") = {".bashrc", ".bash_profile"}
        valid_config_files("zsh") = {".zshrc"}
```

**测试策略**: 基于示例的测试 - 测试常见 shell 类型（bash, zsh），验证配置文件选择正确

---

### 属性 9: 校验和验证正确性

**需求追溯**: 需求 3.2

**属性陈述**: 对于任意文件 f 和预期校验和 c，验证函数应该正确判断文件完整性

**形式化**:
```
∀ file f, checksum c
  ⇒ verify(f, c) = true ⟺ sha256(f) = c
```

**测试策略**: 基于属性的测试 - 生成随机文件和校验和，验证函数正确判断匹配和不匹配情况

---

### 属性 10: 当前版本查询一致性

**需求追溯**: 需求 5.1, 5.4

**属性陈述**: 对于任意状态 S，如果当前版本是 v，则 v 必须在已安装列表中，且可执行文件存在

**形式化**:
```
∀ state S, version v where current_version(S) = v
  ⇒ v ∈ installed(S) ∧
    exists(executable_path(v))
```

**测试策略**: 基于属性的测试 - 设置当前版本后查询，验证版本在已安装列表且可执行文件存在

---

### 属性 11: 镜像选择回退属性

**需求追溯**: 需求 9.2, 9.4

**属性陈述**: 对于任意国家代码 `c`，镜像选择函数应该返回与默认源兼容的配置，并在探测失败或非 `CN` 时回退到默认值。

**形式化**:
```
∀ country_code c, let cfg = selectMirror(c) in
  (c = "CN" ⇒ cfg.APIBase = golang_cn_api ∧ cfg.DownloadBase = studygolang_dl) ∧
  (c ≠ "CN" ⇒ cfg.APIBase = go_dev_api ∧ cfg.DownloadBase = go_dev_dl)
```

**测试策略**: 基于示例的测试 - 模拟 `CN`、`US`、空字符串等输入，验证选择结果符合预期并与默认源兼容。



## 实现细节

### 1. 远程版本获取

**Go 官方版本源**: `https://go.dev/dl/?mode=json`

该 API 返回 JSON 格式的版本列表，包含：
- 版本号
- 下载链接
- 文件名
- SHA256 校验和
- 操作系统和架构信息

**实现步骤**:
1. 发送 HTTP GET 请求到官方 API
2. 解析 JSON 响应
3. 过滤出 Linux 平台的版本
4. 按版本号降序排序
5. 返回版本列表

### 2. 版本安装流程

```
1. 验证版本是否已安装
   ├─ 是 → 跳过安装，返回提示
   └─ 否 → 继续

2. 从远程源获取版本信息
   ├─ 成功 → 继续
   └─ 失败 → 返回错误

3. 下载安装包到临时目录
   ├─ 成功 → 继续
   └─ 失败 → 清理临时文件，返回错误

4. 验证 SHA256 校验和
   ├─ 通过 → 继续
   └─ 失败 → 清理临时文件，返回错误

5. 解压到版本目录
   ├─ 成功 → 继续
   └─ 失败 → 清理不完整文件，返回错误

6. 保存版本元数据
   └─ 完成

7. 清理临时文件
   └─ 完成
```

### 3. 版本切换流程

```
1. 验证版本是否已安装
   ├─ 是 → 继续
   └─ 否 → 返回错误，提示先安装

2. 检测用户的 shell 类型
   └─ 支持: bash, zsh

3. 构建环境变量配置
   ├─ GOROOT = ~/.govm/versions/go<version>
   ├─ PATH = $GOROOT/bin:$PATH
   └─ GOPATH = ~/go (如果未设置)

4. 更新 shell 配置文件
   ├─ 检查是否已有 govm 配置块
   ├─ 有 → 更新配置块
   └─ 无 → 添加新配置块

5. 标记当前版本
   └─ 保存到 ~/.govm/current

6. 提示用户重新加载 shell 配置
   └─ source ~/.bashrc 或 source ~/.zshrc
```

### 4. 目录结构

```
~/.govm/
├── versions/              # 所有已安装的 Go 版本
│   ├── go1.21.0/
│   │   ├── bin/
│   │   ├── src/
│   │   └── pkg/
│   ├── go1.20.5/
│   └── go1.19.10/
├── metadata.json          # 版本元数据
├── current                # 当前使用的版本标记
└── downloads/             # 临时下载目录
```

### 5. 元数据文件格式

**metadata.json**:
```json
{
  "versions": [
    {
      "number": "1.21.0",
      "fullName": "go1.21.0",
      "installPath": "/home/user/.govm/versions/go1.21.0",
      "installedAt": "2024-01-15T10:30:00Z",
      "os": "linux",
      "arch": "amd64"
    }
  ]
}
```

**current**:
```
1.21.0
```

### 6. Shell 配置块格式

**~/.bashrc 或 ~/.zshrc**:
```bash
# >>> govm initialize >>>
export GOROOT="$HOME/.govm/versions/go1.21.0"
export PATH="$GOROOT/bin:$PATH"
export GOPATH="$HOME/go"
# <<< govm initialize <<<
```

### 7. 错误处理策略

| 错误场景 | 处理方式 | 退出码 |
|---------|---------|--------|
| 网络请求失败 | 显示错误信息，提示检查网络 | 1 |
| 版本不存在 | 显示错误信息，建议使用 -remote 查看可用版本 | 2 |
| 校验和验证失败 | 清理下载文件，显示错误信息 | 3 |
| 解压失败 | 清理不完整文件，显示错误信息 | 4 |
| 权限不足 | 提示用户检查目录权限 | 5 |
| 版本未安装 | 显示错误信息，提示先安装 | 6 |
| Shell 配置失败 | 显示警告，提示手动配置 | 7 |

### 8. 平台检测

**支持的平台**:
- Linux + amd64
- Linux + arm64
- Linux + 386

**检测方法**:
```go
import "runtime"

os := runtime.GOOS      // "linux"
arch := runtime.GOARCH  // "amd64", "arm64", "386"
```

### 9. 依赖库

最小化外部依赖，仅使用 Go 标准库：
- `net/http`: HTTP 请求
- `encoding/json`: JSON 解析
- `archive/tar`: tar 解压
- `compress/gzip`: gzip 解压
- `crypto/sha256`: 校验和验证
- `os`: 文件系统操作
- `path/filepath`: 路径处理
- `runtime`: 平台检测

### 10. 性能考虑

- **并发下载**: 暂不支持，保持简单
- **缓存**: 缓存远程版本列表 5 分钟
- **增量更新**: 不支持，每次完整下载
- **压缩**: 使用官方提供的 tar.gz 格式

### 11. 安全考虑

- **校验和验证**: 必须验证 SHA256 校验和
- **HTTPS**: 使用 HTTPS 下载
- **路径遍历**: 解压时验证路径，防止路径遍历攻击
- **权限**: 安装目录使用用户权限，不需要 root
