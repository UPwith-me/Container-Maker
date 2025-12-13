package main

import (
	"fmt"
	"os"

	"github.com/container-make/cm/pkg/template"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"market", "store"},
	Short:   "Browse and install community templates",
	Long: `Discover, search, and install DevContainer templates from the community.

Examples:
  cm marketplace search python    # Search for Python templates
  cm marketplace list             # List all templates
  cm marketplace install go       # Install the Go template
  cm marketplace info python      # Show template details`,
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search templates",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMarketplaceSearch,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMarketplaceSearch(cmd, []string{})
	},
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <template-id>",
	Short: "Install a template",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceInstall,
}

var marketplaceInfoCmd = &cobra.Command{
	Use:   "info <template-id>",
	Short: "Show template details",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceInfo,
}

func init() {
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceInfoCmd)
	rootCmd.AddCommand(marketplaceCmd)
}

func runMarketplaceSearch(cmd *cobra.Command, args []string) error {
	fmt.Println("🏪 Template Marketplace")
	fmt.Println()

	market := template.NewMarketplace()

	query := ""
	if len(args) > 0 {
		query = args[0]
		fmt.Printf("🔍 Searching for: %s\n\n", query)
	}

	templates, err := market.Search(query)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return nil
	}

	fmt.Println(market.FormatTemplatesTable(templates))
	fmt.Println()
	fmt.Println("💡 Use 'cm marketplace install <id>' to install a template")

	return nil
}

func runMarketplaceInstall(cmd *cobra.Command, args []string) error {
	templateID := args[0]

	fmt.Printf("📦 Installing template: %s\n", templateID)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	market := template.NewMarketplace()
	if err := market.Install(templateID, cwd); err != nil {
		return err
	}

	fmt.Println("✅ Template installed successfully!")
	fmt.Println()
	fmt.Println("📁 Created .devcontainer/devcontainer.json")
	fmt.Println("🚀 Run 'cm shell' to start your dev container")

	return nil
}

func runMarketplaceInfo(cmd *cobra.Command, args []string) error {
	templateID := args[0]

	market := template.NewMarketplace()
	tmpl, err := market.GetTemplate(templateID)
	if err != nil {
		return err
	}

	fmt.Println("📋 Template Details")
	fmt.Println()
	fmt.Printf("  ID:          %s\n", tmpl.ID)
	fmt.Printf("  Name:        %s\n", tmpl.Name)
	fmt.Printf("  Author:      %s\n", tmpl.Author)
	fmt.Printf("  Category:    %s\n", tmpl.Category)
	fmt.Printf("  Description: %s\n", tmpl.Description)
	fmt.Printf("  Stars:       ⭐ %d\n", tmpl.Stars)
	fmt.Printf("  Downloads:   📥 %d\n", tmpl.Downloads)
	fmt.Println()
	fmt.Printf("💡 Use 'cm marketplace install %s' to install\n", tmpl.ID)

	return nil
}
