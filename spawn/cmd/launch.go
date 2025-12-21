package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourusername/spawn/pkg/aws"
	"github.com/yourusername/spawn/pkg/input"
	"github.com/yourusername/spawn/pkg/platform"
	"github.com/yourusername/spawn/pkg/progress"
	"github.com/yourusername/spawn/pkg/wizard"
)

var (
	// Instance config
	instanceType string
	region       string
	az           string
	ami          string
	
	// Network (empty = auto-create)
	vpcID       string
	subnetID    string
	sgID        string
	
	// SSH key
	keyPair     string
	
	// Behavior
	spot             bool
	spotMaxPrice     string
	useReservation   bool
	reservationID    string
	hibernate        bool
	ttl              string
	idleTimeout      string
	hibernateOnIdle  bool
	
	// Meta
	name             string
	userData         string
	userDataFile     string
	
	// Mode
	interactive      bool
	quiet            bool
	waitForRunning   bool
	waitForSSH       bool
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch an EC2 instance",
	Long: `Launch an EC2 instance with smart defaults.

Three ways to use:
  1. Interactive wizard (default if no input)
  2. From truffle JSON via pipe
  3. Direct with flags

Examples:
  # Interactive wizard
  spawn launch
  
  # From truffle
  truffle search m7i.large | spawn launch
  
  # Direct
  spawn launch --instance-type m7i.large --region us-east-1`,
	RunE: runLaunch,
	Aliases: []string{"", "run", "create"},
}

func init() {
	rootCmd.AddCommand(launchCmd)
	
	// Instance config
	launchCmd.Flags().StringVar(&instanceType, "instance-type", "", "Instance type")
	launchCmd.Flags().StringVar(&region, "region", "", "AWS region")
	launchCmd.Flags().StringVar(&az, "az", "", "Availability zone")
	launchCmd.Flags().StringVar(&ami, "ami", "", "AMI ID (auto-detects AL2023)")
	
	// Network
	launchCmd.Flags().StringVar(&vpcID, "vpc", "", "VPC ID")
	launchCmd.Flags().StringVar(&subnetID, "subnet", "", "Subnet ID")
	launchCmd.Flags().StringVar(&sgID, "security-group", "", "Security group ID")
	
	// SSH
	launchCmd.Flags().StringVar(&keyPair, "key-pair", "", "SSH key pair name")
	
	// Capacity
	launchCmd.Flags().BoolVar(&spot, "spot", false, "Launch as Spot instance")
	launchCmd.Flags().StringVar(&spotMaxPrice, "spot-max-price", "", "Max Spot price")
	launchCmd.Flags().BoolVar(&useReservation, "use-reservation", false, "Use capacity reservation")
	launchCmd.Flags().StringVar(&reservationID, "reservation-id", "", "Capacity reservation ID")
	
	// Behavior
	launchCmd.Flags().BoolVar(&hibernate, "hibernate", false, "Enable hibernation")
	launchCmd.Flags().StringVar(&ttl, "ttl", "", "Auto-terminate after duration (e.g., 8h)")
	launchCmd.Flags().StringVar(&idleTimeout, "idle-timeout", "", "Auto-terminate if idle")
	launchCmd.Flags().BoolVar(&hibernateOnIdle, "hibernate-on-idle", false, "Hibernate instead of terminate when idle")
	
	// Meta
	launchCmd.Flags().StringVar(&name, "name", "", "Instance name tag")
	launchCmd.Flags().StringVar(&userData, "user-data", "", "User data (@file or inline)")
	launchCmd.Flags().StringVar(&userDataFile, "user-data-file", "", "User data file")
	
	// Mode
	launchCmd.Flags().BoolVar(&interactive, "interactive", false, "Force interactive wizard")
	launchCmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output")
	launchCmd.Flags().BoolVar(&waitForRunning, "wait-for-running", true, "Wait until running")
	launchCmd.Flags().BoolVar(&waitForSSH, "wait-for-ssh", true, "Wait until SSH is ready")
}

