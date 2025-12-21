package wizard

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yourusername/spawn/pkg/aws"
	"github.com/yourusername/spawn/pkg/platform"
)

// Wizard guides users through spawning an instance
type Wizard struct {
	scanner  *bufio.Scanner
	platform *platform.Platform
	config   *aws.LaunchConfig
}

// NewWizard creates a new interactive wizard
func NewWizard(plat *platform.Platform) *Wizard {
	return &Wizard{
		scanner:  bufio.NewScanner(os.Stdin),
		platform: plat,
		config: &aws.LaunchConfig{
			Tags: make(map[string]string),
		},
	}
}

// Run executes the wizard and returns the configuration
func (w *Wizard) Run(ctx context.Context) (*aws.LaunchConfig, error) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  🧙 spawn Setup Wizard                                ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("I'll help you launch an AWS EC2 instance!")
	fmt.Println("Press Enter to use the default shown in [brackets]")
	fmt.Println()

	// Step 1: Instance Type
	if err := w.askInstanceType(); err != nil {
		return nil, err
	}

	// Step 2: Region
	if err := w.askRegion(); err != nil {
		return nil, err
	}

	// Step 3: Spot or On-Demand
	if err := w.askSpot(); err != nil {
		return nil, err
	}

	// Step 4: Auto-termination
	if err := w.askAutoTerminate(); err != nil {
		return nil, err
	}

	// Step 5: SSH Key
	if err := w.askSSHKey(); err != nil {
		return nil, err
	}

	// Step 6: Name (optional)
	if err := w.askName(); err != nil {
		return nil, err
	}

	// Step 7: Summary and confirm
	return w.confirm()
}

func (w *Wizard) askInstanceType() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📦 Step 1 of 6: Choose Instance Type")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Common choices:")
	fmt.Println()
	fmt.Println("  💻 Development & Testing:")
	fmt.Println("     • t3.medium     - $0.04/hr  (2 vCPU, 4 GB)")
	fmt.Println("     • t3.large      - $0.08/hr  (2 vCPU, 8 GB)")
	fmt.Println()
	fmt.Println("  ⚙️  General Purpose:")
	fmt.Println("     • m7i.large     - $0.10/hr  (2 vCPU, 8 GB)")
	fmt.Println("     • m7i.xlarge    - $0.20/hr  (4 vCPU, 16 GB)")
	fmt.Println()
	fmt.Println("  🚀 Compute Optimized:")
	fmt.Println("     • c7i.xlarge    - $0.17/hr  (4 vCPU, 8 GB)")
	fmt.Println("     • c7i.2xlarge   - $0.34/hr  (8 vCPU, 16 GB)")
	fmt.Println()
	fmt.Println("  🎮 GPU / ML Training:")
	fmt.Println("     • g6.xlarge     - $1.21/hr  (L4 GPU, 16 GB)")
	fmt.Println("     • p5.48xlarge   - $98/hr    (8x H100 GPU)")
	fmt.Println()
	fmt.Print("Instance type [t3.medium]: ")

	instanceType := w.readLine()
	if instanceType == "" {
		instanceType = "t3.medium"
	}

	w.config.InstanceType = instanceType

	// Detect characteristics
	arch := aws.DetectArchitecture(instanceType)
	gpu := aws.DetectGPUInstance(instanceType)

	fmt.Println()
	fmt.Printf("  ✅ Detected: %s", arch)
	if gpu {
		fmt.Print(", GPU-enabled")
	}
	fmt.Println()
	fmt.Println()

	return nil
}

