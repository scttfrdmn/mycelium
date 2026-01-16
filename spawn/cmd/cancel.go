package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/mycelium/spawn/pkg/sweep"
	"github.com/spf13/cobra"
)

var cancelSweepID string

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a running parameter sweep",
	Long: `Cancel a running parameter sweep and terminate all instances.

Queries DynamoDB for the sweep state, terminates all running/pending
instances via cross-account access, and updates the sweep status to CANCELLED.

Examples:
  # Cancel a running sweep
  spawn cancel --sweep-id sweep-20260116-abc123
`,
	RunE: runCancel,
}

func init() {
	cancelCmd.Flags().StringVar(&cancelSweepID, "sweep-id", "", "Sweep ID to cancel (required)")
	cancelCmd.MarkFlagRequired("sweep-id")

	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Fprintf(os.Stderr, "\n🛑 Cancelling Parameter Sweep\n")
	fmt.Fprintf(os.Stderr, "   Sweep ID: %s\n\n", cancelSweepID)

	// Load AWS config for mycelium-infra (where DynamoDB lives)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithSharedConfigProfile("mycelium-infra"),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Query sweep state
	fmt.Fprintf(os.Stderr, "📊 Querying sweep state...\n")
	state, err := sweep.QuerySweepStatus(ctx, cfg, cancelSweepID)
	if err != nil {
		return fmt.Errorf("failed to query sweep state: %w", err)
	}

	// Display current status
	fmt.Fprintf(os.Stderr, "\n   Sweep Name: %s\n", state.SweepName)
	fmt.Fprintf(os.Stderr, "   Status: %s\n", state.Status)
	fmt.Fprintf(os.Stderr, "   Region: %s\n", state.Region)
	fmt.Fprintf(os.Stderr, "   Progress: %d/%d launched\n", state.Launched, state.TotalParams)

	// Check if already cancelled or completed
	if state.Status == "CANCELLED" {
		fmt.Fprintf(os.Stderr, "\n⚠️  Sweep is already cancelled\n")
		return nil
	}
	if state.Status == "COMPLETED" {
		fmt.Fprintf(os.Stderr, "\n⚠️  Sweep is already completed\n")
		return nil
	}

	// Count instances to terminate
	instancesToTerminate := []string{}
	for _, inst := range state.Instances {
		if inst.InstanceID != "" && (inst.State == "pending" || inst.State == "running") {
			instancesToTerminate = append(instancesToTerminate, inst.InstanceID)
		}
	}

	fmt.Fprintf(os.Stderr, "\n🔍 Found %d instances to terminate\n\n", len(instancesToTerminate))

	// Terminate instances if any
	if len(instancesToTerminate) > 0 {
		fmt.Fprintf(os.Stderr, "⚡ Terminating instances...\n")

		// Use dev account credentials directly (user should be authenticated to the target account)
		devCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(state.Region))
		if err != nil {
			return fmt.Errorf("failed to load dev account config: %w", err)
		}

		if err := sweep.TerminateSweepInstancesDirect(ctx, devCfg, instancesToTerminate); err != nil {
			return fmt.Errorf("failed to terminate instances: %w", err)
		}
		fmt.Fprintf(os.Stderr, "   Terminated %d instances\n", len(instancesToTerminate))
	}

	// Update sweep status to CANCELLED
	fmt.Fprintf(os.Stderr, "\n📝 Updating sweep status to CANCELLED...\n")
	state.Status = "CANCELLED"
	state.CompletedAt = time.Now().Format(time.RFC3339)
	if err := sweep.SaveSweepState(ctx, cfg, state); err != nil {
		return fmt.Errorf("failed to update sweep status: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n✅ Sweep cancelled successfully!\n")
	fmt.Fprintf(os.Stderr, "   Sweep ID: %s\n", cancelSweepID)
	if len(instancesToTerminate) > 0 {
		fmt.Fprintf(os.Stderr, "   Terminated: %d instances\n", len(instancesToTerminate))
	}
	fmt.Fprintf(os.Stderr, "\n")

	return nil
}
