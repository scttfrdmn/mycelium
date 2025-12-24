package cmd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/spawn/pkg/aws"
)

var statusCmd = &cobra.Command{
	Use:  "status <instance-id>",
	RunE: runStatus,
	Args: cobra.ExactArgs(1),
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Register completion for instance ID argument
	statusCmd.ValidArgsFunction = completeInstanceID
}

func runStatus(cmd *cobra.Command, args []string) error {
	instanceIdentifier := args[0]
	ctx := context.Background()

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