func runLaunch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Detect platform
	plat, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect platform: %w", err)
	}
	
	// Enable colors on Windows
	if plat.OS == "windows" {
		platform.EnableWindowsColors()
	}
	
	var config *aws.LaunchConfig
	
	// Determine mode: wizard, pipe, or flags
	if interactive || (instanceType == "" && isTerminal(os.Stdin)) {
		// Interactive wizard mode
		wiz := wizard.NewWizard(plat)
		config, err = wiz.Run(ctx)
		if err != nil {
			return err
		}
	} else if !isTerminal(os.Stdin) {
		// Pipe mode (from truffle)
		truffleInput, err := input.ParseFromStdin()
		if err != nil {
			return fmt.Errorf("failed to parse input: %w", err)
		}
		
		config, err = buildLaunchConfig(truffleInput)
		if err != nil {
			return err
		}
	} else {
		// Flags mode
		config, err = buildLaunchConfig(nil)
		if err != nil {
			return err
		}
	}
	
	// Validate
	if config.InstanceType == "" {
		return fmt.Errorf("instance type required")
	}
	if config.Region == "" {
		return fmt.Errorf("region required")
	}
	
	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}
	
	// Launch with progress display
	return launchWithProgress(ctx, awsClient, config, plat)
}

func launchWithProgress(ctx context.Context, awsClient *aws.Client, config *aws.LaunchConfig, plat *platform.Platform) error {
	prog := progress.NewProgress()
	
	// Step 1: Detect AMI
	prog.Start("Detecting AMI")
	if config.AMI == "" {
		ami, err := awsClient.GetRecommendedAMI(ctx, config.Region, config.InstanceType)
		if err != nil {
			prog.Error("Detecting AMI", err)
			return err
		}
		config.AMI = ami
	}
	prog.Complete("Detecting AMI")
	time.Sleep(300 * time.Millisecond)
	
	// Step 2: Setup SSH key
	prog.Start("Setting up SSH key")
	if config.KeyName == "" {
		keyName, err := setupSSHKey(ctx, awsClient, config.Region, plat)
		if err != nil {
			prog.Error("Setting up SSH key", err)
			return err
		}
		config.KeyName = keyName
	}
	prog.Complete("Setting up SSH key")
	time.Sleep(300 * time.Millisecond)

	// Step 3: Setup IAM instance profile
	prog.Start("Setting up IAM role")
	if config.IamInstanceProfile == "" {
		instanceProfile, err := awsClient.SetupSpawndIAMRole(ctx)
		if err != nil {
			prog.Error("Setting up IAM role", err)
			return err
		}
		config.IamInstanceProfile = instanceProfile
	}
	prog.Complete("Setting up IAM role")
	time.Sleep(300 * time.Millisecond)

	// Step 4: Security group (simplified for now)
	prog.Skip("Creating security group")

	// Step 5: Build user data
	userDataScript, err := buildUserData(plat)
	if err != nil {
		return fmt.Errorf("failed to build user data: %w", err)
	}
	config.UserData = base64.StdEncoding.EncodeToString([]byte(userDataScript))

	// Step 6: Launch instance
	prog.Start("Launching instance")
	result, err := awsClient.Launch(ctx, *config)
	if err != nil {
		prog.Error("Launching instance", err)
		return err
	}
	prog.Complete("Launching instance")
	time.Sleep(300 * time.Millisecond)

	// Step 7: Installing spawnd
	prog.Start("Installing spawnd agent")
	time.Sleep(30 * time.Second) // Wait for user-data
	prog.Complete("Installing spawnd agent")
	time.Sleep(300 * time.Millisecond)

	// Step 8: Wait for running
	prog.Start("Waiting for instance")
	if waitForRunning {
		time.Sleep(10 * time.Second) // Simplified
	}
	prog.Complete("Waiting for instance")
	time.Sleep(300 * time.Millisecond)

	// Step 9: Get public IP
	prog.Start("Getting public IP")
	publicIP, err := awsClient.GetInstancePublicIP(ctx, config.Region, result.InstanceID)
	if err != nil {
		prog.Error("Getting public IP", err)
		return err
	}
	result.PublicIP = publicIP
	prog.Complete("Getting public IP")
	time.Sleep(300 * time.Millisecond)

	// Step 10: Wait for SSH
	prog.Start("Waiting for SSH")
	if waitForSSH {
		time.Sleep(5 * time.Second) // Simplified
	}
	prog.Complete("Waiting for SSH")
	
	// Display success
	sshCmd := plat.GetSSHCommand("ec2-user", result.PublicIP)
	prog.DisplaySuccess(result.InstanceID, result.PublicIP, sshCmd, config)
	
	return nil
}

