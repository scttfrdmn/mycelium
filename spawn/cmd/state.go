package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/spawn/pkg/aws"
)

// stop command
var stopCmd = &cobra.Command{
	Use:   "stop <instance-id-or-name>",
	Short: "Stop a spawn-managed instance",
	Long: `Stop a spawn-managed EC2 instance (EBS-backed only).

The instance will be stopped, preserving the EBS volume but stopping compute charges.
The TTL countdown will pause while the instance is stopped.

Examples:
  # Stop an instance
  spawn stop i-1234567890abcdef0

  # Stop by name
  spawn stop my-instance`,
	RunE: runStop,
	Args: cobra.ExactArgs(1),
}

// hibernate command
var hibernateCmd = &cobra.Command{
	Use:   "hibernate <instance-id-or-name>",
	Short: "Hibernate a spawn-managed instance",
	Long: `Hibernate a spawn-managed EC2 instance.

The instance will be hibernated, saving RAM contents to EBS and stopping compute charges.
Hibernation allows for faster startup than a regular stop.
The TTL countdown will pause while the instance is hibernated.

Note: Instance must support hibernation and have it enabled at launch time.

Examples:
  # Hibernate an instance
  spawn hibernate i-1234567890abcdef0

  # Hibernate by name
  spawn hibernate my-instance`,
	RunE:    runHibernate,
	Aliases: []string{"sleep"},
	Args:    cobra.ExactArgs(1),
}

