package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/spawn/pkg/aws"
)

var extendCmd = &cobra.Command{
	Use:  "extend <instance-id> <duration>",
	RunE: runExtend,
	Args: cobra.ExactArgs(2),
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(extendCmd)

	// Register completion for instance ID argument
	extendCmd.ValidArgsFunction = completeInstanceID
}

func runExtend(cmd *cobra.Command, args []string) error {
	instanceIdentifier := args[0]
	newTTL := args[1]
	ctx := context.Background()

	// Validate TTL format
	if err := validateTTL(newTTL); err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return i18n.Te("error.aws_client_init", err)
	}

	// Resolve instance (by ID or name)
	instance, err := resolveInstance(ctx, client, instanceIdentifier)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Found instance in %s (current TTL: %s)\n", instance.Region, instance.TTL)

	// Update the TTL tag
	fmt.Fprintf(os.Stderr, "Updating TTL to %s...\n", newTTL)
	err = client.UpdateInstanceTags(ctx, instance.Region, instance.InstanceID, map[string]string{
		"spawn:ttl": newTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to update TTL: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\n✅ TTL extended successfully!\n")
	fmt.Fprintf(os.Stdout, "   Instance: %s\n", instance.InstanceID)
	fmt.Fprintf(os.Stdout, "   Old TTL:  %s\n", instance.TTL)
	fmt.Fprintf(os.Stdout, "   New TTL:  %s\n", newTTL)
	fmt.Fprintf(os.Stdout, "\nThe spored daemon will automatically detect the new TTL and adjust its schedule.\n")

	return nil
}

func validateTTL(ttl string) error {
	// TTL format: <number><unit> where unit is s, m, h, or d
	// Also supports multiple components like "3h30m"
	pattern := regexp.MustCompile(`^(\d+[smhd])+$`)
	if !pattern.MatchString(ttl) {
		return fmt.Errorf("TTL must be in format <number><unit> (e.g., 2h, 30m, 1d)")
	}

	// Parse each component to ensure it's valid
	componentPattern := regexp.MustCompile(`(\d+)([smhd])`)
	matches := componentPattern.FindAllStringSubmatch(ttl, -1)

	if len(matches) == 0 {
		return fmt.Errorf("no valid duration components found")
	}

	totalSeconds := 0
	for _, match := range matches {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return fmt.Errorf("invalid number: %s", match[1])
		}

		unit := match[2]
		switch unit {
		case "s":
			totalSeconds += value
		case "m":
			totalSeconds += value * 60
		case "h":
			totalSeconds += value * 3600
		case "d":
			totalSeconds += value * 86400
		}
	}

	if totalSeconds <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	return nil
}

// Helper function to format duration for display
func formatTTLDuration(ttl string) string {
	componentPattern := regexp.MustCompile(`(\d+)([smhd])`)
	matches := componentPattern.FindAllStringSubmatch(ttl, -1)

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		value := match[1]
		unit := match[2]

		var unitName string
		switch unit {
		case "s":
			unitName = "second"
		case "m":
			unitName = "minute"
		case "h":
			unitName = "hour"
		case "d":
			unitName = "day"
		}

		// Pluralize if needed
		if value != "1" {
			unitName += "s"
		}

		parts = append(parts, fmt.Sprintf("%s %s", value, unitName))
	}

	return strings.Join(parts, " ")
}