func (w *Wizard) askRegion() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🌍 Step 2 of 6: Choose AWS Region")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Common regions:")
	fmt.Println()
	fmt.Println("  🇺🇸 United States:")
	fmt.Println("     • us-east-1     (Virginia)   [Cheapest, most services]")
	fmt.Println("     • us-west-2     (Oregon)     [Good for west coast]")
	fmt.Println()
	fmt.Println("  🇪🇺 Europe:")
	fmt.Println("     • eu-west-1     (Ireland)")
	fmt.Println("     • eu-central-1  (Frankfurt)")
	fmt.Println()
	fmt.Println("  🌏 Asia Pacific:")
	fmt.Println("     • ap-northeast-1  (Tokyo)")
	fmt.Println("     • ap-southeast-1  (Singapore)")
	fmt.Println()

	defaultRegion := "us-east-1"
	fmt.Printf("Region [%s]: ", defaultRegion)

	region := w.readLine()
	if region == "" {
		region = defaultRegion
	}

	w.config.Region = region
	fmt.Println()

	return nil
}

func (w *Wizard) askSpot() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💰 Step 3 of 6: Spot or On-Demand?")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💡 Spot instances are up to 70% cheaper but can be interrupted.")
	fmt.Println()
	fmt.Println("   ✅ Good for: Development, testing, fault-tolerant workloads")
	fmt.Println("   ⚠️  Not for: Production databases, critical services")
	fmt.Println()
	fmt.Print("Use Spot instances? [y/N]: ")

	response := strings.ToLower(w.readLine())
	if response == "y" || response == "yes" {
		w.config.Spot = true
		fmt.Println()
		fmt.Println("  ✅ Using Spot instances (save up to 70%!)")
	} else {
		fmt.Println()
		fmt.Println("  ✅ Using On-Demand (reliable, no interruptions)")
	}
	fmt.Println()

	return nil
}

func (w *Wizard) askAutoTerminate() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⏱️  Step 4 of 6: Auto-Termination (Prevent Surprise Bills!)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("The instance will automatically terminate to prevent charges:")
	fmt.Println()
	fmt.Println("  1️⃣  After time limit (TTL) - e.g., 8h work day")
	fmt.Println("  2️⃣  When idle (no CPU/network) - e.g., forgot to close")
	fmt.Println("  3️⃣  Both (recommended) - whichever comes first")
	fmt.Println("  4️⃣  Manual only - I'll terminate it myself")
	fmt.Println()
	fmt.Print("Choice [3]: ")

	choice := w.readLine()
	if choice == "" {
		choice = "3"
	}

	fmt.Println()

	switch choice {
	case "1":
		fmt.Print("Time limit (e.g., 8h, 24h, 7d) [8h]: ")
		ttl := w.readLine()
		if ttl == "" {
			ttl = "8h"
		}
		w.config.TTL = ttl
		fmt.Printf("  ✅ Will terminate after %s\n", ttl)

	case "2":
		fmt.Print("Idle timeout (e.g., 30m, 1h, 2h) [1h]: ")
		idle := w.readLine()
		if idle == "" {
			idle = "1h"
		}
		w.config.IdleTimeout = idle
		fmt.Printf("  ✅ Will terminate if idle for %s\n", idle)

	case "3":
		fmt.Print("Time limit [8h]: ")
		ttl := w.readLine()
		if ttl == "" {
			ttl = "8h"
		}
		w.config.TTL = ttl

		fmt.Print("Idle timeout [1h]: ")
		idle := w.readLine()
		if idle == "" {
			idle = "1h"
		}
		w.config.IdleTimeout = idle
		fmt.Printf("  ✅ TTL: %s, Idle: %s (whichever comes first)\n", ttl, idle)

	case "4":
		fmt.Println("  ⚠️  Remember to terminate manually to avoid charges!")
		fmt.Println("      Run: aws ec2 terminate-instances --instance-ids i-xxx")
	}

	fmt.Println()
	return nil
}

