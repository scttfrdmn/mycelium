package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// CreatePlacementGroup creates a cluster placement group for MPI
func (c *Client) CreatePlacementGroup(ctx context.Context, name string) error {
	ec2Client := ec2.NewFromConfig(c.cfg)

	_, err := ec2Client.CreatePlacementGroup(ctx, &ec2.CreatePlacementGroupInput{
		GroupName: aws.String(name),
		Strategy:  types.PlacementStrategyCluster,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypePlacementGroup,
				Tags: []types.Tag{
					{Key: aws.String("spawn:managed"), Value: aws.String("true")},
					{Key: aws.String("spawn:purpose"), Value: aws.String("mpi")},
				},
			},
		},
	})

	if err != nil {
		// Check if already exists (error string contains "already exists")
		if strings.Contains(err.Error(), "already exists") {
			return nil // Already exists, not an error
		}
		return fmt.Errorf("create placement group: %w", err)
	}

	return nil
}

// DeletePlacementGroup removes a placement group
func (c *Client) DeletePlacementGroup(ctx context.Context, name string) error {
	ec2Client := ec2.NewFromConfig(c.cfg)

	_, err := ec2Client.DeletePlacementGroup(ctx, &ec2.DeletePlacementGroupInput{
		GroupName: aws.String(name),
	})
	return err
}

// ValidateInstanceTypeForPlacementGroup checks if instance type supports cluster placement
func (c *Client) ValidateInstanceTypeForPlacementGroup(ctx context.Context, instanceType string) error {
	// Only certain instance families support cluster placement groups:
	// - Compute optimized: c4, c5, c5n, c6g, c6gn, c7g
	// - Memory optimized: r4, r5, r5n, r6g, x1, x1e
	// - Storage optimized: d2, h1, i3, i3en
	// - Accelerated: p2, p3, p4, g3, g4dn, inf1

	supportedPrefixes := []string{
		"c4.", "c5.", "c5n.", "c6g.", "c6gn.", "c7g.",
		"r4.", "r5.", "r5n.", "r6g.",
		"x1.", "x1e.",
		"d2.", "h1.", "i3.", "i3en.",
		"p2.", "p3.", "p4.", "g3.", "g4dn.", "inf1.",
	}

	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(instanceType, prefix) {
			return nil
		}
	}

	return fmt.Errorf("instance type %s does not support cluster placement groups", instanceType)
}

// ValidateInstanceTypeForEFA checks if instance type supports EFA
func (c *Client) ValidateInstanceTypeForEFA(ctx context.Context, instanceType string) error {
	// EFA-supported instance types:
	// - c5n.18xlarge, c5n.metal
	// - c6gn.16xlarge
	// - g4dn.8xlarge, g4dn.12xlarge, g4dn.metal
	// - g5.8xlarge, g5.12xlarge, g5.16xlarge, g5.24xlarge, g5.48xlarge
	// - i3en.12xlarge, i3en.24xlarge, i3en.metal
	// - inf1.24xlarge
	// - m5dn.24xlarge, m5n.24xlarge
	// - m6i.32xlarge
	// - p3dn.24xlarge
	// - p4d.24xlarge, p4de.24xlarge
	// - p5.48xlarge
	// - r5dn.24xlarge, r5n.24xlarge
	// - r6i.32xlarge
	// - trn1.32xlarge

	efaSupported := map[string]bool{
		"c5n.18xlarge": true, "c5n.metal": true,
		"c6gn.16xlarge": true,
		"g4dn.8xlarge": true, "g4dn.12xlarge": true, "g4dn.metal": true,
		"g5.8xlarge": true, "g5.12xlarge": true, "g5.16xlarge": true,
		"g5.24xlarge": true, "g5.48xlarge": true,
		"i3en.12xlarge": true, "i3en.24xlarge": true, "i3en.metal": true,
		"inf1.24xlarge": true,
		"m5dn.24xlarge": true, "m5n.24xlarge": true,
		"m6i.32xlarge": true,
		"p3dn.24xlarge": true,
		"p4d.24xlarge": true, "p4de.24xlarge": true,
		"p5.48xlarge": true,
		"r5dn.24xlarge": true, "r5n.24xlarge": true,
		"r6i.32xlarge": true,
		"trn1.32xlarge": true,
	}

	if !efaSupported[instanceType] {
		return fmt.Errorf("instance type %s does not support EFA", instanceType)
	}

	return nil
}
