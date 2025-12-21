package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/truffle/pkg/aws"
	"github.com/yourusername/truffle/pkg/output"
	"github.com/yourusername/truffle/pkg/progress"
)

var (
	skipAZs        bool
	architecture   string
	minVCPUs       int
	minMemory      float64
	instanceFamily string
	timeout        time.Duration
)

var searchCmd = &cobra.Command{
	Use:   "search [instance-type-pattern]",
	Short: "Search for instance types across AWS regions",
	Long: `Search for EC2 instance types and discover their availability across regions and AZs.

Supports wildcard patterns using * and ? for flexible searching.

Examples:
  truffle search m7i.large
  truffle search "m8g.*"
  truffle search "*.large"
  truffle search m7i.large --skip-azs  # Faster, region-level only
  truffle search "c8g.*" --architecture arm64
  truffle search "*.xlarge" --min-vcpu 4 --min-memory 16`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVar(&skipAZs, "skip-azs", false, "Skip availability zone lookup (faster but less detailed)")
	searchCmd.Flags().StringVar(&architecture, "architecture", "", "Filter by architecture (x86_64, arm64, i386)")
	searchCmd.Flags().IntVar(&minVCPUs, "min-vcpu", 0, "Minimum number of vCPUs")
	searchCmd.Flags().Float64Var(&minMemory, "min-memory", 0, "Minimum memory in GiB")
	searchCmd.Flags().StringVar(&instanceFamily, "family", "", "Filter by instance family (e.g., m5, c5)")
	searchCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")
}

func runSearch(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Convert wildcard pattern to regex
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "🔍 Searching for instance types matching: %s\n", pattern)
		if len(regions) > 0 {
			fmt.Fprintf(os.Stderr, "📍 Filtering regions: %s\n", strings.Join(regions, ", "))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Get regions to search
	searchRegions := regions
	if len(searchRegions) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, "🌍 Fetching all AWS regions...")
		}
		searchRegions, err = awsClient.GetAllRegions(ctx)
		if err != nil {
			return fmt.Errorf("failed to get regions: %w", err)
		}
	}

	// Show spinner for non-verbose mode
	var spinner *progress.Spinner
	if !verbose && outputFormat == "table" {
		msg := fmt.Sprintf("Searching %d region(s)...", len(searchRegions))
		spinner = progress.NewSpinner(os.Stderr, msg)
		spinner.Start()
	} else if verbose {
		fmt.Fprintf(os.Stderr, "🔎 Searching across %d regions...\n", len(searchRegions))
	}

	// Search for instance types
	results, err := awsClient.SearchInstanceTypes(ctx, searchRegions, matcher, aws.FilterOptions{
		IncludeAZs:     !skipAZs, // AZs included by default
		Architecture:   architecture,
		MinVCPUs:       minVCPUs,
		MinMemory:      minMemory,
		InstanceFamily: instanceFamily,
		Verbose:        verbose,
	})

	if spinner != nil {
		spinner.Stop()
	}

	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Sort results for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].InstanceType != results[j].InstanceType {
			return results[i].InstanceType < results[j].InstanceType
		}
		return results[i].Region < results[j].Region
	})

	if len(results) == 0 {
		fmt.Println("No matching instance types found.")
		return nil
	}

	// Output results
	printer := output.NewPrinter(!noColor)
	switch outputFormat {
	case "json":
		return printer.PrintJSON(results)
	case "yaml":
		return printer.PrintYAML(results)
	case "csv":
		return printer.PrintCSV(results)
	case "table":
		return printer.PrintTable(results, !skipAZs) // Show AZs by default
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func wildcardToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)
	// Replace wildcards
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")
	// Anchor the pattern
	return "^" + pattern + "$"
}