// start command
var startCmd = &cobra.Command{
	Use:   "start <instance-id-or-name>",
	Short: "Start a stopped or hibernated instance",
	Long: `Start a stopped or hibernated spawn-managed EC2 instance.

The instance will be started and the TTL countdown will resume.

Examples:
  # Start an instance
  spawn start i-1234567890abcdef0

  # Start by name
  spawn start my-instance`,
	RunE: runStart,
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(hibernateCmd)
	rootCmd.AddCommand(startCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	return stopOrHibernate(args[0], false)
}

func runHibernate(cmd *cobra.Command, args []string) error {
	return stopOrHibernate(args[0], true)
}

func stopOrHibernate(identifier string, hibernate bool) error {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Resolve instance
	instance, err := resolveInstance(ctx, client, identifier)
	if err != nil {
		return err
	}

	// Check current state
	if instance.State == "stopped" {
		return fmt.Errorf("instance %s is already stopped", instance.InstanceID)
	}

	if instance.State != "running" {
		return fmt.Errorf("instance %s is not running (state: %s)", instance.InstanceID, instance.State)
	}

	action := "stop"
	if hibernate {
		action = "hibernate"
	}

	fmt.Fprintf(os.Stderr, "Found instance in %s (state: %s)\n", instance.Region, instance.State)
	fmt.Fprintf(os.Stderr, "Requesting %s for instance %s...\n", action, instance.InstanceID)

	// Calculate remaining TTL if set
	remainingTTL := ""
	if instance.TTL != "" {
		// Parse the TTL duration
		ttlDuration, err := parseDuration(instance.TTL)
		if err == nil {
			// Calculate how long the instance has been running
			uptime := time.Since(instance.LaunchTime)
			remaining := ttlDuration - uptime

			if remaining > 0 {
				// Store remaining time so it can be restored on start
				remainingTTL = formatDurationForTTL(remaining)
				fmt.Fprintf(os.Stderr, "Saving remaining TTL: %s (was: %s, uptime: %s)\n",
					remainingTTL, instance.TTL, uptime.Round(time.Minute))
			}
		}
	}

	// Stop the instance
	err = client.StopInstance(ctx, instance.Region, instance.InstanceID, hibernate)
	if err != nil {
		return fmt.Errorf("failed to %s instance: %w", action, err)
	}

	// Tag the instance with the stop reason and remaining TTL
	stopReason := "user-stopped"
	if hibernate {
		stopReason = "user-hibernated"
	}
	tags := map[string]string{
		"spawn:last-stop-reason": stopReason,
		"spawn:last-stop-time":   time.Now().UTC().Format(time.RFC3339),
	}
	if remainingTTL != "" {
		tags["spawn:ttl-remaining"] = remainingTTL
	}
	client.UpdateInstanceTags(ctx, instance.Region, instance.InstanceID, tags)

	if hibernate {
		fmt.Fprintf(os.Stdout, "\n✅ Hibernate request sent!\n")
		fmt.Fprintf(os.Stdout, "   Instance: %s\n", instance.InstanceID)
		fmt.Fprintf(os.Stdout, "   Region:   %s\n", instance.Region)
		fmt.Fprintf(os.Stdout, "\nThe instance will hibernate (save RAM to disk) and stop.\n")
		fmt.Fprintf(os.Stdout, "TTL countdown will pause until the instance is started again.\n")
	} else {
		fmt.Fprintf(os.Stdout, "\n✅ Stop request sent!\n")
		fmt.Fprintf(os.Stdout, "   Instance: %s\n", instance.InstanceID)
		fmt.Fprintf(os.Stdout, "   Region:   %s\n", instance.Region)
		fmt.Fprintf(os.Stdout, "\nThe instance will stop (EBS preserved).\n")
		fmt.Fprintf(os.Stdout, "TTL countdown will pause until the instance is started again.\n")
	}

	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	identifier := args[0]
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Resolve instance
	instance, err := resolveInstance(ctx, client, identifier)
	if err != nil {
		return err
	}

	// Check current state
	if instance.State == "running" {
		return fmt.Errorf("instance %s is already running", instance.InstanceID)
	}

	if instance.State != "stopped" {
		return fmt.Errorf("instance %s cannot be started (state: %s)", instance.InstanceID, instance.State)
	}

	fmt.Fprintf(os.Stderr, "Found instance in %s (state: %s)\n", instance.Region, instance.State)
	fmt.Fprintf(os.Stderr, "Starting instance %s...\n", instance.InstanceID)

	// Start the instance
	err = client.StartInstance(ctx, instance.Region, instance.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	// Check if there's a saved TTL to restore
	tags := map[string]string{
		"spawn:last-start-time": time.Now().UTC().Format(time.RFC3339),
	}

	if ttlRemaining, exists := instance.Tags["spawn:ttl-remaining"]; exists && ttlRemaining != "" {
		// Restore the remaining TTL
		tags["spawn:ttl"] = ttlRemaining
		// Clear the saved remaining TTL (will be set in subsequent update)
		fmt.Fprintf(os.Stderr, "Restoring TTL: %s (was paused during stop)\n", ttlRemaining)
	}

	// Tag the instance with the start event and restored TTL
	client.UpdateInstanceTags(ctx, instance.Region, instance.InstanceID, tags)

	fmt.Fprintf(os.Stdout, "\n✅ Start request sent!\n")
	fmt.Fprintf(os.Stdout, "   Instance: %s\n", instance.InstanceID)
	fmt.Fprintf(os.Stdout, "   Region:   %s\n", instance.Region)
	fmt.Fprintf(os.Stdout, "\nThe instance is starting up...\n")
	fmt.Fprintf(os.Stdout, "TTL countdown will resume once the instance is running.\n")

	// Wait for instance to be running
	fmt.Fprintf(os.Stderr, "\nWaiting for instance to reach running state...")
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		// Refresh instance state
		instances, err := client.ListInstances(ctx, instance.Region, "")
		if err != nil {
			break
		}

		for _, inst := range instances {
			if inst.InstanceID == instance.InstanceID {
				if inst.State == "running" {
					fmt.Fprintf(os.Stderr, " running!\n")
					if inst.PublicIP != "" {
						fmt.Fprintf(os.Stdout, "\n🔌 Connect: spawn connect %s\n", instance.InstanceID)
					}
					return nil
				}
				break
			}
		}
		fmt.Fprintf(os.Stderr, ".")
	}

	fmt.Fprintf(os.Stderr, " (taking longer than expected)\n")
	fmt.Fprintf(os.Stdout, "\nUse 'spawn list' to check the current state.\n")

	return nil
}

// parseDuration parses a TTL duration string (e.g., "2h", "30m", "1h30m")
func parseDuration(ttl string) (time.Duration, error) {
	// Try parsing as Go duration first
	d, err := time.ParseDuration(ttl)
	if err == nil {
		return d, nil
	}

	// Try custom format: <number><unit> where unit is s, m, h, d
	var total time.Duration
	remaining := ttl

	for len(remaining) > 0 {
		// Find the next number-unit pair
		i := 0
		for i < len(remaining) && (remaining[i] >= '0' && remaining[i] <= '9') {
			i++
		}
		if i == 0 {
			return 0, fmt.Errorf("invalid duration format: %s", ttl)
		}

		numStr := remaining[:i]
		if i >= len(remaining) {
			return 0, fmt.Errorf("invalid duration format: missing unit in %s", ttl)
		}

		unit := remaining[i]
		remaining = remaining[i+1:]

		num, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration: %s", numStr)
		}

		switch unit {
		case 's':
			total += time.Duration(num) * time.Second
		case 'm':
			total += time.Duration(num) * time.Minute
		case 'h':
			total += time.Duration(num) * time.Hour
		case 'd':
			total += time.Duration(num) * 24 * time.Hour
		default:
			return 0, fmt.Errorf("invalid unit in duration: %c", unit)
		}
	}

	return total, nil
}

// formatDurationForTTL formats a duration for use as a TTL tag value
func formatDurationForTTL(d time.Duration) string {
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
