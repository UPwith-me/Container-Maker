package remote

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ContextMode represents the remote connection mode
type ContextMode string

const (
	// ContextModeSSH uses direct SSH commands
	ContextModeSSH ContextMode = "ssh"
	// ContextModeDocker uses Docker context (native Docker remote)
	ContextModeDocker ContextMode = "docker-context"
)

// Context manages Docker context for remote connections
type Context struct {
	name    string
	host    string
	mode    ContextMode
	created bool
}

// NewContext creates a new remote context manager
func NewContext(name, host string) *Context {
	return &Context{
		name: sanitizeName(name),
		host: host,
		mode: ContextModeSSH,
	}
}

// UseDockerContext switches to Docker context mode
// This sets DOCKER_HOST to use the remote Docker daemon directly
func (c *Context) UseDockerContext(ctx context.Context) error {
	contextName := fmt.Sprintf("cm-%s", c.name)

	// Check if context already exists
	checkCmd := exec.CommandContext(ctx, "docker", "context", "inspect", contextName)
	if err := checkCmd.Run(); err != nil {
		// Create new context
		fmt.Printf("📦 Creating Docker context '%s'...\n", contextName)
		createCmd := exec.CommandContext(ctx, "docker", "context", "create", contextName,
			"--docker", fmt.Sprintf("host=ssh://%s", c.host),
			"--description", fmt.Sprintf("Container-Maker remote: %s", c.host))

		output, err := createCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create context: %w\nOutput: %s", err, string(output))
		}
		c.created = true
	}

	// Use the context
	useCmd := exec.CommandContext(ctx, "docker", "context", "use", contextName)
	if err := useCmd.Run(); err != nil {
		return fmt.Errorf("failed to switch context: %w", err)
	}

	c.mode = ContextModeDocker
	fmt.Printf("✅ Now using Docker context '%s'\n", contextName)
	fmt.Println("💡 All docker commands will now run on the remote host")

	return nil
}

// SetDockerHost sets DOCKER_HOST environment variable for the current process
// This is an alternative to docker context that works with any docker command
func (c *Context) SetDockerHost() error {
	dockerHost := fmt.Sprintf("ssh://%s", c.host)
	os.Setenv("DOCKER_HOST", dockerHost)
	c.mode = ContextModeDocker

	fmt.Printf("📡 DOCKER_HOST=%s\n", dockerHost)
	return nil
}

// ResetContext switches back to default Docker context
func (c *Context) ResetContext(ctx context.Context) error {
	// Switch to default context
	cmd := exec.CommandContext(ctx, "docker", "context", "use", "default")
	if err := cmd.Run(); err != nil {
		// Try unsetting DOCKER_HOST instead
		os.Unsetenv("DOCKER_HOST")
	}

	c.mode = ContextModeSSH
	fmt.Println("🔄 Switched back to default Docker context")
	return nil
}

// RemoveContext removes the Docker context if it was created
func (c *Context) RemoveContext(ctx context.Context) error {
	if !c.created {
		return nil
	}

	contextName := fmt.Sprintf("cm-%s", c.name)

	// First switch away from this context
	c.ResetContext(ctx)

	// Remove the context
	cmd := exec.CommandContext(ctx, "docker", "context", "rm", contextName, "-f")
	return cmd.Run()
}

// TestConnection tests if the remote Docker is accessible
func (c *Context) TestConnection(ctx context.Context) error {
	var cmd *exec.Cmd

	switch c.mode {
	case ContextModeDocker:
		// Use docker info with current context
		cmd = exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	case ContextModeSSH:
		// Use SSH to run docker info
		cmd = exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=10", c.host,
			"docker", "info", "--format", "{{.ServerVersion}}")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot connect to Docker on %s: %w", c.host, err)
	}

	version := strings.TrimSpace(string(output))
	fmt.Printf("🐳 Remote Docker version: %s\n", version)
	return nil
}

// GetMode returns the current connection mode
func (c *Context) GetMode() ContextMode {
	return c.mode
}

// ListRemoteContainers lists containers on the remote host
func (c *Context) ListRemoteContainers(ctx context.Context) ([]string, error) {
	var cmd *exec.Cmd

	switch c.mode {
	case ContextModeDocker:
		cmd = exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}")
	case ContextModeSSH:
		cmd = exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=10", c.host,
			"docker", "ps", "--format", "{{.Names}}")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			containers = append(containers, line)
		}
	}

	return containers, nil
}

// ExecCommand executes a command in a remote container
func (c *Context) ExecCommand(ctx context.Context, containerName string, command []string) *exec.Cmd {
	switch c.mode {
	case ContextModeDocker:
		// Docker context mode - docker exec works directly
		args := append([]string{"exec", "-it", containerName}, command...)
		return exec.CommandContext(ctx, "docker", args...)
	case ContextModeSSH:
		// SSH mode - wrap in SSH
		dockerCmd := append([]string{"docker", "exec", "-it", containerName}, command...)
		sshArgs := append([]string{"-t", c.host}, dockerCmd...)
		return exec.CommandContext(ctx, "ssh", sshArgs...)
	}
	return nil
}

// sanitizeName creates a safe context name
func sanitizeName(name string) string {
	// Replace unsafe characters
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(safe, "-")
}
