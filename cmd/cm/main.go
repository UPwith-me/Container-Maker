package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/config"
	"github.com/UPwith-me/Container-Maker/pkg/detect"
	"github.com/UPwith-me/Container-Maker/pkg/images"
	mkpkg "github.com/UPwith-me/Container-Maker/pkg/make"
	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/UPwith-me/Container-Maker/pkg/runtime"
	"github.com/UPwith-me/Container-Maker/pkg/template"
	"github.com/UPwith-me/Container-Maker/pkg/tui"
	"github.com/UPwith-me/Container-Maker/pkg/watch"
	"github.com/spf13/cobra"
)

// Version info - set by build flags
var (
	Version   = "2.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "cm",
	Short: "Container-Maker: The Ultimate Developer Experience for Containers",
	Long: `Container-Maker (cm) is a CLI tool that bridges the gap between local Makefiles
and containerized build environments. It reads devcontainer.json configurations
and executes commands in ephemeral or persistent containers.

QUICK START
  cm init              Initialize a new project with templates
  cm shell             Open interactive shell in the container
  cm run make build    Run any command inside the container

ENVIRONMENT MANAGEMENT
  cm setup             Auto-install Docker/Podman
  cm doctor            Diagnose your development environment
  cm status            View running containers (TUI dashboard)

ADVANCED FEATURES
  cm ai generate       AI-powered config generation
  cm marketplace       Browse and install templates
  cm cloud             Manage cloud development environments

EXAMPLES
  # Start a new Python project
  $ cm init --template python

  # Run tests inside the container
  $ cm run pytest tests/

  # Open VS Code in the container
  $ cm code

  # Deploy to cloud
  $ cm cloud deploy --provider aws`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Only show welcome on init command
		if cmd.Name() == "init" {
			tui.RenderWelcome()
		}
		// Check PATH setup on first run (only for root command)
		if cmd.Name() == "cm" {
			tui.CheckAndSetupPath()
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show smart home screen when cm is run without arguments
		return tui.RunHomeScreen()
	},
}

var runCmd = &cobra.Command{

	Use:   "run [command]",
	Short: "Run a command inside the dev container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default config paths
		if configFile == "" {
			if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
				configFile = ".devcontainer/devcontainer.json"
			} else if _, err := os.Stat("devcontainer.json"); err == nil {
				configFile = "devcontainer.json"
			} else {
				return fmt.Errorf("no devcontainer.json found")
			}
		}

		cfg, err := config.ParseConfig(configFile)
		if err != nil {
			return err
		}

		// Check if using Docker Compose
		if runner.IsComposeConfig(cfg) {
			projectDir := filepath.Dir(configFile)
			cr, err := runner.NewComposeRunner(cfg, projectDir)
			if err != nil {
				return err
			}
			return cr.Run(context.Background(), args)
		}

		// Standard container mode
		r, err := runner.NewRunner(cfg)
		if err != nil {
			return err
		}

		return r.Run(context.Background(), args)
	},
}

var prepareCmd = &cobra.Command{
	Use:   "prepare",
	Short: "Build the dev container image",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default config paths
		if configFile == "" {
			if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
				configFile = ".devcontainer/devcontainer.json"
			} else if _, err := os.Stat("devcontainer.json"); err == nil {
				configFile = "devcontainer.json"
			} else {
				return fmt.Errorf("no devcontainer.json found")
			}
		}

		cfg, err := config.ParseConfig(configFile)
		if err != nil {
			return err
		}

		// Check if using Docker Compose
		if runner.IsComposeConfig(cfg) {
			projectDir := filepath.Dir(configFile)
			cr, err := runner.NewComposeRunner(cfg, projectDir)
			if err != nil {
				return err
			}
			return cr.Prepare(context.Background())
		}

		// Standard container mode
		r, err := runner.NewRunner(cfg)
		if err != nil {
			return err
		}

		// Resolve image (Build/Pull + Features)
		tag, err := r.ResolveImage(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("Successfully prepared image: %s\n", tag)

		return nil
	},
}

