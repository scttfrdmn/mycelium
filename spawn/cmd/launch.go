package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/spawn/pkg/aws"
	spawnconfig "github.com/scttfrdmn/mycelium/spawn/pkg/config"
	"github.com/scttfrdmn/mycelium/spawn/pkg/input"
	"github.com/scttfrdmn/mycelium/spawn/pkg/platform"
	"github.com/scttfrdmn/mycelium/spawn/pkg/progress"
	"github.com/scttfrdmn/mycelium/spawn/pkg/wizard"
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
	onComplete       string
	completionFile   string
	completionDelay  string
	sessionTimeout   string

	// Meta
	name             string
	userData         string
	userDataFile     string
	dnsName          string
	dnsDomain        string
	dnsAPIEndpoint   string

	// Mode
	interactive      bool
	quiet            bool
	waitForRunning   bool
	waitForSSH       bool
)

var launchCmd = &cobra.Command{
	Use:     "launch",
	RunE:    runLaunch,
	Aliases: []string{"", "run", "create"},
	// Short and Long will be set after i18n initialization
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
	launchCmd.Flags().StringVar(&onComplete, "on-complete", "", "Action when workload signals completion: terminate, stop, hibernate")
	launchCmd.Flags().StringVar(&completionFile, "completion-file", "/tmp/SPAWN_COMPLETE", "File to watch for completion signal")
	launchCmd.Flags().StringVar(&completionDelay, "completion-delay", "30s", "Grace period after completion signal")
	launchCmd.Flags().StringVar(&sessionTimeout, "session-timeout", "30m", "Auto-logout idle shells (0 to disable)")

	// Meta
	launchCmd.Flags().StringVar(&name, "name", "", "Instance name tag")
	launchCmd.Flags().StringVar(&userData, "user-data", "", "User data (@file or inline)")
	launchCmd.Flags().StringVar(&userDataFile, "user-data-file", "", "User data file")
	launchCmd.Flags().StringVar(&dnsName, "dns", "", "Register DNS name (e.g., my-instance for my-instance.spore.host)")
	launchCmd.Flags().StringVar(&dnsDomain, "dns-domain", "", "Custom DNS domain (overrides default)")
	launchCmd.Flags().StringVar(&dnsAPIEndpoint, "dns-api-endpoint", "", "Custom DNS API endpoint (overrides default)")

	// Mode
	launchCmd.Flags().BoolVar(&interactive, "interactive", false, "Force interactive wizard")
	launchCmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output")
	launchCmd.Flags().BoolVar(&waitForRunning, "wait-for-running", true, "Wait until running")
	launchCmd.Flags().BoolVar(&waitForSSH, "wait-for-ssh", true, "Wait until SSH is ready")

	// Register completions for flags
	launchCmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeRegion(cmd, args, toComplete)
	})
	launchCmd.RegisterFlagCompletionFunc("instance-type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeInstanceType(cmd, args, toComplete)
	})
}

