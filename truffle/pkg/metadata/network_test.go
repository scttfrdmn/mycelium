package metadata

import (
	"testing"
)

func TestGetFamiliesByEFA(t *testing.T) {
	families := GetFamiliesByEFA()

	if len(families) < 20 {
		t.Errorf("GetFamiliesByEFA() returned %d families, expected at least 20", len(families))
	}

	// Check some known EFA-capable families
	expectedFamilies := []string{"p5", "p4d", "c6gn", "hpc7g", "m7i"}
	for _, expected := range expectedFamilies {
		found := false
		for _, family := range families {
			if family == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetFamiliesByEFA() missing expected family: %s", expected)
		}
	}
}

func TestIsEFASupported(t *testing.T) {
	tests := []struct {
		family   string
		expected bool
	}{
		{"p5", true},
		{"p4d", true},
		{"c6gn", true},
		{"hpc7g", true},
		{"t3", false},
		{"t2", false},
		{"m5", false},
	}

	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			got := IsEFASupported(tt.family)
			if got != tt.expected {
				t.Errorf("IsEFASupported(%q) = %v, want %v", tt.family, got, tt.expected)
			}
		})
	}
}

func TestGetFamiliesByNetworkSpeed(t *testing.T) {
	tests := []struct {
		name         string
		minGbps      int
		wantMin      int
		wantContains []string
	}{
		{
			name:         "10 Gbps minimum",
			minGbps:      10,
			wantMin:      30,
			wantContains: []string{"m5", "c5", "r5"},
		},
		{
			name:         "100 Gbps minimum",
			minGbps:      100,
			wantMin:      10,
			wantContains: []string{"p5", "p4d", "hpc7g"},
		},
		{
			name:         "400 Gbps minimum",
			minGbps:      400,
			wantMin:      1,
			wantContains: []string{"p5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			families := GetFamiliesByNetworkSpeed(tt.minGbps)

			if len(families) < tt.wantMin {
				t.Errorf("GetFamiliesByNetworkSpeed(%d) returned %d families, want >= %d",
					tt.minGbps, len(families), tt.wantMin)
			}

			for _, want := range tt.wantContains {
				found := false
				for _, family := range families {
					if family == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetFamiliesByNetworkSpeed(%d) missing family %q in %v",
						tt.minGbps, want, families)
				}
			}
		})
	}
}

func TestNetworkBandwidthTiers(t *testing.T) {
	// Verify all tiers have valid data
	for speed, capability := range NetworkBandwidthTiers {
		t.Run(speed, func(t *testing.T) {
			if capability.MaxBandwidthGbps <= 0 {
				t.Errorf("Tier %s has invalid bandwidth: %d", speed, capability.MaxBandwidthGbps)
			}
			if len(capability.Families) == 0 {
				t.Errorf("Tier %s has no families", speed)
			}
		})
	}
}

func TestNetworkAliases(t *testing.T) {
	tests := []struct {
		alias string
		want  string
	}{
		{"efa", "efa"},
		{"10g", "10gbps"},
		{"100g", "100gbps"},
		{"highnet", "50gbps"},
		{"ultranet", "100gbps"},
		{"lowlatency", "efa"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got, ok := NetworkAliases[tt.alias]
			if !ok {
				t.Fatalf("NetworkAliases[%q] not found", tt.alias)
			}
			if got != tt.want {
				t.Errorf("NetworkAliases[%q] = %v, want %v", tt.alias, got, tt.want)
			}
		})
	}
}
