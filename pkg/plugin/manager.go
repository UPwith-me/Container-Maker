package plugin

import (
	"context"
	"fmt"
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
		// Register built-in plugins here usually
		instance.Register(NewAuditPlugin())
	})
	return instance
}

// Register registers a new plugin
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta := p.Metadata()
	if _, exists := m.plugins[meta.ID]; exists {
		return fmt.Errorf("plugin %s already registered", meta.ID)
	}

	m.plugins[meta.ID] = p
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

// InitAll initializes all plugins
func (m *Manager) InitAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if err := p.Init(ctx, nil); err != nil {
			return fmt.Errorf("failed to init plugin %s: %w", p.Metadata().Name, err)
		}
	}
	return nil
}

// Hooks

func (m *Manager) PreStart(ctx context.Context, ws *workspace.Workspace) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if lp, ok := p.(LifecyclePlugin); ok {
			if err := lp.PreStart(ctx, ws); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) PostStart(ctx context.Context, ws *workspace.Workspace) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if lp, ok := p.(LifecyclePlugin); ok {
			if err := lp.PostStart(ctx, ws); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- Built-in Audit Plugin Example ---

type AuditPlugin struct{}

func NewAuditPlugin() *AuditPlugin {
	return &AuditPlugin{}
}

func (p *AuditPlugin) Metadata() PluginMetadata {
	return PluginMetadata{
		ID:          "builtin.audit",
		Name:        "Audit Logger",
		Version:     "1.0.0",
		Description: "Logs lifecycle events for auditing",
		Author:      "Container-Maker Team",
	}
}

func (p *AuditPlugin) Init(ctx context.Context, config map[string]interface{}) error {
	return nil
}

func (p *AuditPlugin) Close() error {
	return nil
}

func (p *AuditPlugin) PreStart(ctx context.Context, ws *workspace.Workspace) error {
	fmt.Printf("[AUDIT] Preparing to start workspace '%s' at %s\n", ws.Name, time.Now().Format(time.RFC3339))
	return nil
}

func (p *AuditPlugin) PostStart(ctx context.Context, ws *workspace.Workspace) error {
	fmt.Printf("[AUDIT] Successfully started workspace '%s' with %d services\n", ws.Name, len(ws.Services))
	return nil
}

func (p *AuditPlugin) PreStop(ctx context.Context, ws *workspace.Workspace) error {
	return nil
}

func (p *AuditPlugin) PostStop(ctx context.Context, ws *workspace.Workspace) error {
	return nil
}
