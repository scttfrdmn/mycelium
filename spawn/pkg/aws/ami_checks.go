package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// AMIHealthCheck represents the health status of an AMI
type AMIHealthCheck struct {
	BaseAMIOutdated bool
	BaseAMIAge      time.Duration
	Warnings        []string
}

// CheckAMIHealth checks if an AMI has an outdated base AMI
func (c *Client) CheckAMIHealth(ctx context.Context, ami AMIInfo, region string) (*AMIHealthCheck, error) {
	check := &AMIHealthCheck{
		Warnings: make([]string, 0),
	}

	// Check base AMI age
	baseAMIID := ami.Tags["spawn:base-ami"]
	if baseAMIID == "" {
		// No base AMI tracked - AMI created before tracking was added
		return check, nil
	}

	baseAMIAge, outdated, err := c.checkBaseAMI(ctx, region, baseAMIID, ami.Architecture, ami.GPU)
	if err != nil {
		// Can't check - don't warn
		return check, nil
	}

	check.BaseAMIAge = baseAMIAge
	check.BaseAMIOutdated = outdated

	if outdated {
		days := int(baseAMIAge.Hours() / 24)
		if days > 90 {
			check.Warnings = append(check.Warnings, fmt.Sprintf("base AMI is %d days old (rebuild recommended)", days))
		} else if days > 30 {
			check.Warnings = append(check.Warnings, fmt.Sprintf("base AMI is %d days old", days))
		}
	}

	return check, nil
}

// checkBaseAMI checks if the base AMI used to create this AMI is outdated
func (c *Client) checkBaseAMI(ctx context.Context, region string, baseAMIID string, arch string, gpu bool) (time.Duration, bool, error) {
	// Get the base AMI creation date
	cfg, err := c.getRegionalConfig(ctx, region)
	if err != nil {
		return 0, false, fmt.Errorf("failed to get regional config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// Get base AMI details
	baseResult, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{baseAMIID},
	})
	if err != nil || len(baseResult.Images) == 0 {
		return 0, false, fmt.Errorf("failed to get base AMI details: %w", err)
	}

	baseImage := baseResult.Images[0]
	baseCreationDate, err := time.Parse(time.RFC3339, *baseImage.CreationDate)
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse base AMI creation date: %w", err)
	}

	// Get current recommended AMI
	currentAMI, err := c.GetRecommendedAMI(ctx, region, "")
	if err != nil {
		return 0, false, fmt.Errorf("failed to get current recommended AMI: %w", err)
	}

	// If different, the base is outdated
	outdated := baseAMIID != currentAMI

	// Calculate age
	age := time.Since(baseCreationDate)

	return age, outdated, nil
}
