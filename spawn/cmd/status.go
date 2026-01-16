package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/spawn/pkg/aws"
	"github.com/scttfrdmn/mycelium/spawn/pkg/sweep"
)

var (
	statusSweepID string
)

var statusCmd = &cobra.Command{
	Use:  "status <instance-id>",
	RunE: runStatus,
	Args: cobra.MaximumNArgs(1),
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVar(&statusSweepID, "sweep-id", "", "Check parameter sweep status instead of instance status")

	// Register completion for instance ID argument
	statusCmd.ValidArgsFunction = completeInstanceID
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if sweep status requested
	if statusSweepID != "" {
		return runSweepStatus(ctx, statusSweepID)
	}

	// Instance status mode (original behavior)
	if len(args) == 0 {
		return fmt.Errorf("instance ID required, or use --sweep-id for sweep status")
	}

	instanceIdentifier := args[0]

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

	// Find SSH key
	keyPath, err := findSSHKey(instance.KeyName)
	if err != nil {
		return fmt.Errorf("failed to find SSH key: %w", err)
	}

	// Run spored status via SSH
	sshArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("ec2-user@%s", instance.PublicIP),
		"sudo /usr/local/bin/spored status 2>&1",
	}

	sshCmd := exec.Command("ssh", sshArgs...)
	output, err := sshCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get status: %w\nOutput: %s", err, string(output))
	}

	fmt.Print(string(output))
	return nil
}

