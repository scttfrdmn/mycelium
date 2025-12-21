package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/truffle/pkg/aws"
	"github.com/yourusername/truffle/pkg/output"
)

var (
	spotMaxPrice     float64
	spotShowSavings  bool
	spotSortByPrice  bool
	spotOnlyActive   bool
	spotLookbackHours int
)

var spotCmd = &cobra.Command{
	Use:   "spot [instance-type-pattern]",
	Short: "Search for Spot instance pricing and availability",
	Long: `Search for EC2 Spot instances with pricing information.

This command shows current Spot prices across regions and availability zones,
helping you find the best Spot instance opportunities and maximize savings.

Spot instances are identical to On-Demand instances but can be interrupted by AWS.
They typically cost 50-90% less than On-Demand pricing.

Examples:
  # Get current Spot prices for m7i.large
  truffle spot m7i.large

  # Find Spot instances under $0.10/hour
  truffle spot "m8g.*" --max-price 0.10

  # Show savings vs On-Demand
  truffle spot m7i.xlarge --show-savings

  # Sort by price (cheapest first)
  truffle spot "c7i.*" --sort-by-price

  # Find Spot in specific regions
  truffle spot r7i.2xlarge --regions us-east-1,us-west-2`,
	Args: cobra.ExactArgs(1),
	RunE: runSpot,
}

func init() {
	rootCmd.AddCommand(spotCmd)

	spotCmd.Flags().Float64Var(&spotMaxPrice, "max-price", 0, "Maximum Spot price per hour (USD)")
	spotCmd.Flags().BoolVar(&spotShowSavings, "show-savings", false, "Show savings vs On-Demand pricing")
	spotCmd.Flags().BoolVar(&spotSortByPrice, "sort-by-price", false, "Sort by price (cheapest first)")
	spotCmd.Flags().BoolVar(&spotOnlyActive, "active-only", false, "Only show AZs with active Spot capacity")
	spotCmd.Flags().IntVar(&spotLookbackHours, "lookback-hours", 1, "Hours to look back for price history (1-720)")
	spotCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")
}

func runSpot(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Convert wildcard pattern to regex
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "💰 Searching for Spot instances matching: %s\n", pattern)
		if spotMaxPrice > 0 {
			fmt.Fprintf(os.Stderr, "💵 Max price filter: $%.4f/hour\n", spotMaxPrice)
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

	if verbose {
		fmt.Fprintf(os.Stderr, "🔎 Searching Spot prices across %d regions...\n", len(searchRegions))
	}

	// First find instance types (need to match pattern)
	results, err := awsClient.SearchInstanceTypes(ctx, searchRegions, matcher, aws.FilterOptions{
		IncludeAZs:     true, // Always get AZs for Spot
		Architecture:   architecture,
		MinVCPUs:       minVCPUs,
		MinMemory:      minMemory,
		InstanceFamily: instanceFamily,
		Verbose:        verbose,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No matching instance types found.")
		return nil
	}

	// Get Spot pricing for found instances
	if verbose {
		fmt.Fprintln(os.Stderr, "💰 Fetching Spot pricing data...")
	}

	spotResults, err := awsClient.GetSpotPricing(ctx, results, aws.SpotOptions{
		MaxPrice:      spotMaxPrice,
		ShowSavings:   spotShowSavings,
		LookbackHours: spotLookbackHours,
		OnlyActive:    spotOnlyActive,
		Verbose:       verbose,
	})
	if err != nil {
		return fmt.Errorf("failed to get Spot pricing: %w", err)
	}

	if len(spotResults) == 0 {
		fmt.Println("No Spot pricing data available for matching instances.")
		return nil
	}

	// Sort results
	if spotSortByPrice {
		sort.Slice(spotResults, func(i, j int) bool {
			return spotResults[i].SpotPrice < spotResults[j].SpotPrice
		})
	} else {
		// Default: sort by instance type, then region, then AZ
		sort.Slice(spotResults, func(i, j int) bool {
			if spotResults[i].InstanceType != spotResults[j].InstanceType {
				return spotResults[i].InstanceType < spotResults[j].InstanceType
			}
			if spotResults[i].Region != spotResults[j].Region {
				return spotResults[i].Region < spotResults[j].Region
			}
			return spotResults[i].AvailabilityZone < spotResults[j].AvailabilityZone
		})
	}

	// Print summary
	printSpotSummary(spotResults)

	// Output results
	printer := output.NewPrinter(!noColor)
	switch outputFormat {
	case "json":
		return printer.PrintSpotJSON(spotResults)
	case "yaml":
		return printer.PrintSpotYAML(spotResults)
	case "csv":
		return printer.PrintSpotCSV(spotResults)
	case "table":
		return printer.PrintSpotTable(spotResults, spotShowSavings)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func printSpotSummary(results []aws.SpotPriceResult) {
	if len(results) == 0 {
		return
	}

	// Calculate stats
	instanceTypes := make(map[string]bool)
	regions := make(map[string]bool)
	azs := make(map[string]bool)
	
	var totalPrice, minPrice, maxPrice float64
	minPrice = 999999.0
	maxPrice = 0.0
	totalSavings := 0.0
	savingsCount := 0

	for _, r := range results {
		instanceTypes[r.InstanceType] = true
		regions[r.Region] = true
		azs[r.AvailabilityZone] = true
		
		totalPrice += r.SpotPrice
		if r.SpotPrice < minPrice {
			minPrice = r.SpotPrice
		}
		if r.SpotPrice > maxPrice {
			maxPrice = r.SpotPrice
		}
		
		if r.SavingsPercent > 0 {
			totalSavings += r.SavingsPercent
			savingsCount++
		}
	}

	avgPrice := totalPrice / float64(len(results))
	avgSavings := 0.0
	if savingsCount > 0 {
		avgSavings = totalSavings / float64(savingsCount)
	}

	fmt.Printf("\n💰 Spot Instance Summary:\n")
	fmt.Printf("   Instance Types: %d\n", len(instanceTypes))
	fmt.Printf("   Regions: %d\n", len(regions))
	fmt.Printf("   Availability Zones: %d\n", len(azs))
	fmt.Printf("   Price Range: $%.4f - $%.4f per hour\n", minPrice, maxPrice)
	fmt.Printf("   Average Price: $%.4f per hour\n", avgPrice)
	if avgSavings > 0 {
		fmt.Printf("   Average Savings: %.1f%% vs On-Demand\n", avgSavings)
	}
	fmt.Println()
}
