package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/truffle/pkg/aws"
	"github.com/yourusername/truffle/pkg/output"
)

var (
	crInstanceTypes []string
	crOnlyAvailable bool
	crOnlyActive    bool
	crMinCapacity   int
	crGPUOnly       bool
	crShowBlocks    bool
	crShowODCR      bool
)

var capacityCmd = &cobra.Command{
	Use:   "capacity",
	Short: "Check ML capacity reservations (Capacity Blocks & ODCRs)",
	Long: `Check ML capacity reservations across regions and availability zones.

AWS offers TWO types of capacity reservations for ML workloads:

1. CAPACITY BLOCKS FOR ML (--blocks)
   - Designed for HIGH-PERFORMANCE ML TRAINING (P5, P4d, Trn1)
   - Reserved for 1-14 days, book up to 8 weeks in advance
   - Co-located in EC2 UltraClusters (low-latency networking)
   - Perfect for: Large LLM training, distributed training runs
   - Pricing: Reservation fee + OS fee

2. ON-DEMAND CAPACITY RESERVATIONS / ODCR (--odcr, default)
   - For CONTINUOUS or UNPREDICTABLE workloads
   - No fixed duration, create/cancel anytime
   - Perfect for: ML inference, development, high-availability services
   - Pricing: Pay only when instances are running

Use Cases:
  Capacity Blocks: Training GPT-4 scale models, multi-node training
  ODCRs: Production inference APIs, ML development environments

Examples:
  # Check ODCRs (default - for inference/continuous workloads)
  truffle capacity

  # Check Capacity Blocks for ML (for training workloads)
  truffle capacity --blocks

  # GPU-only ODCRs
  truffle capacity --gpu-only --odcr

  # Specific instance types
  truffle capacity --instance-types p5.48xlarge,g6.xlarge

  # Only show reservations with available capacity
  truffle capacity --available-only

  # Find reservations with at least 10 instances available
  truffle capacity --min-capacity 10`,
	RunE: runCapacity,
}

func init() {
	rootCmd.AddCommand(capacityCmd)

	capacityCmd.Flags().StringSliceVar(&crInstanceTypes, "instance-types", []string{}, "Filter by instance types (comma-separated)")
	capacityCmd.Flags().BoolVar(&crOnlyAvailable, "available-only", false, "Only show reservations with available capacity")
	capacityCmd.Flags().BoolVar(&crOnlyActive, "active-only", true, "Only show active reservations (default: true)")
	capacityCmd.Flags().IntVar(&crMinCapacity, "min-capacity", 0, "Minimum available capacity")
	capacityCmd.Flags().BoolVar(&crGPUOnly, "gpu-only", false, "Only show GPU/ML instance reservations")
	capacityCmd.Flags().BoolVar(&crShowBlocks, "blocks", false, "Show Capacity Blocks for ML (training workloads)")
	capacityCmd.Flags().BoolVar(&crShowODCR, "odcr", true, "Show On-Demand Capacity Reservations (default)")
	capacityCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")
}

func runCapacity(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Fprintln(os.Stderr, "🔍 Searching for On-Demand Capacity Reservations...")
		if crGPUOnly {
			fmt.Fprintln(os.Stderr, "🎮 GPU/ML instances only")
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
		fmt.Fprintf(os.Stderr, "🔎 Searching across %d regions...\n", len(searchRegions))
	}

	// Get capacity reservations
	results, err := awsClient.GetCapacityReservations(ctx, searchRegions, aws.CapacityReservationOptions{
		InstanceTypes: crInstanceTypes,
		OnlyAvailable: crOnlyAvailable,
		OnlyActive:    crOnlyActive,
		MinCapacity:   int32(crMinCapacity),
		Verbose:       verbose,
	})
	if err != nil {
		return fmt.Errorf("failed to get capacity reservations: %w", err)
	}

	// Filter GPU instances if requested
	if crGPUOnly {
		results = filterGPUInstances(results)
	}

	if len(results) == 0 {
		fmt.Println("No capacity reservations found matching criteria.")
		return nil
	}

	// Sort by available capacity (most available first), then by instance type
	sort.Slice(results, func(i, j int) bool {
		if results[i].AvailableCapacity != results[j].AvailableCapacity {
			return results[i].AvailableCapacity > results[j].AvailableCapacity
		}
		if results[i].InstanceType != results[j].InstanceType {
			return results[i].InstanceType < results[j].InstanceType
		}
		return results[i].Region < results[j].Region
	})

	// Print summary
	printCapacitySummary(results)

	// Output results
	printer := output.NewPrinter(!noColor)
	switch outputFormat {
	case "json":
		return printer.PrintCapacityJSON(results)
	case "yaml":
		return printer.PrintCapacityYAML(results)
	case "csv":
		return printer.PrintCapacityCSV(results)
	case "table":
		return printer.PrintCapacityTable(results)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func filterGPUInstances(results []aws.CapacityReservationResult) []aws.CapacityReservationResult {
	// GPU/ML instance families
	gpuFamilies := map[string]bool{
		"p5":   true, // NVIDIA H100
		"p4":   true, // NVIDIA A100
		"p3":   true, // NVIDIA V100
		"g6":   true, // NVIDIA L4/L40S
		"g5":   true, // NVIDIA A10G
		"g4":   true, // NVIDIA T4
		"inf2": true, // AWS Inferentia2
		"inf1": true, // AWS Inferentia
		"trn1": true, // AWS Trainium
		"vt1":  true, // Video transcoding
	}

	filtered := make([]aws.CapacityReservationResult, 0)
	for _, r := range results {
		family := extractFamily(r.InstanceType)
		if gpuFamilies[family] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func extractFamily(instanceType string) string {
	// Extract family from instance type (e.g., "p5" from "p5.48xlarge")
	parts := strings.Split(instanceType, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return instanceType
}

func printCapacitySummary(results []aws.CapacityReservationResult) {
	instanceTypes := make(map[string]bool)
	regions := make(map[string]bool)
	azs := make(map[string]bool)
	
	var totalCapacity, availableCapacity, usedCapacity int32
	activeCount := 0

	for _, r := range results {
		instanceTypes[r.InstanceType] = true
		regions[r.Region] = true
		azs[r.AvailabilityZone] = true
		
		totalCapacity += r.TotalCapacity
		availableCapacity += r.AvailableCapacity
		usedCapacity += r.UsedCapacity
		
		if r.State == "active" {
			activeCount++
		}
	}

	utilizationPercent := 0.0
	if totalCapacity > 0 {
		utilizationPercent = float64(usedCapacity) / float64(totalCapacity) * 100
	}

	fmt.Printf("\n📊 Capacity Reservation Summary:\n")
	fmt.Printf("   Total Reservations: %d\n", len(results))
	fmt.Printf("   Active Reservations: %d\n", activeCount)
	fmt.Printf("   Instance Types: %d\n", len(instanceTypes))
	fmt.Printf("   Regions: %d\n", len(regions))
	fmt.Printf("   Availability Zones: %d\n", len(azs))
	fmt.Printf("   Total Capacity: %d instances\n", totalCapacity)
	fmt.Printf("   Available Capacity: %d instances (%.1f%% free)\n", availableCapacity, 100-utilizationPercent)
	fmt.Printf("   Used Capacity: %d instances (%.1f%% utilized)\n", usedCapacity, utilizationPercent)
	fmt.Println()
}
