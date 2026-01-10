package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/runtime"
)

// Snapshot represents a saved environment state
type Snapshot struct {
	Name        string    `json:"name"`
	ImageID     string    `json:"image_id"`
	ImageTag    string    `json:"image_tag"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description,omitempty"`
	SourceImage string    `json:"source_image,omitempty"`
}

type SnapshotRegistry struct {
	Snapshots map[string]Snapshot `json:"snapshots"`
}

type Manager struct {
	runtime runtime.ContainerRuntime
}

func NewManager(r runtime.ContainerRuntime) *Manager {
	return &Manager{runtime: r}
}

func registryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cm", "snapshots.json"), nil
}

func (m *Manager) loadRegistry() (*SnapshotRegistry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SnapshotRegistry{Snapshots: make(map[string]Snapshot)}, nil
		}
		return nil, err
	}

	var reg SnapshotRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return &SnapshotRegistry{Snapshots: make(map[string]Snapshot)}, nil
	}
	if reg.Snapshots == nil {
		reg.Snapshots = make(map[string]Snapshot)
	}
	return &reg, nil
}

func (m *Manager) saveRegistry(reg *SnapshotRegistry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmp, err := os.CreateTemp(dir, "snapshots-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

// CreateSnapshot creates a new snapshot from a running container
func (m *Manager) CreateSnapshot(ctx context.Context, containerID, name, description string) (*Snapshot, error) {
	// 1. Validate name
	if name == "" {
		return nil, fmt.Errorf("snapshot name required")
	}

	reg, err := m.loadRegistry()
	if err != nil {
		return nil, err
	}
	if _, exists := reg.Snapshots[name]; exists {
		return nil, fmt.Errorf("snapshot '%s' already exists", name)
	}

	// 2. Commit container
	// Tag format: cm-snapshot-<name>-<timestamp>
	timestamp := time.Now().Format("20060102-150405")
	tag := fmt.Sprintf("cm-snapshot-%s-%s", name, timestamp)
	repo := "cm-snapshots" // Local repo name

	commitOpts := runtime.CommitOptions{
		Repository: repo,
		Tag:        tag,
		Comment:    fmt.Sprintf("Snapshot %s: %s", name, description),
		Author:     "Container-Maker",
		Pause:      true, // Pause for consistency
	}

	imageID, err := m.runtime.CommitContainer(ctx, containerID, commitOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to commit container: %w", err)
	}

	// 3. Save to registry
	snap := Snapshot{
		Name:        name,
		ImageID:     imageID,
		ImageTag:    fmt.Sprintf("%s:%s", repo, tag),
		CreatedAt:   time.Now(),
		Description: description,
	}
	reg.Snapshots[name] = snap

	if err := m.saveRegistry(reg); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return &snap, nil
}

// ListSnapshots returns all snapshots sorted by creation time (newest first)
func (m *Manager) ListSnapshots() ([]Snapshot, error) {
	reg, err := m.loadRegistry()
	if err != nil {
		return nil, err
	}

	var list []Snapshot
	for _, s := range reg.Snapshots {
		list = append(list, s)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return list, nil
}

// RestoreSnapshot restores a snapshot by returning its Image Tag.
// The caller is responsible for re-configuring the project to use this image.
func (m *Manager) RestoreSnapshot(name string) (*Snapshot, error) {
	reg, err := m.loadRegistry()
	if err != nil {
		return nil, err
	}

	snap, exists := reg.Snapshots[name]
	if !exists {
		return nil, fmt.Errorf("snapshot '%s' not found", name)
	}

	// Verify image exists
	ctx := context.Background()
	if !m.runtime.ImageExists(ctx, snap.ImageTag) {
		return nil, fmt.Errorf("snapshot image %s not found (may have been pruned)", snap.ImageTag)
	}

	return &snap, nil
}

// DeleteSnapshot removes a snapshot from registry and attempts to remove the image
func (m *Manager) DeleteSnapshot(ctx context.Context, name string) error {
	reg, err := m.loadRegistry()
	if err != nil {
		return err
	}

	snap, exists := reg.Snapshots[name]
	if !exists {
		return fmt.Errorf("snapshot '%s' not found", name)
	}

	// Remove from map
	delete(reg.Snapshots, name)
	if err := m.saveRegistry(reg); err != nil {
		return err
	}

	// Attempt to remove image (best effort)
	if err := m.runtime.RemoveImage(ctx, snap.ImageTag, true); err != nil {
		// Just log or ignore since metadata is gone
		return nil
	}

	return nil
}