var applyShell bool
var shellType string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project or generate shell scripts",
	Long:  `Initialize a new DevContainer project or generate shell integration scripts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --apply or --shell is used, run shell integration logic
		if applyShell || cmd.Flags().Changed("shell") {
			return runShellIntegration(cmd, args)
		}

		// Otherwise, run the interactive wizard
		fmt.Println("🚀 Initializing new DevContainer project...")
		template, err := tui.RunInitWizard()
		if err != nil {
			return err
		}

		if template == "" {
			return nil // Cancelled
		}

		// Check if devcontainer.json already exists
		configPath := ".devcontainer/devcontainer.json"
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("⚠️  %s already exists. Overwrite? [y/N] ", configPath)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Generate config content
		content := tui.GenerateConfig(template)

		// Create directory
		if err := os.MkdirAll(".devcontainer", 0755); err != nil {
			return fmt.Errorf("failed to create .devcontainer directory: %w", err)
		}

		// Write file
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		tui.RenderBox("Success!", fmt.Sprintf("Created %s\nSelected Template: %s", configPath, template))
		return nil
	},
}

func runShellIntegration(_ *cobra.Command, _ []string) error {
	// Shell integration script content
	shellScript := `
# Container-Maker Shell Integration
alias devcontainer='cm run'
function dcm() {
  cm run --config .devcontainer/devcontainer.json -- "$@"
}
# End Container-Maker Integration
`
	fishScript := `
# Container-Maker Shell Integration
alias devcontainer 'cm run'
function dcm
  cm run --config .devcontainer/devcontainer.json -- $argv
end
# End Container-Maker Integration
`

	if !applyShell {
		// Just print the script
		fmt.Println("# Add this to your shell configuration (.bashrc, .zshrc, etc.)")
		fmt.Println(shellScript)
		fmt.Println("# For Fish shell, use:")
		fmt.Println(fishScript)
		return nil
	}

	// Auto-detect shell type if not specified
	detectedShell := shellType
	if detectedShell == "" {
		shellEnv := os.Getenv("SHELL")
		if strings.Contains(shellEnv, "zsh") {
			detectedShell = "zsh"
		} else if strings.Contains(shellEnv, "fish") {
			detectedShell = "fish"
		} else {
			detectedShell = "bash"
		}
	}

	// Determine config file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	var configPath string
	var script string
	switch detectedShell {
	case "zsh":
		configPath = filepath.Join(homeDir, ".zshrc")
		script = shellScript
	case "fish":
		configPath = filepath.Join(homeDir, ".config", "fish", "config.fish")
		script = fishScript
	default: // bash
		configPath = filepath.Join(homeDir, ".bashrc")
		script = shellScript
	}

	// Check if already integrated
	existingContent, err := os.ReadFile(configPath)
	if err == nil && strings.Contains(string(existingContent), "Container-Maker Shell Integration") {
		fmt.Printf("Shell integration already exists in %s\n", configPath)
		return nil
	}

	// Ensure parent directory exists (for fish)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Append to config file
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + script); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	fmt.Printf("Successfully added Container-Maker integration to %s\n", configPath)
	fmt.Printf("Detected shell: %s\n", detectedShell)
	fmt.Println("Please restart your shell or run: source " + configPath)
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running container status dashboard",
	Long:  `Launch an interactive dashboard to view running containers, their stats, ports, and access logs or shell.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunStatusDashboard()
	},
}

var shellStop bool
var shellRebuild bool
var shellPause bool
var shellResume bool

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start or enter a persistent dev container",
	Long: `Start a persistent dev container and enter an interactive shell.

Flags:
  --stop     Stop and remove the container
  --pause    Save container state and stop (frees memory, preserves environment)
  --resume   Restore from saved snapshot
  --rebuild  Rebuild the container from scratch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}

		pr, err := runner.NewPersistentRunner(cfg, projectDir)
		if err != nil {
			return err
		}

		if shellStop {
			return pr.Stop(context.Background())
		}

		if shellPause {
			return pr.Pause(context.Background())
		}

		if shellResume {
			return pr.Resume(context.Background())
		}

		if cmd.Flags().Changed("status") {
			pr.Status(context.Background())
			return nil
		}

		return pr.Shell(context.Background())
	},
}

var execCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute a command in the persistent container",
	Long:  `Execute a command in the persistent dev container. If no container is running, one will be started automatically.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}

		pr, err := runner.NewPersistentRunner(cfg, projectDir)
		if err != nil {
			return err
		}

		return pr.Exec(context.Background(), args)
	},
}

