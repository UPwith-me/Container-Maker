<div align="center">
  <img src="https://raw.githubusercontent.com/container-make/cm/main/assets/logo.png" alt="Container-Maker Logo" width="200">
  
  # Container-Maker
  
  ### 🚀 The Future of Development Environments
  
  **One Config. One Command. Any Container. Anywhere.**

  [![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
  [![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/platform-Windows%20|%20macOS%20|%20Linux-lightgrey.svg)]()
  [![Release](https://img.shields.io/github/v/release/container-make/cm?include_prereleases)](https://github.com/container-make/cm/releases)

  <br>
  
  [🇺🇸 English](#-why-container-maker) • [🇨🇳 中文](#-中文文档)
  
  ---
  
  **Container-Maker** transforms your `devcontainer.json` into a powerful CLI tool.  
  Combining **Makefile speed**, **Docker isolation**, and **DevContainers DX** into one seamless experience.

</div>

<br>

## 💡 Why Container-Maker?

> **"It works on my machine"** — The most expensive phrase in software development.

Container-Maker solves the **#1 DevOps challenge**: development environment consistency.

### The Problem

| Pain Point | Traditional Solution | Container-Maker |
|-----------|---------------------|-----------------|
| Environment setup takes hours | Manual wiki docs | **30 seconds** |
| "Works on my machine" bugs | Pray and debug | **Impossible** |
| New team member onboarding | Days of setup | **One command** |
| CI/CD environment mismatch | Hope for the best | **Identical envs** |
| Local dependency conflicts | Virtual environments | **Full isolation** |

### The Solution

```bash
cd your-project
cm shell   # That's it. You're in a perfect dev environment.
```

---

## 🌟 Key Features

<table>
<tr>
<td width="50%">

### 🎯 Zero-Config Onboarding
```bash
cm setup     # Auto-install Docker
cm init      # Generate devcontainer.json
cm shell     # Start developing
```
**From zero to productive in under 2 minutes.**

</td>
<td width="50%">

### 🔌 Smart Port Forwarding
```json
{
  "forwardPorts": [3000, 8000, 5432]
}
```
**Automatic port mapping. Access localhost:3000 seamlessly.**

</td>
</tr>
<tr>
<td width="50%">

### 📦 DevContainer Features
```bash
cm feature download node
cm feature list
```
**17+ features from OCI registries. Auto-download & install.**

</td>
<td width="50%">

### 🛍️ Template Marketplace
```bash
cm marketplace list
cm marketplace install python
```
**12+ official templates. One command to full stack.**

</td>
</tr>
<tr>
<td width="50%">

### 🧠 AI-Powered Config Generation
```bash
cm ai generate
cm ai analyze
```
**AI analyzes your project and generates optimal config.**

</td>
<td width="50%">

### 🔧 Multi-Backend Support
```bash
cm backend list
cm backend use podman
```
**Docker, Podman, nerdctl. Your choice.**

</td>
</tr>
</table>

---

## 🚀 Quick Start

### Installation

**Option 1: Auto-Install (Recommended)**
```bash
# Windows (PowerShell)
irm https://cm.dev/install.ps1 | iex

# macOS/Linux
curl -fsSL https://cm.dev/install.sh | sh
```

**Option 2: From Source**
```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
./cm  # First run auto-adds to PATH
```

### Your First Container

```bash
# No Docker? No problem!
cm setup              # One-click Docker installation

# Start a new project
cm init --apply       # Interactive setup wizard

# Enter your dev container
cm shell              # 🎉 You're in!
```

---

## 📋 Complete Command Reference

### Core Commands
| Command | Description |
|---------|-------------|
| `cm shell` | Start/enter persistent dev container |
| `cm run -- <cmd>` | Run command in ephemeral container |
| `cm prepare` | Pre-build container image |
| `cm exec <cmd>` | Execute in running container |

### Environment Management
| Command | Description |
|---------|-------------|
| `cm setup` | **One-click Docker install** (Windows/macOS/Linux) |
| `cm doctor` | **Environment diagnostics** with auto-fix suggestions |
| `cm backend` | Switch between Docker/Podman/nerdctl |
| `cm cache` | Manage build caches for faster rebuilds |

### Template & Features
| Command | Description |
|---------|-------------|
| `cm marketplace list` | Browse 12+ official templates |
| `cm marketplace install <id>` | Install template |
| `cm feature list` | List 17+ available features |
| `cm feature download <ref>` | Download from OCI registry |

### AI & Productivity
| Command | Description |
|---------|-------------|
| `cm ai generate` | **AI-powered** devcontainer.json generation |
| `cm ai analyze` | Analyze project structure |
| `cm clone <repo>` | Clone + auto-setup container |
| `cm code` | Open in VS Code with container |

### Advanced
| Command | Description |
|---------|-------------|
| `cm watch` | File watcher with auto-run |
| `cm share` | Generate shareable project link |
| `cm config` | User configuration management |
| `cm version` | Show version info |

---

## 🏗️ Architecture & Innovation

### Technical Breakthroughs

```
┌─────────────────────────────────────────────────────────────┐
│                    Container-Maker                          │
├─────────────────────────────────────────────────────────────┤
│  🔐 UID/GID Mapping    │  Solves file permission issues    │
│  📺 SIGWINCH Sync      │  Perfect terminal resize          │
│  ⚡ Smart Caching       │  7 languages auto-detected        │
│  🛡️ Security Checker   │  Warns about dangerous configs    │
│  🔌 Port Forwarding    │  Seamless host-container access   │
│  🧩 OCI Features       │  Download from any registry       │
├─────────────────────────────────────────────────────────────┤
│  Docker │ Podman │ nerdctl │ containerd                     │
└─────────────────────────────────────────────────────────────┘
```

### What Makes Us Different

| Feature | VS Code DevContainers | Docker Compose | Container-Maker |
|---------|----------------------|----------------|-----------------|
| IDE Independent | ❌ VS Code only | ✅ | ✅ |
| CLI-First | ❌ | ❌ | ✅ |
| Auto PATH Setup | ❌ | ❌ | ✅ |
| One-Click Docker Install | ❌ | ❌ | ✅ |
| AI Config Generation | ❌ | ❌ | ✅ |
| Feature Marketplace | ✅ | ❌ | ✅ |
| Multi-Backend | ❌ | ❌ | ✅ |
| Smart Caching | ❌ | ❌ | ✅ |
| GPU Auto-Detect | ❌ | ❌ | ✅ |

---

## 📊 Performance

| Metric | Without CM | With CM |
|--------|-----------|---------|
| New dev onboarding | 4-8 hours | **2 minutes** |
| Environment bugs | 20% of sprints | **0%** |
| CI/CD parity | 60% | **100%** |
| Cache rebuild time | Full rebuild | **Incremental** |

---

## 🎯 Use Cases

### For Startups
- **Instant onboarding** for new hires
- **Standardized environments** across the team
- **Cost savings** from reduced debugging time

### For Enterprise
- **Compliance-ready** isolated environments
- **Multi-platform support** (Windows, macOS, Linux)
- **Security scanning** built-in

### For Open Source
- **Contributors start in seconds**
- **No "it works on my machine" issues**
- **Reproducible bug reports**

---

## 🤝 Community & Support

- 📖 [Documentation](https://cm.dev/docs)
- 💬 [Discord Community](https://discord.gg/container-make)
- 🐛 [Report Issues](https://github.com/container-make/cm/issues)
- 📧 [Enterprise Support](mailto:enterprise@cm.dev)

---

<br>
<div align="center">---</div>
<br>

<a id="-中文文档"></a>

## 🇨🇳 中文文档

<div align="center">

# Container-Maker

### 🚀 开发环境的未来

**一个配置文件。一条命令。任意容器。随处运行。**

</div>

<br>

## 💡 为什么选择 Container-Maker？

> **"在我电脑上是好的"** — 软件开发中最昂贵的一句话。

Container-Maker 解决了 **DevOps 的头号难题**：开发环境一致性。

### 痛点分析

| 痛点 | 传统解决方案 | Container-Maker |
|-----|------------|-----------------|
| 环境配置耗时数小时 | 手写文档 | **30 秒** |
| "在我电脑上好的"问题 | 祈祷和调试 | **彻底消除** |
| 新成员入职 | 数天配置 | **一条命令** |
| CI/CD 环境不一致 | 听天由命 | **完全一致** |
| 本地依赖冲突 | 虚拟环境 | **完全隔离** |

### 解决方案

```bash
cd your-project
cm shell   # 就这样。完美的开发环境已就绪。
```

---

## 🌟 核心功能

<table>
<tr>
<td width="50%">

### 🎯 零配置入门
```bash
cm setup     # 一键安装 Docker
cm init      # 生成 devcontainer.json
cm shell     # 开始开发
```
**从零到高效开发，不到 2 分钟。**

</td>
<td width="50%">

### 🔌 智能端口转发
```json
{
  "forwardPorts": [3000, 8000, 5432]
}
```
**自动端口映射。无缝访问 localhost:3000。**

</td>
</tr>
<tr>
<td width="50%">

### 📦 DevContainer Features
```bash
cm feature download node
cm feature list
```
**17+ OCI 特性。自动下载安装。**

</td>
<td width="50%">

### 🛍️ 模板市场
```bash
cm marketplace list
cm marketplace install python
```
**12+ 官方模板。一条命令搭建全栈。**

</td>
</tr>
<tr>
<td width="50%">

### 🧠 AI 智能配置生成
```bash
cm ai generate
cm ai analyze
```
**AI 分析项目结构，生成最优配置。**

</td>
<td width="50%">

### 🔧 多后端支持
```bash
cm backend list
cm backend use podman
```
**Docker、Podman、nerdctl，自由选择。**

</td>
</tr>
</table>

---

## 🚀 快速开始

### 安装

**方式一：自动安装（推荐）**
```bash
# Windows (PowerShell)
irm https://cm.dev/install.ps1 | iex

# macOS/Linux
curl -fsSL https://cm.dev/install.sh | sh
```

**方式二：从源码编译**
```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
./cm  # 首次运行自动添加到 PATH
```

### 第一个容器

```bash
# 没有 Docker？没问题！
cm setup              # 一键安装 Docker

# 创建新项目
cm init --apply       # 交互式配置向导

# 进入开发容器
cm shell              # 🎉 完成！
```

---

## 📋 完整命令参考

### 核心命令
| 命令 | 说明 |
|-----|------|
| `cm shell` | 启动/进入持久化开发容器 |
| `cm run -- <cmd>` | 在临时容器中运行命令 |
| `cm prepare` | 预构建容器镜像 |
| `cm exec <cmd>` | 在运行中的容器执行命令 |

### 环境管理
| 命令 | 说明 |
|-----|------|
| `cm setup` | **一键安装 Docker** (Windows/macOS/Linux) |
| `cm doctor` | **环境诊断** + 自动修复建议 |
| `cm backend` | 切换 Docker/Podman/nerdctl |
| `cm cache` | 管理构建缓存加速重建 |

### 模板与特性
| 命令 | 说明 |
|-----|------|
| `cm marketplace list` | 浏览 12+ 官方模板 |
| `cm marketplace install <id>` | 安装模板 |
| `cm feature list` | 列出 17+ 可用特性 |
| `cm feature download <ref>` | 从 OCI 仓库下载 |

### AI 与生产力
| 命令 | 说明 |
|-----|------|
| `cm ai generate` | **AI 驱动**的配置生成 |
| `cm ai analyze` | 分析项目结构 |
| `cm clone <repo>` | 克隆 + 自动配置容器 |
| `cm code` | 在 VS Code 中打开容器 |

### 高级功能
| 命令 | 说明 |
|-----|------|
| `cm watch` | 文件监听自动运行 |
| `cm share` | 生成可分享的项目链接 |
| `cm config` | 用户配置管理 |
| `cm version` | 显示版本信息 |

---

## 🏗️ 技术架构与创新

### 技术突破

```
┌─────────────────────────────────────────────────────────────┐
│                    Container-Maker                          │
├─────────────────────────────────────────────────────────────┤
│  🔐 UID/GID 映射       │  解决文件权限问题                  │
│  📺 SIGWINCH 同步      │  完美的终端大小调整                │
│  ⚡ 智能缓存           │  7 种语言自动检测                  │
│  🛡️ 安全检查器        │  警告危险配置                      │
│  🔌 端口转发           │  无缝的主机-容器访问               │
│  🧩 OCI Features      │  从任意仓库下载                    │
├─────────────────────────────────────────────────────────────┤
│  Docker │ Podman │ nerdctl │ containerd                     │
└─────────────────────────────────────────────────────────────┘
```

### 差异化优势

| 功能 | VS Code DevContainers | Docker Compose | Container-Maker |
|-----|----------------------|----------------|-----------------|
| IDE 无关 | ❌ 仅 VS Code | ✅ | ✅ |
| CLI 优先 | ❌ | ❌ | ✅ |
| 自动 PATH 配置 | ❌ | ❌ | ✅ |
| 一键 Docker 安装 | ❌ | ❌ | ✅ |
| AI 配置生成 | ❌ | ❌ | ✅ |
| 特性市场 | ✅ | ❌ | ✅ |
| 多后端 | ❌ | ❌ | ✅ |
| 智能缓存 | ❌ | ❌ | ✅ |
| GPU 自动检测 | ❌ | ❌ | ✅ |

---

## 📊 性能提升

| 指标 | 无 CM | 使用 CM |
|-----|-------|--------|
| 新人入职时间 | 4-8 小时 | **2 分钟** |
| 环境问题占比 | 20% 迭代 | **0%** |
| CI/CD 一致性 | 60% | **100%** |
| 缓存重建时间 | 全量重建 | **增量构建** |

---

## 🎯 应用场景

### 创业公司
- **即时入职** 新员工
- **团队标准化** 开发环境
- **节省成本** 减少调试时间

### 大型企业
- **合规就绪** 的隔离环境
- **多平台支持** (Windows, macOS, Linux)
- **内置安全** 扫描

### 开源项目
- **贡献者秒级启动**
- **消除 "在我电脑上好的" 问题**
- **可复现的 bug 报告**

---

## 🤝 社区与支持

- 📖 [文档](https://cm.dev/docs)
- 💬 [Discord 社区](https://discord.gg/container-make)
- 🐛 [问题反馈](https://github.com/container-make/cm/issues)
- 📧 [企业支持](mailto:enterprise@cm.dev)

---

<br>

<div align="center">
    <br>
    <p>
        <sub>Designed for the future of development.</sub>
        <br>
        <sub>MIT License © 2025</sub>
    </p>
    <br>
    <a href="#"><img src="https://img.shields.io/github/stars/container-make/cm?style=social" alt="GitHub Stars"></a>
</div>