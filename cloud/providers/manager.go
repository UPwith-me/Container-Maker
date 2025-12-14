// Package providers provides cloud provider management and registry
package providers

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages multiple cloud providers and their credentials
type Manager struct {
	mu        sync.RWMutex
	providers map[ProviderType]Provider
}

// NewManager creates a new provider manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[ProviderType]Provider),
	}
}

// Register registers a provider implementation
func (m *Manager) Register(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.Name()] = provider
}

// Get returns a provider by name
func (m *Manager) Get(name ProviderType) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// List returns all registered providers
func (m *Manager) List() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Provider, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result
}

// ListAvailable returns providers that have valid credentials
func (m *Manager) ListAvailable(ctx context.Context) []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Provider, 0)
	for _, p := range m.providers {
		if p.IsAvailable(ctx) {
			result = append(result, p)
		}
	}
	return result
}

// InitializeWithCredentials initializes providers with user credentials
func (m *Manager) InitializeWithCredentials(creds map[ProviderType]map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for providerType, credentials := range creds {
		if provider, ok := m.providers[providerType]; ok {
			if err := provider.Configure(credentials); err != nil {
				return fmt.Errorf("failed to configure %s: %w", providerType, err)
			}
		}
	}
	return nil
}

// GetDefaultManager returns a manager with all built-in providers registered
func GetDefaultManager() *Manager {
	m := NewManager()

	// Register all built-in providers
	m.Register(NewDockerProvider())
	m.Register(NewAWSProvider())
	m.Register(NewGCPProvider())
	m.Register(NewAzureProvider())
	m.Register(NewDigitalOceanProvider())
	m.Register(NewLinodeProvider())
	m.Register(NewVultrProvider())
	m.Register(NewHetznerProvider())
	m.Register(NewOCIProvider())
	m.Register(NewAlibabaProvider())
	m.Register(NewTencentProvider())
	m.Register(NewLambdaLabsProvider())
	m.Register(NewRunPodProvider())
	m.Register(NewVastAIProvider())

	return m
}

// ProviderInfo contains provider metadata for API responses
type ProviderInfo struct {
	Name                string   `json:"name"`
	DisplayName         string   `json:"display_name"`
	Description         string   `json:"description"`
	Website             string   `json:"website"`
	Status              string   `json:"status"` // available, configured, unavailable
	Features            []string `json:"features"`
	RequiredCredentials []string `json:"required_credentials"`
}

// GetProviderInfo returns API-friendly provider information
func GetProviderInfo(p Provider, ctx context.Context) ProviderInfo {
	status := "unavailable"
	if p.IsAvailable(ctx) {
		status = "available"
	}

	return ProviderInfo{
		Name:                string(p.Name()),
		DisplayName:         p.DisplayName(),
		Description:         p.Description(),
		Website:             p.Website(),
		Status:              status,
		Features:            p.Features(),
		RequiredCredentials: p.RequiredCredentials(),
	}
}