// loadConfig loads the devcontainer.json and returns config and project directory
// loadConfig loads the devcontainer.json and returns config and project directory
// If no config exists, it triggers auto-detection
func loadConfig() (*config.DevContainerConfig, string, error) {
	projectDir, _ := os.Getwd()
	configPath := configFile

	// Try to find existing config
	if configPath == "" {
		if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
			configPath = ".devcontainer/devcontainer.json"
		} else if _, err := os.Stat("devcontainer.json"); err == nil {
			configPath = "devcontainer.json"
		}
	}

	// If config exists, use it
	if configPath != "" {
		cfg, err := config.ParseConfig(configPath)
		if err != nil {
			return nil, "", err
		}

		if strings.Contains(configPath, ".devcontainer") {
			projectDir = filepath.Dir(filepath.Dir(configPath))
			if projectDir == "" || projectDir == "." {
				projectDir, _ = os.Getwd()
			}
		}

		return cfg, projectDir, nil
	}

	// No config found - try auto-detection
	return loadConfigWithAutoDetect(projectDir)
}

// loadConfigWithAutoDetect uses project type detection to create a temporary config
func loadConfigWithAutoDetect(projectDir string) (*config.DevContainerConfig, string, error) {
	result := detect.DetectProjectType(projectDir)

	if result.Primary == nil {
		// No project detected - use TUI quickstart
		if err := tui.RunQuickStart(); err != nil {
			return nil, "", err
		}

		// Check if config was created
		if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
			cfg, err := config.ParseConfig(".devcontainer/devcontainer.json")
			if err != nil {
				return nil, "", err
			}
			return cfg, projectDir, nil
		}

		return nil, "", fmt.Errorf("setup cancelled")
	}

	// Prompt user
	image, saveConfig, err := detect.PromptAutoDetect(result)
	if err != nil {
		return nil, "", err
	}

	// Save config if requested
	if saveConfig {
		if err := detect.CreateDevcontainerConfig(projectDir, image, result.Primary.Language); err != nil {
			fmt.Printf("⚠️  Failed to create devcontainer.json: %v\n", err)
		} else {
			fmt.Println("✅ Created .devcontainer/devcontainer.json")
		}
	}

	// Create temporary config
	cfg := &config.DevContainerConfig{
		Image: image,
	}

	return cfg, projectDir, nil
}

func main() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(prepareCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(execCmd)

	runCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")
	prepareCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")
	initCmd.Flags().BoolVarP(&applyShell, "apply", "a", false, "Automatically apply shell integration to config file")
	initCmd.Flags().StringVarP(&shellType, "shell", "s", "", "Shell type (bash, zsh, fish). Auto-detected if not specified")

	shellCmd.Flags().BoolVar(&shellStop, "stop", false, "Stop the persistent container")
	shellCmd.Flags().BoolVar(&shellRebuild, "rebuild", false, "Rebuild the container")
	shellCmd.Flags().BoolVar(&shellPause, "pause", false, "Save container state and stop (frees memory)")
	shellCmd.Flags().BoolVar(&shellResume, "resume", false, "Restore from saved snapshot")
	shellCmd.Flags().Bool("status", false, "Show persistent container status")
	shellCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")

	execCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")

	makeCmd.Flags().BoolVar(&makeList, "list", false, "List available Makefile targets")
	makeCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")

	Execute()
}

var makeList bool

