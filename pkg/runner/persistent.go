package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/container-make/cm/pkg/config"
	"github.com/container-make/cm/pkg/runtime"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/term"
)

// PersistentRunner manages persistent dev containers
type PersistentRunner struct {
	Client     *client.Client // Keep for backward compatibility
	Runtime    runtime.ContainerRuntime
	Config     *config.DevContainerConfig
	StateFile  string
	ProjectDir string
	Backend    string // "docker", "podman", etc.
}

// ContainerState stores the state of a persistent container
type ContainerState struct {
	ContainerID   string    `json:"containerId"`
	ContainerName string    `json:"containerName"`
	CreatedAt     time.Time `json:"createdAt"`
	ConfigHash    string    `json:"configHash"`
	ImageTag      string    `json:"imageTag"`
	SnapshotImage string    `json:"snapshotImage,omitempty"` // Saved snapshot image
	IsPaused      bool      `json:"isPaused,omitempty"`      // Container was paused (snapshot saved)
	Backend       string    `json:"backend,omitempty"`       // Which backend was used
}

// NewPersistentRunner creates a new persistent runner
func NewPersistentRunner(cfg *config.DevContainerConfig, projectDir string) (*PersistentRunner, error) {
	stateFile := filepath.Join(projectDir, ".devcontainer", ".cm-state.json")

	// Try to get the active runtime
	rt, err := runtime.GetActiveRuntime()
	if err != nil {
		// Fall back to Docker client directly for backward compatibility
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, err
		}

		return &PersistentRunner{
			Client:     cli,
			Config:     cfg,
			StateFile:  stateFile,
			ProjectDir: projectDir,
			Backend:    "docker",
		}, nil
	}

	// Use the runtime interface
	return &PersistentRunner{
		Runtime:    rt,
		Config:     cfg,
		StateFile:  stateFile,
		ProjectDir: projectDir,
		Backend:    rt.Name(),
	}, nil
}

// GetContainerName returns the container name for this project
func (r *PersistentRunner) GetContainerName() string {
	projectName := filepath.Base(r.ProjectDir)
	// Sanitize name for Docker
	projectName = strings.ToLower(projectName)
	projectName = strings.ReplaceAll(projectName, " ", "-")
	return fmt.Sprintf("cm-%s-dev", projectName)
}

// GetSnapshotImageName returns the snapshot image name for this project
func (r *PersistentRunner) GetSnapshotImageName() string {
	return fmt.Sprintf("%s-snapshot:latest", r.GetContainerName())
}

// CalculateConfigHash calculates a hash of the current configuration
func (r *PersistentRunner) CalculateConfigHash() string {
	data, _ := json.Marshal(r.Config)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8])
}

