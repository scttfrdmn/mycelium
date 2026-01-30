package metadata

import (
	"testing"
)

func TestSizeCategories(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantSizes int
	}{
		{"tiny", "tiny", 2},
		{"small", "small", 2},
		{"medium", "medium", 2},
		{"large", "large", 2},
		{"huge", "huge", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, ok := SizeCategories[tt.category]
			if !ok {
				t.Fatalf("category %q not found", tt.category)
			}

			if cat.Name != tt.category {
				t.Errorf("name = %v, want %v", cat.Name, tt.category)
			}

			if len(cat.Sizes) < tt.wantSizes {
				t.Errorf("sizes count = %v, want >= %v", len(cat.Sizes), tt.wantSizes)
			}
		})
	}
}

func TestGetSizesForCategory(t *testing.T) {
	tests := []struct {
		category string
		wantMin  int
		wantNil  bool
	}{
		{"tiny", 2, false},
		{"small", 2, false},
		{"medium", 2, false},
		{"large", 2, false},
		{"huge", 5, false},
		{"nonexistent", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			sizes := GetSizesForCategory(tt.category)

			if tt.wantNil {
				if sizes != nil {
					t.Errorf("GetSizesForCategory(%q) = %v, want nil", tt.category, sizes)
				}
				return
			}

			if len(sizes) < tt.wantMin {
				t.Errorf("GetSizesForCategory(%q) returned %d sizes, want >= %d",
					tt.category, len(sizes), tt.wantMin)
			}
		})
	}
}