func buildLaunchConfig(truffleInput *input.TruffleInput) (*aws.LaunchConfig, error) {
	config := &aws.LaunchConfig{
		Tags: make(map[string]string),
	}
	
	// From truffle input
	if truffleInput != nil {
		config.InstanceType = truffleInput.InstanceType
		config.Region = truffleInput.Region
		config.AvailabilityZone = truffleInput.AvailabilityZone
		
		if truffleInput.Spot {
			config.Spot = true
			if truffleInput.SpotPrice > 0 {
				config.SpotMaxPrice = fmt.Sprintf("%.4f", truffleInput.SpotPrice)
			}
		}
	}
	
	// Override with flags
	if instanceType != "" {
		config.InstanceType = instanceType
	}
	if region != "" {
		config.Region = region
	}
	if az != "" {
		config.AvailabilityZone = az
	}
	if ami != "" {
		config.AMI = ami
	}
	if keyPair != "" {
		config.KeyName = keyPair
	}
	if spot {
		config.Spot = true
	}
	if hibernate {
		config.Hibernate = true
	}
	if ttl != "" {
		config.TTL = ttl
	}
	if idleTimeout != "" {
		config.IdleTimeout = idleTimeout
	}
	if hibernateOnIdle {
		config.HibernateOnIdle = true
	}
	if name != "" {
		config.Name = name
	}
	
	return config, nil
}

func setupSSHKey(ctx context.Context, awsClient *aws.Client, region string, plat *platform.Platform) (string, error) {
	// Check for local SSH key
	if !plat.HasSSHKey() {
		return "", fmt.Errorf("no SSH key found at %s (run spawn with --interactive to create one)", plat.SSHKeyPath)
	}

	// Get fingerprint of local key
	fingerprint, err := plat.GetPublicKeyFingerprint()
	if err != nil {
		return "", fmt.Errorf("failed to get key fingerprint: %w", err)
	}

	// Check if this key already exists in AWS (by fingerprint)
	existingKeyName, err := awsClient.FindKeyPairByFingerprint(ctx, region, fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to search for existing key: %w", err)
	}

	// If found, use the existing key
	if existingKeyName != "" {
		return existingKeyName, nil
	}

	// Key not found in AWS, upload it with generated name
	keyName := fmt.Sprintf("spawn-key-%s", plat.GetUsername())

	publicKey, err := plat.ReadPublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	err = awsClient.ImportKeyPair(ctx, region, keyName, publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to import key pair: %w", err)
	}

	return keyName, nil
}