var makeCmd = &cobra.Command{
	Use:   "make [target...]",
	Short: "Run Makefile targets in the dev container",
	Long: `Run make targets inside the dev container.

Examples:
  cm make              # Run default target
  cm make build        # Run build target
  cm make clean build  # Run multiple targets
  cm make test V=1     # Pass variables to make
  cm make --list       # List available targets`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for Makefile
		cwd, _ := os.Getwd()
		if !mkpkg.HasMakefile(cwd) {
			return fmt.Errorf("No Makefile found in current directory.\nHint: Create a Makefile or use 'cm exec make ...' for custom paths")
		}

		// Handle --list flag
		if makeList {
			makefilePath, _ := mkpkg.FindMakefile(cwd)
			info, err := mkpkg.ParseMakefile(makefilePath)
			if err != nil {
				return fmt.Errorf("failed to parse Makefile: %w", err)
			}
			fmt.Println(mkpkg.ListTargets(info))
			return nil
		}

		// Load config
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}

		// Show current image info
		if cfg.Image != "" {
			fmt.Printf("📋 Using image: %s\n", cfg.Image)
		}

		// Create persistent runner
		pr, err := runner.NewPersistentRunner(cfg, projectDir)
		if err != nil {
			return err
		}

		// Build make command
		makeArgs := []string{"make"}
		makeArgs = append(makeArgs, args...)

		// Execute in container
		err = pr.Exec(context.Background(), makeArgs)

		// Check for 'make not found' error and provide helpful hints
		if err != nil && strings.Contains(err.Error(), "127") {
			fmt.Println("\n⚠️  'make' is not installed in this container image.")
			fmt.Println("\n💡 Suggested solutions:")
			fmt.Println("   1. Use an image with make pre-installed:")
			fmt.Println("      • gcc:latest (C/C++ projects)")
			fmt.Println("      • golang:latest (Go projects)")
			fmt.Println("      • node:latest (Node.js projects)")
			fmt.Println("      • mcr.microsoft.com/devcontainers/base:debian")
			fmt.Println("\n   2. Install make in your current container:")
			fmt.Println("      cm exec apt-get update && apt-get install -y make")
			fmt.Println("      cm shell --pause  # Save the changes")
			fmt.Printf("\n   Current image: %s\n", cfg.Image)
			return nil // Don't show the raw error
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(makeCmd)
}

// Images command group
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage preset development images",
	Long:  `Manage preset development images for quick switching between environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: list images
		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}
		images.UpdateDownloadedStatus(cfg)
		fmt.Println(images.ListImages(cfg))
		return nil
	},
}

var imagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available images",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}
		images.UpdateDownloadedStatus(cfg)
		fmt.Println(images.ListImages(cfg))
		return nil
	},
}

var imagesSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run interactive setup wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		selected, err := images.RunSetupWizard()
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Println("Setup cancelled.")
			return nil
		}
		return images.PullSelectedImages(selected)
	},
}

var imagesUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch current project to use specified image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}

		preset, found := images.GetImage(cfg, name)
		if !found {
			return fmt.Errorf("image '%s' not found. Use 'cm images' to see available images", name)
		}

		// Update devcontainer.json
		devcontainerPath := ".devcontainer/devcontainer.json"
		if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
			// Create directory
			os.MkdirAll(".devcontainer", 0755)
		}

		// Write simple config
		content := fmt.Sprintf(`{
  "name": "%s",
  "image": "%s"
}`, name, preset.Image)

		if err := os.WriteFile(devcontainerPath, []byte(content), 0644); err != nil {
			return err
		}

		fmt.Printf("✅ Switched to '%s' (%s)\n", name, preset.Image)
		fmt.Printf("   Updated %s\n", devcontainerPath)

		// Stop any running container (config changed)
		fmt.Println("\n💡 Run 'cm shell' or 'cm exec' to use the new image")

		return nil
	},
}

var imagesAddCmd = &cobra.Command{
	Use:   "add <name> <image:tag>",
	Short: "Add a custom image",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		imageName := args[1]

		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}

		if err := images.AddCustomImage(cfg, name, imageName); err != nil {
			return err
		}

		fmt.Printf("✅ Added custom image '%s' (%s)\n", name, imageName)
		return nil
	},
}

var imagesPullCmd = &cobra.Command{
	Use:   "pull <name>",
	Short: "Download an image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}

		preset, found := images.GetImage(cfg, name)
		if !found {
			return fmt.Errorf("image '%s' not found", name)
		}

		if err := images.PullImage(preset.Image); err != nil {
			return err
		}

		preset.Downloaded = true
		images.SaveConfig(cfg)

		return nil
	},
}

var imagesRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a custom image from the list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := images.LoadConfig()
		if err != nil {
			return err
		}

		if err := images.RemoveCustomImage(cfg, name); err != nil {
			return err
		}

		fmt.Printf("✅ Removed '%s' from custom images\n", name)
		return nil
	},
}

func init() {
	imagesCmd.AddCommand(imagesListCmd)
	imagesCmd.AddCommand(imagesSetupCmd)
	imagesCmd.AddCommand(imagesUseCmd)
	imagesCmd.AddCommand(imagesAddCmd)
	imagesCmd.AddCommand(imagesPullCmd)
	imagesCmd.AddCommand(imagesRemoveCmd)
	rootCmd.AddCommand(imagesCmd)
}

// Watch command variables
var watchExtensions string
var watchIgnore string
var watchDelay int
var watchClear bool
var watchNoInitial bool

var watchCmd = &cobra.Command{
	Use:   "watch [flags] -- <command>",
	Short: "Watch for file changes and auto-run commands",
	Long: `Watch for file changes and automatically re-run commands in the container.

