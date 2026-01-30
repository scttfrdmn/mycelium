package metadata

import (
	"testing"
)

func TestGPUDatabase(t *testing.T) {
	tests := []struct {
		name            string
		gpuName         string
		wantVendor      string
		wantUseCase     string
		wantMinMemory   int
		wantMinInstances int
	}{
		{
			name:            "a100",
			gpuName:         "a100",
			wantVendor:      "nvidia",
			wantUseCase:     "training",
			wantMinMemory:   40,
			wantMinInstances: 1,
		},
		{
			name:            "h100",
			gpuName:         "h100",
			wantVendor:      "nvidia",
			wantUseCase:     "training",
			wantMinMemory:   80,
			wantMinInstances: 1,
		},
		{
			name:            "v100",
			gpuName:         "v100",
			wantVendor:      "nvidia",
			wantUseCase:     "training",
			wantMinMemory:   16,
			wantMinInstances: 2,
		},
		{
			name:            "t4",
			gpuName:         "t4",
			wantVendor:      "nvidia",
			wantUseCase:     "inference",
			wantMinMemory:   16,
			wantMinInstances: 5,
		},
		{
			name:            "inferentia",
			gpuName:         "inferentia",
			wantVendor:      "aws",
			wantUseCase:     "inference",
			wantMinMemory:   8,
			wantMinInstances: 2,
		},
		{
			name:            "trainium",
			gpuName:         "trainium",
			wantVendor:      "aws",
			wantUseCase:     "training",
			wantMinMemory:   32,
			wantMinInstances: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := GPUDatabase[tt.gpuName]
			if !ok {
				t.Fatalf("GPU %q not found in database", tt.gpuName)
			}

			if info.Vendor != tt.wantVendor {
				t.Errorf("vendor = %v, want %v", info.Vendor, tt.wantVendor)
			}

			if info.UseCase != tt.wantUseCase {
				t.Errorf("use case = %v, want %v", info.UseCase, tt.wantUseCase)
			}

			if info.MemoryGB < tt.wantMinMemory {
				t.Errorf("memory = %v, want >= %v", info.MemoryGB, tt.wantMinMemory)
			}

			if len(info.InstanceTypes) < tt.wantMinInstances {
				t.Errorf("instance types count = %v, want >= %v",
					len(info.InstanceTypes), tt.wantMinInstances)
			}
		})
	}
}

func TestGPUAliases(t *testing.T) {
	tests := []struct {
		alias string
		want  string
	}{
		{"inf", "inferentia"},
		{"inf1", "inferentia"},
		{"inf2", "inferentia2"},
		{"trn", "trainium"},
		{"trn1", "trainium"},
		{"a10", "a10g"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got, ok := GPUAliases[tt.alias]
			if !ok {
				t.Fatalf("alias %q not found", tt.alias)
			}
			if got != tt.want {
				t.Errorf("GPUAliases[%q] = %v, want %v", tt.alias, got, tt.want)
			}
		})
	}
}

func TestGetGPUsByVendor(t *testing.T) {
	tests := []struct {
		vendor  string
		wantMin int
	}{
		{"nvidia", 5},
		{"aws", 3},
		{"amd", 1},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			gpus := GetGPUsByVendor(tt.vendor)
			if len(gpus) < tt.wantMin {
				t.Errorf("GetGPUsByVendor(%q) returned %d GPUs, want >= %d",
					tt.vendor, len(gpus), tt.wantMin)
			}
		})
	}
}

func TestGetGPUsByUseCase(t *testing.T) {
	tests := []struct {
		useCase string
		wantMin int
	}{
		{"training", 3},
		{"inference", 4},
	}

	for _, tt := range tests {
		t.Run(tt.useCase, func(t *testing.T) {
			gpus := GetGPUsByUseCase(tt.useCase)
			if len(gpus) < tt.wantMin {
				t.Errorf("GetGPUsByUseCase(%q) returned %d GPUs, want >= %d",
					tt.useCase, len(gpus), tt.wantMin)
			}
		})
	}
}
