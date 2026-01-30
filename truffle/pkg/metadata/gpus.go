package metadata

// GPUInfo contains information about a GPU accelerator used in EC2 instances
type GPUInfo struct {
	Name          string   // "A100", "V100", "H100"
	Vendor        string   // "nvidia", "amd", "aws"
	MemoryGB      int      // GPU memory per GPU
	UseCase       string   // "training", "inference", "graphics"
	Families      []string // Instance families (for fuzzy matching)
	InstanceTypes []string // Exact instance types (for precise matching)
}

// GPUDatabase maps GPU names to their information
var GPUDatabase = map[string]GPUInfo{
	// NVIDIA Training GPUs
	"h100": {
		Name:          "H100",
		Vendor:        "nvidia",
		MemoryGB:      80,
		UseCase:       "training",
		Families:      []string{"p5"},
		InstanceTypes: []string{"p5.48xlarge"},
	},
	"a100": {
		Name:          "A100",
		Vendor:        "nvidia",
		MemoryGB:      40,
		UseCase:       "training",
		Families:      []string{"p4d", "p4de"},
		InstanceTypes: []string{"p4d.24xlarge", "p4de.24xlarge"},
	},
	"v100": {
		Name:     "V100",
		Vendor:   "nvidia",
		MemoryGB: 16,
		UseCase:  "training",
		Families: []string{"p3"},
		InstanceTypes: []string{
			"p3.2xlarge", "p3.8xlarge", "p3.16xlarge",
			"p3dn.24xlarge",
		},
	},
	"k80": {
		Name:          "K80",
		Vendor:        "nvidia",
		MemoryGB:      12,
		UseCase:       "legacy",
		Families:      []string{"p2"},
		InstanceTypes: []string{"p2.xlarge", "p2.8xlarge", "p2.16xlarge"},
	},
	"m60": {
		Name:     "M60",
		Vendor:   "nvidia",
		MemoryGB: 8,
		UseCase:  "graphics",
		Families: []string{"g3"},
		InstanceTypes: []string{
			"g3s.xlarge", "g3.4xlarge", "g3.8xlarge", "g3.16xlarge",
		},
	},

	// NVIDIA Inference GPUs
	"a10g": {
		Name:     "A10G",
		Vendor:   "nvidia",
		MemoryGB: 24,
		UseCase:  "inference",
		Families: []string{"g5"},
		InstanceTypes: []string{
			"g5.xlarge", "g5.2xlarge", "g5.4xlarge", "g5.8xlarge",
			"g5.12xlarge", "g5.16xlarge", "g5.24xlarge", "g5.48xlarge",
		},
	},
	"t4": {
		Name:     "T4",
		Vendor:   "nvidia",
		MemoryGB: 16,
		UseCase:  "inference",
		Families: []string{"g4dn"},
		InstanceTypes: []string{
			"g4dn.xlarge", "g4dn.2xlarge", "g4dn.4xlarge", "g4dn.8xlarge",
			"g4dn.12xlarge", "g4dn.16xlarge", "g4dn.metal",
		},
	},
	"l4": {
		Name:     "L4",
		Vendor:   "nvidia",
		MemoryGB: 24,
		UseCase:  "inference",
		Families: []string{"g6"},
		InstanceTypes: []string{
			"g6.xlarge", "g6.2xlarge", "g6.4xlarge", "g6.8xlarge",
			"g6.12xlarge", "g6.16xlarge", "g6.24xlarge", "g6.48xlarge",
		},
	},

	// AMD GPUs
	"radeon pro v520": {
		Name:          "Radeon Pro V520",
		Vendor:        "amd",
		MemoryGB:      8,
		UseCase:       "graphics",
		Families:      []string{"g4ad"},
		InstanceTypes: []string{"g4ad.xlarge", "g4ad.2xlarge", "g4ad.4xlarge", "g4ad.8xlarge", "g4ad.16xlarge"},
	},

	// AWS Accelerators
	"inferentia": {
		Name:     "Inferentia",
		Vendor:   "aws",
		MemoryGB: 8,
		UseCase:  "inference",
		Families: []string{"inf1"},
		InstanceTypes: []string{
			"inf1.xlarge", "inf1.2xlarge", "inf1.6xlarge", "inf1.24xlarge",
		},
	},
	"inferentia2": {
		Name:     "Inferentia2",
		Vendor:   "aws",
		MemoryGB: 32,
		UseCase:  "inference",
		Families: []string{"inf2"},
		InstanceTypes: []string{
			"inf2.xlarge", "inf2.8xlarge", "inf2.24xlarge", "inf2.48xlarge",
		},
	},
	"trainium": {
		Name:     "Trainium",
		Vendor:   "aws",
		MemoryGB: 32,
		UseCase:  "training",
		Families: []string{"trn1", "trn1n"},
		InstanceTypes: []string{
			"trn1.2xlarge", "trn1.32xlarge",
			"trn1n.32xlarge",
		},
	},
}

// GPUAliases maps common GPU names to canonical forms
var GPUAliases = map[string]string{
	"inf":      "inferentia",
	"inf1":     "inferentia",
	"inf2":     "inferentia2",
	"trn":      "trainium",
	"trn1":     "trainium",
	"a10":      "a10g",
	"radeon":   "radeon pro v520",
	"v520":     "radeon pro v520",
	"neuron":   "inferentia",
	"inferent": "inferentia",
}

// GetGPUsByVendor returns all GPUs for a given vendor
func GetGPUsByVendor(vendor string) []GPUInfo {
	var gpus []GPUInfo
	for _, info := range GPUDatabase {
		if info.Vendor == vendor {
			gpus = append(gpus, info)
		}
	}
	return gpus
}

// GetGPUsByUseCase returns all GPUs for a given use case
func GetGPUsByUseCase(useCase string) []GPUInfo {
	var gpus []GPUInfo
	for _, info := range GPUDatabase {
		if info.UseCase == useCase {
			gpus = append(gpus, info)
		}
	}
	return gpus
}
