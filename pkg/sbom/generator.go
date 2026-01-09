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

// SBOM represents a Software Bill of Materials (CycloneDX 1.5)
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
	Type        string   `json:"type"` // application, library, framework
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	PURL        string   `json:"purl,omitempty"`     // Package URL
	Scope       string   `json:"scope,omitempty"`    // required, optional, dev
	Licenses    []string `json:"licenses,omitempty"` // License IDs
	Description string   `json:"description,omitempty"`
}

// GenerateSBOM generates an SBOM for the given path
func GenerateSBOM(path string, projectName string) (*SBOM, error) {
	sbom := &SBOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: fmt.Sprintf("urn:uuid:%d", time.Now().UnixNano()),
		Version:      1,
		Metadata: Metadata{
			Timestamp: time.Now().Format(time.RFC3339),
			Tool: Tool{
				Vendor:  "Container-Maker",
				Name:    "cm-sbom",
				Version: "2.0",
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
		if info.IsDir() && (info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "venv" || info.Name() == "target" || info.Name() == "__pycache__") {
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
			parsePomXml(path, sbom)
		case "Cargo.toml":
			parseCargoToml(path, sbom)
		case "Gemfile.lock":
			parseGemfileLock(path, sbom)
		case "composer.json":
			parseComposerJson(path, sbom)
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

// parsePomXml parses Maven POM files for Java dependencies
func parsePomXml(path string, sbom *SBOM) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	// Simple regex-like parsing for dependencies
	// Looking for <groupId>...</groupId><artifactId>...</artifactId><version>...</version>
	lines := strings.Split(content, "\n")
	var groupId, artifactId, version string
	inDependency := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "<dependency>") {
			inDependency = true
			groupId, artifactId, version = "", "", ""
		}
		if inDependency {
			if strings.Contains(line, "<groupId>") {
				groupId = extractXMLValue(line, "groupId")
			}
			if strings.Contains(line, "<artifactId>") {
				artifactId = extractXMLValue(line, "artifactId")
			}
			if strings.Contains(line, "<version>") {
				version = extractXMLValue(line, "version")
			}
		}
		if strings.Contains(line, "</dependency>") && inDependency {
			inDependency = false
			if groupId != "" && artifactId != "" {
				sbom.Components = append(sbom.Components, Component{
					Type:    "library",
					Name:    fmt.Sprintf("%s:%s", groupId, artifactId),
					Version: version,
					PURL:    fmt.Sprintf("pkg:maven/%s/%s@%s", groupId, artifactId, version),
				})
			}
		}
	}
}

func extractXMLValue(line, tag string) string {
	start := fmt.Sprintf("<%s>", tag)
	end := fmt.Sprintf("</%s>", tag)
	if i := strings.Index(line, start); i != -1 {
		line = line[i+len(start):]
		if j := strings.Index(line, end); j != -1 {
			return strings.TrimSpace(line[:j])
		}
	}
	return ""
}

// parseCargoToml parses Rust Cargo.toml files
func parseCargoToml(path string, sbom *SBOM) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inDependencies := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[dependencies]") || strings.HasPrefix(line, "[dev-dependencies]") {
			inDependencies = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDependencies = false
			continue
		}
		if inDependencies && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				// Handle complex version specs like { version = "1.0" }
				if strings.Contains(version, "version") {
					if i := strings.Index(version, "\""); i != -1 {
						rest := version[i+1:]
						if j := strings.Index(rest, "\""); j != -1 {
							version = rest[:j]
						}
					}
				}
				sbom.Components = append(sbom.Components, Component{
					Type:    "library",
					Name:    name,
					Version: version,
					PURL:    fmt.Sprintf("pkg:cargo/%s@%s", name, version),
				})
			}
		}
	}
}

// parseGemfileLock parses Ruby Gemfile.lock files
func parseGemfileLock(path string, sbom *SBOM) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inSpecs := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "specs:" {
			inSpecs = true
			continue
		}
		if inSpecs && len(line) > 0 && line[0] != ' ' {
			inSpecs = false
			continue
		}
		if inSpecs && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") {
			// Gem line: "    gemname (version)"
			line = strings.TrimSpace(line)
			if i := strings.Index(line, " ("); i != -1 {
				name := line[:i]
				version := strings.Trim(line[i+2:], ")")
				sbom.Components = append(sbom.Components, Component{
					Type:    "library",
					Name:    name,
					Version: version,
					PURL:    fmt.Sprintf("pkg:gem/%s@%s", name, version),
				})
			}
		}
	}
}

// parseComposerJson parses PHP composer.json files
func parseComposerJson(path string, sbom *SBOM) {
	type ComposerJSON struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return
	}

	for name, version := range composer.Require {
		if strings.HasPrefix(name, "php") || strings.HasPrefix(name, "ext-") {
			continue // Skip PHP itself and extensions
		}
		version = cleanVersion(version)
		sbom.Components = append(sbom.Components, Component{
			Type:    "library",
			Name:    name,
			Version: version,
			PURL:    fmt.Sprintf("pkg:composer/%s@%s", name, version),
			Scope:   "required",
		})
	}

	for name, version := range composer.RequireDev {
		version = cleanVersion(version)
		sbom.Components = append(sbom.Components, Component{
			Type:    "library",
			Name:    name,
			Version: version,
			PURL:    fmt.Sprintf("pkg:composer/%s@%s", name, version),
			Scope:   "dev",
		})
	}
}
