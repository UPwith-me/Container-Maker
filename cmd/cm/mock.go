package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/mock"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Manage mock services",
	Long: `Run mock services for development and testing.

Create HTTP mock servers with dynamic responses, latency simulation,
and contract verification capabilities.

COMMANDS
  cm mock serve     Start a mock server
  cm mock verify    Verify a service against a contract`,
}

var (
	mockPort int
	// mockConfig  string // Unused
	mockLatency string
)

var mockServeCmd = &cobra.Command{
	Use:   "serve [config-file]",
	Short: "Start a mock server",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config := mock.MockConfig{
			Name: "default-mock",
			Port: mockPort,
			Mode: mock.ModeDynamic,
		}

		if len(args) > 0 {
			// Load from file
			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}
			if err := yaml.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}
			// Override port if flag is set and not default
			if cmd.Flags().Changed("port") {
				config.Port = mockPort
			}
		}

		// Parse latency flag
		if mockLatency != "" {
			d, err := time.ParseDuration(mockLatency)
			if err == nil {
				config.Latency.Fixed = d
			}
		}

		server := mock.NewHTTPServer(config)

		// Hndle shutdown
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			fmt.Println("\nStopping mock server...")
			_ = server.Stop()
		}()

		return server.Start()
	},
}

var (
	contractPath string
	serviceURL   string
)

var mockVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a service against a contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		if contractPath == "" || serviceURL == "" {
			return fmt.Errorf("both --contract and --url are required")
		}

		verifier := &mock.DefaultContractVerifier{}
		result, err := verifier.VerifyFile(contractPath, serviceURL)
		if err != nil {
			return err
		}

		fmt.Printf("Verifying contract '%s' against %s\n", result.Contract, serviceURL)
		fmt.Printf("Provider: %s, Consumer: %s\n\n", result.Provider, result.Consumer)

		for _, i := range result.Interactions {
			status := "✅ PASS"
			if !i.Passed {
				status = "❌ FAIL"
			}
			fmt.Printf("%s %s\n", status, i.Description)
			if !i.Passed {
				fmt.Printf("  Error: %s\n", i.Error)
			}
		}

		fmt.Println()
		if result.Passed {
			fmt.Printf("✨ ALL VERIFIED (Title: %v)\n", result.Duration)
		} else {
			fmt.Printf("💥 VERIFICATION FAILED\n")
			os.Exit(1)
		}

		return nil
	},
}

// Helper to create a quick mock endpoint
var mockQuickCmd = &cobra.Command{
	Use:   "quick <path> <body>",
	Short: "Quickly start a simple mock endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		body := args[1]

		config := mock.MockConfig{
			Name: "quick-mock",
			Port: mockPort,
			Endpoints: []mock.EndpointConfig{
				{
					Path:   path,
					Method: "GET",
					Body:   body,
				},
			},
		}

		server := mock.NewHTTPServer(config)
		return server.Start()
	},
}

func init() {
	mockServeCmd.Flags().IntVarP(&mockPort, "port", "p", 8080, "Port to listen on")
	mockServeCmd.Flags().StringVarP(&mockLatency, "latency", "l", "", "Simulate fixed latency (e.g. 100ms)")

	mockVerifyCmd.Flags().StringVarP(&contractPath, "contract", "c", "", "Path to contract YAML file")
	mockVerifyCmd.Flags().StringVarP(&serviceURL, "url", "u", "", "URL of service to verify")

	mockQuickCmd.Flags().IntVarP(&mockPort, "port", "p", 8080, "Port to listen on")

	mockCmd.AddCommand(mockServeCmd)
	mockCmd.AddCommand(mockVerifyCmd)
	mockCmd.AddCommand(mockQuickCmd)

	rootCmd.AddCommand(mockCmd)
}
