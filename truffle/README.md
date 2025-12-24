<p align="center">
  <img src="../assets/logo-light.png#gh-light-mode-only" alt="Mycelium Logo" width="500">
  <img src="../assets/logo-dark.png#gh-dark-mode-only" alt="Mycelium Logo" width="500">
</p>

# 🍄 Truffle - AWS EC2 Instance Type Region Finder

Truffle is a powerful CLI tool to discover which AWS regions and **availability zones** support specific EC2 instance types. Perfect for planning multi-region deployments, cost optimization, and infrastructure as code!

> **🎯 AZ-First Design:** Truffle includes availability zone information **by default** because AZ availability is more critical than region availability for production deployments.

## ✨ Features

- 🔍 **Wildcard Search**: Search for instance types using patterns like `m7i.*` or `*.large`
- 🌍 **Multi-Region**: Query all AWS regions or filter specific ones
- **📍 AZ-First**: Availability zone details included by default (can skip with `--skip-azs`)
- **💰 Spot Pricing**: Get real-time Spot instance prices with `truffle spot`
- **🎮 GPU Capacity**: NEW! Check On-Demand Capacity Reservations with `truffle capacity`
- **📦 Go Module**: Use Truffle as a library in your applications!
- 📊 **Multiple Outputs**: Table, JSON, YAML, or CSV formats
- 🎨 **Colorful CLI**: Beautiful, colorized terminal output
- ⚡ **Fast & Concurrent**: Parallel region queries for speed
- 🔧 **Rich Filtering**: Filter by architecture, vCPUs, memory, AZ count, price, capacity, and more
- 🎯 **AZ Command**: Dedicated `truffle az` command for AZ-centric searches
- 💵 **Spot Command**: Dedicated `truffle spot` command for Spot pricing and savings
- 🔋 **Capacity Command**: Dedicated `truffle capacity` command for ODCRs (critical for GPU/ML instances!)
- 🌍 **Multilingual**: 6 languages supported (en, es, fr, de, ja, pt)
- ♿ **Accessibility**: Screen reader support with --accessibility flag

## 🚀 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/truffle.git
cd truffle

# Install dependencies
go mod download

# Build
go build -o truffle

# Optional: Install globally
sudo mv truffle /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/yourusername/truffle@latest
```

## 📋 Prerequisites

- Go 1.22 or higher
- AWS credentials configured (via `~/.aws/credentials`, environment variables, or IAM role)
- AWS permissions: `ec2:DescribeRegions`, `ec2:DescribeInstanceTypes`, `ec2:DescribeInstanceTypeOfferings`

## 🎯 Quick Start

```bash
# Search for a specific instance type (includes AZ details by default)
truffle search m7i.large

# Search with wildcards - Graviton4 instances
truffle search "m8g.*"

# AZ-first search: find instances in at least 3 AZs
truffle az "m7i.*" --min-az-count 3

# NEW! Get Spot instance pricing
truffle spot m7i.large --sort-by-price

# Find cheap Spot instances (under $0.10/hr)
truffle spot "m8g.*" --max-price 0.10 --show-savings

# Search specific availability zones
truffle az m7i.large --az us-east-1a,us-east-1b,us-east-1c

# Fast region-only search (skip AZ lookup)
truffle search c7a.xlarge --skip-azs

# JSON output for scripting
truffle search "c8g.*" --output json
```

## 📍 Why Availability Zones Matter

**Region availability ≠ AZ availability!** 

A region may support an instance type, but not in all (or any!) of its availability zones. For production deployments:

- ✅ **Multi-AZ HA** requires 2-3+ AZs with the instance type
- ✅ **Capacity planning** needs AZ-level granularity
- ✅ **Spot instances** work best with AZ diversity
- ❌ **Region check alone** can lead to deployment failures

**Truffle includes AZ data by default** to prevent these issues. See [AZ_GUIDE.md](AZ_GUIDE.md) for detailed examples.

## 🌍 Internationalization

truffle supports multiple languages for a better user experience worldwide.

### Supported Languages

- 🇬🇧 **English** (en) - Default
- 🇪🇸 **Spanish** (es) - Español
- 🇫🇷 **French** (fr) - Français
- 🇩🇪 **German** (de) - Deutsch
- 🇯🇵 **Japanese** (ja) - 日本語
- 🇵🇹 **Portuguese** (pt) - Português

### Using Different Languages

```bash
# Spanish
truffle --lang es search m7i.large
truffle --lang es spot "m8g.*" --sort-by-price

