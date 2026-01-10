package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/UPwith-me/Container-Maker/pkg/plugin"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := plugin.GetManager()
		// Re-scan to be sure
		if err := mgr.DiscoverPlugins(context.Background()); err != nil {
			return err
		}

		plugins := mgr.GetPlugins()
		if len(plugins) == 0 {
			fmt.Println("No plugins installed.")
			return nil
		}

		fmt.Println("NAME                 VERSION      DESCRIPTION")
		fmt.Println("────────────────────────────────────────────────────────")
		for _, p := range plugins {
			meta := p.Metadata()
			fmt.Printf("%-20s %-12s %s\n", meta.Name, meta.Version, meta.Description)
		}
		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <url>",
	Short: "Install a plugin from a URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		fmt.Printf("⬇️  Downloading plugin from %s...\n", url)

		// 1. Download
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("download failed: %s", resp.Status)
		}

		// 2. Save to ~/.cm/plugins/cm-<name>
		// We need to infer name from URL or header?
		// For simplicity, take base name
		baseName := filepath.Base(url)
		if baseName == "." || baseName == "/" {
			return fmt.Errorf("cannot infer filename from URL")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		pluginDir := filepath.Join(home, ".cm", "plugins")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return err
		}

		targetPath := filepath.Join(pluginDir, baseName)
		file, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(file, resp.Body); err != nil {
			return err
		}

		// 3. Chmod +x
		if runtime.GOOS != "windows" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return err
			}
		}

		fmt.Printf("✅  Plugin installed to %s\n", targetPath)
		fmt.Println("    Run 'cm plugin list' to verify.")
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	rootCmd.AddCommand(pluginCmd)
}
