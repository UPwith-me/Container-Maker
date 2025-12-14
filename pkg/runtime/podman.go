package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// PodmanRuntime implements ContainerRuntime for Podman
type PodmanRuntime struct {
	name    string
	path    string
	version string
}

// NewPodmanRuntime creates a new Podman runtime
func NewPodmanRuntime(name, path string) (*PodmanRuntime, error) {
	if path == "" {
		p, err := exec.LookPath("podman")
		if err != nil {
			return nil, fmt.Errorf("podman not found in PATH")
		}
		path = p
	}

	r := &PodmanRuntime{
		name: name,
		path: path,
	}

	// Get version
	if v, err := r.Version(); err == nil {
		r.version = v
	}

	return r, nil
}

func (r *PodmanRuntime) Name() string { return r.name }
func (r *PodmanRuntime) Type() string { return "podman" }
func (r *PodmanRuntime) Path() string { return r.path }

func (r *PodmanRuntime) Version() (string, error) {
	if r.version != "" {
		return r.version, nil
	}

	cmd := exec.Command(r.path, "version", "--format", "{{.Version}}")
	output, err := cmd.Output()
	if err != nil {
		// Fallback
		cmd = exec.Command(r.path, "--version")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		// Parse "podman version X.Y.Z"
		parts := strings.Fields(string(output))
		if len(parts) >= 3 {
			return parts[2], nil
		}
	}
	return strings.TrimSpace(string(output)), nil
}

func (r *PodmanRuntime) IsAvailable() bool {
	_, err := os.Stat(r.path)
	return err == nil
}

func (r *PodmanRuntime) IsRunning() error {
	// Podman is daemonless, so just check if we can run info
	cmd := exec.Command(r.path, "info", "--format", "json")
	return cmd.Run()
}

func (r *PodmanRuntime) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	args := []string{"create"}

	// Add image
	args = append(args, config.Image)

	// Add name if provided
	if config.Hostname != "" {
		args = append(args, "--hostname", config.Hostname)
	}

	// Environment
	for _, env := range config.Env {
		args = append(args, "-e", env)
	}

	// Working directory
	if config.WorkingDir != "" {
		args = append(args, "-w", config.WorkingDir)
	}

	// User
	if config.User != "" {
		args = append(args, "-u", config.User)
	}

	// TTY
	if config.Tty {
		args = append(args, "-t")
	}
	if config.OpenStdin {
		args = append(args, "-i")
	}

	// Binds (volumes)
	for _, bind := range config.Binds {
		args = append(args, "-v", bind)
	}

	// Port bindings
	for portProto, bindings := range config.PortBindings {
		for _, b := range bindings {
			if b.HostIP != "" {
				args = append(args, "-p", fmt.Sprintf("%s:%s:%s", b.HostIP, b.HostPort, portProto))
			} else {
				args = append(args, "-p", fmt.Sprintf("%s:%s", b.HostPort, portProto))
			}
		}
	}

	// Auto remove
	if config.AutoRemove {
		args = append(args, "--rm")
	}

	// Init
	if config.Init {
		args = append(args, "--init")
	}

	// Privileged
	if config.Privileged {
		args = append(args, "--privileged")
	}

	// Network
	if config.NetworkMode != "" {
		args = append(args, "--network", config.NetworkMode)
	}

	// Capabilities
	for _, cap := range config.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	for _, cap := range config.CapDrop {
		args = append(args, "--cap-drop", cap)
	}

	// Devices
	for _, d := range config.Devices {
		args = append(args, "--device", fmt.Sprintf("%s:%s", d.PathOnHost, d.PathInContainer))
	}

	// Security options
	for _, opt := range config.SecurityOpt {
		args = append(args, "--security-opt", opt)
	}

	// Shared memory size
	if config.ShmSize > 0 {
		args = append(args, "--shm-size", fmt.Sprintf("%d", config.ShmSize))
	}

	// Entrypoint
	if len(config.Entrypoint) > 0 {
		args = append(args, "--entrypoint", strings.Join(config.Entrypoint, " "))
	}

	// Command
	args = append(args, config.Cmd...)

	cmd := exec.CommandContext(ctx, r.path, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("podman create failed: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (r *PodmanRuntime) StartContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, r.path, "start", id)
	return cmd.Run()
}

func (r *PodmanRuntime) StopContainer(ctx context.Context, id string, timeout int) error {
	cmd := exec.CommandContext(ctx, r.path, "stop", "-t", fmt.Sprintf("%d", timeout), id)
	return cmd.Run()
}

func (r *PodmanRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)

	cmd := exec.CommandContext(ctx, r.path, args...)
	return cmd.Run()
}

