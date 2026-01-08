package environment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const (
	// NetworkPrefix is the prefix for all CM-managed networks
	NetworkPrefix = "cm-"

	// NetworkDriver is the default network driver
	NetworkDriver = "bridge"

	// LabelManagedBy identifies CM-managed networks
	LabelManagedBy = "cm.managed_by"
	LabelEnvID     = "cm.environment_id"
	LabelEnvName   = "cm.environment_name"
	LabelProject   = "cm.project"
	LabelCreatedAt = "cm.created_at"
)

// DockerNetworkManager implements NetworkManager using Docker API
type DockerNetworkManager struct {
	client *client.Client
}

// NewDockerNetworkManager creates a new Docker network manager
func NewDockerNetworkManager() (*DockerNetworkManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, WrapError(err, "NETWORK_INIT_ERROR", "failed to create Docker client")
	}

	// Verify Docker is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return nil, ErrDockerNotAvailable.WithCause(err)
	}

	return &DockerNetworkManager{client: cli}, nil
}

// NewDockerNetworkManagerWithClient creates a network manager with an existing client
func NewDockerNetworkManagerWithClient(cli *client.Client) *DockerNetworkManager {
	return &DockerNetworkManager{client: cli}
}

// CreateNetwork creates a new Docker network for an environment
func (m *DockerNetworkManager) CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error) {
	if name == "" {
		return "", ErrInvalidName.WithSuggestion("network name cannot be empty")
	}

	// Ensure network name has our prefix
	networkName := m.normalizeNetworkName(name)

	// Check if network already exists
	existing, err := m.GetNetwork(ctx, networkName)
	if err == nil && existing != nil {
		return existing.ID, nil // Already exists
	}

	// Merge labels with our managed labels
	allLabels := map[string]string{
		LabelManagedBy: "container-maker",
		LabelCreatedAt: time.Now().Format(time.RFC3339),
	}
	for k, v := range labels {
		allLabels[k] = v
	}

	// Create network with optimal settings
	createOpts := networktypes.CreateOptions{
		Driver:     NetworkDriver,
		Attachable: true, // Allow manual attachment
		Labels:     allLabels,
		IPAM: &networktypes.IPAM{
			Driver: "default",
		},
		Options: map[string]string{
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.enable_icc":           "true", // Inter-container communication
		},
	}

	resp, err := m.client.NetworkCreate(ctx, networkName, createOpts)
	if err != nil {
		return "", WrapError(err, "NETWORK_CREATE_ERROR", "failed to create network").WithSuggestion(
			"Check if Docker has sufficient permissions to create networks",
		)
	}

	return resp.ID, nil
}

// DeleteNetwork removes a Docker network
func (m *DockerNetworkManager) DeleteNetwork(ctx context.Context, nameOrID string) error {
	// Try to get network info first
	info, err := m.GetNetwork(ctx, nameOrID)
	if err != nil {
		return err
	}

	// Check if any containers are still connected
	if len(info.Containers) > 0 {
		return ErrNetworkInUse.WithSuggestion(
			fmt.Sprintf("Disconnect these containers first: %v", containerNames(info.Containers)),
		)
	}

	if err := m.client.NetworkRemove(ctx, info.ID); err != nil {
		return WrapError(err, "NETWORK_DELETE_ERROR", "failed to delete network")
	}

	return nil
}

// ForceDeleteNetwork removes a network, disconnecting all containers first
func (m *DockerNetworkManager) ForceDeleteNetwork(ctx context.Context, nameOrID string) error {
	info, err := m.GetNetwork(ctx, nameOrID)
	if err != nil {
		return err
	}

	// Disconnect all containers
	for containerID := range info.Containers {
		if err := m.DisconnectFromNetwork(ctx, info.ID, containerID); err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to disconnect %s: %v\n", containerID[:12], err)
		}
	}

	return m.DeleteNetwork(ctx, info.ID)
}

