package workspace

import (
	"context"
	"strings"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/environment"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Orchestrator manages service lifecycle for a workspace
type Orchestrator struct {
	workspace     *Workspace
	graph         *Graph
	dockerClient  *client.Client
	envManager    *environment.Manager
	state         *WorkspaceState
	mu            sync.RWMutex
}

// NewOrchestrator creates a new orchestrator for a workspace
func NewOrchestrator(ws *Workspace) (*Orchestrator, error) {
	// Create dependency graph
	graph, err := NewGraph(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Create environment manager
	envMgr, err := environment.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create environment manager: %w", err)
	}

	return &Orchestrator{
		workspace:    ws,
		graph:        graph,
		dockerClient: cli,
		envManager:   envMgr,
		state: &WorkspaceState{
			Name:     ws.Name,
			Services: make(map[string]*ServiceState),
		},
	}, nil
}

// Up starts all or specified services
func (o *Orchestrator) Up(ctx context.Context, opts StartOptions) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Determine which services to start
	var toStart []string
	var err error

	if len(opts.Services) > 0 {
		if opts.NoDeps {
			toStart = opts.Services
		} else {
			toStart, err = o.graph.GetStartOrderForServices(opts.Services)
			if err != nil {
				return err
			}
		}
	} else {
		toStart, err = o.graph.StartOrder()
		if err != nil {
			return err
		}
	}

	// Filter by profile if specified
	if opts.Profile != "" {
		profileServices := o.workspace.GetServicesByProfile(opts.Profile)
		profileNames := make(map[string]bool)
		for _, svc := range profileServices {
			profileNames[svc.Name] = true
		}
		var filtered []string
		for _, name := range toStart {
			if profileNames[name] {
				filtered = append(filtered, name)
			}
		}
		toStart = filtered
	}

	fmt.Printf(" Starting %d services in workspace '%s'\n", len(toStart), o.workspace.Name)
	fmt.Println()

	// Start services in order
	for i, name := range toStart {
		svc := o.workspace.Services[name]
		fmt.Printf("[%d/%d] Starting %s...\n", i+1, len(toStart), name)

		if err := o.startService(ctx, svc, opts); err != nil {
			fmt.Printf("?Failed to start %s: %v\n", name, err)
			if !opts.Force {
				return fmt.Errorf("failed to start %s: %w", name, err)
			}
			continue
		}

		fmt.Printf("?%s started\n", name)
	}

	o.state.StartedAt = time.Now()
	o.state.LastUpdateAt = time.Now()

	fmt.Println()
	fmt.Printf(" Workspace '%s' is up!\n", o.workspace.Name)

	return nil
}

// Down stops all or specified services
func (o *Orchestrator) Down(ctx context.Context, opts StopOptions) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Determine which services to stop
	var toStop []string
	var err error

	if len(opts.Services) > 0 {
		toStop, err = o.graph.GetStopOrderForServices(opts.Services)
		if err != nil {
			return err
		}
	} else {
		toStop, err = o.graph.StopOrder()
		if err != nil {
			return err
		}
	}

	fmt.Printf(" Stopping %d services in workspace '%s'\n", len(toStop), o.workspace.Name)
	fmt.Println()

	// Stop services in order
	for i, name := range toStop {
		svc := o.workspace.Services[name]
		fmt.Printf("[%d/%d] Stopping %s...\n", i+1, len(toStop), name)

		if err := o.stopService(ctx, svc, opts); err != nil {
			fmt.Printf("  Warning: failed to stop %s: %v\n", name, err)
			continue
		}

		fmt.Printf("?%s stopped\n", name)
	}

	o.state.LastUpdateAt = time.Now()

	fmt.Println()
	fmt.Printf(" Workspace '%s' is down\n", o.workspace.Name)

	return nil
}

// startService starts a single service
func (o *Orchestrator) startService(ctx context.Context, svc *Service, opts StartOptions) error {
	// Initialize state
	state := &ServiceState{
		Name:   svc.Name,
		Status: ServiceStatusStarting,
	}
	o.state.Services[svc.Name] = state

	// Determine image
	imageName := svc.Image
	if imageName == "" && svc.Template != "" {
		imageName = o.resolveTemplate(svc.Template)
	}
	if imageName == "" && svc.Build != nil {
		var err error
		imageName, err = o.buildImage(ctx, svc, opts.Build)
		if err != nil {
			state.Status = ServiceStatusError
			state.Error = err.Error()
			return err
		}
	}

	if imageName == "" {
		err := fmt.Errorf("no image specified for service %s", svc.Name)
		state.Status = ServiceStatusError
		state.Error = err.Error()
		return err
	}

	// Pull image if needed
	if err := o.ensureImage(ctx, imageName); err != nil {
		state.Status = ServiceStatusError
		state.Error = err.Error()
		return err
	}

	// Build container config
	containerName := fmt.Sprintf("cm-%s-%s", sanitizeName(o.workspace.Name), svc.Name)
	workspaceDir := fmt.Sprintf("/workspaces/%s", filepath.Base(svc.Path))

	containerConfig := &container.Config{
		Image:      imageName,
		Cmd:        svc.Command,
		Entrypoint: svc.Entrypoint,
		WorkingDir: workspaceDir,
		Tty:        true,
		OpenStdin:  true,
		Labels: map[string]string{
			"cm.managed_by":    "container-maker",
			"cm.workspace":     o.workspace.Name,
			"cm.service":       svc.Name,
		},
	}

	// Add environment variables
	for k, v := range svc.Environment {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Host config
	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s", svc.Path, workspaceDir)},
		NetworkMode: container.NetworkMode(o.workspace.GenerateNetworkName()),
	}

	// Add ports
	// Simplified port handling
	for _, port := range svc.Ports {
		// Port mappings would be added here
		_ = port
	}

	// Add resource limits
	if svc.Resources != nil {
		if svc.Resources.Memory != "" {
			memBytes := parseMemoryLimit(svc.Resources.Memory)
			hostConfig.Resources.Memory = memBytes
		}
		if svc.Resources.CPUs > 0 {
			hostConfig.Resources.NanoCPUs = int64(svc.Resources.CPUs * 1e9)
		}
	}

	// Add GPU
	if svc.GPU != nil && (svc.GPU.Count > 0 || len(svc.GPU.DeviceIDs) > 0) {
		hostConfig.Resources.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       svc.GPU.Driver,
				Count:        svc.GPU.Count,
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	// Create container
	resp, err := o.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		state.Status = ServiceStatusError
		state.Error = err.Error()
		return fmt.Errorf("failed to create container: %w", err)
	}

	state.ContainerID = resp.ID
	svc.ContainerID = resp.ID

	// Start container
	if err := o.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		state.Status = ServiceStatusError
		state.Error = err.Error()
		return fmt.Errorf("failed to start container: %w", err)
	}

	state.Status = ServiceStatusRunning
	state.StartedAt = time.Now()

	return nil
}