Examples:
  cm watch -- go test ./...        # Watch and run tests
  cm watch -- npm run build        # Watch and build
  cm watch --ext go,mod -- go test # Only watch .go and .mod files
  cm watch --delay 500 -- make     # 500ms debounce delay
  cm watch --clear -- go build     # Clear screen before each run`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}

		// Parse options
		opts := watch.DefaultOptions()
		opts.ProjectDir = projectDir
		opts.Config = cfg
		opts.Clear = watchClear
		opts.InitialRun = !watchNoInitial

		if watchExtensions != "" {
			opts.Extensions = strings.Split(watchExtensions, ",")
		}

		if watchIgnore != "" {
			opts.IgnoreDirs = append(opts.IgnoreDirs, strings.Split(watchIgnore, ",")...)
		}

		if watchDelay > 0 {
			opts.Delay = time.Duration(watchDelay) * time.Millisecond
		}

		// Create watcher
		w, err := watch.New(opts, args)
		if err != nil {
			return err
		}
		defer w.Close()

		// Handle Ctrl+C gracefully
		ctx, cancel := context.WithCancel(context.Background())
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n👋 Stopping watcher...")
			cancel()
		}()

		// Start watching
		return w.Start(ctx)
	},
}

func init() {
	watchCmd.Flags().StringVar(&watchExtensions, "ext", "", "File extensions to watch (comma-separated, e.g., go,mod)")
	watchCmd.Flags().StringVar(&watchIgnore, "ignore", "", "Additional directories to ignore (comma-separated)")
	watchCmd.Flags().IntVar(&watchDelay, "delay", 300, "Debounce delay in milliseconds")
	watchCmd.Flags().BoolVar(&watchClear, "clear", false, "Clear screen before each run")
	watchCmd.Flags().BoolVar(&watchNoInitial, "no-initial", false, "Don't run command on startup")
	watchCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to devcontainer.json")
	rootCmd.AddCommand(watchCmd)
}

