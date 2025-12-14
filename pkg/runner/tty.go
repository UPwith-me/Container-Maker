package runner

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/term"
)

// TTYManager handles terminal management for container interaction
type TTYManager struct {
	client      *client.Client
	containerID string
	resizeChan  chan os.Signal
	done        chan struct{}
}

// NewTTYManager creates a new TTY manager
func NewTTYManager(cli *client.Client, containerID string) *TTYManager {
	return &TTYManager{
		client:      cli,
		containerID: containerID,
		resizeChan:  make(chan os.Signal, 1),
		done:        make(chan struct{}),
	}
}

// StartResizeMonitor starts monitoring for terminal resize signals
func (t *TTYManager) StartResizeMonitor(ctx context.Context) {
	// Register for SIGWINCH (window size change) - Unix only
	signal.Notify(t.resizeChan, syscall.Signal(0x1c)) // SIGWINCH = 28

	go func() {
		// Initial resize
		t.resizeContainer(ctx)

		for {
			select {
			case <-t.resizeChan:
				t.resizeContainer(ctx)
			case <-t.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopResizeMonitor stops the resize monitor
func (t *TTYManager) StopResizeMonitor() {
	signal.Stop(t.resizeChan)
	close(t.done)
}

// resizeContainer syncs the container's TTY size with the host terminal
func (t *TTYManager) resizeContainer(ctx context.Context) error {
	// Get current terminal size
	width, height := GetTerminalSize()
	if width == 0 || height == 0 {
		return nil
	}

	// Resize container TTY
	return t.client.ContainerResize(ctx, t.containerID, container.ResizeOptions{
		Height: uint(height),
		Width:  uint(width),
	})
}

// GetTerminalSize returns the current terminal dimensions
func GetTerminalSize() (width, height int) {
	fd := int(os.Stdout.Fd())

	w, h, err := term.GetSize(fd)
	if err != nil {
		// Fallback to defaults
		return 80, 24
	}

	return w, h
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// MakeRaw puts the terminal into raw mode
func MakeRaw() (*term.State, error) {
	fd := int(os.Stdin.Fd())
	return term.MakeRaw(fd)
}

// RestoreTerminal restores the terminal to its previous state
func RestoreTerminal(state *term.State) error {
	fd := int(os.Stdin.Fd())
	return term.Restore(fd, state)
}