// ConnectToNetwork connects a container to a network
func (m *DockerNetworkManager) ConnectToNetwork(ctx context.Context, networkID, containerID string, aliases []string) error {
	endpointSettings := &networktypes.EndpointSettings{
		Aliases: aliases,
	}

	if err := m.client.NetworkConnect(ctx, networkID, containerID, endpointSettings); err != nil {
		return WrapError(err, "NETWORK_CONNECT_ERROR", "failed to connect container to network")
	}

	return nil
}

// DisconnectFromNetwork disconnects a container from a network
func (m *DockerNetworkManager) DisconnectFromNetwork(ctx context.Context, networkID, containerID string) error {
	if err := m.client.NetworkDisconnect(ctx, networkID, containerID, false); err != nil {
		return WrapError(err, "NETWORK_DISCONNECT_ERROR", "failed to disconnect container from network")
	}

	return nil
}

// GetNetwork retrieves network information
func (m *DockerNetworkManager) GetNetwork(ctx context.Context, nameOrID string) (*NetworkInfo, error) {
	inspect, err := m.client.NetworkInspect(ctx, nameOrID, networktypes.InspectOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, ErrNetworkNotFound.WithEnv("", nameOrID)
		}
		return nil, WrapError(err, "NETWORK_INSPECT_ERROR", "failed to inspect network")
	}

	return m.convertNetworkResource(&inspect), nil
}