func runSweepStatus(ctx context.Context, sweepID string) error {
	// Load AWS SDK config for mycelium-infra (where DynamoDB table lives)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithSharedConfigProfile("mycelium-infra"),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Query sweep status
	fmt.Fprintf(os.Stderr, "🔍 Querying sweep status...\n\n")
	status, err := sweep.QuerySweepStatus(ctx, cfg, sweepID)
	if err != nil {
		return fmt.Errorf("failed to query sweep status: %w", err)
	}

	// Display sweep information
	fmt.Fprintf(os.Stdout, "╔═══════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stdout, "║  Parameter Sweep Status                                      ║\n")
	fmt.Fprintf(os.Stdout, "╚═══════════════════════════════════════════════════════════════╝\n\n")

	fmt.Fprintf(os.Stdout, "Sweep ID:          %s\n", status.SweepID)
	fmt.Fprintf(os.Stdout, "Sweep Name:        %s\n", status.SweepName)
	fmt.Fprintf(os.Stdout, "Status:            %s\n", colorizeStatus(status.Status))
	fmt.Fprintf(os.Stdout, "Region:            %s\n", status.Region)
	fmt.Fprintf(os.Stdout, "\n")

	// Display timestamps
	createdAt, _ := time.Parse(time.RFC3339, status.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, status.UpdatedAt)
	fmt.Fprintf(os.Stdout, "Created:           %s\n", createdAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(os.Stdout, "Last Updated:      %s\n", updatedAt.Format("2006-01-02 15:04:05 MST"))

	if status.CompletedAt != "" {
		completedAt, _ := time.Parse(time.RFC3339, status.CompletedAt)
		fmt.Fprintf(os.Stdout, "Completed:         %s\n", completedAt.Format("2006-01-02 15:04:05 MST"))
		duration := completedAt.Sub(createdAt)
		fmt.Fprintf(os.Stdout, "Duration:          %s\n", formatDuration(duration))
	}
	fmt.Fprintf(os.Stdout, "\n")

	// Display progress
	fmt.Fprintf(os.Stdout, "Progress:\n")
	fmt.Fprintf(os.Stdout, "  Total Parameters:  %d\n", status.TotalParams)
	fmt.Fprintf(os.Stdout, "  Launched:          %d (%.1f%%)\n", status.Launched, float64(status.Launched)/float64(status.TotalParams)*100)
	fmt.Fprintf(os.Stdout, "  Next to Launch:    %d\n", status.NextToLaunch)
	fmt.Fprintf(os.Stdout, "  Failed:            %d\n", status.Failed)

	// Calculate and display estimated completion time
	if status.Status == "RUNNING" && status.Launched > 0 && status.NextToLaunch < status.TotalParams {
		elapsed := updatedAt.Sub(createdAt)
		avgTimePerLaunch := elapsed / time.Duration(status.Launched)
		remaining := status.TotalParams - status.NextToLaunch

		// Account for max concurrent limiting
		remainingBatches := (remaining + status.MaxConcurrent - 1) / status.MaxConcurrent
		estimatedRemaining := time.Duration(remainingBatches) * avgTimePerLaunch * time.Duration(status.MaxConcurrent)
		estimatedCompletion := time.Now().Add(estimatedRemaining)

		fmt.Fprintf(os.Stdout, "  Est. Completion:   %s (in %s)\n",
			estimatedCompletion.Format("3:04 PM MST"),
			formatDuration(estimatedRemaining))
	}
	fmt.Fprintf(os.Stdout, "\n")

	// Display configuration
	fmt.Fprintf(os.Stdout, "Configuration:\n")
	fmt.Fprintf(os.Stdout, "  Max Concurrent:    %d\n", status.MaxConcurrent)
	fmt.Fprintf(os.Stdout, "  Launch Delay:      %s\n", status.LaunchDelay)
	fmt.Fprintf(os.Stdout, "\n")

	// Calculate active instances
	activeCount := 0
	completedCount := 0
	failedCount := 0
	for _, inst := range status.Instances {
		switch inst.State {
		case "pending", "running":
			activeCount++
		case "terminated", "stopped":
			completedCount++
		case "failed":
			failedCount++
		}
	}

	fmt.Fprintf(os.Stdout, "Instances:\n")
	fmt.Fprintf(os.Stdout, "  Active:            %d\n", activeCount)
	fmt.Fprintf(os.Stdout, "  Completed:         %d\n", completedCount)
	fmt.Fprintf(os.Stdout, "  Failed:            %d\n", failedCount)
	fmt.Fprintf(os.Stdout, "\n")

	// Display error message if any
	if status.ErrorMessage != "" {
		fmt.Fprintf(os.Stdout, "⚠️  Error: %s\n\n", status.ErrorMessage)
	}

	// Display instance details (limited to most recent 10)
	if len(status.Instances) > 0 {
		fmt.Fprintf(os.Stdout, "Recent Instances (showing last 10):\n")
		fmt.Fprintf(os.Stdout, "%-5s %-20s %-15s %-20s\n", "Index", "Instance ID", "State", "Launched At")
		fmt.Fprintf(os.Stdout, "%-5s %-20s %-15s %-20s\n", "-----", "--------------------", "---------------", "--------------------")

		// Show last 10 instances
		startIdx := 0
		if len(status.Instances) > 10 {
			startIdx = len(status.Instances) - 10
		}

		for _, inst := range status.Instances[startIdx:] {
			launchedAt, _ := time.Parse(time.RFC3339, inst.LaunchedAt)
			stateDisplay := colorizeInstanceState(inst.State)
			fmt.Fprintf(os.Stdout, "%-5d %-20s %-15s %-20s\n",
				inst.Index,
				inst.InstanceID,
				stateDisplay,
				launchedAt.Format("2006-01-02 15:04:05"),
			)
		}
		fmt.Fprintf(os.Stdout, "\n")
	}

	// Display failed launches if any
	if failedCount > 0 {
		fmt.Fprintf(os.Stdout, "Failed Launches:\n")
		for _, inst := range status.Instances {
			if inst.State == "failed" {
				fmt.Fprintf(os.Stdout, "  [%d] %s\n", inst.Index, inst.ErrorMessage)
			}
		}
		fmt.Fprintf(os.Stdout, "\n")
	}

	// Display next steps based on status
	if status.Status == "RUNNING" {
		fmt.Fprintf(os.Stdout, "The sweep is currently running in Lambda.\n")
		fmt.Fprintf(os.Stdout, "Re-run this command to see updated progress.\n")
	} else if status.Status == "COMPLETED" {
		fmt.Fprintf(os.Stdout, "✅ Sweep completed successfully!\n")
	} else if status.Status == "FAILED" {
		fmt.Fprintf(os.Stdout, "❌ Sweep failed. Check error message above.\n")
		fmt.Fprintf(os.Stdout, "\nTo resume:\n")
		fmt.Fprintf(os.Stdout, "  spawn resume --sweep-id %s --detach\n", status.SweepID)
	}

	return nil
}

func colorizeStatus(status string) string {
	switch status {
	case "INITIALIZING":
		return "🔄 " + status
	case "RUNNING":
		return "🚀 " + status
	case "COMPLETED":
		return "✅ " + status
	case "FAILED":
		return "❌ " + status
	case "CANCELLED":
		return "⚠️  " + status
	default:
		return status
	}
}

func colorizeInstanceState(state string) string {
	switch state {
	case "pending":
		return "🔄 " + state
	case "running":
		return "🟢 " + state
	case "terminated", "stopped":
		return "⚪ " + state
	case "failed":
		return "❌ " + state
	default:
		return state
	}
}

