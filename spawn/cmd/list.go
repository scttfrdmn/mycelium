package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/spawn/pkg/aws"
)

var (
	listRegion         string
	listAZ             string
	listState          string
	listInstanceType   string
	listInstanceFamily string
	listTag            []string
	listJSON           bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List spawn-managed EC2 instances",
	Long: `List all EC2 instances managed by spawn across regions.

By default, shows all running and stopped instances in all regions.

Examples:
  # List all spawn instances
  spawn list

  # List instances in specific region
  spawn list --region us-east-1

  # List only running instances
  spawn list --state running

  # List stopped instances in us-west-2
  spawn list --region us-west-2 --state stopped

  # List by availability zone
  spawn list --az us-east-1a

  # List by instance type or family
  spawn list --instance-type t3.micro
  spawn list --instance-family m7i

  # List by tag
  spawn list --tag project=ml --tag team=research`,
	RunE:    runList,
	Aliases: []string{"ls"},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listRegion, "region", "", "Filter by AWS region (default: all regions)")
	listCmd.Flags().StringVar(&listAZ, "az", "", "Filter by availability zone")
	listCmd.Flags().StringVar(&listState, "state", "", "Filter by instance state (running, stopped, etc.)")
	listCmd.Flags().StringVar(&listInstanceType, "instance-type", "", "Filter by exact instance type (e.g., t3.micro)")
	listCmd.Flags().StringVar(&listInstanceFamily, "instance-family", "", "Filter by instance family (e.g., m7i, t3)")
	listCmd.Flags().StringArrayVar(&listTag, "tag", []string{}, "Filter by tag (key=value format, can be specified multiple times)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// List instances
	fmt.Fprintf(os.Stderr, "Searching for spawn-managed instances")
	if listRegion != "" {
		fmt.Fprintf(os.Stderr, " in %s", listRegion)
	} else {
		fmt.Fprintf(os.Stderr, " across all regions")
	}
	fmt.Fprintln(os.Stderr, "...")

	instances, err := client.ListInstances(ctx, listRegion, listState)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(instances) == 0 {
		fmt.Println("\nNo spawn-managed instances found.")
		return nil
	}

	// Apply additional filters
	instances = filterInstances(instances)

	if len(instances) == 0 {
		fmt.Println("\nNo instances match the specified filters.")
		return nil
	}

	// Sort by launch time (newest first)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].LaunchTime.After(instances[j].LaunchTime)
	})

	// Output format
	if listJSON {
		return outputJSON(instances)
	}

	return outputTable(instances)
}

func outputTable(instances []aws.InstanceInfo) error {
	fmt.Println() // Blank line after search message

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "INSTANCE ID\tNAME\tTYPE\tSTATE\tAZ\tAGE\tTTL\tPUBLIC IP\tSPOT")

	for _, inst := range instances {
		// Calculate age
		age := formatDuration(time.Since(inst.LaunchTime))

		// Format TTL
		ttl := inst.TTL
		if ttl == "" {
			ttl = "none"
		}

		// Format name
		name := inst.Name
		if name == "" {
			name = "-"
		}

		// Spot indicator
		spotIndicator := ""
		if inst.SpotInstance {
			spotIndicator = "✓"
		}

		// Color state
		state := inst.State
		switch state {
		case "running":
			state = "\033[32m" + state + "\033[0m" // Green
		case "stopped":
			state = "\033[33m" + state + "\033[0m" // Yellow
		case "stopping":
			state = "\033[33m" + state + "\033[0m" // Yellow
		case "pending":
			state = "\033[36m" + state + "\033[0m" // Cyan
		}

		// Public IP
		publicIP := inst.PublicIP
		if publicIP == "" {
			publicIP = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			inst.InstanceID,
			name,
			inst.InstanceType,
			state,
			inst.AvailabilityZone,
			age,
			ttl,
			publicIP,
			spotIndicator,
		)
	}

	return nil
}

func outputJSON(instances []aws.InstanceInfo) error {
	// Simple JSON output
	fmt.Println("[")
	for i, inst := range instances {
		comma := ","
		if i == len(instances)-1 {
			comma = ""
		}

		fmt.Printf(`  {
    "instance_id": "%s",
    "name": "%s",
    "instance_type": "%s",
    "state": "%s",
    "region": "%s",
    "availability_zone": "%s",
    "public_ip": "%s",
    "private_ip": "%s",
    "launch_time": "%s",
    "ttl": "%s",
    "idle_timeout": "%s",
    "key_name": "%s",
    "spot": %t
  }%s
`,
			inst.InstanceID,
			inst.Name,
			inst.InstanceType,
			inst.State,
			inst.Region,
			inst.AvailabilityZone,
			inst.PublicIP,
			inst.PrivateIP,
			inst.LaunchTime.Format(time.RFC3339),
			inst.TTL,
			inst.IdleTimeout,
			inst.KeyName,
			inst.SpotInstance,
			comma,
		)
	}
	fmt.Println("]")

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}

func filterInstances(instances []aws.InstanceInfo) []aws.InstanceInfo {
	var filtered []aws.InstanceInfo

	for _, inst := range instances {
		// Filter by availability zone
		if listAZ != "" && inst.AvailabilityZone != listAZ {
			continue
		}

		// Filter by instance type (exact match)
		if listInstanceType != "" && inst.InstanceType != listInstanceType {
			continue
		}

		// Filter by instance family (prefix match)
		if listInstanceFamily != "" {
			// Extract family from instance type (e.g., "m7i" from "m7i.large")
			parts := strings.Split(inst.InstanceType, ".")
			if len(parts) == 0 || parts[0] != listInstanceFamily {
				continue
			}
		}

		// Filter by tags
		matchesTags := true
		for _, tagFilter := range listTag {
			// Parse tag filter in format "key=value"
			parts := strings.SplitN(tagFilter, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := parts[0]
			value := parts[1]

			// Check if instance has this tag with matching value
			// Special handling for common tags
			if key == "Name" {
				if inst.Name != value {
					matchesTags = false
					break
				}
			} else if key == "spawn:ttl" {
				if inst.TTL != value {
					matchesTags = false
					break
				}
			} else if key == "spawn:idle-timeout" {
				if inst.IdleTimeout != value {
					matchesTags = false
					break
				}
			} else {
				// Check in Tags map
				tagValue, exists := inst.Tags[key]
				if !exists || tagValue != value {
					matchesTags = false
					break
				}
			}
		}

		if !matchesTags {
			continue
		}

		// Instance passed all filters
		filtered = append(filtered, inst)
	}

	return filtered
}
