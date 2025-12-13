package runner

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/container-make/cm/pkg/config"
	"github.com/container-make/cm/pkg/features"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"golang.org/x/term"
)

type Runner struct {
	Client *client.Client
	Config *config.DevContainerConfig
}

func NewRunner(cfg *config.DevContainerConfig) (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Runner{Client: cli, Config: cfg}, nil
}

func (r *Runner) Run(ctx context.Context, command []string) error {
	var imageTag string
	var err error

	// 1. Resolve Image (Build/Pull + Features)
	imageTag, err = r.ResolveImage(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve image: %w", err)
	}
	r.Config.Image = imageTag

	// 2. Create Container
	fmt.Println("Creating container...")

	// Check if we are in a terminal
	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))

	// 2.1 Setup workspace mount
	workspaceBind, workspaceDir, err := r.setupWorkspaceMount()
	if err != nil {
		fmt.Printf("Warning: failed to setup workspace mount: %v\n", err)
	}

	// Basic HostConfig
	hostConfig := &container.HostConfig{
		AutoRemove: true,             // --rm
		Init:       &[]bool{true}[0], // --init
		Binds:      r.Config.Mounts,
	}

	// Add workspace bind mount if available
	if workspaceBind != "" {
		hostConfig.Binds = append(hostConfig.Binds, workspaceBind)
		fmt.Printf("Mounting workspace: %s\n", workspaceBind)
	}

	// 2.2 Apply runArgs to hostConfig
	// Create a temporary containerConfig for parseRunArgs (some args may affect it)
	tempContainerConfig := &container.Config{}
	if len(r.Config.RunArgs) > 0 {
		if err := parseRunArgs(r.Config.RunArgs, hostConfig, tempContainerConfig); err != nil {
			return fmt.Errorf("failed to parse runArgs: %w", err)
		}
	}

	// Port Forwarding
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, p := range r.Config.ForwardPorts {
		var portSpec string
		switch v := p.(type) {
		case float64: // JSON numbers are floats
			portSpec = fmt.Sprintf("%d", int(v))
		case int:
			portSpec = fmt.Sprintf("%d", v)
		case string:
			portSpec = v
		default:
			fmt.Printf("Warning: invalid port format: %v\n", p)
			continue
		}

		// Parse port specification
		// Formats: "8080", "8080/tcp", "8080/udp", "8080:80", "8080:80/tcp"
		hostPort, containerPort, protocol := parsePortSpec(portSpec)

		// Check for port conflict
		if isPortInUse(hostPort, protocol) {
			fmt.Printf("Warning: port %s/%s is already in use on host, skipping\n", hostPort, protocol)
			continue
		}

		port := nat.Port(containerPort + "/" + protocol)
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "127.0.0.1", // Bind to localhost
				HostPort: hostPort,
			},
		}
		fmt.Printf("Forwarding port %s -> %s/%s\n", hostPort, containerPort, protocol)
	}

	hostConfig.PortBindings = portBindings

	// Entrypoint setup
	// We inject a script to handle UID mapping
	entrypointPath := "/tmp/cm-entrypoint.sh"

	// Merge environment variables
	envVars := mergeEnvMaps(r.Config.ContainerEnv, r.Config.RemoteEnv)

	// Pass target user to entrypoint if specified in config
	if r.Config.User != "" {
		envVars = append(envVars, fmt.Sprintf("CM_TARGET_USER=%s", r.Config.User))
	}

	// 2.3 Setup SSH agent forwarding
	sshBind, sshEnv := r.setupSSHForwarding()
	if sshBind != "" {
		hostConfig.Binds = append(hostConfig.Binds, sshBind)
		fmt.Println("SSH agent forwarding enabled")
	}
	if sshEnv != "" {
		envVars = append(envVars, sshEnv)
	}

	// ContainerConfig
	containerConfig := &container.Config{
		Image:        r.Config.Image,
		Cmd:          command,
		Env:          envVars,
		User:         "root", // Always start as root to allow user creation, script will drop privileges
		Tty:          isTerminal,
		OpenStdin:    true,
		Entrypoint:   []string{"/bin/sh", entrypointPath},
		ExposedPorts: exposedPorts,
	}

	// Set working directory if workspace is configured
	if workspaceDir != "" {
		containerConfig.WorkingDir = workspaceDir
	}

	resp, err := r.Client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	fmt.Printf("Container created: %s\n", resp.ID)

	// 2.5 Inject Entrypoint Script
	if err := r.copyEntrypointToContainer(ctx, resp.ID, entrypointPath); err != nil {
		// Clean up
		r.Client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return fmt.Errorf("failed to inject entrypoint: %w", err)
	}

	// 3. Start Container
	fmt.Println("Starting container...")
	if err := r.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// 3.1 Lifecycle Hooks: PostCreateCommand & PostStartCommand
	// Since we are ephemeral, we run both here.
	if err := r.executeLifecycleHook(ctx, resp.ID, "postCreateCommand", r.Config.PostCreateCommand); err != nil {
		fmt.Printf("Warning: postCreateCommand failed: %v\n", err)
	}
	if err := r.executeLifecycleHook(ctx, resp.ID, "postStartCommand", r.Config.PostStartCommand); err != nil {
		fmt.Printf("Warning: postStartCommand failed: %v\n", err)
	}

	// 3.2 Features Warning
	if len(r.Config.Features) > 0 {
		fmt.Println("Warning: 'features' are detected in devcontainer.json but are not yet supported by Container-Make. They will be ignored.")
	}

	// 4. Handle Signals & TTY
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Resize TTY if applicable
	if isTerminal {
		// Set initial size
		width, height, _ := term.GetSize(int(os.Stdin.Fd()))
		r.Client.ContainerResize(ctx, resp.ID, container.ResizeOptions{
			Height: uint(height),
			Width:  uint(width),
		})

		// Put terminal in raw mode
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Printf("Warning: failed to set raw mode: %v\n", err)
		} else {
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}
	}

	go func() {
		<-sigChan
		// Restore terminal before printing (if in raw mode)
		// Note: defer handles restoration on return, but here we might want to ensure clean output
		// For now, just stop container.
		timeout := 10 // seconds
		r.Client.ContainerStop(ctx, resp.ID, container.StopOptions{Timeout: &timeout})
	}()

	// 5. Attach / Logs
	attachResp, err := r.Client.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  isTerminal, // Only attach stdin in TTY mode
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach: %w", err)
	}
	defer attachResp.Close()

	// 5.1 Lifecycle Hook: PostAttachCommand
	if err := r.executeLifecycleHook(ctx, resp.ID, "postAttachCommand", r.Config.PostAttachCommand); err != nil {
		fmt.Printf("Warning: postAttachCommand failed: %v\n", err)
	}

	// Use a channel to signal when output streaming is done
	outputDone := make(chan error, 1)

	// Stream output in a goroutine
	go func() {
		if isTerminal {
			// In TTY mode, stdout and stderr are merged, and we copy stdin
			go io.Copy(attachResp.Conn, os.Stdin)
			_, err := io.Copy(os.Stdout, attachResp.Reader)
			outputDone <- err
		} else {
			// In non-TTY mode, use StdCopy to demultiplex
			_, err := stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
			outputDone <- err
		}
	}()

	// 6. Wait for container to exit
	statusCh, errCh := r.Client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
	}

	// Wait for output to finish (with timeout)
	select {
	case <-outputDone:
	case <-time.After(2 * time.Second):
		// Timeout waiting for output, but container has exited
	}

	return nil
}

