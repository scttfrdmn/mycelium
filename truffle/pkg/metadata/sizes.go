package metadata

// SizeCategory represents a category of instance sizes
type SizeCategory struct {
	Name  string
	Sizes []string // Instance size suffixes
}

// SizeCategories maps size category names to their suffixes
var SizeCategories = map[string]SizeCategory{
	"tiny": {
		Name:  "tiny",
		Sizes: []string{"nano", "micro"},
	},
	"small": {
		Name:  "small",
		Sizes: []string{"small", "medium"},
	},
	"medium": {
		Name:  "medium",
		Sizes: []string{"large", "xlarge"},
	},
	"large": {
		Name:  "large",
		Sizes: []string{"2xlarge", "4xlarge"},
	},
	"huge": {
		Name:  "huge",
		Sizes: []string{"8xlarge", "12xlarge", "16xlarge", "24xlarge", "32xlarge", "48xlarge", "56xlarge", "112xlarge", "metal"},
	},
}

// GetSizesForCategory returns the instance size suffixes for a category
func GetSizesForCategory(category string) []string {
	if cat, ok := SizeCategories[category]; ok {
		return cat.Sizes
	}
	return nil
}
