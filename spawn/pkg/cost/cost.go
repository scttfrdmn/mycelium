package cost

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	dynamoTableName = "spawn-sweep-orchestration"
)

// RegionalCost represents cost for a specific region
type RegionalCost struct {
	Region         string
	InstanceHours  float64
	EstimatedCost  float64
	InstanceCount  int
	InstanceType   string
}

// InstanceTypeCost represents cost for a specific instance type
type InstanceTypeCost struct {
	InstanceType   string
	InstanceHours  float64
	EstimatedCost  float64
	InstanceCount  int
}

// CostBreakdown represents a detailed cost breakdown
type CostBreakdown struct {
	SweepID           string
	TotalCost         float64
	TotalInstanceHours float64
	Budget            float64
	BudgetRemaining   float64
	BudgetExceeded    bool
	ByRegion          []RegionalCost
	ByInstanceType    []InstanceTypeCost
}

// SweepInstance represents an instance in the sweep
type SweepInstance struct {
	Index          int     `dynamodbav:"index"`
	Region         string  `dynamodbav:"region"`
	InstanceID     string  `dynamodbav:"instance_id"`
	RequestedType  string  `dynamodbav:"requested_type,omitempty"`
	ActualType     string  `dynamodbav:"actual_type,omitempty"`
	State          string  `dynamodbav:"state"`
	LaunchedAt     string  `dynamodbav:"launched_at"`
	TerminatedAt   string  `dynamodbav:"terminated_at,omitempty"`
	ErrorMessage   string  `dynamodbav:"error_message,omitempty"`
	InstanceHours  float64 `dynamodbav:"instance_hours,omitempty"`
	EstimatedCost  float64 `dynamodbav:"estimated_cost,omitempty"`
}

// SweepRecord represents the minimal sweep record for cost calculation
type SweepRecord struct {
	SweepID       string          `dynamodbav:"sweep_id"`
	EstimatedCost float64         `dynamodbav:"estimated_cost,omitempty"`
	Budget        float64         `dynamodbav:"budget,omitempty"`
	Instances     []SweepInstance `dynamodbav:"instances"`
}

// Client provides cost tracking operations
type Client struct {
	db *dynamodb.Client
}

// NewClient creates a new cost client
func NewClient(db *dynamodb.Client) *Client {
	return &Client{db: db}
}

// GetCostBreakdown retrieves and calculates cost breakdown for a sweep
func (c *Client) GetCostBreakdown(ctx context.Context, sweepID string) (*CostBreakdown, error) {
	// Get sweep record from DynamoDB
	result, err := c.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(dynamoTableName),
		Key: map[string]types.AttributeValue{
			"sweep_id": &types.AttributeValueMemberS{Value: sweepID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get sweep record: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("sweep not found: %s", sweepID)
	}

	var sweep SweepRecord
	if err := attributevalue.UnmarshalMap(result.Item, &sweep); err != nil {
		return nil, fmt.Errorf("unmarshal sweep: %w", err)
	}

	// Calculate breakdown
	breakdown := &CostBreakdown{
		SweepID:         sweepID,
		TotalCost:       sweep.EstimatedCost,
		Budget:          sweep.Budget,
		ByRegion:        make([]RegionalCost, 0),
		ByInstanceType:  make([]InstanceTypeCost, 0),
	}

	// Aggregate by region
	regionMap := make(map[string]*RegionalCost)
	typeMap := make(map[string]*InstanceTypeCost)

	for _, inst := range sweep.Instances {
		// Skip instances that never launched
		if inst.InstanceID == "" {
			continue
		}

		instType := inst.ActualType
		if instType == "" {
			instType = inst.RequestedType
		}

		// Aggregate by region
		if regionMap[inst.Region] == nil {
			regionMap[inst.Region] = &RegionalCost{
				Region: inst.Region,
			}
		}
		regionMap[inst.Region].InstanceHours += inst.InstanceHours
		regionMap[inst.Region].EstimatedCost += inst.EstimatedCost
		regionMap[inst.Region].InstanceCount++

		// Aggregate by instance type
		if typeMap[instType] == nil {
			typeMap[instType] = &InstanceTypeCost{
				InstanceType: instType,
			}
		}
		typeMap[instType].InstanceHours += inst.InstanceHours
		typeMap[instType].EstimatedCost += inst.EstimatedCost
		typeMap[instType].InstanceCount++

		breakdown.TotalInstanceHours += inst.InstanceHours
	}

	// Convert maps to slices
	for _, rc := range regionMap {
		breakdown.ByRegion = append(breakdown.ByRegion, *rc)
	}
	for _, tc := range typeMap {
		breakdown.ByInstanceType = append(breakdown.ByInstanceType, *tc)
	}

	// Sort by cost (highest first)
	sort.Slice(breakdown.ByRegion, func(i, j int) bool {
		return breakdown.ByRegion[i].EstimatedCost > breakdown.ByRegion[j].EstimatedCost
	})
	sort.Slice(breakdown.ByInstanceType, func(i, j int) bool {
		return breakdown.ByInstanceType[i].EstimatedCost > breakdown.ByInstanceType[j].EstimatedCost
	})

	// Calculate budget status
	if breakdown.Budget > 0 {
		breakdown.BudgetRemaining = breakdown.Budget - breakdown.TotalCost
		breakdown.BudgetExceeded = breakdown.TotalCost > breakdown.Budget
	}

	return breakdown, nil
}
