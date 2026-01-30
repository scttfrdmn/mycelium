package find

import (
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		wantVendors     []string
		wantProcs       []string
		wantGPUs        []string
		wantSizes       []string
		wantVCPU        int
		wantMemory      float64
		wantArch        string
		wantNetworkGbps int
		wantEFA         bool
		wantErr         bool
	}{
		{
			name:        "single vendor",
			query:       "intel",
			wantVendors: []string{"intel"},
		},
		{
			name:        "single vendor - graviton",
			query:       "graviton",
			wantVendors: []string{"aws"},
		},
		{
			name:      "processor code name",
			query:     "ice lake",
			wantProcs: []string{"ice lake"},
		},
		{
			name:      "processor code name - milan",
			query:     "milan",
			wantProcs: []string{"milan"},
		},
		{
			name:      "multi-word processor",
			query:     "sapphire rapids",
			wantProcs: []string{"sapphire rapids"},
		},
		{
			name:     "gpu type",
			query:    "a100",
			wantGPUs: []string{"a100"},
		},
		{
			name:     "gpu alias",
			query:    "inf",
			wantGPUs: []string{"inferentia"},
		},
		{
			name:      "size category",
			query:     "large",
			wantSizes: []string{"large"},
		},
		{
			name:       "vendor with vcpu",
			query:      "amd 16 cores",
			wantVendors: []string{"amd"},
			wantVCPU:   16,
		},
		{
			name:       "vendor with memory",
			query:      "graviton 32gb",
			wantVendors: []string{"aws"},
			wantMemory: 32,
		},
		{
			name:       "combined specs",
			query:      "amd 16 cores 64gb",
			wantVendors: []string{"amd"},
			wantVCPU:   16,
			wantMemory: 64,
		},
		{
			name:        "vendor and size",
			query:       "graviton large",
			wantVendors: []string{"aws"},
			wantSizes:   []string{"large"},
		},
		{
			name:      "processor with specs",
			query:     "milan 64 cores",
			wantProcs: []string{"milan"},
			wantVCPU:  64,
		},
		{
			name:       "architecture",
			query:      "arm64",
			wantArch:   "arm64",
		},
		{
			name:       "x86_64 architecture",
			query:      "x86_64",
			wantArch:   "x86_64",
		},
		{
			name:     "multi-word gpu",
			query:    "radeon pro v520",
			wantGPUs: []string{"radeon pro v520"},
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
		{
			name:       "vcpu with different unit",
			query:      "8 vcpus",
			wantVCPU:   8,
		},
		{
			name:       "memory with gib",
			query:      "32gib",
			wantMemory: 32,
		},
		{
			name:       "efa network",
			query:      "efa",
			wantEFA:    true,
		},
		{
			name:          "100gbps network",
			query:         "100gbps",
			wantNetworkGbps: 100,
		},
		{
			name:          "efa with graviton",
			query:         "efa graviton",
			wantVendors:   []string{"aws"},
			wantEFA:       true,
		},
		{
			name:          "h100 with efa",
			query:         "h100 efa",
			wantGPUs:      []string{"h100"},
			wantEFA:       true,
		},
		{
			name:          "100g alias",
			query:         "100g",
			wantNetworkGbps: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !stringSlicesEqual(got.Vendors, tt.wantVendors) {
				t.Errorf("Vendors = %v, want %v", got.Vendors, tt.wantVendors)
			}

			if !stringSlicesEqual(got.Processors, tt.wantProcs) {
				t.Errorf("Processors = %v, want %v", got.Processors, tt.wantProcs)
			}

			if !stringSlicesEqual(got.GPUs, tt.wantGPUs) {
				t.Errorf("GPUs = %v, want %v", got.GPUs, tt.wantGPUs)
			}

			if !stringSlicesEqual(got.Sizes, tt.wantSizes) {
				t.Errorf("Sizes = %v, want %v", got.Sizes, tt.wantSizes)
			}

			if got.MinVCPU != tt.wantVCPU {
				t.Errorf("MinVCPU = %v, want %v", got.MinVCPU, tt.wantVCPU)
			}

			if got.MinMemory != tt.wantMemory {
				t.Errorf("MinMemory = %v, want %v", got.MinMemory, tt.wantMemory)
			}

			if got.Architecture != tt.wantArch {
				t.Errorf("Architecture = %v, want %v", got.Architecture, tt.wantArch)
			}

			if got.MinNetworkGbps != tt.wantNetworkGbps {
				t.Errorf("MinNetworkGbps = %v, want %v", got.MinNetworkGbps, tt.wantNetworkGbps)
			}

			if got.RequireEFA != tt.wantEFA {
				t.Errorf("RequireEFA = %v, want %v", got.RequireEFA, tt.wantEFA)
			}
		})
	}
}

func TestParsedQuery_ResolveInstanceFamilies(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantMin   int
		wantFamilies []string
	}{
		{
			name:      "graviton",
			query:     "graviton",
			wantMin:   10,
		},
		{
			name:      "ice lake",
			query:     "ice lake",
			wantMin:   3,
			wantFamilies: []string{"m6i", "c6i", "r6i"},
		},
		{
			name:      "a100",
			query:     "a100",
			wantMin:   1,
			wantFamilies: []string{"p4d", "p4de"},
		},
		{
			name:      "intel vendor",
			query:     "intel",
			wantMin:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			families := pq.ResolveInstanceFamilies()

			if len(families) < tt.wantMin {
				t.Errorf("ResolveInstanceFamilies() returned %d families, want >= %d",
					len(families), tt.wantMin)
			}

			if len(tt.wantFamilies) > 0 {
				for _, wantFamily := range tt.wantFamilies {
					found := false
					for _, family := range families {
						if family == wantFamily {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ResolveInstanceFamilies() missing family %q in %v",
							wantFamily, families)
					}
				}
			}
		})
	}
}

func TestParsedQuery_DeriveArchitecture(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantArch string
	}{
		{
			name:     "graviton",
			query:    "graviton",
			wantArch: "arm64",
		},
		{
			name:     "intel",
			query:    "intel",
			wantArch: "x86_64",
		},
		{
			name:     "ice lake",
			query:    "ice lake",
			wantArch: "x86_64",
		},
		{
			name:     "milan",
			query:    "milan",
			wantArch: "x86_64",
		},
		{
			name:     "explicit arm64",
			query:    "arm64",
			wantArch: "arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			arch := pq.DeriveArchitecture()
			if arch != tt.wantArch {
				t.Errorf("DeriveArchitecture() = %v, want %v", arch, tt.wantArch)
			}
		})
	}
}

func TestParsedQuery_BuildSizePattern(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantContains []string
	}{
		{
			name:         "large",
			query:        "large",
			wantContains: []string{"2xlarge", "4xlarge"},
		},
		{
			name:         "small",
			query:        "small",
			wantContains: []string{"small", "medium"},
		},
		{
			name:         "no size",
			query:        "intel",
			wantContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			pattern := pq.BuildSizePattern()

			if len(tt.wantContains) == 0 {
				if pattern != ".*" {
					t.Errorf("BuildSizePattern() = %v, want .*", pattern)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(pattern, want) {
					t.Errorf("BuildSizePattern() = %v, should contain %v", pattern, want)
				}
			}
		})
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