func runLaunch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect platform
	plat, err := platform.Detect()
	if err != nil {
		return i18n.Te("error.platform_detect_failed", err)
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
			return i18n.Te("error.input_parse_failed", err)
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
		return i18n.Te("error.instance_type_required", nil)
	}

	// Auto-detect region if not specified
	if config.Region == "" {
		fmt.Fprintf(os.Stderr, "🌍 No region specified, auto-detecting closest region...\n")
		detectedRegion, err := detectBestRegion(ctx, config.InstanceType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Could not auto-detect region: %v\n", err)
			fmt.Fprintf(os.Stderr, "   Using default: us-east-1\n")
			config.Region = "us-east-1"
		} else {
			fmt.Fprintf(os.Stderr, "✓ Selected region: %s\n", detectedRegion)
			config.Region = detectedRegion
		}
	}

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return i18n.Te("error.aws_client_init", err)
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
		instanceProfile, err := awsClient.SetupSporedIAMRole(ctx)
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

	// Step 7: Installing spore agent
	prog.Start("Installing spore agent")
	time.Sleep(30 * time.Second) // Wait for user-data
	prog.Complete("Installing spore agent")
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

	// Step 11: Register DNS (if requested)
	var dnsRecord string
	if dnsName != "" {
		// Load DNS configuration with precedence
		dnsConfig, err := spawnconfig.LoadDNSConfig(ctx, dnsDomain, dnsAPIEndpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n⚠️  Failed to load DNS config: %v\n", err)
		} else {
			prog.Start("Registering DNS")
			fqdn, err := registerDNS(plat, result.InstanceID, result.PublicIP, dnsName, dnsConfig.Domain, dnsConfig.APIEndpoint)
			if err != nil {
				prog.Error("Registering DNS", err)
				// Non-fatal: continue even if DNS registration fails
				fmt.Fprintf(os.Stderr, "\n⚠️  DNS registration failed: %v\n", err)
			} else {
				dnsRecord = fqdn
				prog.Complete("Registering DNS")
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Display success
	sshCmd := plat.GetSSHCommand("ec2-user", result.PublicIP)
	prog.DisplaySuccess(result.InstanceID, result.PublicIP, sshCmd, config)

	// Show DNS info if registered
	if dnsRecord != "" {
		fmt.Fprintf(os.Stdout, "\n🌐 DNS: %s\n", dnsRecord)
		fmt.Fprintf(os.Stdout, "   Connect: ssh %s@%s\n", plat.GetUsername(), dnsRecord)
	}

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
	if dnsName != "" {
		config.DNSName = dnsName
	}
	if idleTimeout != "" {
		config.IdleTimeout = idleTimeout
	}
	if hibernateOnIdle {
		config.HibernateOnIdle = true
	}
	if onComplete != "" {
		config.OnComplete = onComplete
	}
	if completionFile != "" {
		config.CompletionFile = completionFile
	}
	if completionDelay != "" {
		config.CompletionDelay = completionDelay
	}
	if sessionTimeout != "" {
		config.SessionTimeout = sessionTimeout
	}
	if name != "" {
		config.Name = name
	}

	return config, nil
}

func setupSSHKey(ctx context.Context, awsClient *aws.Client, region string, plat *platform.Platform) (string, error) {
	// Check for local SSH key
	if !plat.HasSSHKey() {
		// Auto-create SSH key if running in a terminal
		if isTerminal(os.Stdin) {
			fmt.Fprintf(os.Stderr, "\n⚠️  No SSH key found at %s\n", plat.SSHKeyPath)
			fmt.Fprintf(os.Stderr, "   Creating SSH key automatically...\n")

			if err := plat.CreateSSHKey(); err != nil {
				return "", fmt.Errorf("failed to create SSH key: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ SSH key created: %s\n\n", plat.SSHKeyPath)
		} else {
			// Non-interactive stdin (piped input) - provide helpful error
			return "", fmt.Errorf("no SSH key found at %s\n\nTo create one:\n  ssh-keygen -t rsa -b 4096 -f %s -N ''\n\nOr run spawn directly (not piped):\n  spawn launch --instance-type m7i.large --region us-east-1",
				plat.SSHKeyPath, plat.SSHKeyPath)
		}
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
	
	// Build user-data with spored installer (S3-based with SHA256 verification)
	script := fmt.Sprintf(`#!/bin/bash
set -e

# User configuration
LOCAL_USERNAME="%s"
LOCAL_SSH_KEY_BASE64="%s"
`, username, publicKeyBase64) + `

# Detect architecture
ARCH=$(uname -m)
echo "Installing spored for architecture: $ARCH"

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
        BINARY="spored-linux-amd64"
        ;;
    aarch64)
        BINARY="spored-linux-arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Download from S3 (public buckets, regional for low latency)
S3_BASE_URL="https://spawn-binaries-${REGION}.s3.amazonaws.com"
FALLBACK_URL="https://spawn-binaries-us-east-1.s3.amazonaws.com"

echo "Downloading spored binary..."

# Try regional bucket first, fallback to us-east-1
if curl -f -o /usr/local/bin/spored "${S3_BASE_URL}/${BINARY}" 2>/dev/null; then
    CHECKSUM_URL="${S3_BASE_URL}/${BINARY}.sha256"
    echo "Downloaded from ${REGION}"
else
    echo "Regional bucket unavailable, using us-east-1"
    curl -f -o /usr/local/bin/spored "${FALLBACK_URL}/${BINARY}" || {
        echo "Failed to download spored binary"
        exit 1
    }
    CHECKSUM_URL="${FALLBACK_URL}/${BINARY}.sha256"
fi

# Download and verify SHA256 checksum
echo "Verifying checksum..."
curl -f -o /tmp/spored.sha256 "${CHECKSUM_URL}" || {
    echo "Failed to download checksum"
    exit 1
}

cd /usr/local/bin
EXPECTED_CHECKSUM=$(cat /tmp/spored.sha256)
ACTUAL_CHECKSUM=$(sha256sum spored | awk '{print $1}')

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    echo "❌ Checksum verification failed!"
    echo "   Expected: $EXPECTED_CHECKSUM"
    echo "   Actual:   $ACTUAL_CHECKSUM"
    rm -f /usr/local/bin/spored
    exit 1
fi

echo "✅ Checksum verified: $EXPECTED_CHECKSUM"
chmod +x /usr/local/bin/spored

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

# Configure automatic logout for idle sessions
# This prevents indefinite logins when users leave sessions idle
echo "Configuring session timeouts..."

# Get session timeout from EC2 tags
INSTANCE_ID=$(ec2-metadata --instance-id | cut -d " " -f 2)
SESSION_TIMEOUT=$(aws ec2 describe-tags --region $REGION \
    --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:session-timeout" \
    --query 'Tags[0].Value' --output text 2>/dev/null || echo "30m")

# Convert duration to seconds
parse_duration() {
    local duration="$1"
    local total=0

    # Handle 0 or disabled
    if [ "$duration" = "0" ] || [ "$duration" = "disabled" ]; then
        echo "0"
        return
    fi

    # Parse duration (e.g., 30m, 1h30m, 2h)
    while [[ $duration =~ ([0-9]+)([smhd]) ]]; do
        local value="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"

        case "$unit" in
            s) total=$((total + value)) ;;
            m) total=$((total + value * 60)) ;;
            h) total=$((total + value * 3600)) ;;
            d) total=$((total + value * 86400)) ;;
        esac

        duration="${duration#*${BASH_REMATCH[0]}}"
    done

    echo "$total"
}

TIMEOUT_SECONDS=$(parse_duration "$SESSION_TIMEOUT")

if [ "$TIMEOUT_SECONDS" -gt 0 ]; then
    # SSH server timeout (disconnect idle SSH connections)
    # Use 1/2 of session timeout for SSH keepalive
    SSH_INTERVAL=$((TIMEOUT_SECONDS / 6))
    if [ $SSH_INTERVAL -lt 60 ]; then
        SSH_INTERVAL=60
    fi

    if ! grep -q "^ClientAliveInterval" /etc/ssh/sshd_config; then
        echo "ClientAliveInterval $SSH_INTERVAL" >> /etc/ssh/sshd_config
        echo "ClientAliveCountMax 3" >> /etc/ssh/sshd_config
        systemctl reload sshd 2>/dev/null || service sshd reload 2>/dev/null || true
    fi

    # Shell timeout (auto-logout idle shells)
    # Set TMOUT for all users via /etc/profile.d/
    cat > /etc/profile.d/session-timeout.sh <<EOFTIMEOUT
# Automatic logout for idle shells ($SESSION_TIMEOUT)
# Prevents indefinite logins when users leave sessions idle
# Note: readonly prevents users from unsetting it
export TMOUT=$TIMEOUT_SECONDS
readonly TMOUT
EOFTIMEOUT

    chmod 644 /etc/profile.d/session-timeout.sh

    echo "✅ Session timeouts configured (Timeout: $SESSION_TIMEOUT)"
else
    echo "✅ Session timeouts disabled (set spawn:session-timeout tag to enable)"
fi

# Create login banner (MOTD) with spore configuration
echo "Creating login banner..."

# Get all spore tags for the banner
TTL_TAG=$(aws ec2 describe-tags --region $REGION \
    --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:ttl" \
    --query 'Tags[0].Value' --output text 2>/dev/null || echo "disabled")

IDLE_TIMEOUT_TAG=$(aws ec2 describe-tags --region $REGION \
    --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:idle-timeout" \
    --query 'Tags[0].Value' --output text 2>/dev/null || echo "disabled")

ON_COMPLETE_TAG=$(aws ec2 describe-tags --region $REGION \
    --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:on-complete" \
    --query 'Tags[0].Value' --output text 2>/dev/null || echo "disabled")

# Create the banner
cat > /etc/motd <<EOFMOTD
╔═══════════════════════════════════════════════════════════════╗
║                  🌱 Welcome to Spore                         ║
╚═══════════════════════════════════════════════════════════════╝

Instance: $INSTANCE_ID
Region:   $REGION

Spore Agent Configuration:
  • TTL:              $TTL_TAG
  • Idle Timeout:     $IDLE_TIMEOUT_TAG
  • On Complete:      $ON_COMPLETE_TAG
  • Session Timeout:  $SESSION_TIMEOUT

⚠️  IMPORTANT: This shell will auto-logout after $SESSION_TIMEOUT of inactivity
⚠️  SSH connections will disconnect if idle for extended periods

To view current status:
  sudo spored status

To extend TTL:
  spawn extend <instance-id> <new-ttl>

Documentation: https://github.com/scttfrdmn/mycelium

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOFMOTD

chmod 644 /etc/motd
echo "✅ Login banner created"

# Create systemd service
cat > /etc/systemd/system/spored.service <<'EOFSERVICE'
[Unit]
Description=Spawn Agent - Instance self-monitoring
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/spored
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOFSERVICE

# Enable and start
systemctl daemon-reload
systemctl enable spored
systemctl start spored

echo "spored installation complete"
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

func registerDNS(plat *platform.Platform, instanceID, publicIP, recordName, domain, apiEndpoint string) (string, error) {
	// Build SSH command to register DNS from within the instance
	sshScript := fmt.Sprintf(`
# Get IMDSv2 token
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600" -s 2>/dev/null)

# Get instance identity
IDENTITY_DOC=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/dynamic/instance-identity/document 2>/dev/null | base64 -w0)
IDENTITY_SIG=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/dynamic/instance-identity/signature 2>/dev/null | tr -d '\n')
PUBLIC_IP=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null)

# Call DNS API
curl -s -X POST %s \
  -H "Content-Type: application/json" \
  -d "{
    \"instance_identity_document\": \"$IDENTITY_DOC\",
    \"instance_identity_signature\": \"$IDENTITY_SIG\",
    \"record_name\": \"%s\",
    \"ip_address\": \"$PUBLIC_IP\",
    \"action\": \"UPSERT\"
  }" 2>/dev/null || echo '{"success":false,"error":"DNS API call failed"}'
`, apiEndpoint, recordName)

	// Execute SSH command
	sshKeyPath := plat.SSHKeyPath
	username := plat.GetUsername()

	// Build SSH command arguments
	sshArgs := []string{
		"-i", sshKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", username, publicIP),
		sshScript,
	}

	// Execute
	cmd := exec.Command("ssh", sshArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute SSH command: %w (output: %s)", err, string(output))
	}

	// Parse response
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Record  string `json:"record"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &response); err != nil {
		return "", fmt.Errorf("failed to parse DNS API response: %w (output: %s)", err, string(output))
	}

	if !response.Success {
		return "", fmt.Errorf("%s", response.Error)
	}

	return response.Record, nil
}