// stopService stops a single service
func (o *Orchestrator) stopService(ctx context.Context, svc *Service, opts StopOptions) error {
	state := o.state.Services[svc.Name]
	if state == nil {
		return nil
	}

	state.Status = ServiceStatusStopping

	if state.ContainerID != "" {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 10
		}

		if err := o.dockerClient.ContainerStop(ctx, state.ContainerID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			// Ignore not found errors
			if !client.IsErrNotFound(err) {
				return err
			}
		}

		if opts.Remove {
			if err := o.dockerClient.ContainerRemove(ctx, state.ContainerID, container.RemoveOptions{
				Force: true,
			}); err != nil && !client.IsErrNotFound(err) {
				return err
			}
		}
	}

	state.Status = ServiceStatusStopped
	svc.ContainerID = ""
	state.ContainerID = ""

	return nil
}

// resolveTemplate maps template names to images
func (o *Orchestrator) resolveTemplate(template string) string {
	templates := map[string]string{
		"python":     "mcr.microsoft.com/devcontainers/python:3.11",
		"node":       "mcr.microsoft.com/devcontainers/javascript-node:20",
		"go":         "mcr.microsoft.com/devcontainers/go:1.21",
		"rust":       "mcr.microsoft.com/devcontainers/rust:latest",
		"java":       "mcr.microsoft.com/devcontainers/java:17",
		"pytorch":    "pytorch/pytorch:latest",
		"tensorflow": "tensorflow/tensorflow:latest-gpu",
	}
	if img, ok := templates[template]; ok {
		return img
	}
	return template
}

// ensureImage pulls an image if not available locally
func (o *Orchestrator) ensureImage(ctx context.Context, imageName string) error {
	_, _, err := o.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists
	}

	fmt.Printf("   Pulling %s...\n", imageName)
	reader, err := o.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Read to completion
	buf := make([]byte, 1024)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			break
		}
	}

	return nil
}

// buildImage builds an image for a service
func (o *Orchestrator) buildImage(ctx context.Context, svc *Service, force bool) (string, error) {
	// Simplified - would implement docker build
	return "", fmt.Errorf("build not implemented for service %s", svc.Name)
}

// Status returns current workspace status
func (o *Orchestrator) Status() *WorkspaceState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state
}

// Restart restarts specific services
func (o *Orchestrator) Restart(ctx context.Context, services []string) error {
	if err := o.Down(ctx, StopOptions{Services: services}); err != nil {
		return err
	}
	return o.Up(ctx, StartOptions{Services: services})
}

// Logs streams logs from a service
func (o *Orchestrator) Logs(ctx context.Context, service string, follow bool, tail int) error {
	state := o.state.Services[service]
	if state == nil || state.ContainerID == "" {
		return fmt.Errorf("service %s is not running", service)
	}

	tailStr := fmt.Sprintf("%d", tail)
	if tail <= 0 {
		tailStr = "100"
	}

	reader, err := o.dockerClient.ContainerLogs(ctx, state.ContainerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tailStr,
		Timestamps: true,
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	buf := make([]byte, 8192)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		if n > 8 {
			os.Stdout.Write(buf[8:n])
		}
	}

	return nil
}

// Exec executes a command in a service
func (o *Orchestrator) Exec(ctx context.Context, service string, command []string) error {
	state := o.state.Services[service]
	if state == nil || state.ContainerID == "" {
		return fmt.Errorf("service %s is not running", service)
	}

	// Would use docker exec
	return fmt.Errorf("exec not implemented")
}

// Close cleans up resources
func (o *Orchestrator) Close() error {
	return o.dockerClient.Close()
}

// parseMemoryLimit converts memory string to bytes
func parseMemoryLimit(s string) int64 {
	var multiplier int64 = 1
	s = strings.ToLower(strings.TrimSpace(s))

	if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	}

	var value int64
	fmt.Sscanf(s, "%d", &value)
	return value * multiplier
}