// parseRunArgs parses Docker run arguments and applies them to container configuration.
// Supports common runArgs like --cap-add, --security-opt, --device, --network, --privileged, etc.
func parseRunArgs(runArgs []string, hostConfig *container.HostConfig, _ *container.Config) error {
	for i := 0; i < len(runArgs); i++ {
		arg := runArgs[i]

		// Handle flags with values
		getValue := func() (string, error) {
			if i+1 >= len(runArgs) {
				return "", fmt.Errorf("missing value for flag %s", arg)
			}
			i++
			return runArgs[i], nil
		}

		switch arg {
		case "--cap-add":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.CapAdd = append(hostConfig.CapAdd, val)

		case "--cap-drop":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.CapDrop = append(hostConfig.CapDrop, val)

		case "--security-opt":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, val)

		case "--device":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.Devices = append(hostConfig.Devices, container.DeviceMapping{
				PathOnHost:        val,
				PathInContainer:   val,
				CgroupPermissions: "rwm",
			})

		case "--network", "--net":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.NetworkMode = container.NetworkMode(val)

		case "--privileged":
			hostConfig.Privileged = true

		case "--ipc":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.IpcMode = container.IpcMode(val)

		case "--pid":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.PidMode = container.PidMode(val)

		case "-v", "--volume":
			val, err := getValue()
			if err != nil {
				return err
			}
			hostConfig.Binds = append(hostConfig.Binds, val)

		default:
			// Ignore unknown flags with warning
			fmt.Printf("Warning: runArgs flag '%s' is not yet supported and will be ignored\n", arg)
		}
	}

	return nil
}

