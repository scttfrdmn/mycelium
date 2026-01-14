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
	"strconv"
	"strings"
	"sync"
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

	// Job array
	count            int
	jobArrayName     string
	instanceNames    string
	command          string

	// IAM
	iamRole            string
	iamPolicy          []string
	iamManagedPolicies []string
	iamPolicyFile      string
	iamTrustServices   []string
	iamRoleTags        []string

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

	// Job array
	launchCmd.Flags().IntVar(&count, "count", 1, "Number of instances to launch (job array)")
	launchCmd.Flags().StringVar(&jobArrayName, "job-array-name", "", "Job array group name (required if --count > 1)")
	launchCmd.Flags().StringVar(&instanceNames, "instance-names", "", "Instance name template (e.g., 'worker-{index}', default: '{job-array-name}-{index}')")
	launchCmd.Flags().StringVar(&command, "command", "", "Command to run on all instances (executed after spored setup)")

	// IAM
	launchCmd.Flags().StringVar(&iamRole, "iam-role", "", "IAM role name (creates if doesn't exist)")
	launchCmd.Flags().StringSliceVar(&iamPolicy, "iam-policy", []string{}, "Service-level policies (e.g., s3:ReadOnly,dynamodb:WriteOnly)")
	launchCmd.Flags().StringSliceVar(&iamManagedPolicies, "iam-managed-policies", []string{}, "AWS managed policy ARNs")
	launchCmd.Flags().StringVar(&iamPolicyFile, "iam-policy-file", "", "Custom IAM policy JSON file")
	launchCmd.Flags().StringSliceVar(&iamTrustServices, "iam-trust-services", []string{"ec2"}, "Services that can assume role")
	launchCmd.Flags().StringSliceVar(&iamRoleTags, "iam-role-tags", []string{}, "Tags for IAM role (key=value format)")

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
		// Check if user specified custom IAM configuration
		if iamRole != "" || len(iamPolicy) > 0 || len(iamManagedPolicies) > 0 || iamPolicyFile != "" {
			// User-specified IAM configuration
			iamConfig := aws.IAMRoleConfig{
				RoleName:        iamRole,
				Policies:        iamPolicy,
				ManagedPolicies: iamManagedPolicies,
				PolicyFile:      iamPolicyFile,
				TrustServices:   iamTrustServices,
				Tags:            parseIAMRoleTags(iamRoleTags),
			}

			instanceProfile, err := awsClient.CreateOrGetInstanceProfile(ctx, iamConfig)
			if err != nil {
				prog.Error("Setting up IAM role", err)
				return fmt.Errorf("failed to create IAM instance profile: %w", err)
			}
			config.IamInstanceProfile = instanceProfile
		} else {
			// Default: use spored IAM role
			instanceProfile, err := awsClient.SetupSporedIAMRole(ctx)
			if err != nil {
				prog.Error("Setting up IAM role", err)
				return err
			}
			config.IamInstanceProfile = instanceProfile
		}
	}
	prog.Complete("Setting up IAM role")
	time.Sleep(300 * time.Millisecond)

	// Step 4: Security group (simplified for now)
	prog.Skip("Creating security group")

	// Step 5: Build user data
	userDataScript, err := buildUserData(plat, config)
	if err != nil {
		return fmt.Errorf("failed to build user data: %w", err)
	}
	config.UserData = base64.StdEncoding.EncodeToString([]byte(userDataScript))

	// Check if job array mode (count > 1)
	if count > 1 {
		// Job array launch path
		if jobArrayName == "" {
			return fmt.Errorf("--job-array-name is required when --count > 1")
		}
		return launchJobArray(ctx, awsClient, config, plat, prog)
	}

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