// ListNetworks lists all CM-managed networks
func (m *DockerNetworkManager) ListNetworks(ctx context.Context, labels map[string]string) ([]*NetworkInfo, error) {
	// Build filter
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", LabelManagedBy+"=container-maker")

	for k, v := range labels {
		filterArgs.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	networks, err := m.client.NetworkList(ctx, networktypes.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, WrapError(err, "NETWORK_LIST_ERROR", "failed to list networks")
	}

	result := make([]*NetworkInfo, len(networks))
	for i, n := range networks {
		result[i] = m.convertNetworkSummary(&n)
	}

	return result, nil
}

// ListAllNetworks lists all networks (including non-CM ones)
func (m *DockerNetworkManager) ListAllNetworks(ctx context.Context) ([]*NetworkInfo, error) {
	networks, err := m.client.NetworkList(ctx, networktypes.ListOptions{})
	if err != nil {
		return nil, WrapError(err, "NETWORK_LIST_ERROR", "failed to list networks")
	}

	result := make([]*NetworkInfo, len(networks))
	for i, n := range networks {
		result[i] = m.convertNetworkSummary(&n)
	}

	return result, nil
}

// CreateEnvironmentNetwork creates a network specifically for an environment
func (m *DockerNetworkManager) CreateEnvironmentNetwork(ctx context.Context, env *Environment) (string, error) {
	labels := map[string]string{
		LabelEnvID:   env.ID,
		LabelEnvName: env.Name,
	}
	if env.ProjectDir != "" {
		labels[LabelProject] = env.ProjectDir
	}

	networkName := fmt.Sprintf("%s%s", NetworkPrefix, env.Name)
	return m.CreateNetwork(ctx, networkName, labels)
}

// LinkEnvironments connects two environments by joining their networks
func (m *DockerNetworkManager) LinkEnvironments(ctx context.Context, env1, env2 *Environment) error {
	// Get or create network for env1
	network1, err := m.ensureEnvironmentNetwork(ctx, env1)
	if err != nil {
		return err
	}

	// Get or create network for env2
	network2, err := m.ensureEnvironmentNetwork(ctx, env2)
	if err != nil {
		return err
	}

	// Connect env1's container to env2's network (and vice versa)
	if env1.ContainerID != "" && network2 != "" {
		if err := m.ConnectToNetwork(ctx, network2, env1.ContainerID, []string{env1.Name}); err != nil {
			// Ignore already connected error
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	if env2.ContainerID != "" && network1 != "" {
		if err := m.ConnectToNetwork(ctx, network1, env2.ContainerID, []string{env2.Name}); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	return nil
}

// UnlinkEnvironments disconnects two environments
func (m *DockerNetworkManager) UnlinkEnvironments(ctx context.Context, env1, env2 *Environment) error {
	// Get networks
	network1Name := fmt.Sprintf("%s%s", NetworkPrefix, env1.Name)
	network2Name := fmt.Sprintf("%s%s", NetworkPrefix, env2.Name)

	// Disconnect containers from each other's networks
	if env1.ContainerID != "" {
		_ = m.DisconnectFromNetwork(ctx, network2Name, env1.ContainerID)
	}
	if env2.ContainerID != "" {
		_ = m.DisconnectFromNetwork(ctx, network1Name, env2.ContainerID)
	}

	return nil
}

// PruneNetworks removes unused CM networks
func (m *DockerNetworkManager) PruneNetworks(ctx context.Context) ([]string, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", LabelManagedBy+"=container-maker")

	report, err := m.client.NetworksPrune(ctx, filterArgs)
	if err != nil {
		return nil, WrapError(err, "NETWORK_PRUNE_ERROR", "failed to prune networks")
	}

	return report.NetworksDeleted, nil
}

// GetNetworkForEnvironment returns the network for an environment
func (m *DockerNetworkManager) GetNetworkForEnvironment(ctx context.Context, envID string) (*NetworkInfo, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", LabelEnvID+"="+envID)

	networks, err := m.client.NetworkList(ctx, networktypes.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, WrapError(err, "NETWORK_LIST_ERROR", "failed to find environment network")
	}

	if len(networks) == 0 {
		return nil, ErrNetworkNotFound.WithEnv(envID, "")
	}

	return m.convertNetworkSummary(&networks[0]), nil
}

// ensureEnvironmentNetwork ensures a network exists for an environment
func (m *DockerNetworkManager) ensureEnvironmentNetwork(ctx context.Context, env *Environment) (string, error) {
	if env.NetworkID != "" {
		// Verify it still exists
		_, err := m.GetNetwork(ctx, env.NetworkID)
		if err == nil {
			return env.NetworkID, nil
		}
	}

	// Create new network
	return m.CreateEnvironmentNetwork(ctx, env)
}

// normalizeNetworkName ensures network name has our prefix
func (m *DockerNetworkManager) normalizeNetworkName(name string) string {
	if strings.HasPrefix(name, NetworkPrefix) {
		return name
	}
	return NetworkPrefix + name
}

// convertNetworkResource converts Docker API network to our NetworkInfo
func (m *DockerNetworkManager) convertNetworkResource(n *networktypes.Inspect) *NetworkInfo {
	containers := make(map[string]string)
	for id, endpoint := range n.Containers {
		containers[id] = endpoint.Name
	}

	return &NetworkInfo{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Containers: containers,
		Labels:     n.Labels,
		CreatedAt:  n.Created,
	}
}

// convertNetworkSummary converts Docker API network summary to our NetworkInfo
func (m *DockerNetworkManager) convertNetworkSummary(n *networktypes.Summary) *NetworkInfo {
	return &NetworkInfo{
		ID:        n.ID,
		Name:      n.Name,
		Driver:    n.Driver,
		Scope:     n.Scope,
		Internal:  n.Internal,
		Labels:    n.Labels,
		CreatedAt: n.Created,
	}
}

// containerNames extracts container names from a map
func containerNames(containers map[string]string) []string {
	names := make([]string, 0, len(containers))
	for _, name := range containers {
		names = append(names, name)
	}
	return names
}

// IsManagedNetwork checks if a network is managed by Container-Maker
func IsManagedNetwork(info *NetworkInfo) bool {
	if info == nil || info.Labels == nil {
		return false
	}
	return info.Labels[LabelManagedBy] == "container-maker"
}