func (r *Runner) copyEntrypointToContainer(ctx context.Context, containerID, _ string) error {
	script := GetEntrypoint()

	// Create a tar archive in memory
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	hdr := &tar.Header{
		Name: "cm-entrypoint.sh", // Filename in tar, will be extracted to path's directory? No, CopyToContainer extracts to the destination path which must be a directory.
		// Wait, CopyToContainer path is the destination directory.
		// So if I want /tmp/cm-entrypoint.sh, I should copy to /tmp.
		Mode: 0755,
		Size: int64(len(script)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(script)); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	// Copy to container
	// Path must be a directory
	return r.Client.CopyToContainer(ctx, containerID, "/tmp", buf, container.CopyToContainerOptions{})
}

func (r *Runner) Build(ctx context.Context) (string, error) {
	if r.Config.Build == nil {
		return "", fmt.Errorf("no build configuration")
	}

	// Determine context and dockerfile
	buildContext := r.Config.Build.Context
	if buildContext == "" {
		buildContext = "."
	}
	dockerfile := r.Config.Build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	// Generate a tag based on the config hash or project name
	// For simplicity, let's use "cm-dev-env" for now, or maybe hash the path
	tag := "cm-dev-env:latest"

	fmt.Printf("Building image %s from %s...\n", tag, dockerfile)

	// Construct docker build command
	args := []string{"build", "-t", tag, "-f", dockerfile}

	// Add build args
	for k, v := range r.Config.Build.Args {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	// Add cache support from environment variables
	if cacheFrom := os.Getenv("CM_CACHE_FROM"); cacheFrom != "" {
		args = append(args, "--cache-from", cacheFrom)
		fmt.Printf("Using cache from: %s\n", cacheFrom)
	}
	if cacheTo := os.Getenv("CM_CACHE_TO"); cacheTo != "" {
		args = append(args, "--cache-to", cacheTo)
		fmt.Printf("Caching to: %s\n", cacheTo)
	}

	args = append(args, buildContext)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return tag, nil
}

// Pull pulls a Docker image with progress display
func (r *Runner) Pull(ctx context.Context) error {
	if r.Config.Image == "" {
		return fmt.Errorf("no image specified in configuration")
	}

	fmt.Printf("Pulling image %s...\n", r.Config.Image)

	// Check if image already exists
	_, _, err := r.Client.ImageInspectWithRaw(ctx, r.Config.Image)
	if err == nil {
		fmt.Printf("Image %s already exists locally.\n", r.Config.Image)
		return nil
	}

	// Pull the image
	reader, err := r.Client.ImagePull(ctx, r.Config.Image, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Stream pull progress to stdout
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	fmt.Printf("\nSuccessfully pulled %s\n", r.Config.Image)
	return nil
}

// ResolveImage ensures the container image (base + features) is ready
func (r *Runner) ResolveImage(ctx context.Context) (string, error) {
	var baseImage string
	var err error

	// 1. Resolve Base Image
	if r.Config.Build != nil {
		baseImage, err = r.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to build base image: %w", err)
		}
	} else if r.Config.Image != "" {
		if err := r.Pull(ctx); err != nil {
			return "", fmt.Errorf("failed to pull base image: %w", err)
		}
		baseImage = r.Config.Image
	} else {
		return "", fmt.Errorf("no image or build configuration found")
	}

	// 2. Apply Features (if any)
	if len(r.Config.Features) == 0 {
		return baseImage, nil
	}

	finalImage, err := r.applyFeatures(ctx, baseImage)
	if err != nil {
		return "", fmt.Errorf("failed to apply features: %w", err)
	}

	return finalImage, nil
}

// applyFeatures builds a new image with features installed on top of the base image
func (r *Runner) applyFeatures(ctx context.Context, baseImage string) (string, error) {
	fmt.Println("🔍 Resolving DevContainer Features...")

	// parse feature refs
	refs, err := features.ParseFeaturesFromConfig(r.Config.Features)
	if err != nil {
		return "", err
	}

	if len(refs) == 0 {
		return baseImage, nil
	}

	// Create temp build context
	tmpDir, err := os.MkdirTemp("", "cm-features-build-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	installer := features.NewFeatureInstaller(tmpDir)

	// Download features
	for _, ref := range refs {
		feature, err := features.DownloadFeature(ref, tmpDir)
		if err != nil {
			fmt.Printf("Warning: Failed to download feature %s: %v\n", ref.Source, err)
			continue
		}
		installer.AddFeature(feature)
	}

	// Generate Dockerfile
	dockerfileContent := fmt.Sprintf("FROM %s\n", baseImage)
	dockerfileContent += installer.GenerateDockerfileSnippet()

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return "", err
	}

	// Build feature layer
	// We tag it based on a hash of features, or just a generic dev tag for now
	featureTag := fmt.Sprintf("%s-with-features", baseImage)
	// Sanitize tag
	featureTag = strings.ReplaceAll(featureTag, ":", "-") + ":latest"

	fmt.Printf("🛠️  Building image with features -> %s\n", featureTag)

	args := []string{"build", "-t", featureTag, "-f", dockerfilePath, tmpDir}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker build failed: %w", err)
	}

	return featureTag, nil
}

func (r *Runner) executeLifecycleHook(ctx context.Context, containerID, name string, cmd interface{}) error {
	if cmd == nil {
		return nil
	}

	var commands []string
	switch v := cmd.(type) {
	case string:
		commands = []string{v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				commands = append(commands, s)
			}
		}
	}

	if len(commands) == 0 {
		return nil
	}

	fmt.Printf("Executing %s (%d command(s))...\n", name, len(commands))
	for i, c := range commands {
		startTime := time.Now()
		fmt.Printf("  [%d/%d] Running: %s\n", i+1, len(commands), truncateString(c, 60))

		// Create Exec
		execConfig := container.ExecOptions{
			Cmd:          []string{"/bin/sh", "-c", c},
			AttachStdout: true,
			AttachStderr: true,
		}
		execIDResp, err := r.Client.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			return fmt.Errorf("failed to create exec for %s: %w", name, err)
		}

		// Attach to Exec
		resp, err := r.Client.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{})
		if err != nil {
			return fmt.Errorf("failed to attach exec for %s: %w", name, err)
		}

		// Stream output
		stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
		resp.Close()

		// Check exit code
		inspectResp, err := r.Client.ContainerExecInspect(ctx, execIDResp.ID)
		if err != nil {
			fmt.Printf("  Warning: could not inspect exec status: %v\n", err)
		} else if inspectResp.ExitCode != 0 {
			duration := time.Since(startTime)
			return fmt.Errorf("%s command failed with exit code %d (took %v): %s",
				name, inspectResp.ExitCode, duration.Round(time.Millisecond), c)
		}

		fmt.Printf("  ✓ Completed in %v\n", time.Since(startTime).Round(time.Millisecond))
	}

	return nil
}

