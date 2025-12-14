package runner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OCIFeatureDownloader handles downloading DevContainer Features from OCI registries
type OCIFeatureDownloader struct {
	cacheDir string
	backend  string
}

// FeatureManifest represents the OCI manifest for a feature
type FeatureManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int    `json:"size"`
	} `json:"layers"`
}

// NewOCIFeatureDownloader creates a new OCI feature downloader
func NewOCIFeatureDownloader(backend string) *OCIFeatureDownloader {
	home, _ := os.UserHomeDir()
	return &OCIFeatureDownloader{
		cacheDir: filepath.Join(home, ".cm", "features"),
		backend:  backend,
	}
}

// DownloadFeature downloads a feature from an OCI registry
// featureRef format: ghcr.io/devcontainers/features/go:1
func (d *OCIFeatureDownloader) DownloadFeature(ctx context.Context, featureRef string) (string, error) {
	// Parse feature reference
	registry, namespace, name, tag := parseFeatureRef(featureRef)

	// Check cache first
	cacheKey := fmt.Sprintf("%s-%s-%s-%s", registry, strings.ReplaceAll(namespace, "/", "-"), name, tag)
	cachePath := filepath.Join(d.cacheDir, cacheKey)
	if _, err := os.Stat(filepath.Join(cachePath, "install.sh")); err == nil {
		return cachePath, nil // Already cached
	}

	fmt.Printf("📥 Downloading feature: %s\n", featureRef)

	// Try different download methods
	var err error

	// Method 1: Direct GitHub raw download (fastest for devcontainers/features)
	if registry == "ghcr.io" && strings.HasPrefix(namespace, "devcontainers/features") {
		err = d.downloadFromGitHub(ctx, name, tag, cachePath)
		if err == nil {
			return cachePath, nil
		}
	}

	// Method 2: OCI Registry API
	err = d.downloadFromOCI(ctx, registry, namespace, name, tag, cachePath)
	if err == nil {
		return cachePath, nil
	}

	// Method 3: Use oras CLI if available
	err = d.downloadWithOras(ctx, featureRef, cachePath)
	if err == nil {
		return cachePath, nil
	}

	return "", fmt.Errorf("failed to download feature %s: all methods failed", featureRef)
}

// parseFeatureRef parses a feature reference into components
func parseFeatureRef(ref string) (registry, namespace, name, tag string) {
	tag = "latest"

	// Extract tag
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		possibleTag := ref[idx+1:]
		if !strings.Contains(possibleTag, "/") {
			tag = possibleTag
			ref = ref[:idx]
		}
	}

	// Split into parts
	parts := strings.Split(ref, "/")
	if len(parts) >= 3 {
		registry = parts[0]
		name = parts[len(parts)-1]
		namespace = strings.Join(parts[1:len(parts)-1], "/")
	} else if len(parts) == 2 {
		registry = "ghcr.io"
		namespace = "devcontainers/features"
		name = parts[1]
	} else {
		registry = "ghcr.io"
		namespace = "devcontainers/features"
		name = parts[0]
	}

	return
}

// downloadFromGitHub downloads from GitHub raw content
func (d *OCIFeatureDownloader) downloadFromGitHub(ctx context.Context, name, tag string, destPath string) error {
	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	// Files to download
	files := []string{"install.sh", "devcontainer-feature.json"}

	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/devcontainers/features/main/src/%s", name)

	for _, file := range files {
		url := fmt.Sprintf("%s/%s", baseURL, file)
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		destFile := filepath.Join(destPath, file)
		f, err := os.Create(destFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, resp.Body)
		f.Close()
		if err != nil {
			return err
		}

		// Make install.sh executable
		if file == "install.sh" {
			os.Chmod(destFile, 0755)
		}
	}

	// Verify install.sh exists
	if _, err := os.Stat(filepath.Join(destPath, "install.sh")); err != nil {
		return fmt.Errorf("install.sh not found")
	}

	return nil
}

// downloadFromOCI downloads from OCI registry using HTTP API
func (d *OCIFeatureDownloader) downloadFromOCI(ctx context.Context, registry, namespace, name, tag string, destPath string) error {
	// Build manifest URL
	// For ghcr.io: https://ghcr.io/v2/devcontainers/features/go/manifests/1
	manifestURL := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", registry, namespace, name, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return err
	}

	// Accept OCI manifest media types
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Try with token for ghcr.io
		if registry == "ghcr.io" {
			return d.downloadFromOCIWithToken(ctx, registry, namespace, name, tag, destPath)
		}
		return fmt.Errorf("manifest fetch failed: %d", resp.StatusCode)
	}

	var manifest FeatureManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return err
	}

	// Download first layer (should be the feature tarball)
	if len(manifest.Layers) == 0 {
		return fmt.Errorf("no layers in manifest")
	}

	layer := manifest.Layers[0]
	return d.downloadAndExtractLayer(ctx, registry, namespace, name, layer.Digest, destPath)
}

