// Using Truffle as a Go Module

Truffle can be used both as a CLI tool AND as a library in your own Go applications!

## 🚀 Installation

Add Truffle to your Go project:

```bash
go get github.com/yourusername/truffle
```

## 📦 Import Packages

```go
import (
    "github.com/yourusername/truffle/pkg/aws"
    "github.com/yourusername/truffle/pkg/output"
)
```

## 🎯 Basic Usage

### Initialize Client

```go
import (
    "context"
    "github.com/yourusername/truffle/pkg/aws"
)

func main() {
    ctx := context.Background()
    
    // Create AWS client (uses standard AWS SDK credentials)
    client, err := aws.NewClient(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Search for Instance Types

```go
import "regexp"

// Get regions
regions, _ := client.GetAllRegions(ctx)

// Create pattern matcher
pattern := regexp.MustCompile("^m7i\\.large$")

// Search with options
results, err := client.SearchInstanceTypes(ctx, regions, pattern, aws.FilterOptions{
    IncludeAZs:     true,
    Architecture:   "x86_64",
    MinVCPUs:       2,
    MinMemory:      8,
    InstanceFamily: "m7i",
})

// Use results
for _, r := range results {
    fmt.Printf("%s in %s: %d vCPUs, %.1f GiB\n",
        r.InstanceType, r.Region, r.VCPUs, float64(r.MemoryMiB)/1024)
}
```

### Get Spot Pricing

```go
// First, search for instances
results, _ := client.SearchInstanceTypes(ctx, regions, pattern, opts)

// Then get Spot pricing
spotResults, err := client.GetSpotPricing(ctx, results, aws.SpotOptions{
    MaxPrice:      0.10,
    ShowSavings:   true,
    LookbackHours: 24,
})

for _, spot := range spotResults {
    fmt.Printf("%s in %s: $%.4f/hr\n",
        spot.InstanceType, spot.AvailabilityZone, spot.SpotPrice)
}
```

### Check Capacity Reservations

```go
// Check for GPU instance reservations
reservations, err := client.GetCapacityReservations(ctx, regions, aws.CapacityReservationOptions{
    InstanceTypes: []string{"p5.48xlarge", "g6.xlarge"},
    OnlyAvailable: true,
    MinCapacity:   1,
})

for _, r := range reservations {
    fmt.Printf("%s: %d available in %s\n",
        r.InstanceType, r.AvailableCapacity, r.AvailabilityZone)
}
```

## 📊 Data Structures

### InstanceTypeResult

```go
type InstanceTypeResult struct {
    InstanceType   string   // e.g., "m7i.large"
    Region         string   // e.g., "us-east-1"
    AvailableAZs   []string // e.g., ["us-east-1a", "us-east-1b"]
    VCPUs          int32    // Number of vCPUs
    MemoryMiB      int64    // Memory in MiB
    Architecture   string   // e.g., "x86_64", "arm64"
    InstanceFamily string   // e.g., "m7i"
}
```

### SpotPriceResult

```go
type SpotPriceResult struct {
    InstanceType     string  // Instance type
    Region           string  // AWS region
    AvailabilityZone string  // Specific AZ
    SpotPrice        float64 // Current Spot price/hour
    OnDemandPrice    float64 // On-Demand price/hour
    SavingsPercent   float64 // Savings percentage
    Timestamp        string  // Price timestamp
}
```

### CapacityReservationResult

```go
type CapacityReservationResult struct {
    ReservationID     string  // ODCR ID
    InstanceType      string  // Instance type
    Region            string  // AWS region
    AvailabilityZone  string  // Specific AZ
    TotalCapacity     int32   // Total reserved
    AvailableCapacity int32   // Available now
    UsedCapacity      int32   // Currently used
    State             string  // "active", "expired", etc.
    EndDate           string  // Expiration date
    Platform          string  // "Linux/UNIX", etc.
}
```

## 🎨 Output Formatting

Use the output package for pretty printing:

```go
import "github.com/yourusername/truffle/pkg/output"

printer := output.NewPrinter(true) // true = use colors

// Print as table
printer.PrintTable(results, true)

// Print as JSON
printer.PrintJSON(results)

// Print as YAML
printer.PrintYAML(results)

