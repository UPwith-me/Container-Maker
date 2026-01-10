# container-maker v3.1.0: Industrial Edition 🏭

**Release Date**: 2026-01-10
**Tag**: `v3.1.0` (Updated)

This major release transforms Container-Maker from a development tool into an **Industrial-Grade Workspace Platform**. It introduces strict security controls, offline capabilities, and powerful extension mechanisms.

## 🚀 Key Features

### 1. Environment Snapshots (`cm snapshot`)
*   **Instant Rollback**: Capture the exact state (filesystem + memory) of your dev container.
*   **Versioning**: Name and tag snapshots (e.g., `pre-upgrade`, `stable-v1`).
*   **Integration**: Works interchangeably with Docker and Podman commits.

### 2. Offline Air-Gap Support (`cm export` / `cm load`)
*   **Single-File Bundles**: Export the entire project environment (Image + Config + Code) into a `.cm` archive.
*   **Zero-Dependency Restore**: Restore perfectly on a machine with no internet access.
*   **Streaming Architecture**: Handles multi-GB environments with minimal memory footprint.

### 3. Integrated Security Scanning (`cm scan`)
*   **Trivy Powered**: Built-in vulnerability scanner for container images.
*   **Severity Filters**: Fail builds on `CRITICAL` or `HIGH` CVEs.
*   **Actionable Reports**: JSON/Table outputs with fix recommendations.

### 4. Plugin System & Extensibility (`cm plugin`)
*   **Process-Based Plugins**: Extend CLI capabilities using any language (Go, Python, Bash).
*   **Dynamic Discovery**: Drop executables into `~/.cm/plugins/` to register new commands.
*   **Lifecycle Hooks**: Inject custom logic before/after container start.

### 5. Intelligent Resource Profiling (`cm profile`)
*   **AI Analytics**: Monitor CPU/Memory usage and suggest optimal `devcontainer.json` limits.
*   **P95 Algorithm**: Weighted sliding window analysis for accurate recommendations.

### 6. Global Configuration (`cm config`)
*   **Atomic Persistence**: Robust configuration management with safe writes.
*   **Update Channels**: Switch between `stable` and `beta` update streams.

## 🛠️ Enhancements & Fixes
*   **Security**: Patched Zip Slip vulnerability in `cm load`. Use strict path sanitization.
*   **Security**: Patched PowerShell Injection in `path_setup.go`.
*   **Reliability**: Added `RemoveImage` capability to Runtime interface for cleaner teardowns.
*   **UX**: Improved `cm setup` path detection on Windows.

## 📦 Installation

```bash
# Windows
irm https://github.com/UPwith-me/Container-Maker/releases/download/v3.1.0/cm-windows-amd64.exe -OutFile cm.exe

# Linux/macOS
curl -L https://github.com/UPwith-me/Container-Maker/releases/download/v3.1.0/cm-linux-amd64 -o cm
chmod +x cm
```

**Full Changelog**: https://github.com/UPwith-me/Container-Maker/compare/v3.0.0...v3.1.0