// downloadFromOCIWithToken downloads with anonymous token for ghcr.io
func (d *OCIFeatureDownloader) downloadFromOCIWithToken(ctx context.Context, registry, namespace, name, tag string, destPath string) error {
	// Get anonymous token from ghcr.io
	tokenURL := fmt.Sprintf("https://%s/token?scope=repository:%s/%s:pull", registry, namespace, name)

	tokenResp, err := http.Get(tokenURL)
	if err != nil {
		return err
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return err
	}

	if tokenData.Token == "" {
		return fmt.Errorf("failed to get token")
	}

	// Fetch manifest with token
	manifestURL := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", registry, namespace, name, tag)
	req, _ := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	req.Header.Set("Authorization", "Bearer "+tokenData.Token)
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("manifest fetch with token failed: %d", resp.StatusCode)
	}

	var manifest FeatureManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return err
	}

	if len(manifest.Layers) == 0 {
		return fmt.Errorf("no layers")
	}

	// Download layer with token
	layer := manifest.Layers[0]
	blobURL := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", registry, namespace, name, layer.Digest)

	blobReq, _ := http.NewRequestWithContext(ctx, "GET", blobURL, nil)
	blobReq.Header.Set("Authorization", "Bearer "+tokenData.Token)

	blobResp, err := http.DefaultClient.Do(blobReq)
	if err != nil {
		return err
	}
	defer blobResp.Body.Close()

	if blobResp.StatusCode != 200 {
		return fmt.Errorf("blob fetch failed: %d", blobResp.StatusCode)
	}

	return d.extractTarGz(blobResp.Body, destPath)
}

// downloadAndExtractLayer downloads and extracts a layer blob
func (d *OCIFeatureDownloader) downloadAndExtractLayer(ctx context.Context, registry, namespace, name, digest string, destPath string) error {
	blobURL := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", registry, namespace, name, digest)

	resp, err := http.Get(blobURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("blob download failed: %d", resp.StatusCode)
	}

	return d.extractTarGz(resp.Body, destPath)
}

// extractTarGz extracts a tar.gz stream to destination
func (d *OCIFeatureDownloader) extractTarGz(reader io.Reader, destPath string) error {
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	gzr, err := gzip.NewReader(reader)
	if err != nil {
		// Try as plain tar
		return d.extractTar(reader, destPath)
	}
	defer gzr.Close()

	return d.extractTar(gzr, destPath)
}

// extractTar extracts a tar stream to destination
func (d *OCIFeatureDownloader) extractTar(reader io.Reader, destPath string) error {
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// downloadWithOras uses oras CLI to download (fallback)
func (d *OCIFeatureDownloader) downloadWithOras(ctx context.Context, featureRef, destPath string) error {
	// Check if oras is available
	if _, err := exec.LookPath("oras"); err != nil {
		return fmt.Errorf("oras not found")
	}

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "oras", "pull", featureRef, "-o", destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// InstallFeatureInContainer installs a downloaded feature into a container
func (d *OCIFeatureDownloader) InstallFeatureInContainer(ctx context.Context, containerID, featurePath string, options map[string]interface{}) error {
	installScript := filepath.Join(featurePath, "install.sh")
	if _, err := os.Stat(installScript); err != nil {
		return fmt.Errorf("install.sh not found in feature")
	}

	// Copy install.sh to container
	cmd := exec.CommandContext(ctx, d.backend, "cp", installScript, containerID+":/tmp/feature-install.sh")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Build environment variables from options
	envArgs := []string{"exec"}
	for k, v := range options {
		envArgs = append(envArgs, "-e", fmt.Sprintf("%s=%v", strings.ToUpper(k), v))
	}
	envArgs = append(envArgs, containerID, "sh", "-c", "chmod +x /tmp/feature-install.sh && /tmp/feature-install.sh")

	// Execute install script
	execCmd := exec.CommandContext(ctx, d.backend, envArgs...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// GetFeatureMetadata reads the devcontainer-feature.json
func (d *OCIFeatureDownloader) GetFeatureMetadata(featurePath string) (*FeatureMetadata, error) {
	metaPath := filepath.Join(featurePath, "devcontainer-feature.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta FeatureMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// ListCachedFeatures returns all cached features
func (d *OCIFeatureDownloader) ListCachedFeatures() ([]string, error) {
	if _, err := os.Stat(d.cacheDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(d.cacheDir)
	if err != nil {
		return nil, err
	}

	features := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			features = append(features, entry.Name())
		}
	}

	return features, nil
}

// ClearCache removes all cached features
func (d *OCIFeatureDownloader) ClearCache() error {
	return os.RemoveAll(d.cacheDir)
}
