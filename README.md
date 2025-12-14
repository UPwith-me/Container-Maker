<div align="center">

<img src="assets/logo.png" width="300" alt="Container-Maker Logo" />

<h1>
    <br>
    ⚡ CONTAINER-MAKER ⚡
    <br>
</h1>

<h3>The Ultimate Developer Experience Platform for the Container Era</h3>
<h3>容器时代的终极开发体验平台</h3>

<p>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Built_with-Go_1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-ff5252?style=for-the-badge" alt="License"></a>
    <a href="#"><img src="https://img.shields.io/badge/Platform-Windows_|_Linux_|_macOS-181717?style=for-the-badge&logo=linux" alt="Platform"></a>
</p>

<p>
    <a href="#-english"><b>English</b></a> • <a href="#-chinese"><b>中文白皮书</b></a>
</p>

<br>

<p align="center" style="max-width: 700px; margin: auto;">
    <b>Container-Maker (cm)</b> redefines the local development lifecycle. It isn't just a CLI; it's an <b>infrastructure-as-code agent</b> that instantly turns any machine into a production-grade development studio.
    <br><br>
    By fusing the <b>speed of Makefiles</b>, the <b>isolation of Docker</b>, and the <b>intelligence of VS Code DevContainers</b>, it delivers a zero-configuration, reproducible, and blazing-fast development experience.
</p>

<br>

</div>

---

<a id="-english"></a>

## 📖 English

### 🚀 Innovation Highlights

Container-Maker solves the "works on my machine" problem once and for all, while introducing groundbreaking ease-of-use features:

*   **⚡ Zero-Config Onboarding**: New users simply run `cm setup` to auto-detect their OS and install the optimal container engine (Docker/Podman).
*   **🔌 Smart Agent Integration**: Automatically detects its first run, adds itself to the system PATH, and refreshes the shell session instantly—no restart required.
*   **🤖 AI-Driven Configuration**: Integrated AI engine (`cm ai generate`) analyzes your project and builds the perfect `devcontainer.json` automatically.
*   **🌐 Universal Portability**: One configuration works across Windows, Linux, macOS, and WSL2. We handle the complex TTY signals, UID/GID mapping, and socket forwarding transparently.
*   **🛡️ Enterprise-Grade Security**: Built-in security scanner warns about dangerous mounts (`docker.sock`, privileged mode) and facilitates Rootless Docker adoption.
*   **📦 Intelligent Caching**: Automatic persistent caching for major languages (Go, Rust, Node, Python, C++, Java, .NET) accelerates incremental builds by up to 10x.

### 💎 Core Value Proposition

<div align="center">
<table>
  <tr>
    <td width="33%" valign="top">
      <h3>🎯 Single Source of Truth</h3>
      <p><b>Configuration as Code.</b> Your <code>devcontainer.json</code> defines the entire universe. No more maintaining separate Dockerfiles, Makefiles, or shell scripts for local dev.</p>
    </td>
    <td width="33%" valign="top">
      <h3>💎 Native Fidelity</h3>
      <p><b>Seamless Integration.</b> <code>vim</code>, <code>htop</code>, and interactive shells work exactly as they do locally. We engineered a custom signal proxy to handle window resizing (SIGWINCH) and interrupts perfectly.</p>
    </td>
    <td width="33%" valign="top">
      <h3>🚀 BuildKit Powered</h3>
      <p><b>Blazing Speed.</b> Leverages Docker BuildKit for aggressive layer caching. Your environment spins up in seconds, not minutes, with intelligent pre-building.</p>
    </td>
  </tr>
</table>
</div>

### 🌟 Feature Ecosystem

#### 1. Smart Environment Management
*   **Auto PATH Integration**: On first launch, `cm` intelligently offers to register itself globally, handling PowerShell/Bash PATH updates and session refreshing automatically.
*   **Full-Stack Environment**: One command (`cm shell`) spins up complex stacks including Databases (PostgreSQL, Redis), Vector DBs (Qdrant), and Monitoring (Prometheus/Grafana) via seamless Docker Compose integration.
*   **Environment Diagnostics**: `cm doctor` performs deep checks on GPU drivers, network connectivity, disk space, and runtime health.

#### 2. Intelligence & Automation
*   **AI Config Generator**: `cm ai generate` uses LLMs to inspect your codebase and generate optimized, best-practice DevContainer configurations.
*   **Template Marketplace**: Instant access to 17+ curated templates for AI/ML (PyTorch, TensorFlow), Web (React, Node), and Systems (Rust, Go).
    *   `cm marketplace search --gpu` to find GPU-accelerated templates.

#### 3. Seamless Developer Experience
*   **DevContainer Features (OCI)**: Fully supports the DevContainer Features spec. Download and install tools (e.g., `ghcr.io/devcontainers/features/go`) directly from OCI registries.
*   **TUI Dashboard**: A beautiful, interactive Terminal UI (`cm status`) to monitor running containers, logs, and resource usage.
*   **Smart Port Forwarding**: Automatic detection and forwarding of ports defined in `forwardPorts`, supporting TCP/UDP and range mapping.