# Japanese
truffle --lang ja search "c7i.*"
truffle --lang ja capacity --gpu-only

# French
truffle --lang fr --help
truffle --lang fr az m7i.large

# German
truffle --lang de search m7i.large --min-vcpu 4

# Portuguese
truffle --lang pt spot m7i.large --show-savings
```

### Environment Variable

Set your preferred language globally:

```bash
# Set language in your shell profile (~/.bashrc, ~/.zshrc)
export TRUFFLE_LANG=es

# Now all truffle commands use Spanish
truffle search m7i.large
truffle spot "m8g.*"
truffle --help
```

### Language Detection Priority

truffle detects language in this order:

1. `--lang` flag (highest priority)
2. `TRUFFLE_LANG` environment variable
3. Config file (`~/.truffle/config.yaml`)
4. System locale (`LANG`, `LC_ALL`)
5. Default to English

### What Gets Translated

All user-facing text is translated:

- ✅ Command descriptions and help text
- ✅ Table headers and column names
- ✅ Summary messages and statistics
- ✅ Progress indicators
- ✅ Success/warning/error messages
- ✅ Flag descriptions

**Technical terms stay in English** (AWS, EC2, vCPU, Spot, GPU, ODCR) for consistency with AWS documentation.

### Accessibility Features

Screen reader-friendly output:

```bash
# Disable emoji only
truffle --no-emoji search m7i.large

# Full accessibility mode (no emoji, no color, screen reader-friendly)
truffle --accessibility search m7i.large
truffle --accessibility spot "m8g.*"
```

**Accessibility mode:**
- Replaces emoji with text symbols (`[✓]`, `[✗]`, `[!]`, `[*]`)
- Disables color output
- Uses ASCII-only table borders
- Works with JAWS, NVDA, VoiceOver

### Examples in Different Languages

**Spanish Search:**
```bash
$ truffle --lang es search m7i.large

Buscando tipos de instancia que coincidan con: m7i.large

+---------------+-----------+-------+--------------+---------------+
| Tipo          | Región    | vCPUs | Memoria (GiB)| Arquitectura  |
+---------------+-----------+-------+--------------+---------------+
| m7i.large     | us-east-1 |     2 |          8.0 | x86_64        |
| m7i.large     | us-west-2 |     2 |          8.0 | x86_64        |
+---------------+-----------+-------+--------------+---------------+

Se encontraron 2 tipos de instancia en 2 región(es)
```

**Japanese Spot Pricing:**
```bash
$ truffle --lang ja spot m7i.large

m7i.largeのスポット価格を取得中...

+---------------+-----------+---------+----------------+
| インスタンス  | リージョン| 価格/時間| 節約率         |
+---------------+-----------+---------+----------------+
| m7i.large     | us-east-1 | $0.0331 | 68%            |
| m7i.large     | us-west-2 | $0.0344 | 67%            |
+---------------+-----------+---------+----------------+
```

**French Capacity Check:**
```bash
$ truffle --lang fr capacity --gpu-only

Vérification des réservations de capacité GPU...

+---------------+-----------+----------------+-------------------+
| Type Instance | Région    | Capacité Dispo | Capacité Totale   |
+---------------+-----------+----------------+-------------------+
| p5.48xlarge   | us-east-1 |             12 |                20 |
| g6.xlarge     | us-west-2 |              8 |                10 |
+---------------+-----------+----------------+-------------------+

