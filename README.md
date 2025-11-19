<div align="center">

<!-- TITLE & LOGO -->
<h1>
    <br>
    ⚡ CONTAINER-MAKE ⚡
    <br>
</h1>

<h3>The Developer Experience Tool for the Container Era</h3>
<h3>容器时代的极致开发体验工具</h3>

<p>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Built_with-Go_1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-ff5252?style=for-the-badge" alt="License"></a>
    <a href="#"><img src="https://img.shields.io/badge/Platform-Windows_|_Linux_|_macOS-181717?style=for-the-badge&logo=linux" alt="Platform"></a>
</p>

<p>
    <a href="#-english"><b>English</b></a> • <a href="#-chinese"><b>中文文档</b></a>
</p>

<br>

<!-- INTRO -->
<p align="center" style="max-width: 600px; margin: auto;">
    <b>Container-Make (cm)</b> transforms your <code>devcontainer.json</code> into a powerful CLI artifact.<br>
    It fuses the <b>speed</b> of Makefiles, the <b>isolation</b> of Docker, and the <b>intelligence</b> of VSCode DevContainers into a single, lethal binary.
</p>

<br>

</div>

<a id="-english"></a>

## 📖 English

### ✨ Why Container-Make?

<div align="center">
<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🎯 Single Source of Truth</h3>
      <p>Your <code>devcontainer.json</code> defines the universe. No more maintaining separate Dockerfiles or Makefiles for local dev.</p>
    </td>
    <td width="50%" valign="top">
      <h3>💎 Native Fidelity</h3>
      <p><code>vim</code>, <code>htop</code>, and interactive shells work exactly as they do locally. We handle the complex TTY and signal forwarding for you.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🚀 BuildKit Powered</h3>
      <p>Leverages Docker BuildKit for aggressive caching. Your environment spins up in seconds, not minutes.</p>
    </td>
    <td width="50%" valign="top">
      <h3>🛡️ Zero Pollution</h3>
      <p>Dependencies live in the container, not on your host OS. Keep your machine clean.</p>
    </td>
  </tr>
</table>
</div>

### 🛠️ Workflow

#### 1. Install
Build from source or download the binary.

```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
```

#### 2. Initialize
Generate shell aliases for a seamless experience.

```bash
./cm init
# Follow the on-screen instructions to update your .bashrc/.zshrc
```

#### 3. Execute
Go to any project with a `.devcontainer` folder and run commands.

```bash
# Prepare the environment (Pre-build image)
cm prepare

# Run any command inside the container
cm run -- go build -o app main.go
cm run -- npm install
cm run -- python train.py

# Drop into an interactive shell
cm run -- /bin/bash
```

### ⚙️ Configuration
We support the standard `devcontainer.json` specification.

```jsonc
// .devcontainer/devcontainer.json
{
    "image": "mcp/firecrawl:latest",
    "forwardPorts": [8080],
    "containerEnv": {
        "APP_ENV": "development"
    },
    "postStartCommand": "echo 'Ready to code!'"
}
```

<br>
<div align="center">---</div>
<br>

<a id="-chinese"></a>

## 🇨🇳 中文文档

**Container-Make (cm)** 将您的 `devcontainer.json` 转化为一个强大的命令行工具。它集成了 **Makefile** 的极致速度、**Docker** 的绝对隔离以及 **DevContainers** 的现代开发体验。

### ✨ 核心价值

<div align="center">
<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🎯 单一真理来源</h3>
      <p>使用 <code>devcontainer.json</code> 定义整个开发宇宙。无需再为本地开发维护额外的 Dockerfile 或 Makefile。</p>
    </td>
    <td width="50%" valign="top">
      <h3>💎 原生级保真</h3>
      <p><code>vim</code>、<code>htop</code> 和交互式 Shell 的体验与宿主机完全一致。我们为您处理了复杂的 TTY 和信号转发。</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🚀 BuildKit 驱动</h3>
      <p>利用 Docker BuildKit 的激进缓存策略。环境启动仅需秒级，而非分钟级。</p>
    </td>
    <td width="50%" valign="top">
      <h3>🛡️ 零环境污染</h3>
      <p>所有依赖均活在容器内，保持宿主机纯净。告别 "it works on my machine"。</p>
    </td>
  </tr>
</table>
</div>

### 🛠️ 工作流

#### 1. 安装
从源码编译或下载二进制文件。

```bash
git clone https://github.com/container-make/cm.git
cd cm && go build -o cm ./cmd/cm
```

#### 2. 初始化
生成 Shell 别名，获得无缝体验。

```bash
./cm init
# 按照屏幕提示更新您的 .bashrc 或 .zshrc
```

#### 3. 执行
进入任何包含 `.devcontainer` 文件夹的项目即可执行。

```bash
# 准备环境 (预构建镜像)
cm prepare

# 在容器内运行任意命令
cm run -- go build -o app main.go
cm run -- npm install
cm run -- python train.py

# 进入交互式终端
cm run -- /bin/bash
```

### ⚙️ 配置指南
我们支持标准的 `devcontainer.json` 规范。

```jsonc
// .devcontainer/devcontainer.json
{
    // 基础镜像
    "image": "mcp/firecrawl:latest",

    // 端口自动转发 (映射到 localhost)
    "forwardPorts": [8080],

    // 注入环境变量
    "containerEnv": {
        "APP_ENV": "development"
    },

    // 生命周期钩子
    "postStartCommand": "echo '环境已就绪！'"
}
```

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
