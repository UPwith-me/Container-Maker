// Package plugin provides an extensibility framework for Container-Maker.
// It allows extending functionality via lifecycle hooks and custom commands.
package plugin

import (
	"context"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// PluginMetadata describes a plugin
type PluginMetadata struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author" yaml:"author"`
	Website     string `json:"website" yaml:"website"`
}

// Plugin interface must be implemented by all plugins
type Plugin interface {
	// Metadata returns the plugin metadata
	Metadata() PluginMetadata

	// Init initializes the plugin
	Init(ctx context.Context, config map[string]interface{}) error

	// Close cleans up plugin resources
	Close() error
}

// LifecyclePlugin allows hooking into workspace lifecycle
type LifecyclePlugin interface {
	Plugin

	// PreStart is called before services start
	PreStart(ctx context.Context, ws *workspace.Workspace) error

	// PostStart is called after services start
	PostStart(ctx context.Context, ws *workspace.Workspace) error

	// PreStop is called before services stop
	PreStop(ctx context.Context, ws *workspace.Workspace) error

	// PostStop is called after services stop
	PostStop(ctx context.Context, ws *workspace.Workspace) error
}

// CLIPlugin allows adding custom CLI commands (placeholder for future expansion)
type CLIPlugin interface {
	Plugin

	// GetCommands returns custom commands to register
	// GetCommands() []*cobra.Command
}