Trouvé 2 réservations avec capacité disponible
```

## 📚 Usage

### Search Command

Search for instance types across AWS regions:

```bash
truffle search [pattern] [flags]
```

**Note:** Availability zone information is **included by default**. Use `--skip-azs` for faster region-level queries.

**Examples:**

```bash
# Find all m7i family instances (7th gen Intel)
truffle search "m7i.*"

# Find all xlarge instances
truffle search "*.xlarge"

# Find Graviton4 instances with specific requirements
truffle search "c8g.*" --min-vcpu 4 --min-memory 16

# Filter by ARM architecture (Graviton)
truffle search "*.xlarge" --architecture arm64

# Search in specific regions only
truffle search m7i.large --regions us-west-2,eu-central-1

# Get detailed AZ information for ML instances
truffle search inf2.xlarge --include-azs

# Output for scripting - latest AMD instances
truffle search "c7a.*" --output json | jq '.[] | select(.region == "us-east-1")'
```

**Flags:**

- `--skip-azs`: Skip AZ lookup for faster queries (AZs included by default)
- `--architecture string`: Filter by architecture (x86_64, arm64, i386)
- `--min-vcpu int`: Minimum number of vCPUs
- `--min-memory float`: Minimum memory in GiB
- `--family string`: Filter by instance family (e.g., m7i, c8g)
- `--timeout duration`: Timeout for AWS API calls (default: 5m)

### AZ Command (Availability Zone-First Search)

Search with an AZ-centric perspective:

```bash
truffle az [pattern] [flags]
```

**Examples:**

```bash
# Find which specific AZs have m7i.large
truffle az m7i.large

# Require at least 3 AZs per region (for high availability)
truffle az "m8g.*" --min-az-count 3

# Search in specific availability zones only
truffle az m7i.xlarge --az us-east-1a,us-east-1b,us-east-1c

# Find instances with minimum specs in 3+ AZs
truffle az "*" --min-vcpu 8 --min-memory 32 --min-az-count 3
```

**Flags:**

- `--az strings`: Filter by specific availability zones
- `--min-az-count int`: Minimum number of AZs required per region
- `--regions-only`: Show only regions meeting AZ count requirement

**Why use `truffle az`?**
- Shows AZ availability summary
- Sorts results by AZ count (most AZs first)
- Perfect for multi-AZ deployment planning
- Validates specific AZ requirements

### Spot Command (Spot Instance Pricing)

Get real-time Spot instance pricing:

```bash
truffle spot [pattern] [flags]
```

**Examples:**

```bash
# Get current Spot prices
truffle spot m7i.large

# Find Spot instances under $0.10/hour
truffle spot "m8g.*" --max-price 0.10

# Sort by price (cheapest first)
truffle spot "c7i.*" --sort-by-price

# Show savings vs On-Demand pricing
truffle spot m7i.xlarge --show-savings

# Find cheap Spot across multiple regions
truffle spot r7i.2xlarge --regions us-east-1,us-west-2 --max-price 0.15
```

**Flags:**

- `--max-price float`: Maximum Spot price per hour (USD)
- `--show-savings`: Show savings vs On-Demand pricing
- `--sort-by-price`: Sort by price (cheapest first)
- `--lookback-hours int`: Hours to look back for price history (default: 1)

**Why use `truffle spot`?**
- Find cheapest Spot instances across AZs
- Compare Spot prices in different regions
- Calculate potential savings (typically 50-90%!)
- Plan Spot fleet configurations
- Identify best AZs for Spot diversity

**Perfect for:**
- Kubernetes Spot node groups
- Batch processing workloads
- Development/testing environments
- Cost optimization initiatives

See [SPOT_GUIDE.md](SPOT_GUIDE.md) for comprehensive Spot instance strategies.

### Capacity Command (GPU/ML Instance Reservations) 🆕

Check On-Demand Capacity Reservations (ODCRs) - **critical for getting in-demand GPU instances**:

```bash
truffle capacity [flags]
```

**Examples:**

```bash
# Check all GPU capacity reservations
truffle capacity --gpu-only

