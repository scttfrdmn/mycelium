package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/spawn/pkg/aws"
)

var (
	listRegion         string
	listAZ             string
	listState          string
	listInstanceType   string
	listInstanceFamily string
	listTag            []string
	listJSON           bool
	listJobArrayID     string
	listJobArrayName   string
)

var listCmd = &cobra.Command{
	Use:     "list",
	RunE:    runList,
	Aliases: []string{"ls"},
	// Short and Long will be set after i18n initialization
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
	listCmd.Flags().StringVar(&listJobArrayID, "job-array-id", "", "Filter by job array ID")
	listCmd.Flags().StringVar(&listJobArrayName, "job-array-name", "", "Filter by job array name")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return i18n.Te("error.aws_client_init", err)
	}

	// List instances
	if listRegion != "" {
		fmt.Fprintf(os.Stderr, "%s...\n", i18n.Tf("spawn.list.searching_region", map[string]interface{}{
			"Region": listRegion,
		}))
	} else {
		fmt.Fprintf(os.Stderr, "%s...\n", i18n.T("spawn.list.searching_all_regions"))
	}

	instances, err := client.ListInstances(ctx, listRegion, listState)
	if err != nil {
		return i18n.Te("spawn.list.error.list_failed", err)
	}

	if len(instances) == 0 {
		fmt.Printf("\n%s\n", i18n.T("spawn.list.no_instances"))
		return nil
	}

	// Apply additional filters
	instances = filterInstances(instances)

	if len(instances) == 0 {
		fmt.Printf("\n%s\n", i18n.T("spawn.list.no_instances_match"))
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

	// Group instances by job array ID
	jobArrays := make(map[string][]aws.InstanceInfo)
	standaloneInstances := []aws.InstanceInfo{}

	for _, inst := range instances {
		if inst.JobArrayID != "" {
			jobArrays[inst.JobArrayID] = append(jobArrays[inst.JobArrayID], inst)
		} else {
			standaloneInstances = append(standaloneInstances, inst)
		}
	}

	// Sort job array IDs for consistent output
	var jobArrayIDs []string
	for id := range jobArrays {
		jobArrayIDs = append(jobArrayIDs, id)
	}
	sort.Strings(jobArrayIDs)

	// Display job arrays first
	if len(jobArrays) > 0 {
		fmt.Println("\033[1mJob Arrays:\033[0m")
		for _, arrayID := range jobArrayIDs {
			arrayInstances := jobArrays[arrayID]

			// Sort instances by index within array
			sort.Slice(arrayInstances, func(i, j int) bool {
				return arrayInstances[i].JobArrayIndex < arrayInstances[j].JobArrayIndex
			})

			// Count states
			runningCount := 0
			stoppedCount := 0
			pendingCount := 0
			for _, inst := range arrayInstances {
				switch inst.State {
				case "running":
					runningCount++
				case "stopped", "stopping":
					stoppedCount++
				case "pending":
					pendingCount++
				}
			}

			// Display job array summary
			arrayName := arrayInstances[0].JobArrayName
			if arrayName == "" {
				arrayName = "unnamed"
			}
			fmt.Printf("\n  %s (%d instances", arrayName, len(arrayInstances))
			if runningCount > 0 {
				fmt.Printf(", %d running", runningCount)
			}
			if pendingCount > 0 {
				fmt.Printf(", %d pending", pendingCount)
			}
			if stoppedCount > 0 {
				fmt.Printf(", %d stopped", stoppedCount)
			}
			fmt.Printf(")\n")
			fmt.Printf("  Array ID: %s\n", arrayID)

			// Display instances
			for _, inst := range arrayInstances {
				displayInstance(inst, "    ")
			}
		}
		fmt.Println()
	}

	// Display standalone instances
	if len(standaloneInstances) > 0 {
		if len(jobArrays) > 0 {
			fmt.Println("\033[1mStandalone Instances:\033[0m")
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Header
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i18n.T("spawn.list.header.instance_id"),
			i18n.T("spawn.list.header.name"),
			i18n.T("spawn.list.header.type"),
			i18n.T("spawn.list.header.state"),
			i18n.T("spawn.list.header.az"),
			i18n.T("spawn.list.header.age"),
			i18n.T("spawn.list.header.ttl"),
			i18n.T("spawn.list.header.public_ip"),
			i18n.T("spawn.list.header.spot"),
		)

		for _, inst := range standaloneInstances {
			age := formatDuration(time.Since(inst.LaunchTime))
			ttl := inst.TTL
			if ttl == "" {
				ttl = "none"
			}
			name := inst.Name
			if name == "" {
				name = "-"
			}
			spotIndicator := ""
			if inst.SpotInstance {
				spotIndicator = "✓"
			}
			state := colorizeState(inst.State)
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
	}

	return nil
}

func displayInstance(inst aws.InstanceInfo, prefix string) {
	state := colorizeState(inst.State)
	publicIP := inst.PublicIP
	if publicIP == "" {
		publicIP = "-"
	}
	name := inst.Name
	if name == "" {
		name = "-"
	}
	spotIndicator := ""
	if inst.SpotInstance {
		spotIndicator = " (spot)"
	}

	fmt.Printf("%s[%s] %s  %s  %s  %s  %s  %s%s\n",
		prefix,
		inst.JobArrayIndex,
		name,
		inst.InstanceID,
		inst.InstanceType,
		state,
		inst.AvailabilityZone,
		publicIP,
		spotIndicator,
	)
}

func colorizeState(state string) string {
	switch state {
	case "running":
		return "\033[32m" + state + "\033[0m" // Green
	case "stopped":
		return "\033[33m" + state + "\033[0m" // Yellow
	case "stopping":
		return "\033[33m" + state + "\033[0m" // Yellow
	case "pending":
		return "\033[36m" + state + "\033[0m" // Cyan
	default:
		return state
	}
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

		// Filter by job array ID
		if listJobArrayID != "" && inst.JobArrayID != listJobArrayID {
			continue
		}

		// Filter by job array name
		if listJobArrayName != "" && inst.JobArrayName != listJobArrayName {
			continue
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
