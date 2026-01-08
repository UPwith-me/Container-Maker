package monitor

import (
	"testing"
	"time"
)

func TestContainerMetrics(t *testing.T) {
	metrics := &ContainerMetrics{
		ContainerID:   "abc123",
		ContainerName: "test-container",
		CPUPercent:    25.5,
		MemoryUsed:    1024 * 1024 * 512,  // 512MB
		MemoryLimit:   1024 * 1024 * 1024, // 1GB
		MemoryPercent: 50.0,
		NetworkRx:     1000,
		NetworkTx:     2000,
		PIDs:          10,
		Timestamp:     time.Now(),
	}

	if metrics.ContainerID == "" {
		t.Error("ContainerID should not be empty")
	}
	if metrics.CPUPercent != 25.5 {
		t.Errorf("Expected CPU 25.5%%, got %.1f%%", metrics.CPUPercent)
	}
	if metrics.MemoryPercent != 50.0 {
		t.Errorf("Expected Memory 50%%, got %.1f%%", metrics.MemoryPercent)
	}
	// Silence linter
	_ = metrics.ContainerName
	_ = metrics.MemoryUsed
	_ = metrics.MemoryLimit
	_ = metrics.NetworkRx
	_ = metrics.NetworkTx
	_ = metrics.PIDs
	_ = metrics.Timestamp
}

func TestLogEntry(t *testing.T) {
	entry := &LogEntry{
		ContainerID: "abc123",
		Timestamp:   time.Now(),
		Stream:      "stdout",
		Message:     "Hello, World!",
	}

	if entry.Stream != "stdout" && entry.Stream != "stderr" {
		t.Errorf("Stream should be stdout or stderr, got %s", entry.Stream)
	}
	// Silence linter
	_ = entry.ContainerID
	_ = entry.Timestamp
	_ = entry.Message
}

func TestContainerEvent(t *testing.T) {
	event := &ContainerEvent{
		ContainerID:   "abc123",
		ContainerName: "test-container",
		Type:          "container",
		Action:        "start",
		Timestamp:     time.Now(),
		Attributes:    map[string]string{"image": "ubuntu"},
	}

	if event.Type != "container" {
		t.Error("Event type should be 'container'")
	}
	if event.Action != "start" {
		t.Error("Event action should be 'start'")
	}
	// Silence linter
	_ = event.ContainerID
	_ = event.ContainerName
	_ = event.Timestamp
	_ = event.Attributes
}

func TestContainerInfo(t *testing.T) {
	info := &ContainerInfo{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "ubuntu:22.04",
		State:   "running",
		Status:  "Up 5 minutes",
		Created: time.Now().Add(-5 * time.Minute),
		Ports: []PortMapping{
			{ContainerPort: 8080, HostPort: 8080, Protocol: "tcp"},
		},
		Labels: map[string]string{"cm.managed_by": "container-maker"},
	}

	if info.State != "running" {
		t.Error("State should be 'running'")
	}
	if len(info.Ports) != 1 {
		t.Error("Should have 1 port mapping")
	}
	if info.Ports[0].ContainerPort != 8080 {
		t.Error("Container port should be 8080")
	}
	// Silence linter
	_ = info.ID
	_ = info.Name
	_ = info.Image
	_ = info.Status
	_ = info.Created
	_ = info.Labels
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
		{1536 * 1024 * 1024, "1.5GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.max)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.max, result, tt.expected)
		}
	}
}

func TestRenderBar(t *testing.T) {
	// Test 0%
	bar0 := renderBar(0, 100, 10)
	if len(bar0) == 0 {
		t.Error("Bar should not be empty")
	}

	// Test 50%
	bar50 := renderBar(50, 100, 10)
	if len(bar50) == 0 {
		t.Error("Bar should not be empty")
	}

	// Test 100%
	bar100 := renderBar(100, 100, 10)
	if len(bar100) == 0 {
		t.Error("Bar should not be empty")
	}

	// Test overflow (>100%)
	barOver := renderBar(150, 100, 10)
	if len(barOver) == 0 {
		t.Error("Bar should not be empty")
	}
}

func TestMax(t *testing.T) {
	if max(5, 3) != 5 {
		t.Error("max(5, 3) should be 5")
	}
	if max(3, 5) != 5 {
		t.Error("max(3, 5) should be 5")
	}
	if max(5, 5) != 5 {
		t.Error("max(5, 5) should be 5")
	}
}

func TestFilterOptions(t *testing.T) {
	opts := FilterOptions{
		All:    true,
		Labels: map[string]string{"env": "prod"},
		Name:   "cm-*",
		Status: []string{"running"},
	}

	if !opts.All {
		t.Error("All should be true")
	}
	if opts.Labels["env"] != "prod" {
		t.Error("Labels should contain env=prod")
	}
	// Silence linter
	_ = opts.Name
	_ = opts.Status
}

func TestSystemMetrics(t *testing.T) {
	metrics := &SystemMetrics{
		TotalCPU:       400.0, // 4 cores
		TotalMemory:    16 * 1024 * 1024 * 1024,
		UsedMemory:     8 * 1024 * 1024 * 1024,
		ContainerCount: 10,
		RunningCount:   5,
		StoppedCount:   5,
		Timestamp:      time.Now(),
	}

	if metrics.ContainerCount != metrics.RunningCount+metrics.StoppedCount {
		t.Error("Container counts should add up")
	}
	// Silence linter
	_ = metrics.TotalCPU
	_ = metrics.TotalMemory
	_ = metrics.UsedMemory
	_ = metrics.Timestamp
}

func TestDashboardState(t *testing.T) {
	state := &DashboardState{
		Containers: []*ContainerInfo{
			{ID: "abc", Name: "test", State: "running"},
		},
		Metrics: map[string]*ContainerMetrics{
			"abc": {CPUPercent: 10.0},
		},
	}

	if len(state.Containers) != 1 {
		t.Error("Should have 1 container")
	}
	if state.Metrics["abc"].CPUPercent != 10.0 {
		t.Error("Metrics should be accessible")
	}
}