// LoadState loads the container state from disk
func (r *PersistentRunner) LoadState() (*ContainerState, error) {
	data, err := os.ReadFile(r.StateFile)
	if err != nil {
		return nil, err
	}

	var state ContainerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveState saves the container state to disk
func (r *PersistentRunner) SaveState(state *ContainerState) error {
	// Ensure directory exists
	dir := filepath.Dir(r.StateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save backend info
	state.Backend = r.Backend

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.StateFile, data, 0644)
}

// ClearState removes the state file
func (r *PersistentRunner) ClearState() error {
	return os.Remove(r.StateFile)
}

// getClient returns the Docker client, initializing if needed
func (r *PersistentRunner) getClient(ctx context.Context) (*client.Client, error) {
	if r.Client != nil {
		return r.Client, nil
	}

	// If we have a Docker runtime, get its client
	if dr, ok := r.Runtime.(*runtime.DockerRuntime); ok {
		return dr.Client(), nil
	}

	// Initialize a new client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	r.Client = cli
	return cli, nil
}

// IsContainerRunning checks if the persistent container is running
func (r *PersistentRunner) IsContainerRunning(ctx context.Context) (bool, string, error) {
	state, err := r.LoadState()
	if err != nil {
		return false, "", nil // No state file = no container
	}

	// Use runtime if available
	if r.Runtime != nil {
		info, err := r.Runtime.InspectContainer(ctx, state.ContainerID)
		if err != nil {
			r.ClearState()
			return false, "", nil
		}
		return info.Running, state.ContainerID, nil
	}

	// Fallback to Docker client
	cli, err := r.getClient(ctx)
	if err != nil {
		return false, "", err
	}

	inspect, err := cli.ContainerInspect(ctx, state.ContainerID)
	if err != nil {
		// Container doesn't exist
		r.ClearState()
		return false, "", nil
	}

	return inspect.State.Running, state.ContainerID, nil
}

// EnsureContainer ensures a persistent container is running
func (r *PersistentRunner) EnsureContainer(ctx context.Context, rebuild bool) (string, error) {
	containerName := r.GetContainerName()
	currentHash := r.CalculateConfigHash()

	// Check if we have an existing container
	running, containerID, err := r.IsContainerRunning(ctx)
	if err != nil {
		return "", err
	}

	if running && !rebuild {
		// Check if config changed
		state, _ := r.LoadState()
		if state != nil && state.ConfigHash != currentHash {
			fmt.Println("⚠️  Configuration has changed since container was created.")
			fmt.Print("   Rebuild container? [Y/n] ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "n" {
				rebuild = true
			}
		}

		if !rebuild {
			fmt.Printf("📦 Container '%s' is already running\n", containerName)
			return containerID, nil
		}
	}

	// Need to create or rebuild
	if containerID != "" {
		fmt.Printf("🔄 Stopping existing container '%s'...\n", containerName)
		if r.Runtime != nil {
			r.Runtime.StopContainer(ctx, containerID, 10)
			r.Runtime.RemoveContainer(ctx, containerID, true)
		} else {
			cli, _ := r.getClient(ctx)
			timeout := 10
			cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
			cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
		}
		r.ClearState()
	}

	// Resolve image
	imageTag, err := r.resolveImage(ctx)
	if err != nil {
		return "", err
	}

	fmt.Printf("📦 Creating persistent container '%s' (backend: %s)...\n", containerName, r.Backend)

	// Create container
	containerID, err = r.createContainer(ctx, containerName, imageTag)
	if err != nil {
		return "", err
	}

	// Start container
	if r.Runtime != nil {
		err = r.Runtime.StartContainer(ctx, containerID)
	} else {
		cli, _ := r.getClient(ctx)
		err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	}
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Save state
	state := &ContainerState{
		ContainerID:   containerID,
		ContainerName: containerName,
		CreatedAt:     time.Now(),
		ConfigHash:    currentHash,
		ImageTag:      imageTag,
		Backend:       r.Backend,
	}
	if err := r.SaveState(state); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}

	fmt.Printf("✅ Container '%s' started\n", containerName)

	// Install DevContainer Features
	if len(r.Config.Features) > 0 {
		installer := NewFeatureInstaller(containerID, r.getBackendCommand())
		if err := installer.InstallFeatures(ctx, r.Config.Features); err != nil {
			fmt.Printf("⚠️  Features installation failed: %v\n", err)
		}
	}

	// Execute lifecycle commands
	if err := r.runLifecycleCommand(ctx, containerID, "postCreateCommand", r.Config.PostCreateCommand); err != nil {
		fmt.Printf("⚠️  postCreateCommand failed: %v\n", err)
	}
	if err := r.runLifecycleCommand(ctx, containerID, "postStartCommand", r.Config.PostStartCommand); err != nil {
		fmt.Printf("⚠️  postStartCommand failed: %v\n", err)
	}

	return containerID, nil
}

// resolveImage ensures the image is available (either by pulling or building)
func (r *PersistentRunner) resolveImage(ctx context.Context) (string, error) {
	// Check if we need to build from Dockerfile
	if r.Config.Build != nil && r.Config.Build.Dockerfile != "" {
		return r.buildImage(ctx)
	}

	// Otherwise, pull the image
	if r.Config.Image == "" {
		return "", fmt.Errorf("no image specified and no build configuration found")
	}

	fmt.Printf("🔍 Checking image %s...\n", r.Config.Image)

	// Use runtime if available
	if r.Runtime != nil {
		if !r.Runtime.ImageExists(ctx, r.Config.Image) {
			fmt.Printf("📥 Pulling image %s...\n", r.Config.Image)
			if err := r.Runtime.PullImage(ctx, r.Config.Image); err != nil {
				return "", fmt.Errorf("failed to pull image: %w", err)
			}
			fmt.Printf("✅ Successfully pulled %s\n", r.Config.Image)
		}
		return r.Config.Image, nil
	}

	// Fallback to Docker client
	cli, err := r.getClient(ctx)
	if err != nil {
		return "", err
	}

	_, _, err = cli.ImageInspectWithRaw(ctx, r.Config.Image)
	if err != nil {
		fmt.Printf("📥 Pulling image %s...\n", r.Config.Image)
		reader, err := cli.ImagePull(ctx, r.Config.Image, image.PullOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to pull image: %w", err)
		}
		defer reader.Close()

		// Use beautiful progress display
		progress := NewPullProgressDisplay()
		progress.ProcessPullOutput(reader)

		fmt.Printf("✅ Successfully pulled %s\n", r.Config.Image)
	}

	return r.Config.Image, nil
}

