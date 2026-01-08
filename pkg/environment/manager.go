package environment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/config"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Manager implements EnvironmentManager
type Manager struct {
	store          *FileStateStore
	networkManager *DockerNetworkManager
	dockerClient   *client.Client
}

// NewManager creates a new environment manager
func NewManager() (*Manager, error) {
	store, err := NewFileStateStore()
	if err != nil {
		return nil, err
	}

	networkMgr, err := NewDockerNetworkManager()
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, WrapError(err, "MANAGER_INIT_ERROR", "failed to create Docker client")
	}

	return &Manager{
		store:          store,
		networkManager: networkMgr,
		dockerClient:   cli,
	}, nil
}

// generateID generates a unique environment ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "env-" + hex.EncodeToString(bytes)
}

// validateName validates an environment name
func validateName(name string) error {
	if name == "" {
		return ErrInvalidName.WithSuggestion("environment name cannot be empty")
	}

	if len(name) > 64 {
		return ErrInvalidName.WithSuggestion("environment name cannot exceed 64 characters")
	}

	// Must match Docker naming conventions
	pattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]*$`)
	if !pattern.MatchString(name) {
		return ErrInvalidName.WithSuggestion(
			"environment name must start with a letter and contain only letters, numbers, underscores, dots, and hyphens",
		)
	}

	return nil
}

// Create creates a new environment
func (m *Manager) Create(ctx context.Context, opts EnvironmentCreateOptions) (*Environment, error) {
	// Validate name
	if err := validateName(opts.Name); err != nil {
		return nil, err
	}

	// Check if environment with same name exists
	existing, _ := m.store.GetByName(opts.Name)
	if existing != nil {
		if !opts.Force {
			return nil, ErrEnvironmentExists.WithEnv(existing.ID, opts.Name)
		}
		// Force: delete existing
		if err := m.Delete(ctx, existing.ID, true); err != nil {
			return nil, err
		}
	}

	// Determine project directory
	projectDir := opts.ProjectDir
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return nil, WrapError(err, "PROJECT_DIR_ERROR", "failed to get current directory")
		}
	}
	projectDir, _ = filepath.Abs(projectDir)

	// Create environment
	env := &Environment{
		ID:          generateID(),
		Name:        opts.Name,
		ProjectDir:  projectDir,
		Template:    opts.Template,
		Status:      StatusCreating,
		Labels:      opts.Labels,
		Tags:        opts.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Backend:     "docker",
		Ports:       make(map[string]int),
		LinkedEnvs:  []string{},
		GPUs:        opts.GPUs,
		MemoryLimit: opts.Memory,
		CPULimit:    opts.CPU,
	}

	// Set up labels
	if env.Labels == nil {
		env.Labels = make(map[string]string)
	}

	// Create dedicated network for this environment
	networkID, err := m.networkManager.CreateEnvironmentNetwork(ctx, env)
	if err != nil {
		return nil, err
	}
	env.NetworkID = networkID
	env.NetworkName = NetworkPrefix + env.Name

	// Save initial state
	if err := m.store.Save(env); err != nil {
		// Cleanup network on failure
		m.networkManager.DeleteNetwork(ctx, networkID)
		return nil, err
	}

	// If NoStart is not set, start the environment
	if !opts.NoStart {
		if err := m.startEnvironment(ctx, env, opts); err != nil {
			// Update status to error
			env.Status = StatusError
			env.StatusMsg = err.Error()
			m.store.Save(env)
			return env, err
		}
	} else {
		env.Status = StatusStopped
		m.store.Save(env)
	}

	// Link to other environments if requested
	for _, linkTo := range opts.LinkTo {
		targetEnv, err := m.Get(ctx, linkTo)
		if err != nil {
			fmt.Printf("Warning: failed to link to %s: %v\n", linkTo, err)
			continue
		}
		if err := m.Link(ctx, env.ID, targetEnv.ID, EnvironmentLinkOptions{Bidirectional: true}); err != nil {
			fmt.Printf("Warning: failed to link to %s: %v\n", linkTo, err)
		}
	}

	return env, nil
}

// startEnvironment starts the container for an environment
func (m *Manager) startEnvironment(ctx context.Context, env *Environment, opts EnvironmentCreateOptions) error {
	// Load devcontainer.json or template
	cfg, err := m.loadConfig(env)
	if err != nil {
		return err
	}

	// Resolve image
	imageName := cfg.Image
	if imageName == "" && cfg.Build != nil && cfg.Build.Dockerfile != "" {
		// Build from Dockerfile
		imageName, err = m.buildImage(ctx, env, cfg)
		if err != nil {
			return err
		}
	}

	if imageName == "" {
		return ErrInvalidConfig.WithSuggestion("no image or build configuration specified")
	}

	// Pull image if needed
	if err := m.ensureImage(ctx, imageName); err != nil {
		return err
	}

	// Create container
	containerName := fmt.Sprintf("cm-%s", env.Name)
	workspaceDir := fmt.Sprintf("/workspaces/%s", filepath.Base(env.ProjectDir))

	containerConfig := &container.Config{
		Image:      imageName,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: workspaceDir,
		Tty:        true,
		OpenStdin:  true,
		Labels: map[string]string{
			LabelManagedBy: "container-maker",
			LabelEnvID:     env.ID,
			LabelEnvName:   env.Name,
		},
	}

	// Add environment variables
	for k, v := range cfg.ContainerEnv {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", k, v))
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:%s", env.ProjectDir, workspaceDir)},
		NetworkMode: container.NetworkMode(env.NetworkName),
	}

	// Add mounts from config
	hostConfig.Binds = append(hostConfig.Binds, cfg.Mounts...)

	// Add GPU support
	if len(env.GPUs) > 0 || len(opts.GPUs) > 0 {
		hostConfig.Resources.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				Count:        -1, // All GPUs or specific ones
				Capabilities: [][]string{{"gpu"}},
			},
		}
	}

	// Memory limit
	if env.MemoryLimit != "" {
		memBytes := parseMemory(env.MemoryLimit)
		if memBytes > 0 {
			hostConfig.Resources.Memory = memBytes
		}
	}

	// CPU limit
	if env.CPULimit > 0 {
		hostConfig.Resources.NanoCPUs = int64(env.CPULimit * 1e9)
	}

	// Create the container
	resp, err := m.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return WrapError(err, "CONTAINER_CREATE_ERROR", "failed to create container")
	}

	env.ContainerID = resp.ID
	env.ContainerName = containerName
	env.ImageTag = imageName

	// Start the container
	if err := m.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return WrapError(err, "CONTAINER_START_ERROR", "failed to start container")
	}

	env.Status = StatusRunning
	env.UpdatedAt = time.Now()

	// Save updated state
	return m.store.Save(env)
}

// loadConfig loads the devcontainer configuration for an environment
func (m *Manager) loadConfig(env *Environment) (*config.DevContainerConfig, error) {
	// Try to find devcontainer.json
	paths := []string{
		filepath.Join(env.ProjectDir, ".devcontainer", "devcontainer.json"),
		filepath.Join(env.ProjectDir, "devcontainer.json"),
		filepath.Join(env.ProjectDir, ".devcontainer.json"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			cfg, err := config.ParseConfig(path)
			if err != nil {
				return nil, WrapError(err, "CONFIG_PARSE_ERROR", "failed to parse devcontainer.json")
			}
			env.ConfigFile = path
			return cfg, nil
		}
	}

	// If template is specified, use template config
	if env.Template != "" {
		return m.loadTemplateConfig(env.Template)
	}

	return nil, ErrInvalidConfig.WithSuggestion(
		"No devcontainer.json found. Create one with 'cm init' or specify a template with --template",
	)
}

// loadTemplateConfig loads configuration from a template
func (m *Manager) loadTemplateConfig(templateName string) (*config.DevContainerConfig, error) {
	// Map common template names to images
	templateImages := map[string]string{
		"python":     "mcr.microsoft.com/devcontainers/python:3.11",
		"node":       "mcr.microsoft.com/devcontainers/javascript-node:20",
		"go":         "mcr.microsoft.com/devcontainers/go:1.21",
		"rust":       "mcr.microsoft.com/devcontainers/rust:latest",
		"java":       "mcr.microsoft.com/devcontainers/java:17",
		"cpp":        "mcr.microsoft.com/devcontainers/cpp:latest",
		"dotnet":     "mcr.microsoft.com/devcontainers/dotnet:8.0",
		"pytorch":    "pytorch/pytorch:latest",
		"tensorflow": "tensorflow/tensorflow:latest-gpu",
		"ubuntu":     "ubuntu:22.04",
	}

	img, ok := templateImages[strings.ToLower(templateName)]
	if !ok {
		img = templateName // Assume it's a direct image reference
	}

	return &config.DevContainerConfig{
		Image: img,
	}, nil
}

// ensureImage ensures an image is available locally
func (m *Manager) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, _, err := m.dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists
	}

	// Pull image
	fmt.Printf("📥 Pulling image %s...\n", imageName)
	reader, err := m.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return WrapError(err, "IMAGE_PULL_ERROR", "failed to pull image")
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

	fmt.Printf("✅ Image %s ready\n", imageName)
	return nil
}

// buildImage builds an image for an environment
func (m *Manager) buildImage(ctx context.Context, env *Environment, cfg *config.DevContainerConfig) (string, error) {
	// Implementation for building from Dockerfile
	// This would call docker build
	return "", fmt.Errorf("Dockerfile build not yet implemented for environments")
}

// Start starts a stopped environment
func (m *Manager) Start(ctx context.Context, nameOrID string) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	if env.Status == StatusRunning {
		return nil // Already running
	}

	if env.ContainerID == "" {
		return ErrContainerNotFound.WithEnv(env.ID, env.Name)
	}

	if err := m.dockerClient.ContainerStart(ctx, env.ContainerID, container.StartOptions{}); err != nil {
		return WrapError(err, "CONTAINER_START_ERROR", "failed to start container")
	}

	env.Status = StatusRunning
	env.UpdatedAt = time.Now()
	return m.store.Save(env)
}

// Stop stops a running environment
func (m *Manager) Stop(ctx context.Context, nameOrID string, timeout int) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	if env.Status != StatusRunning {
		return nil // Already stopped
	}

	if env.ContainerID != "" {
		if err := m.dockerClient.ContainerStop(ctx, env.ContainerID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			return WrapError(err, "CONTAINER_STOP_ERROR", "failed to stop container")
		}
	}

	env.Status = StatusStopped
	env.UpdatedAt = time.Now()
	return m.store.Save(env)
}

// Restart restarts an environment
func (m *Manager) Restart(ctx context.Context, nameOrID string) error {
	if err := m.Stop(ctx, nameOrID, 10); err != nil {
		return err
	}
	return m.Start(ctx, nameOrID)
}

// Delete deletes an environment
func (m *Manager) Delete(ctx context.Context, nameOrID string, force bool) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	// Stop if running
	if env.Status == StatusRunning {
		if !force {
			return ErrEnvironmentRunning.WithEnv(env.ID, env.Name).WithSuggestion(
				"Stop the environment first with 'cm env stop' or use --force",
			)
		}
		m.Stop(ctx, nameOrID, 5)
	}

	// Remove container
	if env.ContainerID != "" {
		if err := m.dockerClient.ContainerRemove(ctx, env.ContainerID, container.RemoveOptions{
			Force: force,
		}); err != nil && !client.IsErrNotFound(err) {
			return WrapError(err, "CONTAINER_REMOVE_ERROR", "failed to remove container")
		}
	}

	// Remove network
	if env.NetworkID != "" {
		m.networkManager.ForceDeleteNetwork(ctx, env.NetworkID)
	}

	// Remove from store
	return m.store.Delete(env.ID)
}

// Get retrieves an environment by name or ID
func (m *Manager) Get(ctx context.Context, nameOrID string) (*Environment, error) {
	// Try by ID first
	env, err := m.store.Load(nameOrID)
	if err == nil {
		return m.syncStatus(ctx, env)
	}

	// Try by name
	env, err = m.store.GetByName(nameOrID)
	if err != nil {
		return nil, ErrEnvironmentNotFound.WithEnv("", nameOrID)
	}

	return m.syncStatus(ctx, env)
}

// syncStatus syncs the environment status with actual Docker state
func (m *Manager) syncStatus(ctx context.Context, env *Environment) (*Environment, error) {
	if env.ContainerID == "" {
		return env, nil
	}

	inspect, err := m.dockerClient.ContainerInspect(ctx, env.ContainerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			env.Status = StatusOrphaned
			env.ContainerID = ""
			m.store.Save(env)
		}
		return env, nil
	}

	// Update status based on container state
	if inspect.State.Running {
		env.Status = StatusRunning
	} else if inspect.State.Paused {
		env.Status = StatusPaused
	} else {
		env.Status = StatusStopped
	}

	return env, nil
}

// List returns all environments
func (m *Manager) List(ctx context.Context, opts EnvironmentListOptions) ([]*Environment, error) {
	envs, err := m.store.List()
	if err != nil {
		return nil, err
	}

	// Sync status for each
	for i, env := range envs {
		envs[i], _ = m.syncStatus(ctx, env)
	}

	// Apply filters
	if opts.Filter.Status != "" {
		var filtered []*Environment
		for _, env := range envs {
			if env.Status == opts.Filter.Status {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}

	// Sort
	sort.Slice(envs, func(i, j int) bool {
		switch opts.SortBy {
		case "name":
			if opts.SortDesc {
				return envs[i].Name > envs[j].Name
			}
			return envs[i].Name < envs[j].Name
		case "created":
			if opts.SortDesc {
				return envs[i].CreatedAt.After(envs[j].CreatedAt)
			}
			return envs[i].CreatedAt.Before(envs[j].CreatedAt)
		default:
			return envs[i].Name < envs[j].Name
		}
	})

	// Limit
	if opts.Limit > 0 && len(envs) > opts.Limit {
		envs = envs[:opts.Limit]
	}

	return envs, nil
}

// Exists checks if an environment exists
func (m *Manager) Exists(ctx context.Context, nameOrID string) bool {
	_, err := m.Get(ctx, nameOrID)
	return err == nil
}

// Switch sets the active environment
func (m *Manager) Switch(ctx context.Context, nameOrID string) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	if err := m.store.SetActive(env.ID); err != nil {
		return err
	}

	env.LastUsedAt = time.Now()
	return m.store.Save(env)
}

// GetActive returns the active environment
func (m *Manager) GetActive(ctx context.Context) (*Environment, error) {
	activeID, err := m.store.GetActive()
	if err != nil {
		return nil, err
	}

	if activeID == "" {
		return nil, ErrEnvironmentNotFound.WithSuggestion("No active environment. Use 'cm env switch <name>' to set one")
	}

	return m.Get(ctx, activeID)
}

// Link links two environments together
func (m *Manager) Link(ctx context.Context, env1ID, env2ID string, opts EnvironmentLinkOptions) error {
	if env1ID == env2ID {
		return ErrSelfLink
	}

	env1, err := m.Get(ctx, env1ID)
	if err != nil {
		return err
	}

	env2, err := m.Get(ctx, env2ID)
	if err != nil {
		return err
	}

	// Check if already linked
	for _, linked := range env1.LinkedEnvs {
		if linked == env2ID {
			return ErrLinkExists.WithEnv(env1ID, env1.Name)
		}
	}

	// Connect networks
	if err := m.networkManager.LinkEnvironments(ctx, env1, env2); err != nil {
		return err
	}

	// Update state
	env1.LinkedEnvs = append(env1.LinkedEnvs, env2ID)
	m.store.Save(env1)

	if opts.Bidirectional {
		env2.LinkedEnvs = append(env2.LinkedEnvs, env1ID)
		m.store.Save(env2)
	}

	return nil
}

// Unlink unlinks two environments
func (m *Manager) Unlink(ctx context.Context, env1ID, env2ID string) error {
	env1, err := m.Get(ctx, env1ID)
	if err != nil {
		return err
	}

	env2, err := m.Get(ctx, env2ID)
	if err != nil {
		return err
	}

	// Disconnect networks
	m.networkManager.UnlinkEnvironments(ctx, env1, env2)

	// Update state
	env1.LinkedEnvs = removeFromSlice(env1.LinkedEnvs, env2ID)
	m.store.Save(env1)

	env2.LinkedEnvs = removeFromSlice(env2.LinkedEnvs, env1ID)
	m.store.Save(env2)

	return nil
}

// Shell opens a shell in an environment
func (m *Manager) Shell(ctx context.Context, nameOrID string, shell string) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	if env.Status != StatusRunning {
		if err := m.Start(ctx, nameOrID); err != nil {
			return err
		}
	}

	if shell == "" {
		shell = "/bin/sh"
	}

	// Use docker exec for interactive shell
	fmt.Printf("🚀 Entering shell in '%s'...\n", env.Name)

	// This will be called via exec.Command in the CLI layer
	return nil
}

// Exec executes a command in an environment
func (m *Manager) Exec(ctx context.Context, nameOrID string, cmd []string) error {
	env, err := m.Get(ctx, nameOrID)
	if err != nil {
		return err
	}

	if env.Status != StatusRunning {
		return ErrEnvironmentStopped.WithEnv(env.ID, env.Name)
	}

	// Execute command (to be called via docker exec in CLI layer)
	return nil
}

// Helper functions

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != item {
			result = append(result, v)
		}
	}
	return result
}

func parseMemory(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	var multiplier int64 = 1

	if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "k") {
		multiplier = 1024
		s = s[:len(s)-1]
	}

	var value int64
	fmt.Sscanf(s, "%d", &value)
	return value * multiplier
}
