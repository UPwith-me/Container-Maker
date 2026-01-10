package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ContainerManager handles remote container lifecycle
type ContainerManager struct {
	host    string
	sshOpts []string
}

// NewContainerManager creates a new container manager
func NewContainerManager(host string) *ContainerManager {
	return &ContainerManager{
		host:    host,
		sshOpts: []string{"-o", "ConnectTimeout=30", "-o", "BatchMode=yes"},
	}
}

// ContainerConfig holds configuration for creating a container
type ContainerConfig struct {
	Name      string
	Image     string
	WorkDir   string
	Env       map[string]string
	Ports     []int
	Volumes   []string
	GPU       bool
	Resources ResourceConfig
	Labels    map[string]string
}

// ResourceConfig holds resource limits
type ResourceConfig struct {
	Memory  string // e.g., "4g"
	CPUs    string // e.g., "2"
	ShmSize string // e.g., "2g"
}

// CreateContainer creates a new container on the remote host
func (cm *ContainerManager) CreateContainer(ctx context.Context, cfg *ContainerConfig) error {
	// Build docker run command
	args := []string{"docker", "run", "-d"}

	// Add name
	args = append(args, "--name", cfg.Name)

	// Add labels
	args = append(args, "--label", "cm.managed_by=container-maker")
	args = append(args, "--label", "cm.created_at="+currentTimestamp())
	for k, v := range cfg.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add working directory
	if cfg.WorkDir != "" {
		args = append(args, "-w", cfg.WorkDir)
	}

	// Add environment variables
	for k, v := range cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add port mappings
	for _, port := range cfg.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
	}

	// Add volumes
	for _, vol := range cfg.Volumes {
		args = append(args, "-v", vol)
	}

	// Add GPU support
	if cfg.GPU {
		args = append(args, "--gpus", "all")
	}

	// Add resource limits
	if cfg.Resources.Memory != "" {
		args = append(args, "--memory", cfg.Resources.Memory)
	}
	if cfg.Resources.CPUs != "" {
		args = append(args, "--cpus", cfg.Resources.CPUs)
	}
	if cfg.Resources.ShmSize != "" {
		args = append(args, "--shm-size", cfg.Resources.ShmSize)
	}

	// Keep container running
	args = append(args, "--restart", "unless-stopped")

	// Interactive/TTY for shell access
	args = append(args, "-it")

	// Add image
	args = append(args, cfg.Image)

	// Default command to keep container running
	args = append(args, "tail", "-f", "/dev/null")

	// Build SSH command
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, args...)

	fmt.Printf("📦 Creating container '%s' on %s...\n", cfg.Name, cm.host)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create container: %w\nOutput: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	fmt.Printf("✅ Container created: %s\n", containerID[:12])

	return nil
}

// CreateFromDevcontainer creates a container from a devcontainer.json config
func (cm *ContainerManager) CreateFromDevcontainer(ctx context.Context, projectDir, configPath string) error {
	// Read and parse devcontainer.json
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Extract configuration
	cfg := &ContainerConfig{
		Name:   fmt.Sprintf("cm-%s", filepath.Base(projectDir)),
		Env:    make(map[string]string),
		Labels: make(map[string]string),
	}

	// Get image
	if image, ok := config["image"].(string); ok {
		cfg.Image = image
	} else {
		cfg.Image = "mcr.microsoft.com/devcontainers/base:ubuntu"
	}

	// Get forward ports
	if ports, ok := config["forwardPorts"].([]interface{}); ok {
		for _, p := range ports {
			if port, ok := p.(float64); ok {
				cfg.Ports = append(cfg.Ports, int(port))
			}
		}
	}

	// Get container env
	if env, ok := config["containerEnv"].(map[string]interface{}); ok {
		for k, v := range env {
			if val, ok := v.(string); ok {
				cfg.Env[k] = val
			}
		}
	}

	// Get mounts
	if mounts, ok := config["mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mount, ok := m.(string); ok {
				cfg.Volumes = append(cfg.Volumes, mount)
			}
		}
	}

	// Check for GPU in runArgs
	if runArgs, ok := config["runArgs"].([]interface{}); ok {
		for _, arg := range runArgs {
			if argStr, ok := arg.(string); ok {
				if argStr == "--gpus" || strings.Contains(argStr, "gpu") {
					cfg.GPU = true
				}
			}
		}
	}

	// Add workspace volume
	remotePath := fmt.Sprintf("/workspace/%s", filepath.Base(projectDir))
	cfg.Volumes = append(cfg.Volumes, fmt.Sprintf("%s:%s", remotePath, remotePath))
	cfg.WorkDir = remotePath

	return cm.CreateContainer(ctx, cfg)
}

