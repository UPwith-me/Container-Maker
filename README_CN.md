<div align="center">

<img src="assets/logo.png" width="200" alt="Container-Maker Logo" />

# ⚡ CONTAINER-MAKER

### 面向容器时代的终极开发者体验平台

<p>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Built_with-Go_1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL_3.0-blue?style=for-the-badge" alt="License"></a>
    <a href="#"><img src="https://img.shields.io/badge/Platform-Windows_|_Linux_|_macOS-181717?style=for-the-badge&logo=linux" alt="Platform"></a>
</p>

<p>
    <a href="#-快速开始"><b>快速开始</b></a> •
    <a href="#-核心功能"><b>功能特性</b></a> •
    <a href="#️-云控制平面"><b>云控制平面</b></a> •
    <a href="#-命令参考"><b>命令参考</b></a> •
    <a href="README.md"><b>English</b></a>
</p>

<br>

**Container-Maker (cm)** 填补了本地 Makefile 的简洁性与容器隔离性之间的空白。它是一个零配置的 CLI 工具，通过融合 `make` 的速度与 DevContainers 的智能，将任何机器瞬间转变为生产级开发工作站。

</div>

---

## 📑 目录

- [关于项目](#-关于项目)
- [快速开始](#-快速开始)
  - [安装方式](#安装方式)
  - [5分钟入门](#5分钟入门)
- [核心功能](#-核心功能)
  - [零配置启动](#1-零配置启动-cm-setup)
  - [环境诊断](#2-环境诊断-cm-doctor)
  - [项目初始化](#3-项目初始化-cm-init)
  - [容器交互](#4-容器交互-cm-shell--run--exec)
  - [AI 配置生成](#5-ai-配置生成-cm-ai-generate)
  - [模板市场](#6-模板市场-cm-marketplace)
  - [VS Code 集成](#7-vs-code-集成-cm-code)
- [高级功能](#-高级功能)
  - [DevContainer Features](#devcontainer-features-oci)
  - [Docker Compose 集成](#docker-compose-集成)
  - [智能缓存](#智能缓存)
  - [端口转发](#端口转发)
  - [文件监听](#文件监听-cm-watch)
  - [远程开发](#远程开发-cm-remote)
  - [安全扫描](#安全扫描)
- [云控制平面](#️-云控制平面)
  - [功能概览](#功能概览)
  - [支持的云提供商](#支持的云提供商-14)
  - [CLI 集成](#cli-集成)
  - [Web 控制台](#web-控制台)
- [TUI 仪表盘](#-tui-仪表盘)
- [模板库](#-模板库)
- [命令参考](#-命令参考)
- [配置参考](#️-配置参考)
- [设计巧思](#-设计巧思)
- [安全性](#-安全性)
- [常见问题](#-常见问题)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

---

## 🎯 关于项目

**Container-Maker** 是将容器化开发能力带入命令行的关键缺失环节，让您无需面对复杂的配置即可享受容器的隔离性与一致性。

<table>
<tr>
<td width="33%" valign="top">

### 🎯 单一真相来源
您的 `devcontainer.json` 定义了整个开发环境。无需再维护独立的 Dockerfile、Makefile 或 shell 脚本。

</td>
<td width="33%" valign="top">

### 💎 原生级体验
`vim`、`htop` 和交互式 shell 的工作方式与本地完全一致。自定义信号代理完美处理窗口调整 (SIGWINCH)。

</td>
<td width="33%" valign="top">

### 🚀 BuildKit 加速
利用 Docker BuildKit 进行激进的层缓存。您的环境在几秒内启动，而不是几分钟。

</td>
</tr>
</table>

### 功能对比

| 功能 | Docker CLI | VS Code DevContainers | **Container-Maker** |
|------|------------|----------------------|---------------------|
| 零配置启动 | ❌ | ⚠️ 需要 VS Code | ✅ |
| 独立 CLI 使用 | ✅ | ❌ | ✅ |
| AI 配置生成 | ❌ | ❌ | ✅ |
| 云端部署 | ❌ | ❌ | ✅ |
| TUI 仪表盘 | ❌ | ❌ | ✅ |
| 模板市场 | ❌ | ⚠️ 有限 | ✅ |
| 多运行时支持 | ⚠️ 仅 Docker | ⚠️ 仅 Docker | ✅ Docker/Podman |

---

## 🚀 快速开始

### 安装方式

#### 方式一：下载预编译包（推荐）

```bash
# Windows (PowerShell)
irm https://github.com/UPwith-me/Container-Maker/releases/latest/download/cm-windows-amd64.exe -OutFile cm.exe

# Linux / macOS
curl -Lo cm https://github.com/UPwith-me/Container-Maker/releases/latest/download/cm-linux-amd64
chmod +x cm && sudo mv cm /usr/local/bin/
```

#### 方式二：Go Install

```bash
go install github.com/UPwith-me/Container-Maker/cmd/cm@latest
```

#### 方式三：从源码构建

```bash
git clone https://github.com/UPwith-me/Container-Maker.git
cd Container-Maker
go build -o cm ./cmd/cm
```

### 5分钟入门

```bash
# 步骤 1: 自动检测并安装 Docker/Podman
cm setup

# 步骤 2: 使用指定模板初始化项目
cm init --template python

# 步骤 3: 进入容器
cm shell

# 步骤 4: 运行命令
cm run python main.py

# 步骤 5: 在 VS Code 中打开
cm code
```

---

## ✨ 核心功能

### 1. 零配置启动 (`cm setup`)

自动检测您的操作系统并安装最优的容器运行时。

```bash
cm setup
```

- **Windows**: 安装 Docker Desktop 或 WSL2 + Docker
- **Linux**: 安装 Docker CE 或 Podman
- **macOS**: 安装 Docker Desktop 或 Colima

### 2. 环境诊断 (`cm doctor`)

对开发环境进行深度健康检查。

```bash
cm doctor
```

检查项目包括：
- ✅ 容器运行时 (Docker/Podman)
- ✅ GPU 支持 (NVIDIA/AMD)
- ✅ 网络连通性
- ✅ 磁盘空间
- ✅ Docker Compose 可用性

### 3. 项目初始化 (`cm init`)

从精选模板创建新项目，或让 AI 生成配置。

```bash
# 交互式模式
cm init

# 使用指定模板
cm init --template pytorch

# AI 驱动生成
cm ai generate
```

### 4. 容器交互 (`cm shell` / `run` / `exec`)

多种与容器交互的方式：

| 命令 | 描述 | 使用场景 |
|------|------|----------|
| `cm shell` | 启动持久容器并进入 | 交互式开发 |
| `cm run <cmd>` | 在临时容器中运行命令 | 一次性构建 |
| `cm exec <cmd>` | 在运行中的容器中执行 | 热重载场景 |

```bash
# 启动 shell 会话
cm shell

# 运行测试
cm run pytest tests/

# 在后台容器中执行
cm exec npm run build
```

### 5. AI 配置生成 (`cm ai generate`)

让 AI 分析您的项目并生成优化的配置。

```bash
cm ai generate
```

- 分析 `package.json`、`requirements.txt`、`go.mod` 等
- 推荐最优基础镜像
- 配置缓存策略
- 添加适当的 VS Code 扩展

### 6. 模板市场 (`cm marketplace`)

浏览和安装社区模板。

```bash
# 搜索模板
cm marketplace search pytorch

# 列出 GPU 加速模板
cm marketplace search --gpu

# 安装模板
cm marketplace install ml-pytorch
```

### 7. VS Code 集成 (`cm code`)

在 VS Code 中打开项目，支持完整的 DevContainer。

```bash
cm code
```

- 自动检测 `devcontainer.json`
- 启动带有 Remote-Containers 的 VS Code
- 支持本地和远程容器

---

## 🔧 高级功能

### DevContainer Features (OCI)

从 OCI 注册表安装额外工具：

```bash
# 添加 Go 到容器
cm feature add ghcr.io/devcontainers/features/go

# 添加 Docker-in-Docker
cm feature add ghcr.io/devcontainers/features/docker-in-docker
```

### Docker Compose 集成

无缝支持 `docker-compose.yml`：

```json
{
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "workspaceFolder": "/workspace"
}
```

### 智能缓存

主要语言的自动持久化缓存：

| 语言 | 缓存路径 | 速度提升 |
|------|----------|----------|
| Go | `/go/pkg/mod` | 最高 10x |
| Node.js | `node_modules` | 最高 5x |
| Rust | `/usr/local/cargo` | 最高 8x |
| Python | `~/.cache/pip` | 最高 3x |
| Java | `~/.m2` | 最高 4x |

### 端口转发

自动检测和转发端口：

```json
{
  "forwardPorts": [3000, 8080, "5432:5432"]
}
```

支持：
- 单个端口：`3000`
- 端口范围：`8000-8010`
- 端口映射：`"host:container"`

### 文件监听 (`cm watch`)

文件变更时自动运行命令：

```bash
# 监听并运行测试
cm watch --run "pytest tests/"

# 使用自定义模式监听
cm watch --pattern "*.py" --run "python main.py"
```

### 安全扫描

主动安全警告：

```bash
cm doctor --security
```

检测项目：
- ⚠️ Docker 套接字挂载
- ⚠️ 特权模式
- ⚠️ 敏感环境变量
- ✅ 建议使用 Rootless Docker

### 远程开发 (`cm remote`)

无缝连接远程机器并同步文件：

```bash
# 添加远程主机
cm remote add myserver user@192.168.1.100

# 列出已配置的远程主机
cm remote list

# 测试连接
cm remote test myserver

# 设置当前使用的远程主机
cm remote use myserver

# 在远程容器中打开 shell
cm remote shell
```

**文件同步：**

```bash
# 启动持续同步（本地 → 远程）
cm remote sync start myserver

# 一次性推送到远程
cm remote sync push

# 从远程拉取
cm remote sync pull
```

---

## ☁️ 云控制平面

Container-Maker Cloud 将您的本地开发扩展到云端，提供按需 GPU 实例。

### 功能概览

- **一键 GPU 访问**：配置 NVIDIA T4、A10、A100 实例
- **14+ 云提供商**：AWS、GCP、Azure、DigitalOcean 等
- **按需付费**：无预付费用，按秒计费
- **无缝 CLI 集成**：`cm cloud` 命令

### 支持的云提供商 (14+)

| 提供商 | GPU 支持 | 区域数 |
|--------|----------|--------|
| AWS EC2 | ✅ | 25+ |
| Google Cloud | ✅ | 35+ |
| Azure | ✅ | 60+ |
| DigitalOcean | ❌ | 14 |
| Hetzner | ❌ | 5 |
| Linode | ✅ | 11 |
| Vultr | ✅ | 25 |
| OCI (Oracle) | ✅ | 41 |
| Lambda Labs | ✅ | 5 |
| RunPod | ✅ | 10+ |
| Vast.ai | ✅ | 社区 |
| Paperspace | ✅ | 6 |
| CoreWeave | ✅ | 3 |
| Docker (本地) | ✅ | - |

### CLI 集成

```bash
# 登录云端
cm cloud login

# 列出可用实例
cm cloud instances

# 创建 GPU 实例
cm cloud create --provider aws --type gpu-t4 --name ml-training

# 通过 SSH 连接
cm cloud connect <instance-id>

# 停止实例
cm cloud stop <instance-id>

# 删除实例
cm cloud delete <instance-id>
```

### Web 控制台

访问功能完整的 Web 控制台：

```bash
# 启动本地控制台
cm cloud dashboard

# 或访问托管版本
# https://cloud.container-maker.dev
```

功能：
- 实时实例监控
- WebSocket 日志流
- 交互式终端
- 使用分析和账单

---

## 📊 TUI 仪表盘

美观的终端 UI，用于监控您的容器。

```bash
cm status
```

或者直接运行 `cm` 无参数启动主页。

功能：
- 容器列表及状态
- 资源使用情况 (CPU/内存)
- 日志流
- 快捷操作 (启动/停止/删除)

---

## 📦 模板库

30+ 精选模板，适用于各种场景：

### AI/ML
| 模板 | 描述 |
|------|------|
| `pytorch` | 支持 CUDA 的 PyTorch |
| `tensorflow` | TensorFlow 2.x + GPU |
| `huggingface` | Transformers + Datasets |
| `jupyter` | JupyterLab 科学计算栈 |

### 复杂环境 (新增!)
| 模板 | 描述 |
|------|------|
| `miniconda` | Conda/Anaconda 数据科学环境 |
| `python-poetry` | Poetry 现代包管理 |
| `python-pipenv` | Pipenv 虚拟环境 |
| `cpp-conan` | C++ Conan 包管理器 |
| `cpp-vcpkg` | C++ Vcpkg 库管理 |
| `cpp-cmake` | C++ CMake 项目 |
| `java-maven` | Java Maven 项目 |
| `java-gradle` | Java Gradle 项目 |
| `dotnet` | .NET 8.0 开发环境 |
| `php-composer` | PHP Composer 项目 |

### Web 开发
| 模板 | 描述 |
|------|------|
| `node` | Node.js 20 LTS |
| `react` | React + Vite |
| `nextjs` | Next.js 14 |
| `python-web` | FastAPI / Django |

### 系统编程
| 模板 | 描述 |
|------|------|
| `go` | Go 1.21+ |
| `rust` | Rust + Cargo |
| `cpp` | C++ + CMake |

### DevOps
| 模板 | 描述 |
|------|------|
| `terraform` | Terraform + 云 CLI |
| `kubernetes` | kubectl + Helm |
| `ansible` | Ansible + Python |

---

## 📖 命令参考

### 核心命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm` | 启动 TUI 主页 | `cm` |
| `cm init` | 初始化新项目 | `cm init --template python` |
| `cm shell` | 进入持久容器 | `cm shell` |
| `cm run <cmd>` | 在容器中运行命令 | `cm run make build` |
| `cm exec <cmd>` | 在运行中的容器执行 | `cm exec npm test` |
| `cm prepare` | 构建容器镜像 | `cm prepare` |

### 环境命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm setup` | 安装容器运行时 | `cm setup` |
| `cm doctor` | 运行诊断 | `cm doctor` |
| `cm status` | 显示 TUI 仪表盘 | `cm status` |
| `cm code` | 在 VS Code 中打开 | `cm code` |

### AI 与模板

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm ai generate` | AI 生成配置 | `cm ai generate` |
| `cm marketplace search` | 搜索模板 | `cm marketplace search --gpu` |
| `cm marketplace install` | 安装模板 | `cm marketplace install pytorch` |
| `cm template list` | 列出本地模板 | `cm template list` |

### 云端命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm cloud login` | 认证登录 | `cm cloud login` |
| `cm cloud instances` | 列出实例 | `cm cloud instances` |
| `cm cloud create` | 创建实例 | `cm cloud create --type gpu-t4` |
| `cm cloud connect` | SSH 连接实例 | `cm cloud connect abc123` |
| `cm cloud stop` | 停止实例 | `cm cloud stop abc123` |
| `cm cloud delete` | 删除实例 | `cm cloud delete abc123` |

### 高级命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm feature add` | 添加 OCI 特性 | `cm feature add ghcr.io/devcontainers/features/go` |
| `cm feature list` | 列出特性 | `cm feature list` |
| `cm cache clean` | 清理构建缓存 | `cm cache clean` |
| `cm watch` | 监听文件变更 | `cm watch --run "pytest"` |
| `cm backend` | 管理运行时 | `cm backend list` |
| `cm clone` | 克隆 + 进入容器 | `cm clone github.com/user/repo` |
| `cm share` | 生成分享链接 | `cm share --format markdown` |
| `cm images` | 管理预设镜像 | `cm images list` |
| `cm make` | 运行 Makefile 目标 | `cm make build` |

### 远程开发

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm remote add` | 添加远程主机 | `cm remote add server user@host` |
| `cm remote list` | 列出远程主机 | `cm remote list` |
| `cm remote use` | 设置活动远程 | `cm remote use server` |
| `cm remote test` | 测试连接 | `cm remote test server` |
| `cm remote shell` | 远程 Shell | `cm remote shell` |
| `cm remote sync start` | 启动文件同步 | `cm remote sync start` |
| `cm remote sync push` | 推送到远程 | `cm remote sync push` |
| `cm remote sync pull` | 从远程拉取 | `cm remote sync pull` |

### 团队与组织

| 命令 | 描述 | 示例 |
|------|------|------|
| `cm team set` | 设置组织 | `cm team set mycompany` |
| `cm team templates` | 设置模板仓库 | `cm team templates <url>` |
| `cm team info` | 显示团队配置 | `cm team info` |

---

## ⚙️ 配置参考

### devcontainer.json

```jsonc
{
  // 基础镜像或 Dockerfile
  "image": "mcr.microsoft.com/devcontainers/go:1.21",
  // 或使用 Dockerfile
  "build": {
    "dockerfile": "Dockerfile",
    "context": ".",
    "args": { "VARIANT": "1.21" }
  },

  // 容器选项
  "runArgs": ["--cap-add=SYS_PTRACE"],
  "mounts": ["source=go-mod,target=/go/pkg/mod,type=volume"],
  "containerEnv": {
    "CGO_ENABLED": "0"
  },

  // 生命周期命令
  "postCreateCommand": "go mod download",
  "postStartCommand": "echo 'Ready!'",

  // DevContainer Features
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },

  // 端口转发
  "forwardPorts": [8080, 3000],

  // VS Code 定制
  "customizations": {
    "vscode": {
      "extensions": ["golang.go"],
      "settings": {
        "go.useLanguageServer": true
      }
    }
  }
}
```

---

## 💡 设计巧思

Container-Maker 包含多处贴心设计：

### 🔧 自动 PATH 集成

首次运行时，`cm` 会提示将自己添加到系统 PATH，并**立即刷新您的 shell 会话**——无需重启终端。

```
🚀 Container-Maker 检测到这是首次运行。
   是否将 cm 添加到 PATH？[Y/n]
   ✅ 已添加到 PATH。会话已刷新！
```

### 🔄 智能会话刷新

修改环境变量后，`cm` 自动刷新 PowerShell/Bash 会话，无需关闭并重新打开终端。

### 🎨 丰富的 TUI 体验

无参数运行 `cm` 会启动交互式主页：
- 项目检测
- 快捷操作菜单
- 容器状态一览

### 📦 增量特性安装

Features 只下载一次并缓存。后续项目复用缓存层，实现即时启动。

### 🔍 智能项目检测

`cm` 自动查找 `devcontainer.json`：
1. `.devcontainer/devcontainer.json`
2. `devcontainer.json`
3. `.devcontainer.json`

---

## 🔒 安全性

### Rootless 支持

完全兼容 Rootless Docker 和 Podman：

```bash
cm backend switch podman-rootless
```

### 安全扫描

```bash
cm doctor --security
```

检测并警告：
- Docker 套接字挂载 (`/var/run/docker.sock`)
- 特权容器
- 敏感环境变量
- 过多的 capabilities

### 最佳实践

- 使用官方基础镜像
- 尽可能启用 Rootless 模式
- 除非必要，避免挂载 Docker 套接字
- 审查 `runArgs` 的安全影响

---

## ❓ 常见问题

<details>
<summary><b>问：Container-Maker 需要 VS Code 吗？</b></summary>

不需要！Container-Maker 是独立的 CLI 工具。通过 `cm code` 的 VS Code 集成是可选的。
</details>

<details>
<summary><b>问：可以用 Podman 代替 Docker 吗？</b></summary>

可以！使用 `cm backend switch podman` 切换运行时。
</details>

<details>
<summary><b>问：如何启用 GPU 支持？</b></summary>

1. 安装 NVIDIA Container Toolkit
2. 运行 `cm doctor` 验证
3. 使用 GPU 模板：`cm init --template pytorch`
</details>

<details>
<summary><b>问：我的文件在容器中存储在哪里？</b></summary>

默认情况下，您的项目目录挂载在 `/workspaces/<项目名>`。
</details>

<details>
<summary><b>问：如何在容器重启之间持久化数据？</b></summary>

在 mounts 配置中使用命名卷，或使用内置缓存系统。
</details>

---

## 🤝 贡献指南

我们欢迎贡献！

```bash
# Fork 并克隆
git clone https://github.com/UPwith-me/Container-Maker.git
cd Container-Maker

# 构建
go build -o cm ./cmd/cm

# 测试
go test ./...
```

---

## 📄 许可证

Container-Maker 采用双重许可模式：[AGPL-3.0](LICENSE)（开源使用）和 [商业许可](COMMERCIAL-LICENSE.md)（专有/商业使用）。

详见 [LICENSE](LICENSE) 和 [COMMERCIAL-LICENSE.md](COMMERCIAL-LICENSE.md)。

---

<div align="center">

Made with ❤️ by Container-Maker Team

[⬆ 回到顶部](#-container-maker)

</div>