func (w *Wizard) askSSHKey() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔑 Step 5 of 6: SSH Key Setup")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check for existing key
	if w.platform.HasSSHKey() {
		fmt.Printf("✅ Found existing SSH key: %s\n", w.platform.SSHKeyPath)
		fmt.Println("   Will use this key for connecting to your instance")
		w.config.KeyName = "default-ssh-key"
	} else {
		fmt.Printf("⚠️  No SSH key found at: %s\n", w.platform.SSHKeyPath)
		fmt.Println()
		fmt.Println("   An SSH key is required to connect to your instance.")
		fmt.Println()
		fmt.Print("   Create one now? [Y/n]: ")

		response := strings.ToLower(w.readLine())
		if response == "" || response == "y" || response == "yes" {
			fmt.Println()
			fmt.Println("  🔧 Creating SSH key...")

			if err := w.platform.CreateSSHKey(); err != nil {
				return fmt.Errorf("failed to create SSH key: %w", err)
			}

			fmt.Printf("  ✅ SSH key created at: %s\n", w.platform.SSHKeyPath)
			w.config.KeyName = "default-ssh-key"
		} else {
			return fmt.Errorf("SSH key required to connect to instance")
		}
	}

	fmt.Println()
	return nil
}

func (w *Wizard) askName() error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🏷️  Step 6 of 6: Instance Name (Optional)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Print("Name for this instance [leave blank to skip]: ")

	name := w.readLine()
	if name != "" {
		w.config.Name = name
		fmt.Printf("  ✅ Instance will be named: %s\n", name)
	}

	fmt.Println()
	return nil
}

func (w *Wizard) confirm() (*aws.LaunchConfig, error) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  📋 Configuration Summary                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Display configuration
	fmt.Println("You're about to launch:")
	fmt.Println()
	fmt.Printf("  Instance Type:  %s\n", w.config.InstanceType)
	fmt.Printf("  Region:         %s\n", w.config.Region)

	if w.config.Name != "" {
		fmt.Printf("  Name:           %s\n", w.config.Name)
	}

	if w.config.Spot {
		fmt.Println("  Type:           Spot (up to 70% cheaper)")
	} else {
		fmt.Println("  Type:           On-Demand (reliable)")
	}

	if w.config.TTL != "" {
		fmt.Printf("  Time Limit:     %s\n", w.config.TTL)
	}

	if w.config.IdleTimeout != "" {
		fmt.Printf("  Idle Timeout:   %s\n", w.config.IdleTimeout)
	}

	// Estimate cost
	fmt.Println()
	cost := estimateCost(w.config.InstanceType, w.config.Spot)
	fmt.Printf("💰 Estimated cost: ~$%.2f/hour", cost)

	if w.config.Spot {
		onDemandCost := estimateCost(w.config.InstanceType, false)
		savings := ((onDemandCost - cost) / onDemandCost) * 100
		fmt.Printf(" (%.0f%% savings vs On-Demand)", savings)
	}
	fmt.Println()

	if w.config.TTL != "" {
		duration, _ := time.ParseDuration(w.config.TTL)
		hours := duration.Hours()
		totalCost := cost * hours
		fmt.Printf("   Total for %s: ~$%.2f\n", w.config.TTL, totalCost)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Print("🚀 Launch instance? [Y/n]: ")

	response := strings.ToLower(w.readLine())
	if response == "" || response == "y" || response == "yes" {
		fmt.Println()
		return w.config, nil
	}

	return nil, fmt.Errorf("cancelled by user")
}

func (w *Wizard) readLine() string {
	w.scanner.Scan()
	return strings.TrimSpace(w.scanner.Text())
}

// estimateCost estimates hourly cost for an instance type
func estimateCost(instanceType string, spot bool) float64 {
	// Rough estimates - in production, query AWS Pricing API
	costs := map[string]float64{
		"t3.micro":    0.01,
		"t3.small":    0.02,
		"t3.medium":   0.04,
		"t3.large":    0.08,
		"m7i.large":   0.10,
		"m7i.xlarge":  0.20,
		"c7i.xlarge":  0.17,
		"c7i.2xlarge": 0.34,
		"g6.xlarge":   1.21,
		"p5.48xlarge": 98.0,
	}

	cost, ok := costs[instanceType]
	if !ok {
		// Default estimate if not found
		cost = 0.10
	}

	if spot {
		// Spot is typically 60-70% cheaper
		cost = cost * 0.35
	}

	return cost
}
