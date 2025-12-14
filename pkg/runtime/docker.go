package runtime

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// pullProgress represents a Docker pull progress message
type pullProgress struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// displayPullProgress parses Docker pull output and displays clean progress
func displayPullProgress(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	layers := make(map[string]string)
	var lastStatus string

	for scanner.Scan() {
		var p pullProgress
		if err := json.Unmarshal(scanner.Bytes(), &p); err != nil {
			continue
		}

		// Skip repetitive status updates
		if p.ID != "" {
			layers[p.ID] = p.Status
		}

		// Display key status messages
		switch {
		case strings.HasPrefix(p.Status, "Pulling from"):
			fmt.Printf("  📦 %s\n", p.Status)
		case strings.HasPrefix(p.Status, "Digest:"):
			fmt.Printf("  🔑 %s\n", p.Status)
		case strings.HasPrefix(p.Status, "Status:"):
			fmt.Printf("  ✅ %s\n", p.Status)
		case p.Status == "Downloading" && p.ProgressDetail.Total > 0:
			// Show download progress (update in place)
			pct := float64(p.ProgressDetail.Current) / float64(p.ProgressDetail.Total) * 100
			if p.Status != lastStatus || pct == 100 {
				fmt.Printf("\r  ⬇️  Layer %s: %.0f%%", p.ID[:12], pct)
				lastStatus = p.Status + p.ID
			}
		case p.Status == "Pull complete":
			fmt.Printf("\r  ✓ Layer %s: complete   \n", p.ID[:min(12, len(p.ID))])
		case p.Status == "Already exists":
			// Silently skip
		}
	}
	fmt.Println() // Final newline
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DockerRuntime implements ContainerRuntime for Docker
type DockerRuntime struct {
	name    string
	path    string
	client  *client.Client
	version string
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime(name, path string) (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	r := &DockerRuntime{
		name:   name,
		path:   path,
		client: cli,
	}

	// Get version
	if v, err := cli.ServerVersion(context.Background()); err == nil {
		r.version = v.Version
	}

	return r, nil
}

func (r *DockerRuntime) Name() string { return r.name }
func (r *DockerRuntime) Type() string { return "docker" }
func (r *DockerRuntime) Path() string { return r.path }

func (r *DockerRuntime) Version() (string, error) {
	if r.version != "" {
		return r.version, nil
	}
	v, err := r.client.ServerVersion(context.Background())
	if err != nil {
		return "", err
	}
	r.version = v.Version
	return r.version, nil
}

func (r *DockerRuntime) IsAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (r *DockerRuntime) IsRunning() error {
	_, err := r.client.Ping(context.Background())
	return err
}

func (r *DockerRuntime) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	// Convert ports
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for portProto, bindings := range config.PortBindings {
		port := nat.Port(portProto)
		exposedPorts[port] = struct{}{}
		for _, b := range bindings {
			portBindings[port] = append(portBindings[port], nat.PortBinding{
				HostIP:   b.HostIP,
				HostPort: b.HostPort,
			})
		}
	}

	// Convert devices
	var devices []container.DeviceMapping
	for _, d := range config.Devices {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        d.PathOnHost,
			PathInContainer:   d.PathInContainer,
			CgroupPermissions: d.CgroupPermissions,
		})
	}

	hostConfig := &container.HostConfig{
		Binds:        config.Binds,
		PortBindings: portBindings,
		AutoRemove:   config.AutoRemove,
		Init:         &config.Init,
		Privileged:   config.Privileged,
		NetworkMode:  container.NetworkMode(config.NetworkMode),
		CapAdd:       config.CapAdd,
		CapDrop:      config.CapDrop,
		SecurityOpt:  config.SecurityOpt,
		Resources: container.Resources{
			Devices: devices,
		},
	}

	if config.ShmSize > 0 {
		hostConfig.ShmSize = config.ShmSize
	}

	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.Cmd,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
		User:         config.User,
		Hostname:     config.Hostname,
		Entrypoint:   config.Entrypoint,
		ExposedPorts: exposedPorts,
		Tty:          config.Tty,
		OpenStdin:    config.OpenStdin,
	}

	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (r *DockerRuntime) StartContainer(ctx context.Context, id string) error {
	return r.client.ContainerStart(ctx, id, container.StartOptions{})
}

func (r *DockerRuntime) StopContainer(ctx context.Context, id string, timeout int) error {
	return r.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
}

func (r *DockerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return r.client.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

func (r *DockerRuntime) ExecInContainer(ctx context.Context, id string, cmd []string, opts ExecOptions) error {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: opts.AttachStdout,
		AttachStderr: opts.AttachStderr,
		AttachStdin:  opts.AttachStdin,
		Tty:          opts.Tty,
		User:         opts.User,
		WorkingDir:   opts.WorkingDir,
	}

	execResp, err := r.client.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return err
	}

	resp, err := r.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}
	defer resp.Close()

	// Stream output
	if opts.Tty {
		_, _ = io.Copy(os.Stdout, resp.Reader)
	} else {
		_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
	}

	return nil
}

func (r *DockerRuntime) AttachContainer(ctx context.Context, id string, opts AttachOptions) (*AttachResponse, error) {
	resp, err := r.client.ContainerAttach(ctx, id, container.AttachOptions{
		Stream: opts.Stream,
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
		Logs:   opts.Logs,
	})
	if err != nil {
		return nil, err
	}

	return &AttachResponse{
		Conn:   resp.Conn,
		Reader: resp.Reader,
	}, nil
}

func (r *DockerRuntime) WaitContainer(ctx context.Context, id string) (<-chan int64, <-chan error) {
	statusCh, errCh := r.client.ContainerWait(ctx, id, container.WaitConditionNotRunning)

	exitCh := make(chan int64, 1)
	go func() {
		select {
		case status := <-statusCh:
			exitCh <- status.StatusCode
		case <-ctx.Done():
			exitCh <- -1
		}
	}()

	return exitCh, errCh
}

func (r *DockerRuntime) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
	info, err := r.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ContainerInfo{
		ID:      info.ID,
		Name:    strings.TrimPrefix(info.Name, "/"),
		Image:   info.Config.Image,
		State:   info.State.Status,
		Running: info.State.Running,
	}, nil
}

func (r *DockerRuntime) PullImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, _, err := r.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image already exists
	}

	reader, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Use beautiful progress display instead of raw JSON output
	displayPullProgress(reader)
	return nil
}

func (r *DockerRuntime) BuildImage(ctx context.Context, opts BuildOptions) (string, error) {
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

	if opts.CacheFrom != "" {
		args = append(args, "--cache-from", opts.CacheFrom)
	}

	args = append(args, opts.ContextDir)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
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

func (r *DockerRuntime) ImageExists(ctx context.Context, imageName string) bool {
	_, _, err := r.client.ImageInspectWithRaw(ctx, imageName)
	return err == nil
}

func (r *DockerRuntime) CopyToContainer(ctx context.Context, id, destPath string, content io.Reader) error {
	return r.client.CopyToContainer(ctx, id, destPath, content, container.CopyToContainerOptions{})
}

func (r *DockerRuntime) ResizeContainerTTY(ctx context.Context, id string, height, width uint) error {
	return r.client.ContainerResize(ctx, id, container.ResizeOptions{
		Height: height,
		Width:  width,
	})
}

// CopyFileToContainer is a helper to copy a single file
func (r *DockerRuntime) CopyFileToContainer(ctx context.Context, containerID, destDir, filename string, content []byte) error {
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

// Client returns the underlying Docker client
func (r *DockerRuntime) Client() *client.Client {
	return r.client
}
