package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Launch AWS EC2 instances effortlessly",
	Long: `spawn - Ephemeral AWS EC2 instance launcher (companion to truffle)

spawn makes it super easy to launch AWS instances:
  • Auto-detects AMI (Amazon Linux 2023, including GPU variants)
  • Auto-configures SSH keys (uses ~/.ssh/id_rsa by default)
  • Auto-creates VPC/subnet/security groups (tagged for cleanup)
  • Installs spawnd agent for self-monitoring
  • Auto-terminates on idle or TTL
  • Auto-cleans up all resources

Perfect for:
  • Quick dev/test instances
  • ML training jobs
  • Spot instances
  • Non-experts who just need compute

Examples:
  # From truffle
  truffle search m7i.large | spawn
  
  # Spot instance
  truffle spot m7i.large --max-price 0.10 | spawn --spot
  
  # GPU training with auto-terminate
  truffle capacity --gpu-only | spawn --ttl 24h --hibernate-on-idle
  
  # Direct launch
  spawn --instance-type m7i.large --region us-east-1

Unix Philosophy:
  truffle finds, spawn launches
  Pipe JSON from truffle to spawn for seamless workflow`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