// detectBestRegion automatically selects the closest AWS region
// that has the requested instance type available and is allowed by SCPs.
// It prioritizes in-country/in-continent regions based on IP geolocation.
func detectBestRegion(ctx context.Context, instanceType string) (string, error) {
	// First, get allowed regions from AWS (respects SCPs)
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create AWS client: %w", err)
	}

	allowedRegions, err := awsClient.GetEnabledRegions(ctx)
	if err != nil || len(allowedRegions) == 0 {
		// Fallback to common regions if we can't get the list
		allowedRegions = []string{
			"us-east-1", "us-west-2", "eu-west-1",
			"ap-southeast-1", "us-east-2", "eu-central-1",
		}
	}

	// Try to detect user's location via IP geolocation
	userContinent := detectUserContinent()

	// Measure latency to each allowed region's EC2 endpoint
	type regionScore struct {
		region         string
		latency        time.Duration
		continentMatch bool
	}

	results := make([]regionScore, 0, len(allowedRegions))

	for _, region := range allowedRegions {
		start := time.Now()

		// Quick connectivity test to EC2 endpoint
		endpoint := fmt.Sprintf("ec2.%s.amazonaws.com", region)
		conn, err := net.DialTimeout("tcp", endpoint+":443", 2*time.Second)
		if err != nil {
			// Skip regions we can't reach (may be blocked by SCP or network)
			continue
		}
		conn.Close()

		latency := time.Since(start)
		continentMatch := matchesContinent(region, userContinent)

		results = append(results, regionScore{
			region:         region,
			latency:        latency,
			continentMatch: continentMatch,
		})
	}

	if len(results) == 0 {
		return "", fmt.Errorf("could not connect to any allowed AWS region")
	}

	// Sort by: continent match first, then latency
	sort.Slice(results, func(i, j int) bool {
		// Prioritize continent matches
		if results[i].continentMatch != results[j].continentMatch {
			return results[i].continentMatch
		}
		// Within same continent preference, choose lowest latency
		return results[i].latency < results[j].latency
	})

	// Return the best scored region
	return results[0].region, nil
}

