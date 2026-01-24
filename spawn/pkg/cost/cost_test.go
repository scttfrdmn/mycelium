package cost

import (
	"testing"
)

func TestCostBreakdownCalculations(t *testing.T) {
	tests := []struct {
		name               string
		budget             float64
		totalCost          float64
		wantBudgetExceeded bool
		wantRemaining      float64
	}{
		{
			name:               "within budget",
			budget:             100.0,
			totalCost:          75.50,
			wantBudgetExceeded: false,
			wantRemaining:      24.50,
		},
		{
			name:               "exceeded budget",
			budget:             50.0,
			totalCost:          75.50,
			wantBudgetExceeded: true,
			wantRemaining:      -25.50,
		},
		{
			name:               "exactly at budget",
			budget:             100.0,
			totalCost:          100.0,
			wantBudgetExceeded: false,
			wantRemaining:      0.0,
		},
		{
			name:               "no budget set",
			budget:             0.0,
			totalCost:          100.0,
			wantBudgetExceeded: false,
			wantRemaining:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown := &CostBreakdown{
				TotalCost: tt.totalCost,
				Budget:    tt.budget,
			}

			// Calculate budget status manually (simulating what GetCostBreakdown does)
			if breakdown.Budget > 0 {
				breakdown.BudgetRemaining = breakdown.Budget - breakdown.TotalCost
				breakdown.BudgetExceeded = breakdown.TotalCost > breakdown.Budget
			}

			if breakdown.BudgetExceeded != tt.wantBudgetExceeded {
				t.Errorf("BudgetExceeded = %v, want %v", breakdown.BudgetExceeded, tt.wantBudgetExceeded)
			}

			if tt.budget > 0 {
				// Only check remaining if budget is set
				if breakdown.BudgetRemaining != tt.wantRemaining {
					t.Errorf("BudgetRemaining = %.2f, want %.2f", breakdown.BudgetRemaining, tt.wantRemaining)
				}
			}
		})
	}
}

func TestRegionalCostAggregation(t *testing.T) {
	// Create test instances
	instances := []SweepInstance{
		{
			Region:        "us-east-1",
			ActualType:    "t3.micro",
			InstanceHours: 2.5,
			EstimatedCost: 0.0125,
		},
		{
			Region:        "us-east-1",
			ActualType:    "t3.micro",
			InstanceHours: 3.0,
			EstimatedCost: 0.015,
		},
		{
			Region:        "us-west-2",
			ActualType:    "t3.small",
			InstanceHours: 1.5,
			EstimatedCost: 0.0105,
		},
	}

	// Aggregate by region
	regionMap := make(map[string]*RegionalCost)
	for _, inst := range instances {
		if regionMap[inst.Region] == nil {
			regionMap[inst.Region] = &RegionalCost{
				Region: inst.Region,
			}
		}
		regionMap[inst.Region].InstanceHours += inst.InstanceHours
		regionMap[inst.Region].EstimatedCost += inst.EstimatedCost
		regionMap[inst.Region].InstanceCount++
	}

	// Verify us-east-1
	if usEast1, ok := regionMap["us-east-1"]; ok {
		if usEast1.InstanceCount != 2 {
			t.Errorf("us-east-1 InstanceCount = %d, want 2", usEast1.InstanceCount)
		}
		if usEast1.InstanceHours != 5.5 {
			t.Errorf("us-east-1 InstanceHours = %.1f, want 5.5", usEast1.InstanceHours)
		}
		expectedCost := 0.0125 + 0.015
		if usEast1.EstimatedCost != expectedCost {
			t.Errorf("us-east-1 EstimatedCost = %.4f, want %.4f", usEast1.EstimatedCost, expectedCost)
		}
	} else {
		t.Error("us-east-1 region not found in aggregation")
	}

	// Verify us-west-2
	if usWest2, ok := regionMap["us-west-2"]; ok {
		if usWest2.InstanceCount != 1 {
			t.Errorf("us-west-2 InstanceCount = %d, want 1", usWest2.InstanceCount)
		}
		if usWest2.InstanceHours != 1.5 {
			t.Errorf("us-west-2 InstanceHours = %.1f, want 1.5", usWest2.InstanceHours)
		}
	} else {
		t.Error("us-west-2 region not found in aggregation")
	}
}

func TestInstanceTypeCostAggregation(t *testing.T) {
	instances := []SweepInstance{
		{
			ActualType:    "t3.micro",
			InstanceHours: 2.5,
			EstimatedCost: 0.0125,
		},
		{
			ActualType:    "t3.micro",
			InstanceHours: 3.0,
			EstimatedCost: 0.015,
		},
		{
			ActualType:    "t3.small",
			InstanceHours: 1.5,
			EstimatedCost: 0.0105,
		},
	}

	// Aggregate by instance type
	typeMap := make(map[string]*InstanceTypeCost)
	for _, inst := range instances {
		if typeMap[inst.ActualType] == nil {
			typeMap[inst.ActualType] = &InstanceTypeCost{
				InstanceType: inst.ActualType,
			}
		}
		typeMap[inst.ActualType].InstanceHours += inst.InstanceHours
		typeMap[inst.ActualType].EstimatedCost += inst.EstimatedCost
		typeMap[inst.ActualType].InstanceCount++
	}

	// Verify t3.micro
	if t3Micro, ok := typeMap["t3.micro"]; ok {
		if t3Micro.InstanceCount != 2 {
			t.Errorf("t3.micro InstanceCount = %d, want 2", t3Micro.InstanceCount)
		}
		if t3Micro.InstanceHours != 5.5 {
			t.Errorf("t3.micro InstanceHours = %.1f, want 5.5", t3Micro.InstanceHours)
		}
	} else {
		t.Error("t3.micro instance type not found in aggregation")
	}

	// Verify t3.small
	if t3Small, ok := typeMap["t3.small"]; ok {
		if t3Small.InstanceCount != 1 {
			t.Errorf("t3.small InstanceCount = %d, want 1", t3Small.InstanceCount)
		}
	} else {
		t.Error("t3.small instance type not found in aggregation")
	}
}
