package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// Manager manages plugins
type Manager struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton plugin manager
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			plugins: make(map[string]Plugin),
		}
		// Register built-in plugins
		_ = instance.Register(NewAuditPlugin())
	})
	return instance
}

// DiscoverPlugins scans the plugin directory for executables
func (m *Manager) DiscoverPlugins(ctx context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pluginDir := filepath.Join(home, ".cm", "plugins")

	// Create if not exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		_ = os.MkdirAll(pluginDir, 0755)
		return nil
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || info.IsDir() {
			continue
		}

		// Convention: cm-plugin-* or just cm-* ?
		// "cm-" prefix matches `kubectl` style.
		if !strings.HasPrefix(entry.Name(), "cm-") {
			continue
		}

		fullPath := filepath.Join(pluginDir, entry.Name())
		name := strings.TrimPrefix(entry.Name(), "cm-")

		// Create plugin with name derived from filename (Lazy Loading)
		p := &ProcessPlugin{
			Path: fullPath,
			meta: PluginMetadata{
				Name: name, // Temporary name until full metadata loaded
				ID:   fmt.Sprintf("external.%s", name),
			},
		}

		_ = m.Register(p)
	}
	return nil
}

// Register registers a new plugin
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta := p.Metadata()
	m.plugins[meta.Name] = p // Key by Name for easier CLI invocation
	return nil
}

// GetPlugins returns all registered plugins
func (m *Manager) GetPlugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []Plugin
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

// GetPlugin returns a specific plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// --- Process Plugin Implementation ---

type ProcessPlugin struct {
	Path   string
	meta   PluginMetadata
	loaded bool
	mu     sync.Mutex
}

func (p *ProcessPlugin) loadMetadata(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.loaded {
		return nil
	}

	// Execute: ./plugin --metadata
	// Expects JSON on stdout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.Path, "--metadata")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// Fallback: If --metadata fails, valid executable is still a plugin, but description is empty
		// We treat it as valid to allow simple shell scripts.
		p.loaded = true
		return nil
	}

	var newMeta PluginMetadata
	if err := json.Unmarshal(stdout.Bytes(), &newMeta); err == nil {
		p.meta = newMeta
	}
	// If json fail, keep filename derived name

	p.loaded = true
	return nil
}

func (p *ProcessPlugin) Metadata() PluginMetadata {
	// Optimistic: Return what we have (Name from filename).
	// If caller needs full metadata (Description for help), they should trigger load?
	// But Metadata() is sync.
	// For "Industrial", we trigger load if not loaded, but inside a short timeout or async?
	// To keep CLI fast, we return cached/filename-derived metadata for command registration.
	// Only when 'cm plugin list' is called, we might want full details.

	// Lazy load on first access? No, that blocks 'cm' startup if we iterate all.
	// We only strictly need Name for registration. We have it.
	// We return p.meta (which has Name).

	// If one wants description, they might see empty string unless loadMetadata called.
	// 'cm plugin list' calls Metadata(). We should auto-load there?
	// Yes.

	if !p.loaded {
		// Use a detached context or background?
		// We'll try to load with short timeout if this is called.
		_ = p.loadMetadata(context.Background())
	}

	p.mu.Lock() // Re-acquire lock to read safe? loadMetadata has lock.
	// We need RLock logic if separate.
	// Simplified: access p.meta. strictly speaking requires lock.
	defer p.mu.Unlock()
	return p.meta // Need to ensure ProcessPlugin struct fields are protected or not concurrent
}

// Ensure Metadata() is safe implies we need better locking.
// For now, simple lock is fine.

func (p *ProcessPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	return nil
}

func (p *ProcessPlugin) Close() error {
	return nil
}

func (p *ProcessPlugin) Execute(ctx context.Context, args []string, env []string) error {
	// Pass arguments directly to plugin executable
	cmd := exec.CommandContext(ctx, p.Path, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// --- Built-in Audit Plugin Example (Kept for compatibility) ---

type AuditPlugin struct{}

func NewAuditPlugin() *AuditPlugin { return &AuditPlugin{} }
func (p *AuditPlugin) Metadata() PluginMetadata {
	return PluginMetadata{
		ID: "builtin.audit", Name: "audit", Version: "1.0.0", Description: "Internal Lifecycle Logger",
	}
}
func (p *AuditPlugin) Init(ctx context.Context, config map[string]interface{}) error { return nil }
func (p *AuditPlugin) Close() error                                                  { return nil }
func (p *AuditPlugin) PreStart(ctx context.Context, ws *workspace.Workspace) error   { return nil }
func (p *AuditPlugin) PostStart(ctx context.Context, ws *workspace.Workspace) error  { return nil }
func (p *AuditPlugin) PreStop(ctx context.Context, ws *workspace.Workspace) error    { return nil }
func (p *AuditPlugin) PostStop(ctx context.Context, ws *workspace.Workspace) error   { return nil }