// buildImage builds an image from Dockerfile
func (r *PersistentRunner) buildImage(ctx context.Context) (string, error) {
	dockerfile := r.Config.Build.Dockerfile
	buildContext := r.Config.Build.Context
	if buildContext == "" {
		buildContext = "."
	}

	// Resolve paths relative to project directory
	dockerfilePath := filepath.Join(r.ProjectDir, ".devcontainer", dockerfile)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		// Try relative to project root
		dockerfilePath = filepath.Join(r.ProjectDir, dockerfile)
	}

	contextPath := filepath.Join(r.ProjectDir, ".devcontainer", buildContext)
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		contextPath = filepath.Join(r.ProjectDir, buildContext)
	}

	// Generate image tag
	imageTag := fmt.Sprintf("cm-%s:latest", r.GetContainerName())

	fmt.Printf("🔨 Building image from %s...\n", dockerfile)
	fmt.Printf("   Context: %s\n", contextPath)
	fmt.Printf("   Tag: %s\n", imageTag)

	// Build using docker CLI for better output
	args := []string{"build", "-t", imageTag, "-f", dockerfilePath}

	// Add build args
	for k, v := range r.Config.Build.Args {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, contextPath)

	cmd := exec.CommandContext(ctx, r.getBackendCommand(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	fmt.Printf("✅ Successfully built %s\n", imageTag)
	return imageTag, nil
}

// runLifecycleCommand executes a lifecycle command (postCreateCommand, etc.) in the container
func (r *PersistentRunner) runLifecycleCommand(ctx context.Context, containerID, cmdName string, command interface{}) error {
	if command == nil {
		return nil
	}

	var cmdStr string
	switch c := command.(type) {
	case string:
		cmdStr = c
	case []interface{}:
		parts := make([]string, len(c))
		for i, p := range c {
			parts[i] = fmt.Sprintf("%v", p)
		}
		cmdStr = strings.Join(parts, " ")
	default:
		return nil
	}

	if cmdStr == "" {
		return nil
	}

	fmt.Printf("🔧 Running %s: %s\n", cmdName, cmdStr)

	// Execute command in container
	backendCmd := r.getBackendCommand()
	execCmd := exec.CommandContext(ctx, backendCmd, "exec", containerID, "sh", "-c", cmdStr)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", cmdName, err)
	}

	fmt.Printf("✅ %s completed\n", cmdName)
	return nil
}

