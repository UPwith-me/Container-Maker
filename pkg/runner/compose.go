package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/container-make/cm/pkg/config"
)

// ComposeRunner handles Docker Compose-based dev containers
type ComposeRunner struct {
	Config      *config.DevContainerConfig
	ComposeFile string
	ProjectDir  string
}

// NewComposeRunner creates a new Docker Compose runner
func NewComposeRunner(cfg *config.DevContainerConfig, projectDir string) (*ComposeRunner, error) {
	// Parse dockerComposeFile field
	var composeFile string
	switch v := cfg.DockerComposeFile.(type) {
	case string:
		composeFile = v
	case []interface{}:
		// Use the first file as the main compose file
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				composeFile = s
			}
		}
	}

	if composeFile == "" {
		return nil, fmt.Errorf("no docker compose file specified")
	}

	return &ComposeRunner{
		Config:      cfg,
		ComposeFile: composeFile,
		ProjectDir:  projectDir,
	}, nil
}

// IsComposeConfig checks if the config uses Docker Compose
func IsComposeConfig(cfg *config.DevContainerConfig) bool {
	return cfg.DockerComposeFile != nil
}

// Up starts all services defined in the compose file
func (r *ComposeRunner) Up(ctx context.Context) error {
	args := r.buildBaseArgs()
	args = append(args, "up", "-d")

	// Add specific services if configured
	if len(r.Config.RunServices) > 0 {
		args = append(args, r.Config.RunServices...)
	}

	fmt.Println("Starting Docker Compose services...")
	return r.runCompose(ctx, args)
}

// Down stops and removes all services
func (r *ComposeRunner) Down(ctx context.Context) error {
	args := r.buildBaseArgs()
	args = append(args, "down")

	// Handle shutdown action
	switch r.Config.ShutdownAction {
	case "stopCompose":
		args = append(args, "--remove-orphans")
	case "stopContainer":
		// Only stop the main service
		if r.Config.Service != "" {
			return r.stopService(ctx, r.Config.Service)
		}
	case "none":
		return nil
	}

	fmt.Println("Stopping Docker Compose services...")
	return r.runCompose(ctx, args)
}

// Exec executes a command in the main service container
func (r *ComposeRunner) Exec(ctx context.Context, command []string) error {
	service := r.Config.Service
	if service == "" {
		return fmt.Errorf("no service specified in devcontainer.json")
	}

	args := r.buildBaseArgs()
	args = append(args, "exec")

	// Add user if specified
	if r.Config.User != "" {
		args = append(args, "-u", r.Config.User)
	}

	// Add working directory
	if r.Config.WorkspaceFolder != "" {
		args = append(args, "-w", r.Config.WorkspaceFolder)
	}

	// Add environment variables
	for k, v := range r.Config.ContainerEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range r.Config.RemoteEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, service)
	args = append(args, command...)

	fmt.Printf("Executing in service %s: %s\n", service, strings.Join(command, " "))
	return r.runComposeInteractive(ctx, args)
}

// Run starts services and executes a command
func (r *ComposeRunner) Run(ctx context.Context, command []string) error {
	// Start services
	if err := r.Up(ctx); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Execute lifecycle hooks
	if err := r.executeLifecycleHooks(ctx); err != nil {
		fmt.Printf("Warning: lifecycle hooks failed: %v\n", err)
	}

	// Execute command
	if err := r.Exec(ctx, command); err != nil {
		return err
	}

	// Shutdown based on action
	return r.Down(ctx)
}

// Prepare pulls images and builds services
func (r *ComposeRunner) Prepare(ctx context.Context) error {
	args := r.buildBaseArgs()
	args = append(args, "build")

	fmt.Println("Building Docker Compose services...")
	if err := r.runCompose(ctx, args); err != nil {
		return err
	}

	// Pull images for services that don't have a build config
	args = r.buildBaseArgs()
	args = append(args, "pull", "--ignore-buildable")

	fmt.Println("Pulling Docker Compose images...")
	return r.runCompose(ctx, args)
}

// buildBaseArgs builds the base docker compose args
func (r *ComposeRunner) buildBaseArgs() []string {
	args := []string{"-f", filepath.Join(r.ProjectDir, r.ComposeFile)}

	// Add additional compose files if specified
	if files, ok := r.Config.DockerComposeFile.([]interface{}); ok {
		for i := 1; i < len(files); i++ {
			if f, ok := files[i].(string); ok {
				args = append(args, "-f", filepath.Join(r.ProjectDir, f))
			}
		}
	}

	return args
}

// runCompose executes docker compose with the given args
func (r *ComposeRunner) runCompose(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = r.ProjectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runComposeInteractive executes docker compose with interactive stdin
func (r *ComposeRunner) runComposeInteractive(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = r.ProjectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// stopService stops a specific service
func (r *ComposeRunner) stopService(ctx context.Context, service string) error {
	args := r.buildBaseArgs()
	args = append(args, "stop", service)
	return r.runCompose(ctx, args)
}

// executeLifecycleHooks runs lifecycle commands in the main service
func (r *ComposeRunner) executeLifecycleHooks(ctx context.Context) error {
	service := r.Config.Service
	if service == "" {
		return nil
	}

	hooks := []struct {
		name string
		cmd  interface{}
	}{
		{"onCreateCommand", r.Config.OnCreateCommand},
		{"postCreateCommand", r.Config.PostCreateCommand},
		{"postStartCommand", r.Config.PostStartCommand},
	}

	for _, hook := range hooks {
		if hook.cmd == nil {
			continue
		}

		var commands []string
		switch v := hook.cmd.(type) {
		case string:
			commands = []string{v}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					commands = append(commands, s)
				}
			}
		}

		for _, cmd := range commands {
			fmt.Printf("Executing %s: %s\n", hook.name, cmd)
			args := r.buildBaseArgs()
			args = append(args, "exec", "-T", service, "/bin/sh", "-c", cmd)
			if err := r.runCompose(ctx, args); err != nil {
				return fmt.Errorf("%s failed: %w", hook.name, err)
			}
		}
	}

	return nil
}

// ListServices lists all services in the compose file
func (r *ComposeRunner) ListServices(ctx context.Context) ([]string, error) {
	args := r.buildBaseArgs()
	args = append(args, "config", "--services")

	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = r.ProjectDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := strings.Split(strings.TrimSpace(string(output)), "\n")
	return services, nil
}

// GetServiceContainer gets the container ID for a service
func (r *ComposeRunner) GetServiceContainer(ctx context.Context, service string) (string, error) {
	args := r.buildBaseArgs()
	args = append(args, "ps", "-q", service)

	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = r.ProjectDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get container for service %s: %w", service, err)
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return "", fmt.Errorf("no container found for service %s", service)
	}

	return containerID, nil
}

// GetServicePorts gets the exposed ports for a service
func (r *ComposeRunner) GetServicePorts(ctx context.Context, service string) (map[string]string, error) {
	args := r.buildBaseArgs()
	args = append(args, "port", service)

	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = r.ProjectDir
	output, err := cmd.Output()
	if err != nil {
		// Service may not have ports
		return make(map[string]string), nil
	}

	ports := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: containerPort -> hostIP:hostPort
		parts := strings.Split(line, " -> ")
		if len(parts) == 2 {
			ports[parts[0]] = parts[1]
		}
	}

	return ports, nil
}
