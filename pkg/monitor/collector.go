package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// DockerCollector implements MetricsCollector using Docker API
type DockerCollector struct {
	client    *client.Client
	mu        sync.RWMutex
	lastStats map[string]*prevStats // For calculating rates
}

// prevStats stores previous stats for rate calculation
type prevStats struct {
	cpuUsage    uint64
	systemUsage uint64
	netRx       int64
	netTx       int64
	blockRead   int64
	blockWrite  int64
	timestamp   time.Time
}

// NewDockerCollector creates a new Docker metrics collector
func NewDockerCollector() (*DockerCollector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Docker is not available: %w", err)
	}

	return &DockerCollector{
		client:    cli,
		lastStats: make(map[string]*prevStats),
	}, nil
}

// Collect returns current metrics for a container
func (c *DockerCollector) Collect(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	// Get container info
	inspect, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Get stats (one-shot)
	statsResp, err := c.client.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return c.parseStats(containerID, inspect.Name, &stats), nil
}

// CollectAll returns metrics for all running containers
func (c *DockerCollector) CollectAll(ctx context.Context) ([]*ContainerMetrics, error) {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("status", "running")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []*ContainerMetrics
	for _, ctr := range containers {
		metrics, err := c.Collect(ctx, ctr.ID)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to collect metrics for %s: %v\n", ctr.ID[:12], err)
			continue
		}
		result = append(result, metrics)
	}

	return result, nil
}