#### 4. Performance & Security
*   **Incremental Build Cache**: Language-aware caching strategies mount compiler caches (`/go/pkg`, `node_modules`, `.m2`) into containers automatically.
*   **Security Guardrails**: Proactive scanning for security risks. Alerts on privileged containers or sensitive mount points.
*   **Rootless Ready**: Fully compatible with Rootless Docker and Podman security contexts.

### 🛠️ Quick Start

#### Installation

Build from source (Go 1.21+ required):
```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
```

#### First Run Experience

Simply run `cm` to trigger the **Smart Setup Wizard**:

```bash
./cm
```

It will:
1.  Detect your OS context.
2.  Offer to add `cm` to your global PATH.
3.  Refresh your current shell session automatically.

#### One-Click Environment Setup

Validating a new machine? Use our intelligent setup tool:

```bash
cm setup --auto
```

This will automatically install and configure Docker Desktop, Podman, or Rancher Desktop based on your system profile.

### 📦 Usage Examples

**1. Instant Dev Environment:**
```bash
cd my-project
cm shell   # Parses devcontainer.json and drops you into a fully configured shell
```

**2. Running Commands:**
```bash
cm run -- go test ./...       # Run tests in container
cm run -- npm run build       # Build frontend
cm run -- python train.py     # Train AI model (with GPU support)
```

**3. Manage Dependencies (Features):**
```bash
cm feature download ghcr.io/devcontainers/features/node:1
cm feature list
```

**4. Explore Templates:**
```bash
cm marketplace list
cm template use pytorch
```

### 📋 Command Reference

| Category | Command | Description |
|----------|---------|-------------|
| **Core** | `cm shell` | Start/enter persistent development container |
| | `cm run` | Execute one-off command in container |
| | `cm setup` | Auto-install Docker/Container runtime |
| | `cm init` | Initialize project wizard |
| **AI & Templates** | `cm ai generate` | AI-generated configuration |
| | `cm marketplace` | Browse/Install community templates |
| | `cm template` | Manage local templates |
| **Features** | `cm feature` | OCI Feature download & management |
| **Ops & Status** | `cm status` | Interactive TUI dashboard |
| | `cm doctor` | System health & diagnostic check |
| | `cm cache` | Manage build caches & persistence |
| **Config** | `cm config` | Global configuration management |
| | `cm backend` | Switch between Docker/Podman |

---

<a id="-chinese"></a>

## 🇨🇳 中文白皮书

### 🚀 创新与突破

Container-Maker (cm) 不仅仅是一个工具，它是专为解决“在我的机器上能跑”这一世纪难题而生的**基础设施即代码（IaC）智能代理**。它引入了多项突破性技术：

*   **⚡ 零配置智能引导**: 新用户只需运行 `cm setup`，系统即会自动检测操作系统环境，并一键部署最优的容器运行时（Docker/Podman），真正实现开箱即用。
*   **🔌 智能代理集成**: 首次运行时自动检测，主动请求将 `cm` 添加到系统全局 PATH，并能即时刷新当前的 PowerShell/Bash 会话，无需重启终端即可生效。
*   **🤖 AI 驱动的配置生成**: 内置 AI 引擎 (`cm ai generate`) 可深入分析您的项目源代码，自动构建符合最佳实践的 `devcontainer.json` 开发环境配置。
*   **🌐 全平台无缝兼容**: 一套配置，通用 Windows、Linux、macOS 和 WSL2。我们在底层攻克了 TTY 信号透传、UID/GID 动态映射、Socket 转发等技术难题，确保原生般的体验。
*   **🛡️ 企业级安全防护**: 内置安全扫描器，实时检测危险挂载（如 `docker.sock`）、特权模式等风险，并完美支持 Rootless Docker 架构。
*   **📦 智能增量构建缓存**: 针对主流语言（Go, Rust, Node, Python, C++, Java, .NET）实现了智能缓存挂载策略，将增量构建速度提升最高 10 倍。

### 💎 核心价值主张

<div align="center">
<table>
  <tr>
    <td width="33%" valign="top">
      <h3>🎯 单一真理来源</h3>
      <p><b>配置即一切。</b> 使用简单的 <code>devcontainer.json</code> 定义整个开发宇宙。彻底告别维护复杂的 Dockerfile、Makefile 或本地脚本的时代。</p>
    </td>
    <td width="33%" valign="top">
      <h3>💎 原生级极致体验</h3>
      <p><b>无感集成。</b> <code>vim</code>、<code>htop</code> 和交互式 Shell 的体验与宿主机完全一致。我们独创的信号代理技术完美解决了窗口缩放 (SIGWINCH) 和中断信号的同步问题。</p>
    </td>
    <td width="33%" valign="top">
      <h3>🚀 BuildKit 极速引擎</h3>
      <p><b>秒级启动。</b> 深度集成 Docker BuildKit，利用激进的层级缓存策略。环境启动仅需秒级，让开发者的灵感不再被等待打断。</p>
    </td>
  </tr>
</table>
</div>

