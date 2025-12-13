package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display Container-Maker version, build date, and Git commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Container-Maker (cm)")
		fmt.Println()
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Also add --version flag to root command
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(`Container-Maker (cm) v{{.Version}}
`)
}
