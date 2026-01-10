package plugin

import (
	"context"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// PluginMetadata holds plugin manifest information
type PluginMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author,omitempty"`
}

// Plugin defines the interface all plugins must implement
type Plugin interface {
	// Metadata returns the plugin metadata
	Metadata() PluginMetadata

	// Init initializes the plugin with configuration
	Init(ctx context.Context, config map[string]interface{}) error

	// Close cleans up plugin resources
	Close() error
}

// LifecyclePlugin allows hooking into the workspace lifecycle
type LifecyclePlugin interface {
	Plugin
	PreStart(ctx context.Context, ws *workspace.Workspace) error
	PostStart(ctx context.Context, ws *workspace.Workspace) error
	PreStop(ctx context.Context, ws *workspace.Workspace) error
	PostStop(ctx context.Context, ws *workspace.Workspace) error
}

// ExecutablePlugin is a plugin that runs as an external process
type ExecutablePlugin interface {
	Plugin
	// Execute runs a custom command provided by the plugin
	Execute(ctx context.Context, args []string, env []string) error
}