### 🌟 功能生态全景

#### 1. 智能环境管理体系
*   **自动 PATH 集成与会话刷新**: 智能识别首次运行状态，提供一键式全局 PATH 注册功能。支持 PowerShell 和 Unix Shell 的会话级环境变量动态刷新，真正做到安装即用。
*   **全栈环境编排**: 通过 `cm shell` 可一键拉起包含数据库 (PostgreSQL, Redis)、向量引擎 (Qdrant)、监控系统 (Prometheus/Grafana) 的复杂微服务架构。
*   **环境全维诊断**: `cm doctor` 提供专家级的环境体检，覆盖 GPU 驱动状态、网络连通性、磁盘配额及运行时健康度。

#### 2. 智能化与自动化
*   **AI 配置生成器**: 利用大语言模型能力，`cm ai generate` 能够理解您的代码逻辑，生成最匹配的开发容器配置。
*   **模板市场**: 内置 17+ 款精心调优的官方模板，覆盖 AI/ML (PyTorch, TensorFlow)、Web 全栈、系统编程 (Rust, Go) 等领域。
    *   支持 `cm marketplace search --gpu` 快速筛选 GPU 加速模板。

#### 3. 卓越的开发者体验
*   **DevContainer Features (OCI)**: 完整支持 OCI 标准的 DevContainer Features。可直接从 Ghcr.io 等注册表下载并安装工具链（如 Go, Node, K8s 工具），支持版本锁定与参数配置。
*   **TUI 交互式仪表盘**: 提供极具科技感的终端用户界面 (`cm status`)，实时监控容器状态、日志流及系统资源占用。
*   **智能端口转发**: 能够解析并自动转发 `forwardPorts` 定义的端口，支持 TCP/UDP 协议及端口范围映射。

#### 4. 极致性能与安全
*   **语言感知型缓存**: 自动识别项目语言并挂载相应的编译器缓存目录（如 `/go/pkg`, `node_modules`, `.m2`），显著加速重复构建过程。
*   **安全合规护栏**: 主动式安全审计功能，对特权容器、敏感路径挂载进行实时警告。
*   **Rootless 架构支持**: 完美适配无根 Docker (Rootless Docker) 及 Podman 安全上下文，满足企业级合规要求。

### 🛠️ 快速开始

#### 安装

从源码编译 (需要 Go 1.21+):
```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
```

#### 初次运行体验

直接运行 `cm` 即可触发 **智能设置向导**：

```bash
./cm
```

系统将自动：
1.  识别您的操作系统环境。
2.  请求并配置全局 PATH 环境变量。
3.  即时刷新当前 Shell 会话，让 `cm` 命令全局可用。

#### 一键环境部署

在新机器上配置开发环境？使用我们的智能部署工具：

```bash
cm setup --auto
```

该命令将根据您的系统配置，自动下载并安装最佳匹配的 Docker Desktop、Podman 或 Rancher Desktop。

### 📦 使用范例

**1. 瞬间进入开发环境:**
```bash
cd my-project
cm shell   # 自动解析配置，启动并在毫秒级进入持久化开发容器
```

**2. 在容器内执行命令:**
```bash
cm run -- go test ./...       # 在隔离环境中运行测试
cm run -- npm run build       # 构建前端资产
cm run -- python train.py     # 训练 AI 模型 (自动调用 GPU)
```

**3. 管理环境扩展 (Features):**
```bash
cm feature download ghcr.io/devcontainers/features/node:1  # 从 OCI 源下载 Node.js 环境
cm feature list                                            # 查看已安装的扩展
```

**4. 探索官方模板:**
```bash
cm marketplace list        # 浏览模板市场
cm template use pytorch    # 应用 PyTorch 深度学习模板
```

### 📋 命令速查手册

| 类别 | 命令 | 功能描述 |
|------|------|----------|
| **核心功能** | `cm shell` | 启动或进入持久化开发容器 |
| | `cm run` | 在容器中执行一次性命令 |
| | `cm setup` | 智能自动化安装 Docker/容器运行时 |
| | `cm init` | 交互式项目初始化向导 |
| **AI 与模板** | `cm ai generate` | AI 智能生成项目配置 |
| | `cm marketplace` | 浏览与安装社区/官方模板 |
| | `cm template` | 管理本地模板库 |
| **扩展管理** | `cm feature` | OCI Features 下载与生命周期管理 |
| **运维与监控** | `cm status` | 交互式 TUI 状态仪表盘 |
| | `cm doctor` | 系统环境全维诊断专家 |
| | `cm cache` | 构建缓存管理与持久化 |
| **配置** | `cm config` | 全局用户配置管理 |
| | `cm backend` | 容器运行时切换 (Docker/Podman) |

<br>

<!-- FOOTER -->
<div align="center">
    <br>
    <p>
        <sub>Designed for the future of development.</sub>
        <br>
        <sub>MIT License © 2025 Devin HE</sub>
    </p>
    <br>
    <a href="#"><img src="https://img.shields.io/github/stars/container-make/cm?style=social" alt="GitHub Stars"></a>
</div>