# Find available GPU capacity
truffle capacity --gpu-only --available-only

# Specific GPU instances
truffle capacity --instance-types p5.48xlarge,g6.xlarge

# Minimum capacity needed
truffle capacity --gpu-only --min-capacity 10

# Multiple regions
truffle capacity --instance-types inf2.xlarge --regions us-east-1,us-west-2
```

**Flags:**

- `--instance-types strings`: Filter by instance types
- `--gpu-only`: Only show GPU/ML instances (p5, g6, inf2, trn1, etc.)
- `--available-only`: Only show reservations with available capacity
- `--min-capacity int`: Minimum available capacity required
- `--active-only`: Only show active reservations (default: true)

**Why use `truffle capacity`?**
- Find GPU instances (p5, g6, trn1, inf2) with guaranteed capacity
- No more "InsufficientInstanceCapacity" errors!
- Critical for ML training and inference workloads
- See which AZs have available capacity NOW
- Plan large-scale ML deployments

**Perfect for:**
- Large language model training (p5.48xlarge with H100s)
- ML inference scaling (inf2, g6 instances)
- Distributed training runs
- Production ML services requiring capacity guarantees

See [GPU_CAPACITY_GUIDE.md](GPU_CAPACITY_GUIDE.md) for comprehensive GPU/ML capacity strategies.

## 📦 Using as a Go Module

Truffle can be used as a library in your Go applications!

### Installation

```bash
go get github.com/yourusername/truffle
```

### Basic Usage

```go
import (
    "context"
    "regexp"
    "github.com/yourusername/truffle/pkg/aws"
)

func main() {
    ctx := context.Background()
    client, _ := aws.NewClient(ctx)
    
    // Search for instances
    regions, _ := client.GetAllRegions(ctx)
    pattern := regexp.MustCompile("^m7i\\.large$")
    results, _ := client.SearchInstanceTypes(ctx, regions, pattern, 
        aws.FilterOptions{IncludeAZs: true})
    
    // Get Spot pricing
    spotPrices, _ := client.GetSpotPricing(ctx, results, 
        aws.SpotOptions{MaxPrice: 0.10})
    
    // Check capacity reservations (GPU instances!)
    reservations, _ := client.GetCapacityReservations(ctx, regions,
        aws.CapacityReservationOptions{
            InstanceTypes: []string{"p5.48xlarge"},
            OnlyAvailable: true,
        })
}
```

### Use Cases

- **Terraform Providers**: Dynamic instance availability data
- **Kubernetes Operators**: GPU capacity validation
- **CI/CD Pipelines**: Pre-deployment validation
- **ML Platforms**: Capacity monitoring and alerting
- **Cost Optimization**: Automated Spot price analysis

See [MODULE_USAGE.md](MODULE_USAGE.md) for complete documentation and examples!

### List Command

List available instance types, families, or sizes:

```bash
# List all instance families
truffle list --family

# List all instance sizes
truffle list --sizes

# List all instance types in a region
truffle list --region us-east-1
```

### Global Flags

- `-o, --output string`: Output format (table, json, yaml, csv) (default: table)
- `--no-color`: Disable colorized output
- `-r, --regions strings`: Filter by specific regions (comma-separated)
- `-v, --verbose`: Enable verbose output
- `-h, --help`: Help for any command

## 🎨 Output Formats

### Table (Default)

Beautiful, human-readable table output with colors:

```
🍄 Found 3 instance type(s) across 2 region(s)

+---------------+-----------+-------+--------------+---------------+
| Instance Type | Region    | vCPUs | Memory (GiB) | Architecture  |
+---------------+-----------+-------+--------------+---------------+
| m5.large      | us-east-1 |     2 |          8.0 | x86_64       |
|               | us-west-2 |     2 |          8.0 | x86_64       |
+---------------+-----------+-------+--------------+---------------+
```

### JSON

Structured output for parsing and automation:

```bash
truffle search m5.large --output json
```

```json
[
  {
    "instance_type": "m5.large",
    "region": "us-east-1",
    "vcpus": 2,
    "memory_mib": 8192,
    "architecture": "x86_64",
    "instance_family": "m5"
  }
]
```

### YAML

Clean, readable format:

```bash
truffle search m5.large --output yaml
```

```yaml
- instance_type: m5.large
  region: us-east-1
  vcpus: 2
  memory_mib: 8192
  architecture: x86_64
  instance_family: m5