// Stream returns a channel that receives metrics updates
func (c *DockerCollector) Stream(ctx context.Context, containerID string, interval time.Duration) (<-chan *ContainerMetrics, error) {
	ch := make(chan *ContainerMetrics)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics, err := c.Collect(ctx, containerID)
				if err != nil {
					continue // Skip on error
				}
				select {
				case ch <- metrics:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// StreamAll streams metrics for all containers
func (c *DockerCollector) StreamAll(ctx context.Context, interval time.Duration) (<-chan *ContainerMetrics, error) {
	ch := make(chan *ContainerMetrics)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics, err := c.CollectAll(ctx)
				if err != nil {
					continue
				}
				for _, m := range metrics {
					select {
					case ch <- m:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// parseStats converts Docker stats to our metrics format
func (c *DockerCollector) parseStats(containerID, name string, stats *container.StatsResponse) *ContainerMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := c.lastStats[containerID]
	now := time.Now()

	metrics := &ContainerMetrics{
		ContainerID:   containerID,
		ContainerName: strings.TrimPrefix(name, "/"),
		Timestamp:     now,
		PIDs:          int(stats.PidsStats.Current),
	}

	// CPU percentage
	if prev != nil && stats.CPUStats.SystemUsage > prev.systemUsage {
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - prev.cpuUsage)
		systemDelta := float64(stats.CPUStats.SystemUsage - prev.systemUsage)
		if systemDelta > 0 && cpuDelta > 0 {
			cpuCount := float64(stats.CPUStats.OnlineCPUs)
			if cpuCount == 0 {
				cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
			}
			metrics.CPUPercent = (cpuDelta / systemDelta) * cpuCount * 100.0
			metrics.CPUCount = int(cpuCount)
		}
	}

	// Memory
	metrics.MemoryUsed = int64(stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"])
	metrics.MemoryLimit = int64(stats.MemoryStats.Limit)
	if metrics.MemoryLimit > 0 {
		metrics.MemoryPercent = float64(metrics.MemoryUsed) / float64(metrics.MemoryLimit) * 100.0
	}
	metrics.MemoryCache = int64(stats.MemoryStats.Stats["cache"])

	// Network I/O
	var netRx, netTx int64
	for _, netStats := range stats.Networks {
		netRx += int64(netStats.RxBytes)
		netTx += int64(netStats.TxBytes)
	}
	metrics.NetworkRx = netRx
	metrics.NetworkTx = netTx

	// Calculate network rates
	if prev != nil {
		duration := now.Sub(prev.timestamp).Seconds()
		if duration > 0 {
			metrics.NetworkRxRate = float64(netRx-prev.netRx) / duration
			metrics.NetworkTxRate = float64(netTx-prev.netTx) / duration
		}
	}

	// Block I/O
	var blockRead, blockWrite int64
	for _, io := range stats.BlkioStats.IoServiceBytesRecursive {
		switch io.Op {
		case "Read", "read":
			blockRead += int64(io.Value)
		case "Write", "write":
			blockWrite += int64(io.Value)
		}
	}
	metrics.BlockRead = blockRead
	metrics.BlockWrite = blockWrite

	// Calculate block rates
	if prev != nil {
		duration := now.Sub(prev.timestamp).Seconds()
		if duration > 0 {
			metrics.BlockReadRate = float64(blockRead-prev.blockRead) / duration
			metrics.BlockWriteRate = float64(blockWrite-prev.blockWrite) / duration
		}
	}

	// Save for next calculation
	c.lastStats[containerID] = &prevStats{
		cpuUsage:    stats.CPUStats.CPUUsage.TotalUsage,
		systemUsage: stats.CPUStats.SystemUsage,
		netRx:       netRx,
		netTx:       netTx,
		blockRead:   blockRead,
		blockWrite:  blockWrite,
		timestamp:   now,
	}

	return metrics
}

// ListContainers returns all containers
func (c *DockerCollector) ListContainers(ctx context.Context, all bool) ([]*ContainerInfo, error) {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []*ContainerInfo
	for _, ctr := range containers {
		info := &ContainerInfo{
			ID:       ctr.ID,
			Name:     strings.TrimPrefix(ctr.Names[0], "/"),
			Image:    ctr.Image,
			State:    ctr.State,
			Status:   ctr.Status,
			Created:  time.Unix(ctr.Created, 0),
			Labels:   ctr.Labels,
			Networks: make([]string, 0),
		}

		for name := range ctr.NetworkSettings.Networks {
			info.Networks = append(info.Networks, name)
		}

		for _, port := range ctr.Ports {
			info.Ports = append(info.Ports, PortMapping{
				ContainerPort: int(port.PrivatePort),
				HostPort:      int(port.PublicPort),
				Protocol:      port.Type,
				HostIP:        port.IP,
			})
		}

		result = append(result, info)
	}

	return result, nil
}

// GetContainer returns a specific container
func (c *DockerCollector) GetContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	inspect, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	created, _ := time.Parse(time.RFC3339, inspect.Created)

	info := &ContainerInfo{
		ID:      inspect.ID,
		Name:    strings.TrimPrefix(inspect.Name, "/"),
		Image:   inspect.Config.Image,
		State:   inspect.State.Status,
		Status:  inspect.State.Status,
		Created: created,
		Labels:  inspect.Config.Labels,
	}

	for name := range inspect.NetworkSettings.Networks {
		info.Networks = append(info.Networks, name)
	}

	return info, nil
}

// StreamLogs streams logs from a container
func (c *DockerCollector) StreamLogs(ctx context.Context, containerID string, tail int) (<-chan *LogEntry, error) {
	ch := make(chan *LogEntry)

	tailStr := fmt.Sprintf("%d", tail)
	if tail <= 0 {
		tailStr = "all"
	}

	reader, err := c.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       tailStr,
		Timestamps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	go func() {
		defer close(ch)
		defer reader.Close()

		buf := make([]byte, 8192)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := reader.Read(buf)
				if err != nil {
					if err != io.EOF {
						fmt.Printf("Log read error: %v\n", err)
					}
					return
				}
				if n > 8 {
					// Docker log format: 8 bytes header + message
					stream := "stdout"
					if buf[0] == 2 {
						stream = "stderr"
					}
					message := string(buf[8:n])

					// Parse timestamp if present
					var ts time.Time
					if len(message) > 30 && message[4] == '-' && message[7] == '-' {
						ts, _ = time.Parse(time.RFC3339Nano, message[:30])
						message = message[31:]
					} else {
						ts = time.Now()
					}

					select {
					case ch <- &LogEntry{
						ContainerID: containerID,
						Timestamp:   ts,
						Stream:      stream,
						Message:     strings.TrimSpace(message),
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// StreamEvents streams Docker events
func (c *DockerCollector) StreamEvents(ctx context.Context) (<-chan *ContainerEvent, error) {
	ch := make(chan *ContainerEvent)

	eventsCh, errCh := c.client.Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
		),
	})

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					fmt.Printf("Events error: %v\n", err)
					return
				}
			case event := <-eventsCh:
				ce := &ContainerEvent{
					ContainerID:   event.Actor.ID,
					ContainerName: event.Actor.Attributes["name"],
					Type:          string(event.Type),
					Action:        string(event.Action),
					Timestamp:     time.Unix(event.Time, event.TimeNano%1e9),
					Attributes:    event.Actor.Attributes,
				}
				select {
				case ch <- ce:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// Close closes the Docker client
func (c *DockerCollector) Close() error {
	return c.client.Close()
}
