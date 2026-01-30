package metadata

// ProcessorInfo contains information about a CPU processor used in EC2 instances
type ProcessorInfo struct {
	CodeName     string   // "ice lake", "milan", "sapphire rapids"
	Vendor       string   // "intel", "amd", "aws"
	Architecture string   // "x86_64", "arm64"
	Generation   string   // "3rd gen", "4th gen"
	Families     []string // ["m6i", "c6i", "r6i"]
}

// ProcessorDatabase maps processor code names to their information
var ProcessorDatabase = map[string]ProcessorInfo{
	// Intel processors
	"ice lake": {
		CodeName:     "Ice Lake",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "3rd gen",
		Families:     []string{"m6i", "c6i", "r6i", "r6id", "r6idn", "m6id", "m6idn", "c6id", "c6in"},
	},
	"cascade lake": {
		CodeName:     "Cascade Lake",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "2nd gen",
		Families:     []string{"m5", "c5", "r5", "m5n", "m5dn", "c5n", "r5n", "r5dn", "m5d", "c5d", "r5d"},
	},
	"sapphire rapids": {
		CodeName:     "Sapphire Rapids",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "4th gen",
		Families:     []string{"m7i", "c7i", "r7i", "r7iz", "m7i-flex"},
	},
	"skylake": {
		CodeName:     "Skylake",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "1st gen",
		Families:     []string{"m5", "c5", "r5", "z1d"},
	},
	"haswell": {
		CodeName:     "Haswell",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "legacy",
		Families:     []string{"m4", "c4", "r4", "t2"},
	},
	"broadwell": {
		CodeName:     "Broadwell",
		Vendor:       "intel",
		Architecture: "x86_64",
		Generation:   "legacy",
		Families:     []string{"m4", "c4", "t2"},
	},

	// AMD processors
	"milan": {
		CodeName:     "Milan",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "3rd gen",
		Families:     []string{"m6a", "c6a", "r6a", "r6id", "hpc6a"},
	},
	"rome": {
		CodeName:     "Rome",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "2nd gen",
		Families:     []string{"m5a", "c5a", "r5a", "m5ad", "r5ad", "m5dn"},
	},
	"genoa": {
		CodeName:     "Genoa",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "4th gen",
		Families:     []string{"m7a", "c7a", "r7a", "hpc7a"},
	},
	"bergamo": {
		CodeName:     "Bergamo",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "4th gen",
		Families:     []string{"m7a", "c7a"},
	},
	"zen 3": {
		CodeName:     "Zen 3",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "3rd gen",
		Families:     []string{"m6a", "c6a", "r6a", "hpc6a"},
	},
	"zen 4": {
		CodeName:     "Zen 4",
		Vendor:       "amd",
		Architecture: "x86_64",
		Generation:   "4th gen",
		Families:     []string{"m7a", "c7a", "r7a", "hpc7a"},
	},

	// AWS Graviton processors
	"graviton": {
		CodeName:     "Graviton",
		Vendor:       "aws",
		Architecture: "arm64",
		Generation:   "1st gen",
		Families:     []string{"a1"},
	},
	"graviton2": {
		CodeName:     "Graviton2",
		Vendor:       "aws",
		Architecture: "arm64",
		Generation:   "2nd gen",
		Families:     []string{"m6g", "c6g", "r6g", "t4g", "m6gd", "c6gd", "r6gd", "c6gn", "im4gn", "is4gen", "x2gd"},
	},
	"graviton3": {
		CodeName:     "Graviton3",
		Vendor:       "aws",
		Architecture: "arm64",
		Generation:   "3rd gen",
		Families:     []string{"m7g", "c7g", "r7g", "c7gn", "hpc7g", "c7gd", "m7gd", "r7gd"},
	},
	"graviton3e": {
		CodeName:     "Graviton3E",
		Vendor:       "aws",
		Architecture: "arm64",
		Generation:   "3rd gen",
		Families:     []string{"c7gn", "hpc7g"},
	},
	"graviton4": {
		CodeName:     "Graviton4",
		Vendor:       "aws",
		Architecture: "arm64",
		Generation:   "4th gen",
		Families:     []string{"r8g"},
	},
}

// VendorAliases maps common vendor names to canonical forms
var VendorAliases = map[string]string{
	"intel":    "intel",
	"amd":      "amd",
	"aws":      "aws",
	"graviton": "aws",
	"arm":      "aws",
	"amazon":   "aws",
}

// GetProcessorsByVendor returns all processors for a given vendor
func GetProcessorsByVendor(vendor string) []ProcessorInfo {
	var processors []ProcessorInfo
	for _, info := range ProcessorDatabase {
		if info.Vendor == vendor {
			processors = append(processors, info)
		}
	}
	return processors
}

// GetFamiliesByVendor returns all instance families for a given vendor
func GetFamiliesByVendor(vendor string) []string {
	familySet := make(map[string]bool)
	for _, info := range ProcessorDatabase {
		if info.Vendor == vendor {
			for _, family := range info.Families {
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