```

### CSV

For spreadsheet import:

```bash
truffle search m5.large --output csv > instances.csv
```

## 🔧 Configuration

### AWS Credentials

**Recommended: Use `aws login` (Super Easy!)** ⭐

```bash
# Modern AWS authentication - handles everything!
aws login
```

**That's it!** The `aws login` command:
- ✅ Authenticates automatically
- ✅ Handles MFA
- ✅ Works with SSO/IAM Identity Center
- ✅ Refreshes credentials automatically

**For SSO/IAM Identity Center:**
```bash
aws login --profile my-sso-profile
export AWS_PROFILE=my-sso-profile
truffle search m7i.large
```

**Alternative Authentication Methods:**

Truffle also supports traditional AWS SDK for Go v2 authentication:

1. **Environment Variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   export AWS_REGION=us-east-1
   ```

2. **AWS Credentials File** (`~/.aws/credentials`):
   ```ini
   [default]
   aws_access_key_id = your_access_key
   aws_secret_access_key = your_secret_key
   ```

3. **IAM Role** (when running on EC2/ECS/Lambda)

### Required AWS Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeRegions",
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeInstanceTypeOfferings"
      ],
      "Resource": "*"
    }
  ]
}
```

## 💡 Use Cases

### Multi-Region Deployments

```bash
# Find which regions support your desired instance type
truffle search m5.large --include-azs --output json > regions.json
```

### Cost Optimization

```bash
# Compare similar instance types across regions
truffle search "m5.large" --output csv
truffle search "m5a.large" --output csv  # AMD variant
```

### Terraform Planning

```bash
# Generate list of regions for Terraform variables
truffle search c5.xlarge --output json | jq -r '.[].region' | sort -u
```

### CI/CD Integration

```bash
#!/bin/bash
# Check if instance type is available in target region
AVAILABLE=$(truffle search m5.large --regions us-west-2 --output json | jq length)
if [ "$AVAILABLE" -eq 0 ]; then
  echo "Instance type not available in us-west-2"
  exit 1
fi
```

### Capacity Planning

```bash
# Find all instance types with at least 8 vCPUs and 32 GiB memory
truffle search "*" --min-vcpu 8 --min-memory 32 --output table
```

## 🛠️ Development

### Project Structure

```
truffle/
├── main.go              # Entry point
├── cmd/                 # CLI commands
│   ├── root.go         # Root command
│   ├── search.go       # Search command
│   └── list.go         # List command
├── pkg/
│   ├── aws/            # AWS client
│   │   └── client.go
│   └── output/         # Output formatters
│       └── printer.go
├── go.mod
└── README.md
```

### Building

```bash
# Build for current platform
go build -o truffle

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o truffle-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o truffle-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o truffle-windows-amd64.exe
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [tablewriter](https://github.com/olekukonko/tablewriter) - Table formatting
- [color](https://github.com/fatih/color) - Terminal colors

## 📮 Support

- 🐛 [Report a Bug](https://github.com/yourusername/truffle/issues)
- 💡 [Request a Feature](https://github.com/yourusername/truffle/issues)
- 📖 [Documentation](https://github.com/yourusername/truffle/wiki)

## 🔮 Roadmap

- [ ] Add caching for faster repeated queries
- [ ] Support for AWS China and GovCloud regions
- [ ] Instance type comparison feature
- [ ] Pricing information integration
- [ ] Interactive TUI mode
- [ ] Export to various IaC formats (Terraform, CloudFormation)

---

Made with ❤️ and ☕ by the AWS community