// EnsureContainer ensures a container exists and is running
func (cm *ContainerManager) EnsureContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	// Check if container exists
	exists, running, err := cm.ContainerStatus(ctx, cfg.Name)
	if err != nil {
		return "", err
	}

	if !exists {
		// Create new container
		if err := cm.CreateContainer(ctx, cfg); err != nil {
			return "", err
		}
		return cfg.Name, nil
	}

	if !running {
		// Start existing container
		fmt.Printf("🔄 Starting existing container '%s'...\n", cfg.Name)
		if err := cm.StartContainer(ctx, cfg.Name); err != nil {
			return "", err
		}
	}

	return cfg.Name, nil
}

// ContainerStatus checks if a container exists and is running
func (cm *ContainerManager) ContainerStatus(ctx context.Context, name string) (exists, running bool, err error) {
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, "docker", "inspect", "--format", "{{.State.Running}}", name)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	output, err := cmd.Output()
	if err != nil {
		// Container doesn't exist
		return false, false, nil
	}

	running = strings.TrimSpace(string(output)) == "true"
	return true, running, nil
}

// StartContainer starts a stopped container
func (cm *ContainerManager) StartContainer(ctx context.Context, name string) error {
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, "docker", "start", name)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	return cmd.Run()
}

// StopContainer stops a running container
func (cm *ContainerManager) StopContainer(ctx context.Context, name string) error {
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, "docker", "stop", name)

	fmt.Printf("🛑 Stopping container '%s'...\n", name)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	return cmd.Run()
}

// RemoveContainer removes a container
func (cm *ContainerManager) RemoveContainer(ctx context.Context, name string, force bool) error {
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, "docker", "rm")
	if force {
		sshArgs = append(sshArgs, "-f")
	}
	sshArgs = append(sshArgs, name)

	fmt.Printf("🗑️  Removing container '%s'...\n", name)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	return cmd.Run()
}

// ListContainers lists all CM-managed containers on the remote
func (cm *ContainerManager) ListContainers(ctx context.Context) ([]string, error) {
	sshArgs := append(cm.sshOpts, cm.host)
	sshArgs = append(sshArgs, "docker", "ps", "-a",
		"--filter", "label=cm.managed_by=container-maker",
		"--format", "{{.Names}}")

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var containers []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			containers = append(containers, line)
		}
	}

	return containers, nil
}

// ExecInContainer executes a command in a container
func (cm *ContainerManager) ExecInContainer(ctx context.Context, name string, command []string) *exec.Cmd {
	sshArgs := append(cm.sshOpts, cm.host)
	dockerArgs := append([]string{"docker", "exec", "-it", name}, command...)
	sshArgs = append(sshArgs, dockerArgs...)

	return exec.CommandContext(ctx, "ssh", append([]string{"-t"}, sshArgs...)...)
}

// SyncProjectToContainer syncs local project to container workspace
func (cm *ContainerManager) SyncProjectToContainer(ctx context.Context, localPath, containerName, remotePath string) error {
	fmt.Printf("📂 Syncing %s to container...\n", localPath)

	// First sync to remote host
	rsyncArgs := []string{
		"-avz", "--delete",
		"--exclude", ".git",
		"--exclude", "node_modules",
		"--exclude", "__pycache__",
		"--exclude", ".venv",
		"-e", "ssh",
		localPath + "/",
		fmt.Sprintf("%s:%s/", cm.host, remotePath),
	}

	cmd := exec.CommandContext(ctx, "rsync", rsyncArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync failed: %w", err)
	}

	fmt.Println("✅ Sync complete")
	return nil
}

// currentTimestamp returns current timestamp for labels
func currentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