// detectUserContinent attempts to determine the user's continent from their public IP
func detectUserContinent() string {
	// Try ipapi.co (free, no API key needed for moderate usage)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://ipapi.co/json/")
	if err != nil {
		return "" // Failed, will fall back to latency-only
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var result struct {
		CountryCode string `json:"country_code"`
		Continent   string `json:"continent_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	// Map continent codes: AF, AN, AS, EU, NA, OC, SA
	return result.Continent
}

// matchesContinent checks if an AWS region matches the user's continent
func matchesContinent(region, continentCode string) bool {
	if continentCode == "" {
		return false // Unknown continent, no preference
	}

	// Map AWS region prefixes to continent codes
	regionToContinentMap := map[string]string{
		"us-":      "NA", // North America
		"ca-":      "NA", // Canada
		"eu-":      "EU", // Europe
		"me-":      "AS", // Middle East (Asia)
		"af-":      "AF", // Africa
		"ap-":      "AS", // Asia Pacific
		"sa-":      "SA", // South America
		"il-":      "AS", // Israel (Middle East)
		"ap-south": "AS", // India
	}

	// Check region prefix
	for prefix, continent := range regionToContinentMap {
		if len(region) >= len(prefix) && region[:len(prefix)] == prefix {
			return continent == continentCode
		}
	}

	return false
}
