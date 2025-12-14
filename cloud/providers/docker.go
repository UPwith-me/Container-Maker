// Package providers provides a Docker-based local provider for development and testing
package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DockerProvider implements Provider interface using local Docker
type DockerProvider struct {
	dockerPath string
}

// NewDockerProvider creates a new Docker provider for local development
func NewDockerProvider() *DockerProvider {
	return &DockerProvider{
		dockerPath: "docker",
	}
}

func (p *DockerProvider) Name() ProviderType {
	return ProviderDocker
}

func (p *DockerProvider) DisplayName() string {
	return "Local Docker"
}

func (p *DockerProvider) Regions() []Region {
	return []Region{
		{ID: "local", Name: "Local Machine", Country: "Local", Available: true, GPUAvailable: false},
	}
}

func (p *DockerProvider) InstanceTypes() []InstancePricing {
	return []InstancePricing{
		{Type: InstanceTypeCPUSmall, HourlyRate: 0, VCPU: 2, MemoryGB: 4},
		{Type: InstanceTypeCPUMedium, HourlyRate: 0, VCPU: 4, MemoryGB: 8},
		{Type: InstanceTypeCPULarge, HourlyRate: 0, VCPU: 8, MemoryGB: 16},
	}
}

func (p *DockerProvider) CreateInstance(ctx context.Context, config InstanceConfig) (*Instance, error) {
	id := "cm-cloud-" + uuid.New().String()[:8]

	// Build docker run command
	args := []string{
		"run", "-d",
		"--name", id,
		"--hostname", config.Name,
	}

	// Add environment variables
	for k, v := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add port mappings
	for _, port := range config.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
	}

	// Add SSH port (22 -> random high port)
	args = append(args, "-p", "22")

	// Add image
	image := config.Image
	if image == "" {
		image = "ubuntu:22.04"
	}
	args = append(args, image)

	// Keep container running
	args = append(args, "sleep", "infinity")

	cmd := exec.CommandContext(ctx, p.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %v - %s", err, string(output))
	}

	// Get assigned SSH port
	portCmd := exec.CommandContext(ctx, p.dockerPath, "port", id, "22")
	portOutput, _ := portCmd.Output()
	sshPort := 22
	if len(portOutput) > 0 {
		parts := strings.Split(strings.TrimSpace(string(portOutput)), ":")
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &sshPort)
		}
	}

	now := time.Now()
	return &Instance{
		ID:         id,
		Name:       config.Name,
		Type:       config.Type,
		Status:     StatusRunning,
		Provider:   ProviderDocker,
		Region:     "local",
		PublicIP:   "127.0.0.1",
		SSHPort:    sshPort,
		CreatedAt:  now,
		UpdatedAt:  now,
		HourlyRate: 0,
	}, nil
}

func (p *DockerProvider) GetInstance(ctx context.Context, id string) (*Instance, error) {
	cmd := exec.CommandContext(ctx, p.dockerPath, "inspect", "--format",
		"{{.State.Status}}|{{.Config.Hostname}}|{{.Created}}", id)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("instance not found: %s", id)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid inspect output")
	}

	status := StatusRunning
	if parts[0] != "running" {
		status = StatusStopped
	}

	return &Instance{
		ID:       id,
		Name:     parts[1],
		Status:   status,
		Provider: ProviderDocker,
		Region:   "local",
		PublicIP: "127.0.0.1",
	}, nil
}

func (p *DockerProvider) ListInstances(ctx context.Context, ownerID string) ([]*Instance, error) {
	cmd := exec.CommandContext(ctx, p.dockerPath, "ps", "-a", "--filter", "name=cm-cloud-",
		"--format", "{{.Names}}|{{.Status}}|{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var instances []*Instance
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		status := StatusRunning
		if strings.Contains(parts[1], "Exited") {
			status = StatusStopped
		}

		instances = append(instances, &Instance{
			ID:       parts[0],
			Name:     parts[0],
			Status:   status,
			Provider: ProviderDocker,
			Region:   "local",
			PublicIP: "127.0.0.1",
		})
	}

	return instances, nil
}

func (p *DockerProvider) StartInstance(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, p.dockerPath, "start", id)
	return cmd.Run()
}

func (p *DockerProvider) StopInstance(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, p.dockerPath, "stop", id)
	return cmd.Run()
}

func (p *DockerProvider) DeleteInstance(ctx context.Context, id string) error {
	// Stop first
	p.StopInstance(ctx, id)

	cmd := exec.CommandContext(ctx, p.dockerPath, "rm", "-f", id)
	return cmd.Run()
}

func (p *DockerProvider) GetSSHEndpoint(ctx context.Context, id string) (string, int, error) {
	// For Docker, SSH is on localhost
	inst, err := p.GetInstance(ctx, id)
	if err != nil {
		return "", 0, err
	}
	return "127.0.0.1", inst.SSHPort, nil
}

func (p *DockerProvider) ExecCommand(ctx context.Context, id string, command []string) (string, string, int, error) {
	args := append([]string{"exec", id}, command...)
	cmd := exec.CommandContext(ctx, p.dockerPath, args...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", string(output), 1, err
		}
	}

	return string(output), "", exitCode, nil
}

func (p *DockerProvider) GetLogs(ctx context.Context, id string, tail int) (string, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	args = append(args, id)

	cmd := exec.CommandContext(ctx, p.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (p *DockerProvider) StreamLogs(ctx context.Context, id string) (<-chan string, error) {
	ch := make(chan string, 100)

	go func() {
		defer close(ch)
		cmd := exec.CommandContext(ctx, p.dockerPath, "logs", "-f", id)
		stdout, _ := cmd.StdoutPipe()
		cmd.Start()

		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			select {
			case ch <- string(buf[:n]):
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}
