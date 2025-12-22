package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/spawn/pkg/aws"
)

var (
	connectUser       string
	connectKey        string
	connectPort       int
	connectSessionMgr bool
)

var connectCmd = &cobra.Command{
	Use:   "connect <instance-id>",
	Short: "Connect to a spawn-managed instance via SSH",
	Long: `Connect to a spawn-managed instance using SSH.

Automatically detects the SSH key and public IP from instance metadata.
Falls back to AWS Systems Manager Session Manager if no SSH access available.

Examples:
  # Connect to instance
  spawn connect i-1234567890abcdef0

  # Specify SSH key explicitly
  spawn connect i-1234567890abcdef0 --key ~/.ssh/my-key.pem

  # Use Session Manager instead of SSH
  spawn connect i-1234567890abcdef0 --session-manager

  # Connect with specific user
  spawn connect i-1234567890abcdef0 --user ubuntu`,
	RunE:    runConnect,
	Aliases: []string{"ssh"},
	Args:    cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVar(&connectUser, "user", "", "SSH username (default: ec2-user)")
	connectCmd.Flags().StringVar(&connectKey, "key", "", "SSH private key path")
	connectCmd.Flags().IntVar(&connectPort, "port", 22, "SSH port")
	connectCmd.Flags().BoolVar(&connectSessionMgr, "session-manager", false, "Use AWS Session Manager instead of SSH")
}

func runConnect(cmd *cobra.Command, args []string) error {
	instanceIdentifier := args[0]
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Resolve instance (by ID or name)
	instance, err := resolveInstance(ctx, client, instanceIdentifier)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Found instance in %s (state: %s)\n", instance.Region, instance.State)

	// Check if instance is running
	if instance.State != "running" {
		return fmt.Errorf("instance is not running (state: %s)", instance.State)
	}

	// Use Session Manager if requested or if no public IP
	if connectSessionMgr || instance.PublicIP == "" {
		return connectViaSessionManager(instance.InstanceID, instance.Region)
	}

	// Determine SSH user
	user := connectUser
	if user == "" {
		user = "ec2-user" // Default for Amazon Linux
	}

	// Determine SSH key
	keyPath := connectKey
	if keyPath == "" {
		// Try to find the key based on the instance key name
		keyPath, err = findSSHKey(instance.KeyName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not find SSH key for %s: %v\n", instance.KeyName, err)
			fmt.Fprintf(os.Stderr, "Falling back to Session Manager...\n\n")
			return connectViaSessionManager(instance.InstanceID, instance.Region)
		}
	}

	// Build SSH command
	sshArgs := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", fmt.Sprintf("%d", connectPort),
		fmt.Sprintf("%s@%s", user, instance.PublicIP),
	}

	fmt.Fprintf(os.Stderr, "Connecting via SSH: ssh %s\n\n", strings.Join(sshArgs, " "))

	// Execute SSH
	sshCmd := exec.Command("ssh", sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func connectViaSessionManager(instanceID, region string) error {
	// Check if AWS CLI and Session Manager plugin are installed
	_, err := exec.LookPath("aws")
	if err != nil {
		return fmt.Errorf("AWS CLI not found. Install it to use Session Manager: https://aws.amazon.com/cli/")
	}

	fmt.Fprintf(os.Stderr, "Connecting via AWS Session Manager...\n\n")

	// Build AWS SSM start-session command
	ssmCmd := exec.Command("aws", "ssm", "start-session",
		"--target", instanceID,
		"--region", region,
	)

	ssmCmd.Stdin = os.Stdin
	ssmCmd.Stdout = os.Stdout
	ssmCmd.Stderr = os.Stderr

	err = ssmCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start Session Manager session: %w\nMake sure the Session Manager plugin is installed: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html", err)
	}

	return nil
}

func findSSHKey(keyName string) (string, error) {
	if keyName == "" {
		return "", fmt.Errorf("no key name associated with instance")
	}

	// Common SSH key locations and naming patterns
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Try various common key names
	keyPatterns := []string{
		filepath.Join(sshDir, keyName),           // Exact name
		filepath.Join(sshDir, keyName+".pem"),    // With .pem
		filepath.Join(sshDir, keyName+".key"),    // With .key
		filepath.Join(sshDir, "id_rsa"),          // Default RSA key
		filepath.Join(sshDir, "id_ed25519"),      // Default Ed25519 key
		filepath.Join(sshDir, "id_ecdsa"),        // Default ECDSA key
	}

	for _, path := range keyPatterns {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no SSH key found for key name: %s", keyName)
}
