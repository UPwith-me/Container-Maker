package export

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/config"
	"github.com/UPwith-me/Container-Maker/pkg/runtime"
)

// Manifest describes the bundle content
type Manifest struct {
	Version   string    `json:"version"` // Bundle format version
	CreatesAt time.Time `json:"created_at"`
	Project   string    `json:"project"`
	Image     string    `json:"image"`
	Files     []string  `json:"files"` // List of included source files
}

// CreateBundle creates a compressed bundle of the environment
func CreateBundle(ctx context.Context, rt runtime.ContainerRuntime, cfg *config.DevContainerConfig, projectDir string, outputFile string) error {
	// Create output file
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Gzip Writer
	gw := gzip.NewWriter(f)
	defer gw.Close()

	// Tar Writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	manifest := Manifest{
		Version:   "1.0",
		CreatesAt: time.Now(),
		Project:   cfg.Name,
		Image:     cfg.Image,
		Files:     []string{},
	}

	// 1. Export Image (Heavy Operation)
	fmt.Printf("📦 Exporting image: %s (this may take a while)...\n", cfg.Image)
	imageStream, err := rt.SaveImage(ctx, cfg.Image)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	defer imageStream.Close()

	// Add image to tar
	// We don't know the size beforehand easily unless we buffer, but streaming is better.
	// tar.Header size is required.
	// PROBLEM: io.Copy to tar requires known size in header!
	// Solution: Stream to temp file first to get size, then copy to tar.
	// Or use "docker save" output directly as a file in the tar.
	// Yes, temp file is safer for tar.

	tmpImage, err := os.CreateTemp("", "cm-image-*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(tmpImage.Name())
	defer tmpImage.Close()

	if _, err := io.Copy(tmpImage, imageStream); err != nil {
		return fmt.Errorf("failed to buffer image: %w", err)
	}

	info, err := tmpImage.Stat()
	if err != nil {
		return err
	}

	if err := addFileToTar(tw, tmpImage.Name(), "image.tar", info.Size()); err != nil {
		return err
	}

	// 2. Export Config
	// Add devcontainer.json
	// Assuming it's in projectDir/.devcontainer/devcontainer.json or similar?
	// Using cfg path logic would be better but let's assume standard locations or pass explicit path.
	// For now, we rely on scanning projectDir for config files.

	// 3. Export Source Code
	fmt.Printf("📂 Packagiug source code...\n")
	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}

		// Simple Ignore list
		if shouldIgnore(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		manifest.Files = append(manifest.Files, relPath)
		return addFileToTar(tw, path, filepath.Join("code", relPath), info.Size())
	})
	if err != nil {
		return fmt.Errorf("failed to package code: %w", err)
	}

	// 4. Write Manifest
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := addBytesToTar(tw, manifestBytes, "cm-manifest.json"); err != nil {
		return err
	}

	fmt.Printf("✅ Bundle created: %s\n", outputFile)
	return nil
}

func shouldIgnore(path string) bool {
	// Normalized path
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, ".git/") || path == ".git" {
		return true
	}
	if strings.Contains(path, "node_modules/") || strings.Contains(path, "vendor/") {
		return true
	}
	if strings.HasSuffix(path, ".cm") { // Don't include other bundles
		return true
	}
	return false
}

func addFileToTar(tw *tar.Writer, srcPath, tarName string, size int64) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	header := &tar.Header{
		Name:    tarName,
		Size:    size,
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

func addBytesToTar(tw *tar.Writer, data []byte, tarName string) error {
	header := &tar.Header{
		Name:    tarName,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
