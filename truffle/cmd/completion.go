package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// completeInstanceType provides completion for EC2 instance types
func completeInstanceType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Common instance types grouped by category
	instanceTypes := []string{
		// General Purpose (latest generation)
		"t3.micro\tBurstable, 2 vCPU, 1 GB RAM",
		"t3.small\tBurstable, 2 vCPU, 2 GB RAM",
		"t3.medium\tBurstable, 2 vCPU, 4 GB RAM",
		"t3.large\tBurstable, 2 vCPU, 8 GB RAM",
		"m7i.large\tGeneral purpose, 2 vCPU, 8 GB RAM",
		"m7i.xlarge\tGeneral purpose, 4 vCPU, 16 GB RAM",
		"m7i.2xlarge\tGeneral purpose, 8 vCPU, 32 GB RAM",
		"m7i.4xlarge\tGeneral purpose, 16 vCPU, 64 GB RAM",
		"m7a.large\tGeneral purpose AMD, 2 vCPU, 8 GB RAM",
		"m7a.xlarge\tGeneral purpose AMD, 4 vCPU, 16 GB RAM",
		"m8g.medium\tGraviton4, 1 vCPU, 4 GB RAM",
		"m8g.large\tGraviton4, 2 vCPU, 8 GB RAM",
		"m8g.xlarge\tGraviton4, 4 vCPU, 16 GB RAM",

		// Compute Optimized
		"c7i.large\tCompute optimized, 2 vCPU, 4 GB RAM",
		"c7i.xlarge\tCompute optimized, 4 vCPU, 8 GB RAM",
		"c7i.2xlarge\tCompute optimized, 8 vCPU, 16 GB RAM",
		"c7i.4xlarge\tCompute optimized, 16 vCPU, 32 GB RAM",
		"c7a.large\tCompute AMD, 2 vCPU, 4 GB RAM",
		"c7a.xlarge\tCompute AMD, 4 vCPU, 8 GB RAM",
		"c8g.medium\tCompute Graviton4, 1 vCPU, 2 GB RAM",
		"c8g.large\tCompute Graviton4, 2 vCPU, 4 GB RAM",
		"c8g.xlarge\tCompute Graviton4, 4 vCPU, 8 GB RAM",

		// Memory Optimized
		"r7i.large\tMemory optimized, 2 vCPU, 16 GB RAM",
		"r7i.xlarge\tMemory optimized, 4 vCPU, 32 GB RAM",
		"r7i.2xlarge\tMemory optimized, 8 vCPU, 64 GB RAM",
		"r7i.4xlarge\tMemory optimized, 16 vCPU, 128 GB RAM",
		"r7a.large\tMemory AMD, 2 vCPU, 16 GB RAM",
		"r7a.xlarge\tMemory AMD, 4 vCPU, 32 GB RAM",
		"r8g.medium\tMemory Graviton4, 1 vCPU, 8 GB RAM",
		"r8g.large\tMemory Graviton4, 2 vCPU, 16 GB RAM",
		"r8g.xlarge\tMemory Graviton4, 4 vCPU, 32 GB RAM",

		// GPU Instances
		"g5.xlarge\tGPU (1x A10G), 4 vCPU, 16 GB RAM",
		"g5.2xlarge\tGPU (1x A10G), 8 vCPU, 32 GB RAM",
		"g5.4xlarge\tGPU (1x A10G), 16 vCPU, 64 GB RAM",
		"g5.8xlarge\tGPU (1x A10G), 32 vCPU, 128 GB RAM",
		"g6.xlarge\tGPU (1x L4), 4 vCPU, 16 GB RAM",
		"g6.2xlarge\tGPU (1x L4), 8 vCPU, 32 GB RAM",
		"p3.2xlarge\tGPU (1x V100), 8 vCPU, 61 GB RAM",
		"p3.8xlarge\tGPU (4x V100), 32 vCPU, 244 GB RAM",
		"p4d.24xlarge\tGPU (8x A100), 96 vCPU, 1152 GB RAM",
		"p5.48xlarge\tGPU (8x H100), 192 vCPU, 2048 GB RAM",

		// Storage Optimized
		"i4i.large\tStorage optimized, 2 vCPU, 16 GB RAM",
		"i4i.xlarge\tStorage optimized, 4 vCPU, 32 GB RAM",
		"i4i.2xlarge\tStorage optimized, 8 vCPU, 64 GB RAM",

		// High Memory
		"x2idn.16xlarge\tHigh memory, 64 vCPU, 1024 GB RAM",
		"x2idn.24xlarge\tHigh memory, 96 vCPU, 1536 GB RAM",
		"x2idn.32xlarge\tHigh memory, 128 vCPU, 2048 GB RAM",

		// Wildcard patterns
		"m7i.*\tAll m7i instance types",
		"c7i.*\tAll c7i instance types",
		"r7i.*\tAll r7i instance types",
		"g5.*\tAll g5 GPU types",
		"*.large\tAll large instance types",
		"*.xlarge\tAll xlarge instance types",
	}

	var filtered []string
	for _, instanceType := range instanceTypes {
		if toComplete == "" || strings.HasPrefix(instanceType, toComplete) {
			filtered = append(filtered, instanceType)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeRegion provides completion for AWS regions
func completeRegion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// All AWS regions with descriptions
	regions := []string{
		"us-east-1\tUS East (N. Virginia)",
		"us-east-2\tUS East (Ohio)",
		"us-west-1\tUS West (N. California)",
		"us-west-2\tUS West (Oregon)",
		"af-south-1\tAfrica (Cape Town)",
		"ap-east-1\tAsia Pacific (Hong Kong)",
		"ap-south-1\tAsia Pacific (Mumbai)",
		"ap-south-2\tAsia Pacific (Hyderabad)",
		"ap-northeast-1\tAsia Pacific (Tokyo)",
		"ap-northeast-2\tAsia Pacific (Seoul)",
		"ap-northeast-3\tAsia Pacific (Osaka)",
		"ap-southeast-1\tAsia Pacific (Singapore)",
		"ap-southeast-2\tAsia Pacific (Sydney)",
		"ap-southeast-3\tAsia Pacific (Jakarta)",
		"ap-southeast-4\tAsia Pacific (Melbourne)",
		"ca-central-1\tCanada (Central)",
		"ca-west-1\tCanada (Calgary)",
		"eu-central-1\tEurope (Frankfurt)",
		"eu-central-2\tEurope (Zurich)",
		"eu-west-1\tEurope (Ireland)",
		"eu-west-2\tEurope (London)",
		"eu-west-3\tEurope (Paris)",
		"eu-south-1\tEurope (Milan)",
		"eu-south-2\tEurope (Spain)",
		"eu-north-1\tEurope (Stockholm)",
		"il-central-1\tIsrael (Tel Aviv)",
		"me-south-1\tMiddle East (Bahrain)",
		"me-central-1\tMiddle East (UAE)",
		"sa-east-1\tSouth America (São Paulo)",
	}

	var filtered []string
	for _, region := range regions {
		if toComplete == "" || strings.HasPrefix(region, toComplete) {
			filtered = append(filtered, region)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeArchitecture provides completion for CPU architectures
func completeArchitecture(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	architectures := []string{
		"x86_64\tIntel/AMD 64-bit",
		"arm64\tAWS Graviton (ARM 64-bit)",
		"i386\t32-bit x86",
	}

	var filtered []string
	for _, arch := range architectures {
		if toComplete == "" || strings.HasPrefix(arch, toComplete) {
			filtered = append(filtered, arch)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// completeInstanceFamily provides completion for instance families
func completeInstanceFamily(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	families := []string{
		"m7i\tGeneral Purpose (7th gen Intel)",
		"m7a\tGeneral Purpose (7th gen AMD)",
		"m8g\tGeneral Purpose (Graviton4)",
		"c7i\tCompute Optimized (7th gen Intel)",
		"c7a\tCompute Optimized (7th gen AMD)",
		"c8g\tCompute Optimized (Graviton4)",
		"r7i\tMemory Optimized (7th gen Intel)",
		"r7a\tMemory Optimized (7th gen AMD)",
		"r8g\tMemory Optimized (Graviton4)",
		"g5\tGPU (NVIDIA A10G)",
		"g6\tGPU (NVIDIA L4)",
		"p3\tGPU (NVIDIA V100)",
		"p4d\tGPU (NVIDIA A100)",
		"p5\tGPU (NVIDIA H100)",
		"i4i\tStorage Optimized (NVMe SSD)",
		"x2idn\tHigh Memory (up to 2TB RAM)",
		"t3\tBurstable Performance",
	}

	var filtered []string
	for _, family := range families {
		if toComplete == "" || strings.HasPrefix(family, toComplete) {
			filtered = append(filtered, family)
		}
	}

	return filtered, cobra.ShellCompDirectiveNoFileComp
}