// createContainer creates a new persistent container
func (r *PersistentRunner) createContainer(ctx context.Context, name, imageTag string) (string, error) {
	// Setup workspace mount
	cwd, _ := os.Getwd()
	projectName := filepath.Base(r.ProjectDir)
	workspaceDir := fmt.Sprintf("/workspaces/%s", projectName)
	workspaceBind := fmt.Sprintf("%s:%s", cwd, workspaceDir)

	// Use runtime if available
	if r.Runtime != nil {
		cfg := &runtime.ContainerConfig{
			Image:      imageTag,
			Cmd:        []string{"sleep", "infinity"},
			WorkingDir: workspaceDir,
			Tty:        true,
			OpenStdin:  true,
			Binds:      append([]string{workspaceBind}, r.Config.Mounts...),
		}

		// Add environment variables
		for k, v := range r.Config.ContainerEnv {
			cfg.Env = append(cfg.Env, fmt.Sprintf("%s=%s", k, v))
		}
		for k, v := range r.Config.RemoteEnv {
			cfg.Env = append(cfg.Env, fmt.Sprintf("%s=%s", k, v))
		}

		return r.Runtime.CreateContainer(ctx, cfg)
	}

	// Fallback to Docker client
	hostConfig := &container.HostConfig{
		Binds: []string{workspaceBind},
	}

	// Add mounts from config
	hostConfig.Binds = append(hostConfig.Binds, r.Config.Mounts...)

	// Add port bindings from forwardPorts
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, p := range r.Config.ForwardPorts {
		var port string
		switch v := p.(type) {
		case float64:
			port = fmt.Sprintf("%d", int(v))
		case string:
			port = v
		default:
			continue
		}
		containerPort := nat.Port(port + "/tcp")
		exposedPorts[containerPort] = struct{}{}
		portBindings[containerPort] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: port},
		}
	}
	if len(portBindings) > 0 {
		hostConfig.PortBindings = portBindings
		fmt.Printf("🔌 Forwarding ports: %v\n", r.Config.ForwardPorts)
	}

	containerConfig := &container.Config{
		Image:        imageTag,
		Cmd:          []string{"sleep", "infinity"}, // Keep container running
		WorkingDir:   workspaceDir,
		Tty:          true,
		OpenStdin:    true,
		ExposedPorts: exposedPorts,
	}

	// Add environment variables
	for k, v := range r.Config.ContainerEnv {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range r.Config.RemoteEnv {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cli, err := r.getClient(ctx)
	if err != nil {
		return "", err
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// getBackendCommand returns the CLI command for the current backend
func (r *PersistentRunner) getBackendCommand() string {
	if r.Runtime != nil {
		switch r.Runtime.Type() {
		case "podman":
			return "podman"
		case "nerdctl":
			return "nerdctl"
		}
	}
	return "docker"
}

// Shell enters an interactive shell in the persistent container
func (r *PersistentRunner) Shell(ctx context.Context) error {
	containerID, err := r.EnsureContainer(ctx, false)
	if err != nil {
		return err
	}

	fmt.Println("🚀 Entering shell...")

	// Use the appropriate backend command for interactive shell
	backendCmd := r.getBackendCommand()
	cmd := exec.CommandContext(ctx, backendCmd, "exec", "-it", containerID, "/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Exec executes a command in the persistent container
func (r *PersistentRunner) Exec(ctx context.Context, command []string) error {
	containerID, err := r.EnsureContainer(ctx, false)
	if err != nil {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))

	// Use runtime if available
	if r.Runtime != nil {
		return r.Runtime.ExecInContainer(ctx, containerID, command, runtime.ExecOptions{
			AttachStdout: true,
			AttachStderr: true,
			AttachStdin:  isTerminal,
			Tty:          isTerminal,
		})
	}

	// Fallback to Docker client
	cli, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	execConfig := container.ExecOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  isTerminal,
		Tty:          isTerminal,
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
		Tty: isTerminal,
	})
	if err != nil {
		return fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// Stream output
	if isTerminal {
		go io.Copy(attachResp.Conn, os.Stdin)
	}
	io.Copy(os.Stdout, attachResp.Reader)

	// Get exit code
	inspectResp, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil // Ignore inspect errors
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", inspectResp.ExitCode)
	}

	return nil
}

// Stop stops and removes the persistent container
func (r *PersistentRunner) Stop(ctx context.Context) error {
	state, err := r.LoadState()
	if err != nil {
		fmt.Println("No persistent container found.")
		return nil
	}

	containerName := state.ContainerName
	fmt.Printf("🛑 Stopping container '%s'...\n", containerName)

	if r.Runtime != nil {
		r.Runtime.StopContainer(ctx, state.ContainerID, 10)
		if err := r.Runtime.RemoveContainer(ctx, state.ContainerID, true); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	} else {
		cli, err := r.getClient(ctx)
		if err != nil {
			return err
		}

		timeout := 10
		if err := cli.ContainerStop(ctx, state.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
			// Container might already be stopped
		}

		if err := cli.ContainerRemove(ctx, state.ContainerID, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	r.ClearState()
	fmt.Printf("✅ Container '%s' stopped and removed\n", containerName)
	return nil
}

// Status returns the status of the persistent container
func (r *PersistentRunner) Status(ctx context.Context) {
	state, err := r.LoadState()
	if err != nil {
		fmt.Println("No persistent container found.")
		return
	}

	running, _, _ := r.IsContainerRunning(ctx)
	status := "stopped"
	if running {
		status = "running"
	}

	fmt.Printf("Container: %s\n", state.ContainerName)
	fmt.Printf("Status:    %s\n", status)
	fmt.Printf("Image:     %s\n", state.ImageTag)
	fmt.Printf("Backend:   %s\n", r.Backend)
	fmt.Printf("Created:   %s\n", state.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Config:    %s\n", state.ConfigHash[:8])
	if state.SnapshotImage != "" {
		fmt.Printf("Snapshot:  %s\n", state.SnapshotImage)
	}
	if state.IsPaused {
		fmt.Println("📦 Container is PAUSED (use --resume to restore)")
	}
}

// Pause saves the container state to an image and stops it (frees memory)
func (r *PersistentRunner) Pause(ctx context.Context) error {
	state, err := r.LoadState()
	if err != nil {
		return fmt.Errorf("no persistent container found")
	}

	running, containerID, _ := r.IsContainerRunning(ctx)
	if !running {
		if state.IsPaused {
			fmt.Println("📦 Container is already paused.")
			return nil
		}
		return fmt.Errorf("container is not running")
	}

	snapshotImage := r.GetSnapshotImageName()
	fmt.Printf("📸 Saving container state to '%s'...\n", snapshotImage)

	// Commit container to image (Docker-specific, fallback for other backends)
	cli, err := r.getClient(ctx)
	if err != nil {
		return err
	}

	commitResp, err := cli.ContainerCommit(ctx, containerID, container.CommitOptions{
		Reference: snapshotImage,
		Comment:   "Container-Make snapshot",
		Pause:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}
	fmt.Printf("✅ Snapshot saved: %s\n", commitResp.ID[:12])

	// Stop and remove container
	fmt.Println("🛑 Stopping container to free memory...")
	if r.Runtime != nil {
		r.Runtime.StopContainer(ctx, containerID, 10)
		r.Runtime.RemoveContainer(ctx, containerID, true)
	} else {
		timeout := 10
		cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	}

	// Update state
	state.SnapshotImage = snapshotImage
	state.IsPaused = true
	state.ContainerID = ""
	r.SaveState(state)

	fmt.Println("✅ Container paused. Memory freed.")
	fmt.Println("   Use 'cm shell --resume' to restore your environment.")
	return nil
}

// Resume restores a paused container from its snapshot
func (r *PersistentRunner) Resume(ctx context.Context) error {
	state, err := r.LoadState()
	if err != nil {
		return fmt.Errorf("no saved state found")
	}

	if !state.IsPaused || state.SnapshotImage == "" {
		fmt.Println("No paused snapshot found. Starting fresh container...")
		return r.Shell(ctx)
	}

	// Check if snapshot image exists
	if r.Runtime != nil {
		if !r.Runtime.ImageExists(ctx, state.SnapshotImage) {
			return fmt.Errorf("snapshot image not found: %s", state.SnapshotImage)
		}
	} else {
		cli, err := r.getClient(ctx)
		if err != nil {
			return err
		}
		_, _, err = cli.ImageInspectWithRaw(ctx, state.SnapshotImage)
		if err != nil {
			return fmt.Errorf("snapshot image not found: %s", state.SnapshotImage)
		}
	}

	containerName := r.GetContainerName()
	fmt.Printf("📦 Restoring container from snapshot '%s'...\n", state.SnapshotImage)

	// Create container from snapshot image
	containerID, err := r.createContainer(ctx, containerName, state.SnapshotImage)
	if err != nil {
		return fmt.Errorf("failed to create container from snapshot: %w", err)
	}

	// Start container
	if r.Runtime != nil {
		err = r.Runtime.StartContainer(ctx, containerID)
	} else {
		cli, _ := r.getClient(ctx)
		err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update state
	state.ContainerID = containerID
	state.IsPaused = false
	r.SaveState(state)

	fmt.Println("✅ Container restored from snapshot!")
	fmt.Println("🚀 Entering shell...")

	// Enter shell
	backendCmd := r.getBackendCommand()
	cmd := exec.CommandContext(ctx, backendCmd, "exec", "-it", containerID, "/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