func (r *PodmanRuntime) ExecInContainer(ctx context.Context, id string, cmdArgs []string, opts ExecOptions) error {
	args := []string{"exec"}

	if opts.Tty {
		args = append(args, "-t")
	}
	if opts.AttachStdin {
		args = append(args, "-i")
	}
	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}
	if opts.WorkingDir != "" {
		args = append(args, "-w", opts.WorkingDir)
	}

	args = append(args, id)
	args = append(args, cmdArgs...)

	cmd := exec.CommandContext(ctx, r.path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (r *PodmanRuntime) AttachContainer(ctx context.Context, id string, opts AttachOptions) (*AttachResponse, error) {
	// Podman attach is typically done via exec for our use case
	// This is a simplified implementation
	args := []string{"attach"}
	if !opts.Stdin {
		args = append(args, "--no-stdin")
	}
	args = append(args, id)

	cmd := exec.CommandContext(ctx, r.path, args...)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &AttachResponse{
		Conn:   &podmanConn{stdin: stdin, stdout: stdout, cmd: cmd},
		Reader: stdout,
	}, nil
}

// podmanConn wraps podman attach pipes
type podmanConn struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (c *podmanConn) Read(p []byte) (n int, err error)  { return c.stdout.Read(p) }
func (c *podmanConn) Write(p []byte) (n int, err error) { return c.stdin.Write(p) }
func (c *podmanConn) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	return c.cmd.Wait()
}

func (r *PodmanRuntime) WaitContainer(ctx context.Context, id string) (<-chan int64, <-chan error) {
	exitCh := make(chan int64, 1)
	errCh := make(chan error, 1)

	go func() {
		cmd := exec.CommandContext(ctx, r.path, "wait", id)
		output, err := cmd.Output()
		if err != nil {
			errCh <- err
			return
		}
		var exitCode int64
		_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &exitCode)
		exitCh <- exitCode
	}()

	return exitCh, errCh
}

func (r *PodmanRuntime) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, r.path, "inspect", "--format", "json", id)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var containers []struct {
		ID    string `json:"Id"`
		Name  string `json:"Name"`
		Image string `json:"Image"`
		State struct {
			Status  string `json:"Status"`
			Running bool   `json:"Running"`
		} `json:"State"`
	}

	if err := json.Unmarshal(output, &containers); err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("container not found")
	}

	c := containers[0]
	return &ContainerInfo{
		ID:      c.ID,
		Name:    c.Name,
		Image:   c.Image,
		State:   c.State.Status,
		Running: c.State.Running,
	}, nil
}

func (r *PodmanRuntime) PullImage(ctx context.Context, imageName string) error {
	cmd := exec.CommandContext(ctx, r.path, "pull", imageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *PodmanRuntime) BuildImage(ctx context.Context, opts BuildOptions) (string, error) {
	args := []string{"build"}

	for _, tag := range opts.Tags {
		args = append(args, "-t", tag)
	}

	if opts.Dockerfile != "" {
		args = append(args, "-f", opts.Dockerfile)
	}

	for k, v := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, opts.ContextDir)

	cmd := exec.CommandContext(ctx, r.path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	if len(opts.Tags) > 0 {
		return opts.Tags[0], nil
	}
	return "", nil
}

func (r *PodmanRuntime) ImageExists(ctx context.Context, imageName string) bool {
	cmd := exec.CommandContext(ctx, r.path, "image", "exists", imageName)
	return cmd.Run() == nil
}

func (r *PodmanRuntime) CopyToContainer(ctx context.Context, id, destPath string, content io.Reader) error {
	// Create a temp file for the tar content
	tmpFile, err := os.CreateTemp("", "podman-copy-*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, content); err != nil {
		return err
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, r.path, "cp", tmpFile.Name(), fmt.Sprintf("%s:%s", id, destPath))
	return cmd.Run()
}

func (r *PodmanRuntime) ResizeContainerTTY(ctx context.Context, id string, height, width uint) error {
	// Podman doesn't have a direct resize command in older versions
	// This is typically handled by the terminal
	return nil
}

// CopyFileToContainer is a helper to copy a single file
func (r *PodmanRuntime) CopyFileToContainer(ctx context.Context, containerID, destDir, filename string, content []byte) error {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	hdr := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	return r.CopyToContainer(ctx, containerID, destDir, buf)
}
