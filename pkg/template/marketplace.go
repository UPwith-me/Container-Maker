package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MarketplaceTemplate represents a template in the marketplace
type MarketplaceTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Stars       int       `json:"stars"`
	Downloads   int       `json:"downloads"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Marketplace provides access to community templates
type Marketplace struct {
	baseURL   string
	cacheDir  string
	templates []MarketplaceTemplate
}

// NewMarketplace creates a new marketplace client
func NewMarketplace() *Marketplace {
	home, _ := os.UserHomeDir()
	return &Marketplace{
		baseURL:  "https://raw.githubusercontent.com/devcontainers/templates/main",
		cacheDir: filepath.Join(home, ".cm", "marketplace"),
	}
}

// Search searches for templates in the marketplace
func (m *Marketplace) Search(query string) ([]MarketplaceTemplate, error) {
	// Load cached templates or fetch from remote
	if err := m.loadTemplates(); err != nil {
		return nil, err
	}

	if query == "" {
		return m.templates, nil
	}

	query = strings.ToLower(query)
	var results []MarketplaceTemplate
	for _, t := range m.templates {
		if strings.Contains(strings.ToLower(t.Name), query) ||
			strings.Contains(strings.ToLower(t.Description), query) ||
			strings.Contains(strings.ToLower(t.Category), query) {
			results = append(results, t)
		}
	}

	return results, nil
}

// GetTemplate gets a specific template by ID
func (m *Marketplace) GetTemplate(id string) (*MarketplaceTemplate, error) {
	if err := m.loadTemplates(); err != nil {
		return nil, err
	}

	for _, t := range m.templates {
		if t.ID == id {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("template not found: %s", id)
}

// Install downloads and installs a marketplace template
func (m *Marketplace) Install(id, targetDir string) error {
	tmpl, err := m.GetTemplate(id)
	if err != nil {
		return err
	}

	// Download template with retries
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = http.Get(tmpl.URL)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to download template after 3 attempts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("template not found (HTTP %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Create .devcontainer directory
	devcontainerDir := filepath.Join(targetDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return err
	}

	// Write devcontainer.json
	configPath := filepath.Join(devcontainerDir, "devcontainer.json")
	return os.WriteFile(configPath, content, 0644)
}

// loadTemplates loads templates from cache or fetches from remote
func (m *Marketplace) loadTemplates() error {
	if len(m.templates) > 0 {
		return nil
	}

	// Try to load from cache
	cachePath := filepath.Join(m.cacheDir, "templates.json")
	if data, err := os.ReadFile(cachePath); err == nil {
		if json.Unmarshal(data, &m.templates) == nil && len(m.templates) > 0 {
			return nil
		}
	}

	// Fetch from remote (using devcontainers/templates repo)
	m.templates = m.getDefaultTemplates()

	// Cache the templates
	_ = os.MkdirAll(m.cacheDir, 0755)
	if data, err := json.Marshal(m.templates); err == nil {
		_ = os.WriteFile(cachePath, data, 0644)
	}

	return nil
}

// getDefaultTemplates returns official devcontainer templates
func (m *Marketplace) getDefaultTemplates() []MarketplaceTemplate {
	return []MarketplaceTemplate{
		{
			ID:          "python",
			Name:        "Python",
			Author:      "devcontainers",
			Description: "Develop Python applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/python/.devcontainer/devcontainer.json",
		},
		{
			ID:          "javascript-node",
			Name:        "Node.js & JavaScript",
			Author:      "devcontainers",
			Description: "Develop Node.js based applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/javascript-node/.devcontainer/devcontainer.json",
		},
		{
			ID:          "typescript-node",
			Name:        "Node.js & TypeScript",
			Author:      "devcontainers",
			Description: "Develop TypeScript applications with Node.js",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/typescript-node/.devcontainer/devcontainer.json",
		},
		{
			ID:          "go",
			Name:        "Go",
			Author:      "devcontainers",
			Description: "Develop Go applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/go/.devcontainer/devcontainer.json",
		},
		{
			ID:          "rust",
			Name:        "Rust",
			Author:      "devcontainers",
			Description: "Develop Rust applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/rust/.devcontainer/devcontainer.json",
		},
		{
			ID:          "java",
			Name:        "Java",
			Author:      "devcontainers",
			Description: "Develop Java applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/java/.devcontainer/devcontainer.json",
		},
		{
			ID:          "cpp",
			Name:        "C++",
			Author:      "devcontainers",
			Description: "Develop C++ applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/cpp/.devcontainer/devcontainer.json",
		},
		{
			ID:          "dotnet",
			Name:        ".NET",
			Author:      "devcontainers",
			Description: "Develop .NET applications",
			Category:    "Languages",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/dotnet/.devcontainer/devcontainer.json",
		},
		{
			ID:          "docker-from-docker",
			Name:        "Docker from Docker",
			Author:      "devcontainers",
			Description: "Access your host's Docker install from inside a container",
			Category:    "DevOps",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/docker-from-docker/.devcontainer/devcontainer.json",
		},
		{
			ID:          "kubernetes-helm",
			Name:        "Kubernetes - Local Configuration",
			Author:      "devcontainers",
			Description: "Access a local Kubernetes cluster",
			Category:    "DevOps",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/kubernetes-helm/.devcontainer/devcontainer.json",
		},
		{
			ID:          "ubuntu",
			Name:        "Ubuntu",
			Author:      "devcontainers",
			Description: "A plain Ubuntu container with minimal tooling",
			Category:    "Base",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/ubuntu/.devcontainer/devcontainer.json",
		},
		{
			ID:          "alpine",
			Name:        "Alpine",
			Author:      "devcontainers",
			Description: "A minimal Alpine Linux container",
			Category:    "Base",
			URL:         "https://raw.githubusercontent.com/devcontainers/templates/main/src/alpine/.devcontainer/devcontainer.json",
		},
	}
}

// FormatTemplatesTable formats templates as a table (without fake metrics)
func (m *Marketplace) FormatTemplatesTable(templates []MarketplaceTemplate) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-18s %-32s %-12s %s\n", "ID", "Name", "Category", "Author"))
	sb.WriteString(strings.Repeat("─", 80) + "\n")
	for _, t := range templates {
		sb.WriteString(fmt.Sprintf("%-18s %-32s %-12s %s\n", t.ID, t.Name, t.Category, t.Author))
	}
	sb.WriteString("\n📌 Source: github.com/devcontainers/templates (Official)")
	return sb.String()
}
