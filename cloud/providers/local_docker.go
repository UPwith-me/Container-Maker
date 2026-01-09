// Package providers provides LocalDocker cloud provider implementation
// This provider allows using local Docker as a "cloud" for testing and development
package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LocalDockerProvider implements the Provider interface using local Docker
type LocalDockerProvider struct {
	mu         sync.RWMutex
	instances  map[string]*Instance
	configured bool
}

// NewLocalDockerProvider creates a new LocalDocker provider
func NewLocalDockerProvider() *LocalDockerProvider {
	return &LocalDockerProvider{
		instances:  make(map[string]*Instance),
		configured: true, // Always configured if Docker is available
	}
}

func (p *LocalDockerProvider) Name() ProviderType  { return ProviderDocker }
func (p *LocalDockerProvider) DisplayName() string { return "Local Docker" }
func (p *LocalDockerProvider) Description() string {
	return "Run cloud-like instances on your local Docker daemon. Perfect for testing and development."
}
func (p *LocalDockerProvider) Website() string { return "https://docker.com" }
func (p *LocalDockerProvider) Features() []string {
	return []string{"local", "free", "instant", "no-credentials"}
}
func (p *LocalDockerProvider) RequiredCredentials() []string {
	return []string{} // No credentials needed
}

func (p *LocalDockerProvider) Configure(credentials map[string]string) error {
	// No configuration needed for local Docker
	p.configured = true
	return nil
}

func (p *LocalDockerProvider) IsAvailable(ctx context.Context) bool {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

func (p *LocalDockerProvider) Regions() []Region {
	return []Region{
		{ID: "local", Name: "Local Machine", Country: "LOCAL", Available: true, GPUAvailable: p.hasGPU()},
	}
}

func (p *LocalDockerProvider) hasGPU() bool {
	cmd := exec.Command("docker", "info", "--format", "{{.Runtimes}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "nvidia")
}

func (p *LocalDockerProvider) InstanceTypes() []InstancePricing {
	return []InstancePricing{
		{Type: InstanceTypeCPUSmall, HourlyRate: 0, VCPU: 2, MemoryGB: 4},
		{Type: InstanceTypeCPUMedium, HourlyRate: 0, VCPU: 4, MemoryGB: 8},
		{Type: InstanceTypeCPULarge, HourlyRate: 0, VCPU: 8, MemoryGB: 16},
		{Type: InstanceTypeGPUT4, HourlyRate: 0, VCPU: 4, MemoryGB: 16, GPUType: "Local GPU", GPUMemoryGB: 8},
	}
}

func (p *LocalDockerProvider) CreateInstance(ctx context.Context, config InstanceConfig) (*Instance, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Generate instance ID
	id := "local-" + uuid.New().String()[:8]

	// Determine image
	image := config.Image
	if image == "" {
		image = "ubuntu:22.04"
	}

	// Build docker run command
	containerName := "cm-cloud-" + id
	args := []string{
		"run", "-d",
		"--name", containerName,
		"--hostname", config.Name,
		"-p", "2222:22", // SSH port
	}

	// Add resource limits based on instance type
	switch config.Type {
	case InstanceTypeCPUSmall:
		args = append(args, "--cpus", "2", "--memory", "4g")
	case InstanceTypeCPUMedium:
		args = append(args, "--cpus", "4", "--memory", "8g")
	case InstanceTypeCPULarge:
		args = append(args, "--cpus", "8", "--memory", "16g")
	case InstanceTypeGPUT4, InstanceTypeGPUA10, InstanceTypeGPUA100:
		args = append(args, "--gpus", "all")
	}

	// Add SSH-ready image with init command
	args = append(args, image, "sleep", "infinity")

	// Run container
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %s - %w", string(output), err)
	}

	containerID := strings.TrimSpace(string(output))

	// Create instance record
	now := time.Now().UTC()
	instance := &Instance{
		ID:        id,
		OwnerID:   "local-user",
		Name:      config.Name,
		Type:      config.Type,
		Provider:  ProviderDocker,
		Region:    "local",
		Status:    StatusRunning,
		PublicIP:  "127.0.0.1",
		PrivateIP: "172.17.0.2", // Docker default
		SSHPort:   2222,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"container_id": containerID[:12]},
	}

	p.instances[id] = instance

	fmt.Printf("✅ Created local container: %s (%s)\n", containerName, containerID[:12])
	return instance, nil
}

func (p *LocalDockerProvider) GetInstance(ctx context.Context, id string) (*Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if inst, ok := p.instances[id]; ok {
		return inst, nil
	}
	return nil, fmt.Errorf("instance not found: %s", id)
}

func (p *LocalDockerProvider) ListInstances(ctx context.Context, ownerID string) ([]*Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Instance
	for _, inst := range p.instances {
		result = append(result, inst)
	}

	// Also check for running containers with cm-cloud prefix
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=cm-cloud-", "--format", "{{.Names}}")
	output, _ := cmd.Output()
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if name == "" {
			continue
		}
		id := strings.TrimPrefix(name, "cm-cloud-")
		if _, exists := p.instances[id]; !exists {
			result = append(result, &Instance{
				ID:       id,
				Name:     name,
				Provider: ProviderDocker,
				Status:   StatusRunning,
				PublicIP: "127.0.0.1",
			})
		}
	}

	return result, nil
}

func (p *LocalDockerProvider) StartInstance(ctx context.Context, id string) error {
	containerName := "cm-cloud-" + id
	cmd := exec.CommandContext(ctx, "docker", "start", containerName)
	return cmd.Run()
}

func (p *LocalDockerProvider) StopInstance(ctx context.Context, id string) error {
	containerName := "cm-cloud-" + id
	cmd := exec.CommandContext(ctx, "docker", "stop", containerName)
	return cmd.Run()
}

func (p *LocalDockerProvider) DeleteInstance(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	containerName := "cm-cloud-" + id
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	if err := cmd.Run(); err != nil {
		return err
	}

	delete(p.instances, id)
	fmt.Printf("🗑️ Deleted local container: %s\n", containerName)
	return nil
}

func (p *LocalDockerProvider) GetSSHEndpoint(ctx context.Context, id string) (string, int, error) {
	return "127.0.0.1", 2222, nil
}

func (p *LocalDockerProvider) ExecCommand(ctx context.Context, id string, command []string) (string, string, int, error) {
	containerName := "cm-cloud-" + id
	args := append([]string{"exec", containerName}, command...)
	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return "", string(output), 1, err
		}
	}

	return string(output), "", exitCode, nil
}

func (p *LocalDockerProvider) GetLogs(ctx context.Context, id string, tail int) (string, error) {
	containerName := "cm-cloud-" + id
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tail), containerName)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (p *LocalDockerProvider) StreamLogs(ctx context.Context, id string) (<-chan string, error) {
	ch := make(chan string, 100)
	containerName := "cm-cloud-" + id

	go func() {
		defer close(ch)
		cmd := exec.CommandContext(ctx, "docker", "logs", "-f", containerName)
		output, _ := cmd.StdoutPipe()
		_ = cmd.Start()

		buf := make([]byte, 4096)
		for {
			n, err := output.Read(buf)
			if err != nil {
				return
			}
			ch <- string(buf[:n])
		}
	}()

	return ch, nil
}
