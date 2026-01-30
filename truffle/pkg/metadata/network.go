package metadata

// NetworkCapability represents networking features of instance families
type NetworkCapability struct {
	EFASupported    bool     // Elastic Fabric Adapter support
	MaxBandwidthGbps int     // Maximum network bandwidth in Gbps
	Families        []string // Instance families with this capability
}

// NetworkFeature represents network feature requirements
type NetworkFeature struct {
	Name        string
	Description string
	Families    []string
}

// EFACapableFamilies lists all instance families that support EFA
var EFACapableFamilies = []string{
	// Compute optimized
	"c5n", "c6a", "c6gn", "c6i", "c6id", "c6in", "c7a", "c7g", "c7gd", "c7gn", "c7i",
	// General purpose
	"m5dn", "m5n", "m5zn", "m6a", "m6i", "m6id", "m6idn", "m6in", "m7a", "m7g", "m7gd", "m7i", "m7i-flex",
	// Memory optimized
	"r5dn", "r5n", "r6a", "r6i", "r6id", "r6idn", "r6in", "r7a", "r7g", "r7gd", "r7i", "r7iz", "r8g",
	"x2idn", "x2iedn", "x2iezn",
	// Storage optimized
	"i3en", "i4g", "i4i", "im4gn", "is4gen",
	// Accelerated computing (GPU)
	"p3dn", "p4d", "p4de", "p5",
	// HPC optimized
	"hpc6a", "hpc6id", "hpc7a", "hpc7g",
}

// NetworkBandwidthTiers maps bandwidth requirements to instance families
var NetworkBandwidthTiers = map[string]NetworkCapability{
	"10gbps": {
		MaxBandwidthGbps: 10,
		Families: []string{
			"m5", "m5a", "m5ad", "m5d", "c5", "c5a", "c5ad", "c5d",
			"r5", "r5a", "r5ad", "r5d", "t3", "t3a",
		},
	},
	"25gbps": {
		MaxBandwidthGbps: 25,
		Families: []string{
			"m5n", "m5dn", "m6i", "m6id", "m6a", "m6g", "m6gd",
			"c5n", "c6i", "c6id", "c6a", "c6g", "c6gd",
			"r5n", "r5dn", "r6i", "r6id", "r6a", "r6g", "r6gd",
			"p3", "p3dn", "g4dn", "g5",
		},
	},
	"50gbps": {
		MaxBandwidthGbps: 50,
		Families: []string{
			"m5zn", "m6idn", "m6in", "m7i", "m7i-flex", "m7a", "m7g", "m7gd",
			"c6in", "c6gn", "c7i", "c7a", "c7g", "c7gd", "c7gn",
			"r6idn", "r6in", "r7i", "r7iz", "r7a", "r7g", "r7gd", "r8g",
			"p4d", "p4de",
		},
	},
	"100gbps": {
		MaxBandwidthGbps: 100,
		Families: []string{
			"m6idn", "m6in", "m7i", "m7a", "m7g",
			"c6in", "c6gn", "c7i", "c7a", "c7g", "c7gn",
			"r6idn", "r6in", "r7i", "r7iz", "r7a", "r7g", "r8g",
			"x2idn", "x2iedn", "x2iezn",
			"i4i", "i4g", "im4gn", "is4gen",
			"p4d", "p4de", "p5",
			"hpc6a", "hpc6id", "hpc7a", "hpc7g",
		},
	},
	"200gbps": {
		MaxBandwidthGbps: 200,
		Families: []string{
			"p4d", "p4de", "p5",
			"hpc7g",
		},
	},
	"400gbps": {
		MaxBandwidthGbps: 400,
		Families: []string{
			"p5",
		},
	},
}

// NetworkAliases maps common network terms to canonical forms
var NetworkAliases = map[string]string{
	"efa":        "efa",
	"ena":        "ena",
	"10g":        "10gbps",
	"25g":        "25gbps",
	"50g":        "50gbps",
	"100g":       "100gbps",
	"200g":       "200gbps",
	"400g":       "400gbps",
	"highnet":    "50gbps",
	"ultranet":   "100gbps",
	"lowlatency": "efa",
}

// GetFamiliesByNetworkSpeed returns families that support at least the specified bandwidth
func GetFamiliesByNetworkSpeed(minGbps int) []string {
	familySet := make(map[string]bool)

	for _, capability := range NetworkBandwidthTiers {
		if capability.MaxBandwidthGbps >= minGbps {
			for _, family := range capability.Families {
				familySet[family] = true
			}
		}
	}

	families := make([]string, 0, len(familySet))
	for family := range familySet {
		families = append(families, family)
	}
	return families
}

// GetFamiliesByEFA returns families that support EFA
func GetFamiliesByEFA() []string {
	return EFACapableFamilies
}

// IsEFASupported checks if a family supports EFA
func IsEFASupported(family string) bool {
	for _, f := range EFACapableFamilies {
		if f == family {
			return true
		}
	}
	return false
}