// Template command group
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage devcontainer templates",
	Long:  `Browse and use pre-configured devcontainer templates for various project types.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(template.ListTemplates())
		return nil
	},
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(template.ListTemplates())
		return nil
	},
}

var templateUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Apply a template to current project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cwd, _ := os.Getwd()

		// Get template info first
		info, err := template.TemplateInfo(name)
		if err != nil {
			return err
		}
		fmt.Println(info)

		// Apply template
		fmt.Println("Creating .devcontainer/devcontainer.json...")
		if err := template.ApplyTemplate(name, cwd); err != nil {
			return err
		}

		fmt.Println("✅ Template applied!")
		fmt.Println()
		fmt.Println("Run 'cm shell' to start developing.")

		return nil
	},
}

var templateInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show template details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := template.TemplateInfo(args[0])
		if err != nil {
			return err
		}
		fmt.Println(info)
		return nil
	},
}

var templateSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save current config as a custom template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cwd, _ := os.Getwd()

		fmt.Printf("📦 Saving current config as template '%s'...\n", name)

		if err := template.SaveTemplate(name, cwd); err != nil {
			return err
		}

		fmt.Println("✅ Template saved!")
		fmt.Println()
		fmt.Printf("Use 'cm template use %s' in other projects.\n", name)

		return nil
	},
}

var templateRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a custom template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := template.RemoveTemplate(name); err != nil {
			return err
		}

		fmt.Printf("✅ Template '%s' removed.\n", name)
		return nil
	},
}

var templateSearchGPU bool
var templateSearchCategory string

var templateSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search templates by name or description",
	Long: `Search templates with optional filters.

Examples:
  cm template search python     # Search for "python"
  cm template search --gpu      # Show GPU-required templates
  cm template search --category "Deep Learning"
  cm template search ml --gpu   # Combined search`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := template.SearchOptions{
			GPUOnly:  templateSearchGPU,
			Category: templateSearchCategory,
		}
		if len(args) > 0 {
			opts.Query = args[0]
		}

		results := template.SearchTemplates(opts)
		fmt.Println(template.FormatSearchResults(results, opts.Query))
		return nil
	},
}

func init() {
	templateSearchCmd.Flags().BoolVar(&templateSearchGPU, "gpu", false, "Show only GPU-required templates")
	templateSearchCmd.Flags().StringVar(&templateSearchCategory, "category", "", "Filter by category")

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateUseCmd)
	templateCmd.AddCommand(templateInfoCmd)
	templateCmd.AddCommand(templateSaveCmd)
	templateCmd.AddCommand(templateRemoveCmd)
	templateCmd.AddCommand(templateSearchCmd)
	rootCmd.AddCommand(templateCmd)
}

// Backend command group
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Manage container runtime backends",
	Long:  `Manage container runtime backends like Docker, Podman, and custom runtimes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBackends()
	},
}

var backendListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backends",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBackends()
	},
}

func listBackends() error {
	detector := runtime.NewDetector()
	result := detector.Detect()

	fmt.Println("📦 Container Backends")
	fmt.Println()

	if len(result.Backends) == 0 {
		fmt.Println("  No container runtimes found.")
		fmt.Println()
		fmt.Println("Install Docker or Podman:")
		fmt.Println("  Docker:  https://docker.com/get-started")
		fmt.Println("  Podman:  https://podman.io/getting-started")
		return nil
	}

	// Sort backends: active first, then by name
	backends := result.Backends
	sort.Slice(backends, func(i, j int) bool {
		if backends[i].IsActive != backends[j].IsActive {
			return backends[i].IsActive
		}
		return backends[i].Name < backends[j].Name
	})

	fmt.Printf("  %-8s %-12s %-10s %s\n", "Status", "Name", "Version", "Path")
	fmt.Printf("  %-8s %-12s %-10s %s\n", "──────", "────────────", "──────────", "────────────────────")

	for _, b := range backends {
		status := "○ Ready"
		if b.IsActive {
			status = "● Active"
		} else if !b.Running {
			status = "✗ Stopped"
		}

		name := b.Name
		if b.IsCustom {
			name += " [custom]"
		}

		version := b.Version
		if version == "" {
			version = "-"
		}

		fmt.Printf("  %-8s %-12s %-10s %s\n", status, name, version, b.Path)
	}

	fmt.Println()
	if result.Active != nil {
		fmt.Printf("Current: %s\n", result.Active.Name)
	}
	fmt.Println("Switch with: cm backend use <name>")

	return nil
}

var backendUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a specific backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		detector := runtime.NewDetector()

		// Verify backend exists and is running
		result := detector.Detect()
		var found *runtime.BackendInfo
		for _, b := range result.Backends {
			if b.Name == name {
				found = &b
				break
			}
		}

		if found == nil {
			return fmt.Errorf("backend '%s' not found. Use 'cm backend' to see available backends", name)
		}

		if !found.Running {
			fmt.Printf("⚠️  Backend '%s' is not running.\n", name)
			if found.Type == "docker" {
				fmt.Println("\nTo start Docker:")
				fmt.Println("  Windows/Mac: Open Docker Desktop")
				fmt.Println("  Linux:       sudo systemctl start docker")
			}
			return fmt.Errorf("backend not running")
		}

		if err := detector.SetPreferred(name); err != nil {
			return err
		}

		fmt.Printf("✅ Switched to %s\n", name)
		return nil
	},
}

var backendAddCmd = &cobra.Command{
	Use:   "add <name> <path>",
	Short: "Add a custom backend",
	Long: `Add a custom container runtime backend.