// truncateString truncates a string to max length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// mergeEnvMaps merges containerEnv and remoteEnv into a single environment variable list.
// remoteEnv has higher priority and will override containerEnv for duplicate keys.
func mergeEnvMaps(containerEnv, remoteEnv map[string]string) []string {
	// Start with containerEnv
	merged := make(map[string]string)
	for k, v := range containerEnv {
		merged[k] = v
	}
	// Override with remoteEnv
	for k, v := range remoteEnv {
		merged[k] = v
	}

	// Convert to slice
	var env []string
	for k, v := range merged {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// setupWorkspaceMount configures automatic workspace mounting.
// Priority: user-configured workspaceMount > auto-detected current directory.
// Returns: bind mount string, working directory path, error
func (r *Runner) setupWorkspaceMount() (string, string, error) {
	var bind string
	var workdir string

	// If user explicitly configured workspaceMount, use it
	if r.Config.WorkspaceMount != "" {
		bind = r.Config.WorkspaceMount
		// Parse the mount to extract the container path
		// Format is typically "source=...,target=...,type=bind"
		// For simplicity, we'll just use workspaceFolder if specified
		if r.Config.WorkspaceFolder != "" {
			workdir = r.Config.WorkspaceFolder
		} else {
			// Try to parse target from workspaceMount
			// This is a simplified parser, real implementation should be more robust
			parts := parseMount(r.Config.WorkspaceMount)
			if target, ok := parts["target"]; ok {
				workdir = target
			}
		}
		return bind, workdir, nil
	}

	// Auto-detect current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get project name (basename of current directory)
	projectName := filepath.Base(cwd)

	// Default workspace folder
	workdir = "/workspaces/" + projectName

	// Create bind mount string
	// Use Docker's standard bind mount format
	bind = fmt.Sprintf("%s:%s", cwd, workdir)

	return bind, workdir, nil
}

// parseMount parses a Docker mount string into key-value pairs
// Format: "source=...,target=...,type=..."
func parseMount(mount string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(mount, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

// setupSSHForwarding configures SSH agent forwarding by mounting the host's SSH agent socket.
// Returns: bind mount string (empty if not available), environment variable (empty if not available)
func (r *Runner) setupSSHForwarding() (string, string) {
	// Check for SSH_AUTH_SOCK environment variable (Unix/Linux/Mac)
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock != "" {
		// Verify the socket exists
		if _, err := os.Stat(sshAuthSock); err == nil {
			// Mount the socket to the same path in the container
			bind := fmt.Sprintf("%s:%s", sshAuthSock, sshAuthSock)
			envVar := fmt.Sprintf("SSH_AUTH_SOCK=%s", sshAuthSock)
			return bind, envVar
		}
	}

	// On Windows, check for the named pipe (Windows 10 1809+)
	// The pipe path is: \\.\pipe\openssh-ssh-agent
	if runtime.GOOS == "windows" {
		pipePath := `\\.\pipe\openssh-ssh-agent`
		// Note: Docker Desktop on Windows handles this differently
		// For now, we'll just set the environment variable if the pipe exists
		if _, err := os.Stat(pipePath); err == nil {
			// Windows SSH agent forwarding requires special handling
			// Docker Desktop typically mounts this automatically
			return "", fmt.Sprintf("SSH_AUTH_SOCK=%s", pipePath)
		}
	}

	return "", ""
}

// parsePortSpec parses a port specification string.
// Formats: "8080", "8080/tcp", "8080/udp", "8080:80", "8080:80/tcp"
// Returns: hostPort, containerPort, protocol
func parsePortSpec(spec string) (string, string, string) {
	protocol := "tcp" // Default protocol

	// Check for protocol suffix
	if strings.HasSuffix(spec, "/tcp") {
		protocol = "tcp"
		spec = strings.TrimSuffix(spec, "/tcp")
	} else if strings.HasSuffix(spec, "/udp") {
		protocol = "udp"
		spec = strings.TrimSuffix(spec, "/udp")
	}

	// Check for host:container mapping
	if strings.Contains(spec, ":") {
		parts := strings.SplitN(spec, ":", 2)
		return parts[0], parts[1], protocol
	}

	// Same port for host and container
	return spec, spec, protocol
}

// isPortInUse checks if a port is already in use on the host
func isPortInUse(port string, protocol string) bool {
	network := protocol
	if network == "" {
		network = "tcp"
	}

	addr := fmt.Sprintf("127.0.0.1:%s", port)

	switch network {
	case "tcp":
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false // Port is available
		}
		conn.Close()
		return true // Port is in use
	case "udp":
		// UDP port check is less reliable, just try to listen
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return false
		}
		conn, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			return true // Port is in use
		}
		conn.Close()
		return false
	}

	return false
}
