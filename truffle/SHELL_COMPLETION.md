# Shell Completion for Truffle CLI

Truffle CLI includes intelligent shell completion support for bash, zsh, fish, and PowerShell. Completions provide:

- **Command completion** - Tab-complete truffle commands (search, spot, az, capacity, list)
- **Flag completion** - Tab-complete command flags (--regions, --output, --architecture, etc.)
- **Instance type completion** - Tab-complete common EC2 instance types with specs
- **Region completion** - Tab-complete all AWS regions with descriptions
- **Architecture completion** - Tab-complete CPU architectures (x86_64, arm64, i386)
- **Instance family completion** - Tab-complete instance families (m7i, c7i, g5, etc.)
- **Output format completion** - Tab-complete output formats (table, json, yaml, csv)

## Quick Start

### Bash

**Install completion (one-time setup):**
```bash
# Generate completion script
truffle completion bash > ~/.truffle-completion.bash

# Add to your .bashrc
echo 'source ~/.truffle-completion.bash' >> ~/.bashrc

# Reload shell
source ~/.bashrc
```

### Zsh

**Install completion (one-time setup):**
```bash
# Generate completion script
truffle completion zsh > ~/.truffle-completion.zsh

# Add to your .zshrc
echo 'source ~/.truffle-completion.zsh' >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

### Fish

**Install completion (one-time setup):**
```bash
# Generate and install completion
truffle completion fish > ~/.config/fish/completions/truffle.fish
```

### PowerShell

**Install completion (one-time setup):**
```powershell
# Generate completion script
truffle completion powershell | Out-File -FilePath $PROFILE\..\truffle.ps1 -Encoding utf8

# Add to PowerShell profile
Add-Content -Path $PROFILE -Value ". $PSScriptRoot\truffle.ps1"
```

## Usage Examples

### Complete Commands
```bash
$ truffle <TAB>
az         capacity   completion list       quotas     search     spot       version
```

### Complete Instance Types
```bash
$ truffle search <TAB>
m7i.large      General purpose, 2 vCPU, 8 GB RAM
m7i.xlarge     General purpose, 4 vCPU, 16 GB RAM
m8g.large      Graviton4, 2 vCPU, 8 GB RAM
c7i.large      Compute optimized, 2 vCPU, 4 GB RAM
g5.xlarge      GPU (1x A10G), 4 vCPU, 16 GB RAM
p5.48xlarge    GPU (8x H100), 192 vCPU, 2048 GB RAM
m7i.*          All m7i instance types
*.large        All large instance types

$ truffle search m7i.<TAB>
m7i.large    General purpose, 2 vCPU, 8 GB RAM
m7i.xlarge   General purpose, 4 vCPU, 16 GB RAM
m7i.2xlarge  General purpose, 8 vCPU, 32 GB RAM
m7i.4xlarge  General purpose, 16 vCPU, 64 GB RAM
m7i.*        All m7i instance types
```

### Complete Regions
```bash
$ truffle search m7i.large --regions <TAB>
us-east-1      US East (N. Virginia)
us-east-2      US East (Ohio)
us-west-1      US West (N. California)
us-west-2      US West (Oregon)
eu-west-1      Europe (Ireland)
ap-northeast-1 Asia Pacific (Tokyo)
...

$ truffle search m7i.large --regions us-<TAB>
us-east-1  US East (N. Virginia)
us-east-2  US East (Ohio)
us-west-1  US West (N. California)
us-west-2  US West (Oregon)
```

### Complete Output Formats
```bash
$ truffle search m7i.large --output <TAB>
table  json  yaml  csv
```

### Complete Architectures
```bash
$ truffle search "m*" --architecture <TAB>
x86_64  Intel/AMD 64-bit
arm64   AWS Graviton (ARM 64-bit)
i386    32-bit x86
```

### Complete Instance Families
```bash
$ truffle search --family <TAB>
m7i   General Purpose (7th gen Intel)
m7a   General Purpose (7th gen AMD)
m8g   General Purpose (Graviton4)
c7i   Compute Optimized (7th gen Intel)
c8g   Compute Optimized (Graviton4)
g5    GPU (NVIDIA A10G)
g6    GPU (NVIDIA L4)
p5    GPU (NVIDIA H100)
...
```

### Complete Wildcard Patterns
```bash
$ truffle spot <TAB>
m7i.*     All m7i instance types
c7i.*     All c7i instance types
g5.*      All g5 GPU types
*.large   All large instance types
*.xlarge  All xlarge instance types
```

## Completion Features by Command

| Command | Argument Completion | Flag Completion |
|---------|---------------------|-----------------|
| `truffle search` | Instance types | `--architecture`, `--family` |
| `truffle spot` | Instance types | - |
| `truffle az` | Instance types | - |
| `truffle capacity` | - | - |
| `truffle list` | - | - |
| **All commands** | - | `--regions`, `--output` |

## Benefits

### Discover Instance Types
- See all available instance types with descriptions
- Learn about GPU instances (g5, g6, p3, p4d, p5)
- Understand Graviton options (m8g, c8g, r8g)
- Find the latest generation instances (7th/8th gen)

### Explore Regions
- Complete all 29 AWS regions
- See region descriptions (e.g., "Europe (Stockholm)")
- Quickly specify multi-region searches

### Wildcard Patterns
- Tab-complete wildcard patterns like `m7i.*` or `*.large`
- Discover pattern-based searching capabilities
- Speed up searches for instance families

### Learn As You Go
- Descriptions show key specs (vCPU, RAM, GPU)
- Identify instance categories (compute, memory, GPU)
- Compare instance types side by side

## How It Works

All completions are static (work offline):
- No AWS API calls required
- Instant response time
- Works without internet connection
- Pre-configured with common instance types and all regions

## Troubleshooting

### Completion doesn't work
```bash
# Verify truffle is in PATH
which truffle

# Re-generate completion
truffle completion bash > ~/.truffle-completion.bash
source ~/.truffle-completion.bash
```

### Instance types don't show up
```bash
# Verify completion is installed
type _truffle  # bash
type _truffle  # zsh
```

## Advanced: Custom Completions

You can customize completion behavior by editing `cmd/completion.go`:

### Add custom instance types
```go
func completeInstanceType(...) ([]string, cobra.ShellCompDirective) {
	instanceTypes := []string{
		"your-custom-type\tYour Description",
		// Add your types here
	}
	// ...
}
```

### Add custom regions
```go
func completeRegion(...) ([]string, cobra.ShellCompDirective) {
	regions := []string{
		"your-region-1\tYour Region Description",
		// Add your regions here
	}
	// ...
}
```

## Uninstalling

### Bash
```bash
# Remove from .bashrc
sed -i '/truffle-completion/d' ~/.bashrc

# Remove script
rm ~/.truffle-completion.bash
```

### Zsh
```bash
# Remove from .zshrc
sed -i '/truffle-completion/d' ~/.zshrc

# Remove script
rm ~/.truffle-completion.zsh
```

### Fish
```bash
rm ~/.config/fish/completions/truffle.fish
```

### PowerShell
```powershell
# Remove from profile
# Edit $PROFILE and remove the truffle.ps1 line

# Remove script
Remove-Item "$PSScriptRoot\truffle.ps1"
```

## See Also

- [Cobra Shell Completion](https://github.com/spf13/cobra/blob/main/shell_completions.md) - Underlying completion framework
- [spawn Shell Completion](../spawn/SHELL_COMPLETION.md) - Completion for spawn CLI
