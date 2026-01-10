package remote

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PortForwarder manages port forwarding from local to remote
type PortForwarder struct {
	host       string
	forwards   map[int]*Forward
	mu         sync.RWMutex
	activeConn sync.WaitGroup
}

// Forward represents a single port forward
type Forward struct {
	LocalPort   int
	RemotePort  int
	Protocol    string // "tcp" or "udp"
	listener    net.Listener
	active      bool
	connections int
	cancel      context.CancelFunc
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(host string) *PortForwarder {
	return &PortForwarder{
		host:     host,
		forwards: make(map[int]*Forward),
	}
}

// ForwardPort starts forwarding a local port to a remote port
func (pf *PortForwarder) ForwardPort(ctx context.Context, localPort, remotePort int) error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	// Check if already forwarding
	if _, exists := pf.forwards[localPort]; exists {
		return fmt.Errorf("local port %d is already being forwarded", localPort)
	}

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", localPort, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	forward := &Forward{
		LocalPort:  localPort,
		RemotePort: remotePort,
		Protocol:   "tcp",
		listener:   listener,
		active:     true,
		cancel:     cancel,
	}

	pf.forwards[localPort] = forward

	// Start accepting connections
	go pf.acceptConnections(ctx, forward)

	fmt.Printf("📡 Forwarding localhost:%d → %s:%d\n", localPort, pf.host, remotePort)
	return nil
}

// acceptConnections handles incoming connections
func (pf *PortForwarder) acceptConnections(ctx context.Context, fwd *Forward) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := fwd.listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			pf.activeConn.Add(1)
			fwd.connections++
			go pf.handleConnection(ctx, conn, fwd)
		}
	}
}

// handleConnection forwards a single connection via SSH
func (pf *PortForwarder) handleConnection(ctx context.Context, local net.Conn, fwd *Forward) {
	defer pf.activeConn.Done()
	defer local.Close()

	// Create SSH tunnel to remote port
	// Using ssh -L for port forwarding through SSH
	remoteAddr := fmt.Sprintf("localhost:%d", fwd.RemotePort)

	// Use nc (netcat) on the remote to connect to the target port
	cmd := exec.CommandContext(ctx, "ssh", pf.host,
		"nc", "-w", "60", "localhost", strconv.Itoa(fwd.RemotePort))

	remoteIn, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	remoteOut, err := cmd.StdoutPipe()
	if err != nil {
		remoteIn.Close()
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", remoteAddr, err)
		return
	}

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Local -> Remote
	go func() {
		defer wg.Done()
		_, _ = io.Copy(remoteIn, local)
		remoteIn.Close()
	}()

	// Remote -> Local
	go func() {
		defer wg.Done()
		_, _ = io.Copy(local, remoteOut)
		local.Close()
	}()

	wg.Wait()
	_ = cmd.Wait()
}

// StopForward stops forwarding a specific port
func (pf *PortForwarder) StopForward(localPort int) error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	fwd, exists := pf.forwards[localPort]
	if !exists {
		return fmt.Errorf("port %d is not being forwarded", localPort)
	}

	fwd.cancel()
	fwd.listener.Close()
	fwd.active = false
	delete(pf.forwards, localPort)

	fmt.Printf("🔌 Stopped forwarding port %d\n", localPort)
	return nil
}

// StopAll stops all port forwards
func (pf *PortForwarder) StopAll() {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	for port, fwd := range pf.forwards {
		fwd.cancel()
		fwd.listener.Close()
		fwd.active = false
		delete(pf.forwards, port)
	}

	// Wait for active connections to close
	pf.activeConn.Wait()
}

// List returns all active forwards
func (pf *PortForwarder) List() []Forward {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	var forwards []Forward
	for _, fwd := range pf.forwards {
		forwards = append(forwards, *fwd)
	}
	return forwards
}

// AutoDetectPorts detects ports from a container and forwards them
func (pf *PortForwarder) AutoDetectPorts(ctx context.Context, containerName string) ([]int, error) {
	// Get exposed ports from the container
	cmd := exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=10", pf.host,
		"docker", "port", containerName)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container ports: %w", err)
	}

	var ports []int
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		// Format: "8080/tcp -> 0.0.0.0:8080"
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "/")
		if len(parts) < 1 {
			continue
		}

		port, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		// Find available local port
		localPort := pf.findAvailablePort(port)
		if localPort == 0 {
			continue
		}

		if err := pf.ForwardPort(ctx, localPort, port); err != nil {
			fmt.Printf("Warning: failed to forward port %d: %v\n", port, err)
			continue
		}

		ports = append(ports, port)
	}

	return ports, nil
}

// findAvailablePort finds an available local port, starting from preferred
func (pf *PortForwarder) findAvailablePort(preferred int) int {
	// Try the preferred port first
	if pf.isPortAvailable(preferred) {
		return preferred
	}

	// Try ports in range around preferred
	for offset := 1; offset < 100; offset++ {
		if pf.isPortAvailable(preferred + offset) {
			return preferred + offset
		}
	}

	// Find any available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// isPortAvailable checks if a port is available
func (pf *PortForwarder) isPortAvailable(port int) bool {
	pf.mu.RLock()
	_, taken := pf.forwards[port]
	pf.mu.RUnlock()
	if taken {
		return false
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// ForwardMultiple forwards multiple ports at once
func (pf *PortForwarder) ForwardMultiple(ctx context.Context, ports map[int]int) error {
	var errs []string
	for local, remote := range ports {
		if err := pf.ForwardPort(ctx, local, remote); err != nil {
			errs = append(errs, fmt.Sprintf("%d: %v", local, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("some ports failed to forward: %s", strings.Join(errs, "; "))
	}
	return nil
}

// WaitForPort waits for a port to become available on remote
func (pf *PortForwarder) WaitForPort(ctx context.Context, port int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for port %d", port)
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=2", pf.host,
				"nc", "-z", "localhost", strconv.Itoa(port))
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}
}

// PrintStatus prints the current forwarding status
func (pf *PortForwarder) PrintStatus() {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	if len(pf.forwards) == 0 {
		fmt.Println("No active port forwards")
		return
	}

	fmt.Println("📡 Active Port Forwards:")
	fmt.Println("─────────────────────────")
	for _, fwd := range pf.forwards {
		status := "active"
		if !fwd.active {
			status = "inactive"
		}
		fmt.Printf("  localhost:%d → %s:%d [%s] (%d connections)\n",
			fwd.LocalPort, pf.host, fwd.RemotePort, status, fwd.connections)
	}
}
