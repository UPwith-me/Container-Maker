package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/export"
	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [output-file]",
	Short: "Package the environment for offline use",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}

		output := fmt.Sprintf("%s.cm", cfg.Name)
		if len(args) > 0 {
			output = args[0]
		}
		if cfg.Name == "" {
			output = "project.cm" // Fallback
		}

		// Need runtime to export image
		pr, err := runner.NewPersistentRunner(cfg, projectDir)
		if err != nil {
			return err
		}

		start := time.Now()
		if err := export.CreateBundle(context.Background(), pr.Runtime, cfg, projectDir, output); err != nil {
			os.Remove(output) // Clean up partial
			return err
		}

		fmt.Printf("✨ Export complete in %s!\n", time.Since(start).Round(time.Second))
		fmt.Printf("📦 Bundle size: %s\n", getFileSize(output))
		return nil
	},
}

var loadCmd = &cobra.Command{
	Use:   "load <bundle-file>",
	Short: "Load an environment from an offline bundle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundleFile := args[0]

		fmt.Printf("📦 Loading from %s...\n", bundleFile)

		f, err := os.Open(bundleFile)
		if err != nil {
			return err
		}
		defer f.Close()

		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()

		tr := tar.NewReader(gr)

		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			// 1. Handle manifest (optional verification)
			if header.Name == "cm-manifest.json" {
				fmt.Println("📄 Found Manifest")
				continue
			}

			// 2. Handle Image
			if header.Name == "image.tar" {
				fmt.Println("🖼️  Loading Docker Image (this may take a while)...")
				// Stream directly to docker load?
				// exec.Command("docker", "load", "-i", "-")?
				// or docker client Load method.
				// Since we are inside main, we don't have runtime instance yet easily.
				// We assume standard docker load for simplicity in "Industrial" context where docker is ubiquitous.
				// Using exec.Command for strict stream.

				loadCmd := exec.Command("docker", "load")
				loadCmd.Stdin = tr // Stream tar content directly!
				loadCmd.Stdout = os.Stdout
				// Problem: tr.Next() positions stream at start of file. Reading from tr reads file content.
				// BUT if we pass tr as Stdin, it might read past the file boundary?
				// tar.Reader treats Read as "read until end of current file". So it is safe!
				// Correct. io.Copy(dest, tr) copies only current entry.

				if err := loadCmd.Run(); err != nil {
					return fmt.Errorf("failed to load image: %w", err)
				}
				continue
			}

			// 3. Handle Code
			if filepath.Dir(header.Name) == "code" || filepath.HasPrefix(header.Name, "code/") {
				relPath := header.Name[5:]                // strip "code/"
				targetPath := filepath.Join(".", relPath) // Extract to current dir

				// Critical Security Fix: Prevent Zip Slip
				// Clean the path and check for ".." escalation
				cleanPath := filepath.Clean(targetPath)
				if strings.Contains(cleanPath, "..") || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "../") {
					return fmt.Errorf("security error: illegal file path in bundle: %s", header.Name)
				}

				targetPath = cleanPath

				if header.Typeflag == tar.TypeDir {
					_ = os.MkdirAll(targetPath, 0755)
					continue
				}

				dir := filepath.Dir(targetPath)
				_ = os.MkdirAll(dir, 0755)

				outFile, err := os.Create(targetPath)
				if err != nil {
					return err
				}
				// Copy securely
				// Limit size? No, header.Size is trusted from tar, but io.CopyN is safer to prevent endless stream attack
				if _, err := io.CopyN(outFile, tr, header.Size); err != nil {
					outFile.Close()
					return err
				}
				outFile.Close()
				_ = os.Chmod(targetPath, os.FileMode(header.Mode))
				continue
			}
		}

		fmt.Println("✅ Load complete.")
		return nil
	},
}

func getFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	sizeBytes := info.Size()
	return fmt.Sprintf("%.2f MB", float64(sizeBytes)/1024/1024)
}

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(loadCmd)
}
