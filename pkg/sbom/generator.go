package sbom

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SBOM represents a Software Bill of Materials (simplified CycloneDX)
type SBOM struct {
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber"`
	Version      int         `json:"version"`
	Metadata     Metadata    `json:"metadata"`
	Components   []Component `json:"components"`
}

type Metadata struct {
	Timestamp string    `json:"timestamp"`
	Tool      Tool      `json:"tool"`
	Component Component `json:"component"` // The root component (workspace)
}

type Tool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Component struct {
	Type    string `json:"type"` // application, library
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	PURL    string `json:"purl,omitempty"` // Package URL
}

// GenerateSBOM generates an SBOM for the given path
func GenerateSBOM(path string, projectName string) (*SBOM, error) {
	sbom := &SBOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.4",
		SerialNumber: fmt.Sprintf("urn:uuid:%d", time.Now().UnixNano()),
		Version:      1,
		Metadata: Metadata{
			Timestamp: time.Now().Format(time.RFC3339),
			Tool: Tool{
				Vendor:  "Container-Maker",
				Name:    "cm-sbom",
				Version: "1.0",
			},
			Component: Component{
				Type: "application",
				Name: projectName,
			},
		},
		Components: make([]Component, 0),
	}

	if err := scanDirectory(path, sbom); err != nil {
		return nil, err
	}

	return sbom, nil
}

func scanDirectory(root string, sbom *SBOM) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "venv") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		filename := filepath.Base(path)
		switch filename {
		case "go.mod":
			parseGoMod(path, sbom)
		case "package.json":
			parsePackageJson(path, sbom)
		case "requirements.txt":
			parseRequirementsTxt(path, sbom)
		case "pom.xml":
			// Basic XML parsing placeholder
		}
		return nil
	})
}

func parseGoMod(path string, sbom *SBOM) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") {
			continue
		}
		if strings.HasPrefix(line, ")") {
			return
		}

		// Simple parser for direct requires
		// require github.com/foo/bar v1.0.0
		parts := strings.Fields(line)
		if len(parts) >= 2 && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "module") && !strings.HasPrefix(line, "go ") {
			// Handle "require package version" lines properly?
			// go.mod format is tricky, let's look for lines with at least 2 fields that look like package and version
			// Rough heuristic for demonstration
			if strings.Contains(parts[0], ".") {
				comp := Component{
					Type:    "library",
					Name:    parts[0],
					Version: parts[1],
					PURL:    fmt.Sprintf("pkg:golang/%s@%s", parts[0], parts[1]),
				}
				sbom.Components = append(sbom.Components, comp)
			}
		}
	}
}

func parsePackageJson(path string, sbom *SBOM) {
	type PackageJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	for name, version := range pkg.Dependencies {
		version = cleanVersion(version)
		sbom.Components = append(sbom.Components, Component{
			Type:    "library",
			Name:    name,
			Version: version,
			PURL:    fmt.Sprintf("pkg:npm/%s@%s", name, version),
		})
	}
}

func parseRequirementsTxt(path string, sbom *SBOM) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// foo==1.0.0
		if strings.Contains(line, "==") {
			parts := strings.Split(line, "==")
			if len(parts) == 2 {
				sbom.Components = append(sbom.Components, Component{
					Type:    "library",
					Name:    parts[0],
					Version: parts[1],
					PURL:    fmt.Sprintf("pkg:pypi/%s@%s", parts[0], parts[1]),
				})
			}
		}
	}
}

func cleanVersion(v string) string {
	return strings.TrimPrefix(strings.TrimPrefix(v, "^"), "~")
}
