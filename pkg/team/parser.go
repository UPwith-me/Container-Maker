package team

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TeamTemplate represents a parsed team template
type TeamTemplate struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Version     string   `json:"version" yaml:"version"`
	Maintainer  string   `json:"maintainer" yaml:"maintainer"`
	Tags        []string `json:"tags" yaml:"tags"`
	Path        string   `json:"-" yaml:"-"` // Local path to template
	RepoName    string   `json:"-" yaml:"-"` // Source repository name
}

// Manifest represents a team repository manifest
type Manifest struct {
	Version    string            `yaml:"version"`
	Name       string            `yaml:"name"`
	Maintainer string            `yaml:"maintainer"`
	Variables  map[string]string `yaml:"variables,omitempty"`
}

// GetAllTeamTemplates returns all templates from all cached repos
func GetAllTeamTemplates() (map[string][]TeamTemplate, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]TeamTemplate)

	// List all cached repos
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoName := entry.Name()
		repoDir := filepath.Join(cacheDir, repoName)

		templates, err := parseTemplatesInRepo(repoDir, repoName)
		if err != nil {
			continue // Skip repos with errors
		}

		if len(templates) > 0 {
			result[repoName] = templates
		}
	}

	return result, nil
}

// parseTemplatesInRepo parses all templates in a repository
func parseTemplatesInRepo(repoDir, repoName string) ([]TeamTemplate, error) {
	var templates []TeamTemplate

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and .git
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		templateDir := filepath.Join(repoDir, entry.Name())

		// Check for devcontainer.json
		dcPath := filepath.Join(templateDir, "devcontainer.json")
		if _, err := os.Stat(dcPath); os.IsNotExist(err) {
			// Also check .devcontainer subdirectory
			dcPath = filepath.Join(templateDir, ".devcontainer", "devcontainer.json")
			if _, err := os.Stat(dcPath); os.IsNotExist(err) {
				continue
			}
		}

		template := TeamTemplate{
			Name:     entry.Name(),
			Path:     templateDir,
			RepoName: repoName,
		}

		// Try to load template.yaml for metadata
		templateYaml := filepath.Join(templateDir, "template.yaml")
		if data, err := os.ReadFile(templateYaml); err == nil {
			var meta TeamTemplate
			if err := yaml.Unmarshal(data, &meta); err == nil {
				if meta.Name != "" {
					template.Name = meta.Name
				}
				template.Description = meta.Description
				template.Version = meta.Version
				template.Maintainer = meta.Maintainer
				template.Tags = meta.Tags
			}
		}

		// Fallback: try to extract name from devcontainer.json
		if template.Description == "" {
			if dcData, err := os.ReadFile(dcPath); err == nil {
				var dc map[string]interface{}
				if err := json.Unmarshal(dcData, &dc); err == nil {
					if name, ok := dc["name"].(string); ok && template.Name == entry.Name() {
						template.Name = name
					}
				}
			}
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// GetTeamTemplate finds a specific team template by repo/name
func GetTeamTemplate(repoName, templateName string) (*TeamTemplate, error) {
	allTemplates, err := GetAllTeamTemplates()
	if err != nil {
		return nil, err
	}

	templates, ok := allTemplates[repoName]
	if !ok {
		return nil, os.ErrNotExist
	}

	for _, t := range templates {
		if t.Name == templateName || filepath.Base(t.Path) == templateName {
			return &t, nil
		}
	}

	return nil, os.ErrNotExist
}

// GetTemplatePath returns the path to a team template's directory
func GetTemplatePath(repoName, templateName string) (string, error) {
	template, err := GetTeamTemplate(repoName, templateName)
	if err != nil {
		return "", err
	}
	return template.Path, nil
}