func buildUserData(plat *platform.Platform, config *aws.LaunchConfig) (string, error) {
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

# Setup job array environment variables (if part of a job array)
JOB_ARRAY_ID=$(aws ec2 describe-tags --region $REGION \
    --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:job-array-id" \
    --query 'Tags[0].Value' --output text 2>/dev/null || echo "None")

if [ "$JOB_ARRAY_ID" != "None" ]; then
    echo "Setting up job array environment..."

    JOB_ARRAY_NAME=$(aws ec2 describe-tags --region $REGION \
        --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:job-array-name" \
        --query 'Tags[0].Value' --output text 2>/dev/null || echo "")

    JOB_ARRAY_SIZE=$(aws ec2 describe-tags --region $REGION \
        --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:job-array-size" \
        --query 'Tags[0].Value' --output text 2>/dev/null || echo "0")

    JOB_ARRAY_INDEX=$(aws ec2 describe-tags --region $REGION \
        --filters "Name=resource-id,Values=$INSTANCE_ID" "Name=key,Values=spawn:job-array-index" \
        --query 'Tags[0].Value' --output text 2>/dev/null || echo "0")

    # Create job array environment file
    cat > /etc/profile.d/job-array.sh <<EOFJOBARRAY
# Job Array Environment Variables
# Available to all shells for coordinated distributed computing
export JOB_ARRAY_ID="$JOB_ARRAY_ID"
export JOB_ARRAY_NAME="$JOB_ARRAY_NAME"
export JOB_ARRAY_SIZE="$JOB_ARRAY_SIZE"
export JOB_ARRAY_INDEX="$JOB_ARRAY_INDEX"
EOFJOBARRAY

    chmod 644 /etc/profile.d/job-array.sh

    echo "✅ Job array environment configured (Index: $JOB_ARRAY_INDEX/$JOB_ARRAY_SIZE)"
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

# Create the banner with optional job array info
if [ "$JOB_ARRAY_ID" != "None" ]; then
cat > /etc/motd <<EOFMOTD
╔═══════════════════════════════════════════════════════════════╗
║                  🌱 Welcome to Spore                         ║
╚═══════════════════════════════════════════════════════════════╝

Instance: $INSTANCE_ID
Region:   $REGION

Job Array:
  • Array Name:       $JOB_ARRAY_NAME
  • Array ID:         $JOB_ARRAY_ID
  • Instance Index:   $JOB_ARRAY_INDEX / $JOB_ARRAY_SIZE

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
else
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
fi

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

// Job Array Helper Functions

// generateJobArrayID creates a unique ID for a job array
// Format: {name}-{timestamp}-{random}
// Example: compute-20260113-abc123
func generateJobArrayID(name string) string {
	timestamp := time.Now().Format("20060102")
	// Generate 6-character random suffix (base36: 0-9a-z)
	random := fmt.Sprintf("%06x", time.Now().UnixNano()%0xFFFFFF)
	return fmt.Sprintf("%s-%s-%s", name, timestamp, random)
}

// formatInstanceName applies template substitution for instance names
// Supported variables: {index}, {job-array-name}
// Default template: "{job-array-name}-{index}"
func formatInstanceName(template string, jobArrayName string, index int) string {
	if template == "" {
		template = "{job-array-name}-{index}"
	}

	name := template
	name = strings.ReplaceAll(name, "{index}", fmt.Sprintf("%d", index))
	name = strings.ReplaceAll(name, "{job-array-name}", jobArrayName)

	return name
}

// launchJobArray launches N instances in parallel as a job array
func launchJobArray(ctx context.Context, awsClient *aws.Client, baseConfig *aws.LaunchConfig, plat *platform.Platform, prog *progress.Progress) error {
	// Generate unique job array ID
	jobArrayID := generateJobArrayID(jobArrayName)
	createdAt := time.Now()

	fmt.Fprintf(os.Stderr, "\n🚀 Launching job array: %s (%d instances)\n", jobArrayName, count)
	fmt.Fprintf(os.Stderr, "   Job Array ID: %s\n\n", jobArrayID)

	// Phase 1: Launch all instances in parallel
	prog.Start(fmt.Sprintf("Launching %d instances in parallel", count))

	type launchResult struct {
		index      int
		result     *aws.LaunchResult
		err        error
	}

	results := make(chan launchResult, count)
	var wg sync.WaitGroup

	// Launch each instance in a goroutine
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Clone config for this instance
			instanceConfig := *baseConfig

			// Set job array fields
			instanceConfig.JobArrayID = jobArrayID
			instanceConfig.JobArrayName = jobArrayName
			instanceConfig.JobArraySize = count
			instanceConfig.JobArrayIndex = index
			instanceConfig.JobArrayCommand = command

			// Set instance name from template
			instanceConfig.Name = formatInstanceName(instanceNames, jobArrayName, index)

			// Set DNS name with index suffix if DNS is enabled
			if baseConfig.DNSName != "" {
				instanceConfig.DNSName = fmt.Sprintf("%s-%d", baseConfig.DNSName, index)
			}

			// Launch the instance
			result, err := awsClient.Launch(ctx, instanceConfig)
			results <- launchResult{
				index:  index,
				result: result,
				err:    err,
			}
		}(i)
	}

	// Wait for all launches to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	launchedInstances := make([]*aws.LaunchResult, 0, count)
	var launchErrors []string
	successCount := 0
	failureCount := 0

	for result := range results {
		if result.err != nil {
			launchErrors = append(launchErrors, fmt.Sprintf("Instance %d: %v", result.index, result.err))
			failureCount++
		} else {
			launchedInstances = append(launchedInstances, result.result)
			successCount++
		}
	}

	// Handle partial failures
	if failureCount > 0 {
		prog.Error(fmt.Sprintf("Launching %d instances", count), fmt.Errorf("%d/%d instances failed to launch", failureCount, count))

		// Terminate successfully launched instances
		if successCount > 0 {
			fmt.Fprintf(os.Stderr, "\n⚠️  Cleaning up %d successfully launched instances...\n", successCount)
			for _, inst := range launchedInstances {
				_ = awsClient.Terminate(ctx, baseConfig.Region, inst.InstanceID)
			}
		}

		// Return detailed error
		return fmt.Errorf("job array launch failed: %d/%d instances failed:\n  %s",
			failureCount, count, strings.Join(launchErrors, "\n  "))
	}

	prog.Complete(fmt.Sprintf("Launching %d instances", count))
	time.Sleep(300 * time.Millisecond)

	// Sort instances by index for consistent display
	sort.Slice(launchedInstances, func(i, j int) bool {
		// Extract index from Name (assumes format: name-{index})
		getName := func(r *aws.LaunchResult) int {
			parts := strings.Split(r.Name, "-")
			if len(parts) > 0 {
				if idx, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					return idx
				}
			}
			return 0
		}
		return getName(launchedInstances[i]) < getName(launchedInstances[j])
	})

	// Phase 2: Wait for all instances to reach "running" state
	prog.Start("Waiting for all instances to reach running state")
	maxWaitTime := 2 * time.Minute
	checkInterval := 5 * time.Second
	startTime := time.Now()

	allRunning := false
	for time.Since(startTime) < maxWaitTime {
		allRunning = true
		for _, inst := range launchedInstances {
			state, err := awsClient.GetInstanceState(ctx, baseConfig.Region, inst.InstanceID)
			if err != nil || state != "running" {
				allRunning = false
				break
			}
		}

		if allRunning {
			break
		}

		time.Sleep(checkInterval)
	}

	if !allRunning {
		prog.Error("Waiting for instances", fmt.Errorf("timeout waiting for all instances to reach running state"))
		return fmt.Errorf("timeout: not all instances reached running state within %v", maxWaitTime)
	}

	prog.Complete("Waiting for all instances")
	time.Sleep(300 * time.Millisecond)

	// Phase 3: Get public IPs for all instances
	prog.Start("Getting public IPs")
	for _, inst := range launchedInstances {
		publicIP, err := awsClient.GetInstancePublicIP(ctx, baseConfig.Region, inst.InstanceID)
		if err != nil {
			prog.Error("Getting public IP", err)
			// Non-fatal: continue with other instances
			fmt.Fprintf(os.Stderr, "\n⚠️  Failed to get IP for %s: %v\n", inst.InstanceID, err)
		} else {
			inst.PublicIP = publicIP
		}
	}
	prog.Complete("Getting public IPs")
	time.Sleep(300 * time.Millisecond)

	// Note: Peer discovery is handled dynamically by spored agent
	// Each agent queries EC2 for all instances with the same spawn:job-array-id tag
	// This avoids AWS tag size limitations (256 char max) and scales to any array size

	// Display success for job array
	fmt.Fprintf(os.Stderr, "\n✅ Job array launched successfully!\n\n")
	fmt.Fprintf(os.Stderr, "Job Array: %s\n", jobArrayName)
	fmt.Fprintf(os.Stderr, "Array ID:  %s\n", jobArrayID)
	fmt.Fprintf(os.Stderr, "Created:   %s\n", createdAt.Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "Count:     %d instances\n", count)
	fmt.Fprintf(os.Stderr, "Region:    %s\n\n", baseConfig.Region)

	// Display table of instances
	fmt.Fprintf(os.Stderr, "Instances:\n")
	fmt.Fprintf(os.Stderr, "%-5s %-20s %-19s %-15s\n", "Index", "Instance ID", "Name", "Public IP")
	fmt.Fprintf(os.Stderr, "%-5s %-20s %-19s %-15s\n", "-----", "--------------------", "-------------------", "---------------")

	for i, inst := range launchedInstances {
		ipDisplay := inst.PublicIP
		if ipDisplay == "" {
			ipDisplay = "(pending)"
		}
		fmt.Fprintf(os.Stderr, "%-5d %-20s %-19s %-15s\n", i, inst.InstanceID, inst.Name, ipDisplay)
	}

	fmt.Fprintf(os.Stderr, "\nManagement:\n")
	fmt.Fprintf(os.Stderr, "  • List:      spawn list --job-array-name %s\n", jobArrayName)
	fmt.Fprintf(os.Stderr, "  • Terminate: spawn terminate --job-array-name %s\n", jobArrayName)
	fmt.Fprintf(os.Stderr, "  • Extend:    spawn extend --job-array-name %s --ttl 4h\n", jobArrayName)

	if launchedInstances[0].PublicIP != "" {
		fmt.Fprintf(os.Stderr, "\nConnect to instances:\n")
		for i, inst := range launchedInstances {
			if inst.PublicIP != "" {
				sshCmd := plat.GetSSHCommand("ec2-user", inst.PublicIP)
				fmt.Fprintf(os.Stderr, "  [%d] %s\n", i, sshCmd)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n")

	return nil
}

// PeerInfo represents information about a peer instance in a job array
type PeerInfo struct {
	Index      int    `json:"index"`
	InstanceID string `json:"instance_id"`
	IP         string `json:"ip"`
	DNS        string `json:"dns"`
}

// collectPeerInfo collects peer information for all instances in a job array
func collectPeerInfo(ctx context.Context, awsClient *aws.Client, instances []*aws.LaunchResult, config *aws.LaunchConfig, jobArrayID string) ([]PeerInfo, error) {
	peers := make([]PeerInfo, 0, len(instances))

	// Get account ID for DNS namespace
	accountID, _, err := awsClient.GetCallerIdentityInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}

	// Convert account ID to base36
	accountNum, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse account ID: %w", err)
	}
	accountBase36 := strconv.FormatUint(accountNum, 36)

	for i, inst := range instances {
		// Generate DNS name for this instance
		// Format: {name}-{index}.{account-base36}.spore.host
		dnsName := fmt.Sprintf("%s-%d.%s.spore.host", jobArrayName, i, accountBase36)

		peer := PeerInfo{
			Index:      i,
			InstanceID: inst.InstanceID,
			IP:         inst.PublicIP,
			DNS:        dnsName,
		}
		peers = append(peers, peer)
	}

	return peers, nil
}

// updatePeerTags updates the spawn:job-array-peers tag on each instance with peer information
func updatePeerTags(ctx context.Context, awsClient *aws.Client, region string, instances []*aws.LaunchResult, peerInfo []PeerInfo) error {
	// Marshal peer info to JSON
	peerJSON, err := json.Marshal(peerInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal peer info: %w", err)
	}

	peerJSONStr := string(peerJSON)

	// Update tag on each instance
	for _, inst := range instances {
		err := awsClient.UpdateInstanceTags(ctx, region, inst.InstanceID, map[string]string{
			"spawn:job-array-peers": peerJSONStr,
		})
		if err != nil {
			return fmt.Errorf("failed to update tags on %s: %w", inst.InstanceID, err)
		}
	}

	return nil
}

// parseIAMRoleTags parses IAM role tags from key=value format
func parseIAMRoleTags(tags []string) map[string]string {
	result := make(map[string]string)
	for _, tagStr := range tags {
		parts := strings.SplitN(tagStr, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
