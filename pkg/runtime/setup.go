package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// HostInfo contains detected host information
type HostInfo struct {
	OS           string // windows, linux, darwin
	Arch         string // amd64, arm64
	Distro       string // ubuntu, debian, fedora, centos, alpine, etc.
	DistroVer    string // 22.04, 11, 38, etc.
	IsWSL        bool   // Windows Subsystem for Linux
	IsRoot       bool   // Running as root/admin
	HasDocker    bool   // Docker already installed
	HasPodman    bool   // Podman already installed
	DockerSocket string // Docker socket path if exists
}

// InstallOption represents an installation recommendation
type InstallOption struct {
	Name        string
	Description string
	Command     string
	Priority    int // Higher = more recommended
}

// DetectHost detects the current host environment
func DetectHost() *HostInfo {
	info := &HostInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Check WSL
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/version"); err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
				info.IsWSL = true
			}
		}
	}

	// Check root/admin
	if runtime.GOOS != "windows" {
		info.IsRoot = os.Geteuid() == 0
	}

	// Detect Linux distro
	if runtime.GOOS == "linux" {
		info.detectLinuxDistro()
	}

	// Check existing Docker
	if _, err := exec.LookPath("docker"); err == nil {
		info.HasDocker = true
		info.DockerSocket = "/var/run/docker.sock"
		if runtime.GOOS == "windows" {
			info.DockerSocket = "npipe:////./pipe/docker_engine"
		}
	}

	// Check existing Podman
	if _, err := exec.LookPath("podman"); err == nil {
		info.HasPodman = true
	}

	return info
}

// detectLinuxDistro detects the Linux distribution
func (h *HostInfo) detectLinuxDistro() {
	// Try /etc/os-release first
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				h.Distro = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				h.DistroVer = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}

	// Fallback detection
	if h.Distro == "" {
		if _, err := os.Stat("/etc/debian_version"); err == nil {
			h.Distro = "debian"
		} else if _, err := os.Stat("/etc/redhat-release"); err == nil {
			h.Distro = "rhel"
		} else if _, err := os.Stat("/etc/alpine-release"); err == nil {
			h.Distro = "alpine"
		}
	}
}

// GetInstallOptions returns recommended installation options for the host
func (h *HostInfo) GetInstallOptions() []InstallOption {
	var options []InstallOption

	switch h.OS {
	case "windows":
		options = h.getWindowsOptions()
	case "darwin":
		options = h.getMacOSOptions()
	case "linux":
		if h.IsWSL {
			options = h.getWSLOptions()
		} else {
			options = h.getLinuxOptions()
		}
	}

	return options
}

// getWindowsOptions returns Docker install options for Windows
func (h *HostInfo) getWindowsOptions() []InstallOption {
	return []InstallOption{
		{
			Name:        "Docker Desktop",
			Description: "官方 Docker Desktop for Windows (推荐)",
			Command:     `winget install Docker.DockerDesktop`,
			Priority:    100,
		},
		{
			Name:        "Rancher Desktop",
			Description: "开源替代品，支持 containerd/dockerd",
			Command:     `winget install suse.RancherDesktop`,
			Priority:    80,
		},
		{
			Name:        "Podman Desktop",
			Description: "Red Hat 的 Docker 替代品，无需守护进程",
			Command:     `winget install RedHat.Podman-Desktop`,
			Priority:    70,
		},
	}
}

// getMacOSOptions returns Docker install options for macOS
func (h *HostInfo) getMacOSOptions() []InstallOption {
	options := []InstallOption{
		{
			Name:        "Docker Desktop",
			Description: "官方 Docker Desktop for Mac (推荐)",
			Command:     `brew install --cask docker`,
			Priority:    100,
		},
		{
			Name:        "OrbStack",
			Description: "更快更轻量的 Docker 替代品 (macOS 专属)",
			Command:     `brew install --cask orbstack`,
			Priority:    95,
		},
		{
			Name:        "Colima",
			Description: "开源轻量级容器运行时",
			Command:     `brew install colima docker && colima start`,
			Priority:    80,
		},
		{
			Name:        "Podman",
			Description: "无守护进程的容器引擎",
			Command:     `brew install podman && podman machine init && podman machine start`,
			Priority:    70,
		},
	}

	// Recommend OrbStack for Apple Silicon
	if h.Arch == "arm64" {
		options[1].Priority = 100
		options[1].Description = "更快更轻量的 Docker 替代品 (Apple Silicon 推荐)"
		options[0].Priority = 90
	}

	return options
}

