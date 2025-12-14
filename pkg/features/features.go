package features

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Feature represents a DevContainer Feature
type Feature struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Options     map[string]interface{} `json:"options"`
	InstallSh   string                 // Content of install.sh
}

// FeatureRef represents a reference to a feature in devcontainer.json
type FeatureRef struct {
	Source  string                 // Full feature reference (e.g., "ghcr.io/devcontainers/features/go:1")
	ID      string                 // Feature ID (e.g., "go")
	Version string                 // Version (e.g., "1")
	Options map[string]interface{} // Feature options from config
}

// ParseFeatureRef parses a feature reference string
// Examples:
//   - "ghcr.io/devcontainers/features/go:1"
//   - "ghcr.io/devcontainers/features/docker-in-docker:2"
func ParseFeatureRef(source string, options interface{}) (*FeatureRef, error) {
	ref := &FeatureRef{
		Source: source,
	}

	// Parse version from source
	if idx := strings.LastIndex(source, ":"); idx != -1 {
		ref.Version = source[idx+1:]
		source = source[:idx]
	} else {
		ref.Version = "latest"
	}

	// Extract feature ID (last path component)
	parts := strings.Split(source, "/")
	if len(parts) > 0 {
		ref.ID = parts[len(parts)-1]
	}

	// Parse options
	if options != nil {
		switch v := options.(type) {
		case map[string]interface{}:
			ref.Options = v
		case bool:
			// Simple boolean enable
			ref.Options = make(map[string]interface{})
		}
	}

	return ref, nil
}