// Print as CSV
printer.PrintCSV(results)
```

## 🔧 Advanced Examples

### Multi-Region Comparison

```go
// Compare instance availability across regions
regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
pattern := regexp.MustCompile("^m8g\\.xlarge$")

results, _ := client.SearchInstanceTypes(ctx, regions, pattern, aws.FilterOptions{
    IncludeAZs: true,
})

// Group by region
byRegion := make(map[string][]aws.InstanceTypeResult)
for _, r := range results {
    byRegion[r.Region] = append(byRegion[r.Region], r)
}

for region, instances := range byRegion {
    fmt.Printf("%s: %d AZs\n", region, len(instances[0].AvailableAZs))
}
```

### Find Cheapest Spot Instances

```go
// Search multiple instance types
patterns := []string{"m7i.large", "m7a.large", "m8g.large"}
var allResults []aws.InstanceTypeResult

for _, p := range patterns {
    pattern := regexp.MustCompile("^" + regexp.QuoteMeta(p) + "$")
    results, _ := client.SearchInstanceTypes(ctx, regions, pattern, opts)
    allResults = append(allResults, results...)
}

// Get Spot pricing
spotResults, _ := client.GetSpotPricing(ctx, allResults, aws.SpotOptions{
    MaxPrice: 0.10,
})

// Sort by price
sort.Slice(spotResults, func(i, j int) bool {
    return spotResults[i].SpotPrice < spotResults[j].SpotPrice
})

// Use cheapest
cheapest := spotResults[0]
fmt.Printf("Cheapest: %s in %s at $%.4f/hr\n",
    cheapest.InstanceType, cheapest.AvailabilityZone, cheapest.SpotPrice)
```

### GPU Capacity Monitoring

```go
// Monitor GPU capacity reservations
gpuTypes := []string{
    "p5.48xlarge",
    "g6.xlarge",
    "inf2.xlarge",
}

ticker := time.NewTicker(5 * time.Minute)
for range ticker.C {
    reservations, _ := client.GetCapacityReservations(ctx, regions, 
        aws.CapacityReservationOptions{
            InstanceTypes: gpuTypes,
            OnlyAvailable: true,
        })
    
    for _, r := range reservations {
        if r.AvailableCapacity > 0 {
            log.Printf("Available: %d x %s in %s",
                r.AvailableCapacity, r.InstanceType, r.AvailabilityZone)
        }
    }
}
```

### Custom Filtering

```go
// Complex filtering
results, _ := client.SearchInstanceTypes(ctx, regions, pattern, opts)

// Filter results
var filtered []aws.InstanceTypeResult
for _, r := range results {
    // Custom criteria
    if len(r.AvailableAZs) >= 3 && // At least 3 AZs
       r.VCPUs >= 4 &&             // At least 4 vCPUs
       r.Architecture == "arm64" { // Graviton only
        filtered = append(filtered, r)
    }
}
```

## 📝 Real-World Integration Examples

### Terraform Provider

```go
package provider

import (
    "context"
    "regexp"
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
    "github.com/yourusername/truffle/pkg/aws"
)

func dataSourceInstanceAvailability() *schema.Resource {
    return &schema.Resource{
        ReadContext: dataSourceInstanceAvailabilityRead,
        Schema: map[string]*schema.Schema{
            "instance_type": {
                Type:     schema.TypeString,
                Required: true,
            },
            "region": {
                Type:     schema.TypeString,
                Required: true,
            },
            "availability_zones": {
                Type:     schema.TypeList,
                Computed: true,
                Elem:     &schema.Schema{Type: schema.TypeString},
            },
        },
    }
}

func dataSourceInstanceAvailabilityRead(ctx context.Context, d *schema.ResourceData, meta interface{}) error {
    client, _ := aws.NewClient(ctx)
    
    instanceType := d.Get("instance_type").(string)
    region := d.Get("region").(string)
    
    pattern := regexp.MustCompile("^" + regexp.QuoteMeta(instanceType) + "$")
    results, err := client.SearchInstanceTypes(ctx, []string{region}, pattern, 
        aws.FilterOptions{IncludeAZs: true})
    
    if err != nil {
        return err
    }
    
    if len(results) > 0 {
        d.Set("availability_zones", results[0].AvailableAZs)
    }
    
    d.SetId(instanceType + "-" + region)
    return nil
}
```

### Kubernetes Operator

```go
package controllers

