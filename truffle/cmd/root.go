package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat string
	noColor      bool
	regions      []string
	verbose      bool

	// i18n and accessibility flags
	flagLang          string
	flagNoEmoji       bool
	flagAccessibility bool
)

var rootCmd = &cobra.Command{
	Use: "truffle",
	// Short and Long will be set after i18n initialization
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add i18n and accessibility flags
	rootCmd.PersistentFlags().StringVar(&flagLang, "lang", "", "Language for output (en, es, fr, de, ja)")
	rootCmd.PersistentFlags().BoolVar(&flagNoEmoji, "no-emoji", false, "Disable emoji in output")
	rootCmd.PersistentFlags().BoolVar(&flagAccessibility, "accessibility", false, "Enable accessibility mode (implies --no-emoji)")

	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml, csv)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colorized output")
	rootCmd.PersistentFlags().StringSliceVarP(&regions, "regions", "r", []string{}, "Filter by specific regions (comma-separated)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Initialize i18n before command execution
	cobra.OnInitialize(initI18n)

	// Enable shell completion for all supported shells
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.CompletionOptions.DisableDescriptions = false

	// Register completion for persistent flags
	rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml", "csv"}, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.RegisterFlagCompletionFunc("regions", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeRegion(cmd, args, toComplete)
	})
}

func initI18n() {
	// Initialize i18n with configuration from flags
	cfg := i18n.Config{
		Language:          flagLang,
		Verbose:           false,
		AccessibilityMode: flagAccessibility,
		NoEmoji:           flagNoEmoji,
	}

	if err := i18n.Init(cfg); err != nil {
		log.Printf("Warning: failed to initialize i18n: %v", err)
		// Continue with default English
	}

	// Set command descriptions after i18n is initialized
	updateCommandDescriptions()
}

func updateCommandDescriptions() {
	// Root command
	rootCmd.Short = i18n.T("truffle.root.short")
	rootCmd.Long = i18n.T("truffle.root.long")

	// Search command
	if cmd, _, err := rootCmd.Find([]string{"search"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("truffle.search.short")
		cmd.Long = i18n.T("truffle.search.long")
	}

	// Capacity command
	if cmd, _, err := rootCmd.Find([]string{"capacity"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("truffle.capacity.short")
		cmd.Long = i18n.T("truffle.capacity.long")
	}

	// Spot command
	if cmd, _, err := rootCmd.Find([]string{"spot"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("truffle.spot.short")
		cmd.Long = i18n.T("truffle.spot.long")
	}
}
