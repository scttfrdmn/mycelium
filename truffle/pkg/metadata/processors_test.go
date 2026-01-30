package metadata

import (
	"testing"
)

func TestProcessorDatabase(t *testing.T) {
	tests := []struct {
		name         string
		codeName     string
		wantVendor   string
		wantArch     string
		wantFamilies int // minimum expected families
	}{
		{
			name:         "ice lake",
			codeName:     "ice lake",
			wantVendor:   "intel",
			wantArch:     "x86_64",
			wantFamilies: 3,
		},
		{
			name:         "milan",
			codeName:     "milan",
			wantVendor:   "amd",
			wantArch:     "x86_64",
			wantFamilies: 3,
		},
		{
			name:         "graviton2",
			codeName:     "graviton2",
			wantVendor:   "aws",
			wantArch:     "arm64",
			wantFamilies: 5,
		},
		{
			name:         "sapphire rapids",
			codeName:     "sapphire rapids",
			wantVendor:   "intel",
			wantArch:     "x86_64",
			wantFamilies: 3,
		},
		{
			name:         "genoa",
			codeName:     "genoa",
			wantVendor:   "amd",
			wantArch:     "x86_64",
			wantFamilies: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ProcessorDatabase[tt.codeName]
			if !ok {
				t.Fatalf("processor %q not found in database", tt.codeName)
			}

			if info.Vendor != tt.wantVendor {
				t.Errorf("vendor = %v, want %v", info.Vendor, tt.wantVendor)
			}

			if info.Architecture != tt.wantArch {
				t.Errorf("architecture = %v, want %v", info.Architecture, tt.wantArch)
			}

			if len(info.Families) < tt.wantFamilies {
				t.Errorf("families count = %v, want >= %v", len(info.Families), tt.wantFamilies)
			}
		})
	}
}

func TestVendorAliases(t *testing.T) {
	tests := []struct {
		alias string
		want  string
	}{
		{"intel", "intel"},
		{"amd", "amd"},
		{"aws", "aws"},
		{"graviton", "aws"},
		{"arm", "aws"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got, ok := VendorAliases[tt.alias]
			if !ok {
				t.Fatalf("alias %q not found", tt.alias)
			}
			if got != tt.want {
				t.Errorf("VendorAliases[%q] = %v, want %v", tt.alias, got, tt.want)
			}
		})
	}
}

func TestGetProcessorsByVendor(t *testing.T) {
	tests := []struct {
		vendor   string
		wantMin  int // minimum expected processors
	}{
		{"intel", 3},
		{"amd", 4},
		{"aws", 4},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			processors := GetProcessorsByVendor(tt.vendor)
			if len(processors) < tt.wantMin {
				t.Errorf("GetProcessorsByVendor(%q) returned %d processors, want >= %d",
					tt.vendor, len(processors), tt.wantMin)
			}
		})
	}
}

func TestGetFamiliesByVendor(t *testing.T) {
	tests := []struct {
		vendor  string
		wantMin int // minimum expected families
	}{
		{"intel", 10},
		{"amd", 10},
		{"aws", 10},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			families := GetFamiliesByVendor(tt.vendor)
			if len(families) < tt.wantMin {
				t.Errorf("GetFamiliesByVendor(%q) returned %d families, want >= %d",
					tt.vendor, len(families), tt.wantMin)
			}
		})
	}
}