// ParseFeaturesFromConfig extracts feature references from config
func ParseFeaturesFromConfig(features map[string]interface{}) ([]*FeatureRef, error) {
	var refs []*FeatureRef

	for source, options := range features {
		ref, err := ParseFeatureRef(source, options)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feature %s: %w", source, err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// DownloadFeature downloads a feature tarball from OCI registry
// This is a simplified implementation that handles ghcr.io features
func DownloadFeature(ref *FeatureRef, destDir string) (*Feature, error) {
	// For now, we'll implement a basic download mechanism
	// In production, this should use proper OCI registry API

	fmt.Printf("Downloading feature: %s (version: %s)\n", ref.ID, ref.Version)

	// Create feature directory
	featureDir := filepath.Join(destDir, ref.ID)
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create feature directory: %w", err)
	}

	// For ghcr.io features, we need to use OCI API
	// This is a simplified version - production would need proper authentication
	if strings.HasPrefix(ref.Source, "ghcr.io/devcontainers/features/") {
		return downloadGHCRFeature(ref, featureDir)
	}

	return nil, fmt.Errorf("unsupported feature source: %s", ref.Source)
}

// downloadGHCRFeature downloads a feature from GitHub Container Registry
func downloadGHCRFeature(ref *FeatureRef, destDir string) (*Feature, error) {
	// Construct the OCI blob URL
	// Format: https://ghcr.io/v2/devcontainers/features/<id>/blobs/sha256:...

	// First, get the manifest to find the blob digest
	manifestURL := fmt.Sprintf("https://ghcr.io/v2/devcontainers/features/%s/manifests/%s",
		ref.ID, ref.Version)

	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Feature download may require authentication or different API
		// For now, return a stub feature with a warning
		fmt.Printf("Warning: Could not download feature %s (status: %d). Feature will need manual installation.\n",
			ref.ID, resp.StatusCode)

		return &Feature{
			ID:      ref.ID,
			Version: ref.Version,
			Name:    ref.ID,
			Options: ref.Options,
		}, nil
	}

	// Parse manifest
	var manifest struct {
		Layers []struct {
			Digest    string `json:"digest"`
			MediaType string `json:"mediaType"`
		} `json:"layers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Download the first layer (feature tarball)
	if len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("no layers found in manifest")
	}

	blobURL := fmt.Sprintf("https://ghcr.io/v2/devcontainers/features/%s/blobs/%s",
		ref.ID, manifest.Layers[0].Digest)

	return downloadAndExtractTarball(blobURL, destDir, ref)
}

// downloadAndExtractTarball downloads and extracts a feature tarball
func downloadAndExtractTarball(url string, destDir string, ref *FeatureRef) (*Feature, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decompress gzip
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Extract tar
	tarReader := tar.NewReader(gzReader)

	feature := &Feature{
		ID:      ref.ID,
		Version: ref.Version,
		Options: ref.Options,
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil, err
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return nil, err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return nil, err
			}
			file.Close()

			// Make scripts executable
			if strings.HasSuffix(header.Name, ".sh") {
				_ = os.Chmod(targetPath, 0755)
			}

			// Read install.sh content
			if header.Name == "install.sh" || strings.HasSuffix(header.Name, "/install.sh") {
				content, _ := os.ReadFile(targetPath)
				feature.InstallSh = string(content)
			}

			// Parse devcontainer-feature.json
			if header.Name == "devcontainer-feature.json" ||
				strings.HasSuffix(header.Name, "/devcontainer-feature.json") {
				content, _ := os.ReadFile(targetPath)
				_ = json.Unmarshal(content, feature)
			}
		}
	}

	return feature, nil
}

// GenerateFeatureEnv generates environment variables for feature installation
func GenerateFeatureEnv(feature *Feature) []string {
	var env []string

	for key, value := range feature.Options {
		envKey := strings.ToUpper(key)
		envValue := fmt.Sprintf("%v", value)
		env = append(env, fmt.Sprintf("%s=%s", envKey, envValue))
	}

	return env
}

// FeatureInstaller handles feature installation in containers
type FeatureInstaller struct {
	Features   []*Feature
	WorkingDir string
}

// NewFeatureInstaller creates a new feature installer
func NewFeatureInstaller(workingDir string) *FeatureInstaller {
	return &FeatureInstaller{
		WorkingDir: workingDir,
	}
}

// AddFeature adds a feature to be installed
func (fi *FeatureInstaller) AddFeature(feature *Feature) {
	fi.Features = append(fi.Features, feature)
}

// GenerateDockerfileSnippet generates Dockerfile commands to install features
func (fi *FeatureInstaller) GenerateDockerfileSnippet() string {
	var sb strings.Builder

	sb.WriteString("# DevContainer Features Installation\n")

	for _, feature := range fi.Features {
		if feature.InstallSh == "" {
			sb.WriteString(fmt.Sprintf("# Feature %s: No install.sh found\n", feature.ID))
			continue
		}

		sb.WriteString(fmt.Sprintf("\n# Feature: %s (v%s)\n", feature.ID, feature.Version))

		// Set environment variables
		env := GenerateFeatureEnv(feature)
		for _, e := range env {
			sb.WriteString(fmt.Sprintf("ENV %s\n", e))
		}

		// Copy and run install script
		sb.WriteString(fmt.Sprintf("COPY --chown=root:root %s/install.sh /tmp/%s-install.sh\n",
			feature.ID, feature.ID))
		sb.WriteString(fmt.Sprintf("RUN chmod +x /tmp/%s-install.sh && /tmp/%s-install.sh && rm /tmp/%s-install.sh\n",
			feature.ID, feature.ID, feature.ID))
	}

	return sb.String()
}

// GenerateInstallScript generates a shell script to install all features
func (fi *FeatureInstaller) GenerateInstallScript() string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("set -e\n\n")
	sb.WriteString("# DevContainer Features Installation Script\n")
	sb.WriteString("# Generated by Container-Make\n\n")

	for _, feature := range fi.Features {
		if feature.InstallSh == "" {
			sb.WriteString(fmt.Sprintf("echo 'Skipping %s: No install.sh found'\n", feature.ID))
			continue
		}

		sb.WriteString(fmt.Sprintf("\necho '=== Installing feature: %s (v%s) ==='\n",
			feature.ID, feature.Version))

		// Export environment variables
		env := GenerateFeatureEnv(feature)
		for _, e := range env {
			sb.WriteString(fmt.Sprintf("export %s\n", e))
		}

		// Run install script
		sb.WriteString(fmt.Sprintf("cd '%s/%s' && ./install.sh\n", fi.WorkingDir, feature.ID))
		sb.WriteString(fmt.Sprintf("echo '=== Completed: %s ==='\n", feature.ID))
	}

	sb.WriteString("\necho 'All features installed successfully!'\n")

	return sb.String()
}

// SortByDependencies sorts features by their dependencies (if available)
// Currently a stub - full implementation would parse devcontainer-feature.json for dependencies
func (fi *FeatureInstaller) SortByDependencies() {
	// For now, just sort alphabetically for consistent ordering
	// Full implementation would topologically sort based on "dependsOn" field
	for i := 0; i < len(fi.Features)-1; i++ {
		for j := i + 1; j < len(fi.Features); j++ {
			if fi.Features[i].ID > fi.Features[j].ID {
				fi.Features[i], fi.Features[j] = fi.Features[j], fi.Features[i]
			}
		}
	}
}