Examples:
  cm backend add docker-dev /opt/docker/bin/docker
  cm backend add podman-rootless ~/.local/bin/podman`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := args[1]

		// Detect type from path
		typ := "docker"
		if strings.Contains(strings.ToLower(path), "podman") {
			typ = "podman"
		} else if strings.Contains(strings.ToLower(path), "nerdctl") {
			typ = "nerdctl"
		}

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}

		detector := runtime.NewDetector()
		if err := detector.AddCustomBackend(name, path, typ); err != nil {
			return err
		}

		fmt.Printf("✅ Added custom backend '%s' (%s)\n", name, path)
		fmt.Printf("   Type: %s\n", typ)
		fmt.Println()
		fmt.Printf("Use with: cm backend use %s\n", name)

		return nil
	},
}

var backendRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a custom backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		detector := runtime.NewDetector()

		if err := detector.RemoveCustomBackend(name); err != nil {
			return err
		}

		fmt.Printf("✅ Removed backend '%s'\n", name)
		return nil
	},
}

var backendDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Re-detect all available backends",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Detecting container runtimes...")
		fmt.Println()
		return listBackends()
	},
}

func init() {
	backendCmd.AddCommand(backendListCmd)
	backendCmd.AddCommand(backendUseCmd)
	backendCmd.AddCommand(backendAddCmd)
	backendCmd.AddCommand(backendRemoveCmd)
	backendCmd.AddCommand(backendDetectCmd)
	rootCmd.AddCommand(backendCmd)
}

// Doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment issues",
	Long: `Run diagnostic checks on your development environment.

Checks include:
  • Container runtime (Docker/Podman)
  • GPU support (NVIDIA/AMD)
  • Network connectivity
  • Disk space
  • Docker Compose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🩺 Container-Make Doctor")
		fmt.Println("========================")
		fmt.Println()

		results := runtime.RunDiagnostics()

		for _, r := range results {
			var icon string
			switch r.Status {
			case "ok":
				icon = "✅"
			case "warning":
				icon = "⚠️"
			case "error":
				icon = "❌"
			default:
				icon = "•"
			}

			fmt.Printf("%s %s: %s\n", icon, r.Name, r.Message)
			if r.Details != "" {
				fmt.Printf("   %s\n", r.Details)
			}
			if r.Fix != "" {
				fmt.Printf("   💡 %s\n", r.Fix)
			}
			fmt.Println()
		}

		// Summary
		okCount := 0
		warnCount := 0
		errCount := 0
		for _, r := range results {
			switch r.Status {
			case "ok":
				okCount++
			case "warning":
				warnCount++
			case "error":
				errCount++
			}
		}

		fmt.Println("────────────────────────")
		if errCount > 0 {
			fmt.Printf("❌ %d error(s), %d warning(s), %d ok\n", errCount, warnCount, okCount)
		} else if warnCount > 0 {
			fmt.Printf("⚠️  %d warning(s), %d ok\n", warnCount, okCount)
		} else {
			fmt.Printf("✅ All %d checks passed!\n", okCount)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
