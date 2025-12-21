package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat string
	noColor      bool
	regions      []string
	verbose      bool
)

var rootCmd = &cobra.Command{
	Use:   "truffle",
	Short: "🍄 Truffle - AWS EC2 Instance Type Region Finder",
	Long: `Truffle is a CLI tool to discover which AWS regions and availability zones
support specific EC2 instance types. Perfect for planning multi-region deployments!

Examples:
  # Find all regions with m7i.large instances
  truffle search m7i.large

  # Search with wildcards - Graviton4 instances
  truffle search "m8g.*"

  # Filter by specific regions
  truffle search m7i.large --regions us-east-1,eu-west-1

  # Output as JSON
  truffle search m7i.large --output json

  # List all instance families
  truffle list --family`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml, csv)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorized output")
	rootCmd.PersistentFlags().StringSliceVarP(&regions, "regions", "r", []string{}, "Filter by specific regions (comma-separated)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}