import (
    "context"
    "regexp"
    "github.com/yourusername/truffle/pkg/aws"
)

type InstanceReconciler struct {
    AWSClient *aws.Client
}

func (r *InstanceReconciler) findBestSpotInstance(ctx context.Context, reqs InstanceRequirements) (string, string, error) {
    pattern := regexp.MustCompile(reqs.InstancePattern)
    
    // Search for instances
    results, err := r.AWSClient.SearchInstanceTypes(ctx, reqs.Regions, pattern, 
        aws.FilterOptions{
            MinVCPUs:  reqs.MinVCPUs,
            MinMemory: reqs.MinMemory,
        })
    if err != nil {
        return "", "", err
    }
    
    // Get Spot pricing
    spots, err := r.AWSClient.GetSpotPricing(ctx, results, 
        aws.SpotOptions{
            MaxPrice: reqs.MaxPrice,
        })
    if err != nil {
        return "", "", err
    }
    
    // Return cheapest
    if len(spots) > 0 {
        return spots[0].InstanceType, spots[0].AvailabilityZone, nil
    }
    
    return "", "", fmt.Errorf("no suitable instances found")
}
```

### CI/CD Pipeline

```go
package main

import (
    "context"
    "fmt"
    "os"
    "regexp"
    "github.com/yourusername/truffle/pkg/aws"
)

// Pre-deployment validation
func validateInstanceAvailability() error {
    ctx := context.Background()
    client, _ := aws.NewClient(ctx)
    
    requiredType := os.Getenv("INSTANCE_TYPE")
    targetRegion := os.Getenv("AWS_REGION")
    minAZs := 3
    
    pattern := regexp.MustCompile("^" + regexp.QuoteMeta(requiredType) + "$")
    results, err := client.SearchInstanceTypes(ctx, []string{targetRegion}, pattern, 
        aws.FilterOptions{IncludeAZs: true})
    
    if err != nil {
        return err
    }
    
    if len(results) == 0 {
        return fmt.Errorf("instance type %s not available in %s", 
            requiredType, targetRegion)
    }
    
    if len(results[0].AvailableAZs) < minAZs {
        return fmt.Errorf("insufficient AZs: need %d, have %d",
            minAZs, len(results[0].AvailableAZs))
    }
    
    fmt.Printf("✅ Validation passed: %s available in %d AZs\n",
        requiredType, len(results[0].AvailableAZs))
    return nil
}

func main() {
    if err := validateInstanceAvailability(); err != nil {
        fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
        os.Exit(1)
    }
}
```

## 📚 Complete Example Projects

See the `examples/` directory for full working examples:

- `examples/basic/` - Basic instance search
- `examples/spot-pricing/` - Spot price comparison
- `examples/gpu-capacity/` - GPU capacity monitoring
- `examples/multi-region/` - Multi-region analysis
- `examples/terraform-datasource/` - Terraform integration
- `examples/k8s-operator/` - Kubernetes operator

## 🔗 API Reference

Full API documentation: https://pkg.go.dev/github.com/yourusername/truffle

## 💡 Best Practices

1. **Always use context** for cancellation and timeouts
2. **Handle errors** - AWS APIs can fail
3. **Cache results** when appropriate
4. **Use goroutines** for parallel operations
5. **Filter early** to reduce data transfer

## 🐛 Error Handling

```go
results, err := client.SearchInstanceTypes(ctx, regions, pattern, opts)
if err != nil {
    // Check for specific errors
    if ctx.Err() == context.DeadlineExceeded {
        log.Fatal("Operation timed out")
    }
    log.Fatalf("Search failed: %v", err)
}
```

## 🚀 Performance Tips

```go
// Use specific regions when possible
regions := []string{"us-east-1"} // Faster than all regions

// Skip AZ lookup if not needed
opts := aws.FilterOptions{
    IncludeAZs: false, // Faster queries
}

// Use timeout contexts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

**Ready to integrate Truffle into your applications!** 🎯
