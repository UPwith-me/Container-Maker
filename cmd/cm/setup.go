package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/container-make/cm/pkg/runtime"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "一键安装 Docker 容器环境",
	Long: `智能检测您的系统环境并推荐最佳的容器运行时安装方案。

支持的平台:
  - Windows (Docker Desktop, Rancher Desktop, Podman Desktop)
  - macOS (Docker Desktop, OrbStack, Colima, Podman)
  - Linux (Docker Engine, Podman)
  - WSL (Windows Docker 或独立安装)

示例:
  cm setup              # 交互式安装向导
  cm setup --detect     # 仅检测环境，不安装
  cm setup --auto       # 自动安装推荐方案`,
	RunE: runSetup,
}

var (
	setupDetectOnly bool
	setupAuto       bool
)

func init() {
	setupCmd.Flags().BoolVar(&setupDetectOnly, "detect", false, "仅检测环境，不执行安装")
	setupCmd.Flags().BoolVar(&setupAuto, "auto", false, "自动安装推荐的容器运行时")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Container-Maker 环境配置向导")
	fmt.Println()

	// Detect host
	fmt.Println("🔍 正在检测系统环境...")
	host := runtime.DetectHost()
	fmt.Println()
	fmt.Println(host.FormatHostInfo())

	// Check if already installed
	if host.HasDocker || host.HasPodman {
		fmt.Println("✅ 已检测到容器运行时，无需安装！")
		fmt.Println()

		// Run doctor to verify
		if host.HasDocker {
			fmt.Println("💡 运行 'cm doctor' 检查 Docker 状态")
		}
		return nil
	}

	if setupDetectOnly {
		fmt.Println("💡 使用 'cm setup' 开始安装容器运行时")
		return nil
	}

	// Get installation options
	options := host.GetInstallOptions()
	if len(options) == 0 {
		fmt.Println("❌ 无法为您的系统提供安装建议")
		return nil
	}

	// Sort by priority
	sort.Slice(options, func(i, j int) bool {
		return options[i].Priority > options[j].Priority
	})

	// Auto mode: install the first (highest priority) option
	if setupAuto {
		fmt.Printf("🔧 自动安装: %s\n", options[0].Name)
		return executeInstall(options[0])
	}

	// Interactive mode
	fmt.Println("📋 推荐的安装选项:")
	fmt.Println()

	for i, opt := range options {
		marker := "  "
		if i == 0 {
			marker = "⭐"
		}
		fmt.Printf("%s [%d] %s\n", marker, i+1, opt.Name)
		fmt.Printf("      %s\n", opt.Description)
		fmt.Println()
	}

	fmt.Print("请选择安装选项 (1-", len(options), ") 或 'q' 退出: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "q" || input == "Q" {
		fmt.Println("已取消")
		return nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		fmt.Println("❌ 无效选择")
		return nil
	}

	return executeInstall(options[choice-1])
}

func executeInstall(opt runtime.InstallOption) error {
	fmt.Println()
	fmt.Printf("🔧 正在安装 %s...\n", opt.Name)
	fmt.Println()
	fmt.Println("📝 执行命令:")
	fmt.Printf("   %s\n", opt.Command)
	fmt.Println()

	// Confirm
	fmt.Print("确认执行？[Y/n] ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "" && input != "y" && input != "yes" {
		fmt.Println("已取消")
		return nil
	}

	// Detect shell and execute
	var cmd *exec.Cmd

	switch {
	case isWindows():
		// PowerShell on Windows
		cmd = exec.Command("powershell", "-Command", opt.Command)
	default:
		// Bash on Unix
		cmd = exec.Command("bash", "-c", opt.Command)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("\n❌ 安装失败: %v\n", err)
		fmt.Println()
		fmt.Println("💡 请尝试手动执行上述命令，或检查错误信息")
		return nil
	}

	fmt.Println()
	fmt.Println("✅ 安装完成!")
	fmt.Println()
	fmt.Println("📋 后续步骤:")
	fmt.Println("   1. 如果安装了 Docker Desktop，请启动应用程序")
	fmt.Println("   2. 运行 'cm doctor' 验证安装")
	fmt.Println("   3. 运行 'cm shell' 开始使用容器开发环境")

	if !isWindows() {
		fmt.Println()
		fmt.Println("⚠️  注意: 如果添加了 docker 用户组，需要重新登录或运行:")
		fmt.Println("   newgrp docker")
	}

	return nil
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") ||
		strings.Contains(strings.ToLower(os.Getenv("GOOS")), "windows")
}