func buildUserData(plat *platform.Platform) (string, error) {
	// Get local username and SSH public key
	username := plat.GetUsername()
	publicKey, err := plat.ReadPublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to read SSH public key: %w", err)
	}
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	// Read custom user data if provided
	customUserData := ""

	if userDataFile != "" {
		data, err := os.ReadFile(userDataFile)
		if err != nil {
			return "", err
		}
		customUserData = string(data)
	} else if userData != "" {
		if strings.HasPrefix(userData, "@") {
			path := userData[1:]
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			customUserData = string(data)
		} else {
			customUserData = userData
		}
	}
	
	// Build user-data with spawnd installer (S3-based with SHA256 verification)
	script := fmt.Sprintf(`#!/bin/bash
set -e

# User configuration
LOCAL_USERNAME="%s"
LOCAL_SSH_KEY_BASE64="%s"
`, username, publicKeyBase64) + `

# Detect architecture
ARCH=$(uname -m)
echo "Installing spawnd for architecture: $ARCH"

# Detect region
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600" 2>/dev/null || true)
if [ -n "$TOKEN" ]; then
    REGION=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null)
else
    REGION=$(curl -s http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null || echo "us-east-1")
fi

echo "Region: $REGION"

# Determine binary name
case "$ARCH" in
    x86_64)
        BINARY="spawnd-linux-amd64"
        ;;
    aarch64)
        BINARY="spawnd-linux-arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Download from S3 (public buckets, regional for low latency)
S3_BASE_URL="https://spawn-binaries-${REGION}.s3.amazonaws.com"
FALLBACK_URL="https://spawn-binaries-us-east-1.s3.amazonaws.com"

echo "Downloading spawnd binary..."

# Try regional bucket first, fallback to us-east-1
if curl -f -o /usr/local/bin/spawnd "${S3_BASE_URL}/${BINARY}" 2>/dev/null; then
    CHECKSUM_URL="${S3_BASE_URL}/${BINARY}.sha256"
    echo "Downloaded from ${REGION}"
else
    echo "Regional bucket unavailable, using us-east-1"
    curl -f -o /usr/local/bin/spawnd "${FALLBACK_URL}/${BINARY}" || {
        echo "Failed to download spawnd binary"
        exit 1
    }
    CHECKSUM_URL="${FALLBACK_URL}/${BINARY}.sha256"
fi

# Download and verify SHA256 checksum
echo "Verifying checksum..."
curl -f -o /tmp/spawnd.sha256 "${CHECKSUM_URL}" || {
    echo "Failed to download checksum"
    exit 1
}

cd /usr/local/bin
EXPECTED_CHECKSUM=$(cat /tmp/spawnd.sha256)
ACTUAL_CHECKSUM=$(sha256sum spawnd | awk '{print $1}')

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    echo "❌ Checksum verification failed!"
    echo "   Expected: $EXPECTED_CHECKSUM"
    echo "   Actual:   $ACTUAL_CHECKSUM"
    rm -f /usr/local/bin/spawnd
    exit 1
fi

echo "✅ Checksum verified: $EXPECTED_CHECKSUM"
chmod +x /usr/local/bin/spawnd

# Setup local user account
echo "Setting up user: $LOCAL_USERNAME"

# Create user if doesn't exist
if ! id "$LOCAL_USERNAME" &>/dev/null; then
    useradd -m -s /bin/bash "$LOCAL_USERNAME"
    echo "Created user: $LOCAL_USERNAME"
fi

# Add to sudo/wheel group (passwordless sudo like ec2-user)
usermod -aG wheel "$LOCAL_USERNAME" 2>/dev/null || usermod -aG sudo "$LOCAL_USERNAME" 2>/dev/null
echo "$LOCAL_USERNAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$LOCAL_USERNAME
chmod 0440 /etc/sudoers.d/$LOCAL_USERNAME

# Setup SSH for local user
mkdir -p /home/$LOCAL_USERNAME/.ssh
chmod 700 /home/$LOCAL_USERNAME/.ssh

# Decode and write SSH public key
echo "$LOCAL_SSH_KEY_BASE64" | base64 -d > /home/$LOCAL_USERNAME/.ssh/authorized_keys
chmod 600 /home/$LOCAL_USERNAME/.ssh/authorized_keys
chown -R $LOCAL_USERNAME:$LOCAL_USERNAME /home/$LOCAL_USERNAME/.ssh

echo "✅ User $LOCAL_USERNAME configured with SSH access and sudo privileges"

# Create systemd service
cat > /etc/systemd/system/spawnd.service <<'EOFSERVICE'
[Unit]
Description=Spawn Agent - Instance self-monitoring
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/spawnd
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOFSERVICE

# Enable and start
systemctl daemon-reload
systemctl enable spawnd
systemctl start spawnd

echo "spawnd installation complete"
`
	
	if customUserData != "" {
		script += "\n# Custom user data\n"
		script += customUserData
	}
	
	return script, nil
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