// getWSLOptions returns Docker install options for WSL
func (h *HostInfo) getWSLOptions() []InstallOption {
	return []InstallOption{
		{
			Name:        "Docker Desktop (Windows)",
			Description: "使用 Windows 宿主的 Docker Desktop (推荐)",
			Command:     `echo "请在 Windows 中安装 Docker Desktop 并启用 WSL 集成"`,
			Priority:    100,
		},
		{
			Name:        "Docker Engine (WSL 内)",
			Description: "在 WSL 内直接安装 Docker Engine",
			Command:     h.getDockerInstallCmd(),
			Priority:    80,
		},
		{
			Name:        "Podman",
			Description: "无守护进程的容器引擎",
			Command:     h.getPodmanInstallCmd(),
			Priority:    70,
		},
	}
}

// getLinuxOptions returns Docker install options for Linux
func (h *HostInfo) getLinuxOptions() []InstallOption {
	return []InstallOption{
		{
			Name:        "Docker Engine",
			Description: "官方 Docker Engine (推荐)",
			Command:     h.getDockerInstallCmd(),
			Priority:    100,
		},
		{
			Name:        "Podman",
			Description: "无守护进程的容器引擎，兼容 Docker CLI",
			Command:     h.getPodmanInstallCmd(),
			Priority:    80,
		},
	}
}

// getDockerInstallCmd returns the Docker install command for the current distro
func (h *HostInfo) getDockerInstallCmd() string {
	switch h.Distro {
	case "ubuntu", "debian", "linuxmint", "pop":
		return `curl -fsSL https://get.docker.com | sh && sudo usermod -aG docker $USER`
	case "fedora":
		return `sudo dnf install -y dnf-plugins-core && sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo && sudo dnf install -y docker-ce docker-ce-cli containerd.io && sudo systemctl enable --now docker && sudo usermod -aG docker $USER`
	case "centos", "rhel", "rocky", "almalinux":
		return `sudo yum install -y yum-utils && sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo && sudo yum install -y docker-ce docker-ce-cli containerd.io && sudo systemctl enable --now docker && sudo usermod -aG docker $USER`
	case "arch", "manjaro":
		return `sudo pacman -S docker && sudo systemctl enable --now docker && sudo usermod -aG docker $USER`
	case "alpine":
		return `sudo apk add docker && sudo rc-update add docker default && sudo service docker start && sudo addgroup $USER docker`
	case "opensuse", "sles":
		return `sudo zypper install -y docker && sudo systemctl enable --now docker && sudo usermod -aG docker $USER`
	default:
		return `curl -fsSL https://get.docker.com | sh && sudo usermod -aG docker $USER`
	}
}

// getPodmanInstallCmd returns the Podman install command for the current distro
func (h *HostInfo) getPodmanInstallCmd() string {
	switch h.Distro {
	case "ubuntu", "debian":
		return `sudo apt-get update && sudo apt-get install -y podman`
	case "fedora":
		return `sudo dnf install -y podman`
	case "centos", "rhel", "rocky", "almalinux":
		return `sudo yum install -y podman`
	case "arch", "manjaro":
		return `sudo pacman -S podman`
	case "alpine":
		return `sudo apk add podman`
	case "opensuse", "sles":
		return `sudo zypper install -y podman`
	default:
		return `# Please install Podman manually for your distribution`
	}
}

// FormatHostInfo returns a formatted string of host information
func (h *HostInfo) FormatHostInfo() string {
	var sb strings.Builder

	sb.WriteString("🖥️  主机信息\n")
	sb.WriteString(fmt.Sprintf("   操作系统: %s/%s\n", h.OS, h.Arch))

	if h.Distro != "" {
		distroInfo := h.Distro
		if h.DistroVer != "" {
			distroInfo += " " + h.DistroVer
		}
		sb.WriteString(fmt.Sprintf("   发行版:   %s\n", distroInfo))
	}

	if h.IsWSL {
		sb.WriteString("   环境:     WSL (Windows Subsystem for Linux)\n")
	}

	sb.WriteString("\n📦 容器运行时\n")
	if h.HasDocker {
		sb.WriteString("   Docker:   ✅ 已安装\n")
	} else {
		sb.WriteString("   Docker:   ❌ 未安装\n")
	}

	if h.HasPodman {
		sb.WriteString("   Podman:   ✅ 已安装\n")
	} else {
		sb.WriteString("   Podman:   ❌ 未安装\n")
	}

	return sb.String()
}
