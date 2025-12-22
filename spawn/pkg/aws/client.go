package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type Client struct {
	cfg aws.Config
}

func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{cfg: cfg}, nil
}

// LaunchConfig contains all settings for launching an instance
type LaunchConfig struct {
	InstanceType     string
	Region           string
	AvailabilityZone string
	AMI              string
	KeyName          string
	IamInstanceProfile string
	SecurityGroupIDs []string
	SubnetID         string
	UserData         string
	Spot             bool
	SpotMaxPrice     string
	ReservationID    string
	Hibernate        bool

	// spawn-specific tags
	TTL             string
	IdleTimeout     string
	HibernateOnIdle bool
	CostLimit       float64

	// Metadata
	Name string
	Tags map[string]string
}

// LaunchResult contains information about the launched instance
type LaunchResult struct {
	InstanceID       string
	PublicIP         string
	PrivateIP        string
	AvailabilityZone string
	State            string
	KeyName          string
}

func (c *Client) Launch(ctx context.Context, launchConfig LaunchConfig) (*LaunchResult, error) {
	// Update config for region
	cfg := c.cfg.Copy()
	cfg.Region = launchConfig.Region
	ec2Client := ec2.NewFromConfig(cfg)
	
	// Build tags
	tags := buildTags(launchConfig)
	
	// Build block device mappings
	blockDevices := buildBlockDevices(launchConfig)
	
	// Build run instances input
	input := &ec2.RunInstancesInput{
		InstanceType: types.InstanceType(launchConfig.InstanceType),
		ImageId:      aws.String(launchConfig.AMI),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		KeyName:      aws.String(launchConfig.KeyName),
		UserData:     aws.String(launchConfig.UserData),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         tags,
			},
			{
				ResourceType: types.ResourceTypeVolume,
				Tags:         tags,
			},
		},
		BlockDeviceMappings: blockDevices,
	}

	// Add IAM instance profile if specified
	if launchConfig.IamInstanceProfile != "" {
		input.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(launchConfig.IamInstanceProfile),
		}
	}
	
	// Add network configuration
	if launchConfig.SubnetID != "" {
		input.NetworkInterfaces = []types.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(launchConfig.SubnetID),
				Groups:                   launchConfig.SecurityGroupIDs,
			},
		}
	} else if len(launchConfig.SecurityGroupIDs) > 0 {
		input.SecurityGroupIds = launchConfig.SecurityGroupIDs
	}
	
	// Add placement (AZ and reservation)
	placement := &types.Placement{}
	if launchConfig.AvailabilityZone != "" {
		placement.AvailabilityZone = aws.String(launchConfig.AvailabilityZone)
	}
	input.Placement = placement
	
	// Add hibernation if enabled
	if launchConfig.Hibernate {
		input.HibernationOptions = &types.HibernationOptionsRequest{
			Configured: aws.Bool(true),
		}
	}
	
	// Add Spot configuration if needed
	if launchConfig.Spot {
		input.InstanceMarketOptions = &types.InstanceMarketOptionsRequest{
			MarketType: types.MarketTypeSpot,
			SpotOptions: &types.SpotMarketOptions{
				SpotInstanceType: types.SpotInstanceTypeOneTime,
			},
		}
		
		if launchConfig.SpotMaxPrice != "" {
			input.InstanceMarketOptions.SpotOptions.MaxPrice = aws.String(launchConfig.SpotMaxPrice)
		}
	}
	
	// Launch instance
	result, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}
	
	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no instances returned")
	}
	
	instance := result.Instances[0]
	
	launchResult := &LaunchResult{
		InstanceID:       *instance.InstanceId,
		PrivateIP:        valueOrEmpty(instance.PrivateIpAddress),
		PublicIP:         valueOrEmpty(instance.PublicIpAddress),
		AvailabilityZone: valueOrEmpty(instance.Placement.AvailabilityZone),
		State:            string(instance.State.Name),
		KeyName:          launchConfig.KeyName,
	}
	
	return launchResult, nil
}

func buildTags(config LaunchConfig) []types.Tag {
	tags := []types.Tag{
		{Key: aws.String("spawn:managed"), Value: aws.String("true")},
		{Key: aws.String("spawn:root"), Value: aws.String("true")},
		{Key: aws.String("spawn:created-by"), Value: aws.String("spawn")},
		{Key: aws.String("spawn:version"), Value: aws.String("0.1.0")},
	}
	
	if config.Name != "" {
		tags = append(tags, types.Tag{Key: aws.String("Name"), Value: aws.String(config.Name)})
	}
	
	if config.TTL != "" {
		tags = append(tags, types.Tag{Key: aws.String("spawn:ttl"), Value: aws.String(config.TTL)})
	}
	
	if config.IdleTimeout != "" {
		tags = append(tags, types.Tag{Key: aws.String("spawn:idle-timeout"), Value: aws.String(config.IdleTimeout)})
	}
	
	if config.HibernateOnIdle {
		tags = append(tags, types.Tag{Key: aws.String("spawn:hibernate-on-idle"), Value: aws.String("true")})
	}
	
	// Add custom tags
	for k, v := range config.Tags {
		tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	
	return tags
}

func buildBlockDevices(config LaunchConfig) []types.BlockDeviceMapping {
	// Calculate volume size for hibernation
	volumeSize := int32(20) // Default 20 GB
	
	if config.Hibernate {
		// For hibernation, need RAM + OS + buffer
		// Estimate based on instance type
		volumeSize = estimateVolumeSize(config.InstanceType)
	}
	
	return []types.BlockDeviceMapping{
		{
			DeviceName: aws.String("/dev/xvda"),
			Ebs: &types.EbsBlockDevice{
				VolumeSize:          aws.Int32(volumeSize),
				VolumeType:          types.VolumeTypeGp3,
				DeleteOnTermination: aws.Bool(true),
				Encrypted:           aws.Bool(config.Hibernate), // Required for hibernation
			},
		},
	}
}

func estimateVolumeSize(instanceType string) int32 {
	// Rough estimation of RAM size by instance family
	// This should ideally query EC2 DescribeInstanceTypes
	ramEstimates := map[string]int32{
		"t3":  8,
		"t4g": 8,
		"m7i": 16,
		"m8g": 16,
		"c7i": 16,
		"r7i": 32,
		"p5":  768, // H100 instances have lots of RAM
		"g6":  32,
	}
	
	// Extract family
	for prefix, ram := range ramEstimates {
		if len(instanceType) >= len(prefix) && instanceType[:len(prefix)] == prefix {
			return ram + 10 // RAM + 10GB for OS
		}
	}
	
	return 20 // Default
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CheckKeyPairExists checks if a key pair exists in AWS EC2
func (c *Client) CheckKeyPairExists(ctx context.Context, region, keyName string) (bool, error) {
	cfg := c.cfg
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	}

	_, err := ec2Client.DescribeKeyPairs(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		if contains(err.Error(), "InvalidKeyPair.NotFound") || contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check key pair: %w", err)
	}

	return true, nil
}

// ImportKeyPair imports a public key to AWS EC2
func (c *Client) ImportKeyPair(ctx context.Context, region, keyName string, publicKey []byte) error {
	cfg := c.cfg
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: publicKey,
	}

	_, err := ec2Client.ImportKeyPair(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to import key pair: %w", err)
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetInstancePublicIP queries an instance and returns its public IP
func (c *Client) GetInstancePublicIP(ctx context.Context, region, instanceID string) (string, error) {
	cfg := c.cfg
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	return valueOrEmpty(instance.PublicIpAddress), nil
}

// FindKeyPairByFingerprint searches for a key pair matching the given fingerprint
// Returns the key name if found, empty string if not found
func (c *Client) FindKeyPairByFingerprint(ctx context.Context, region, fingerprint string) (string, error) {
	cfg := c.cfg
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	// List all key pairs
	input := &ec2.DescribeKeyPairsInput{}
	result, err := ec2Client.DescribeKeyPairs(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to list key pairs: %w", err)
	}

	// Search for matching fingerprint
	for _, kp := range result.KeyPairs {
		if kp.KeyFingerprint != nil && *kp.KeyFingerprint == fingerprint {
			if kp.KeyName != nil {
				return *kp.KeyName, nil
			}
		}
	}

	return "", nil // Not found
}

// InstanceInfo contains metadata about a spawn-managed instance
type InstanceInfo struct {
	InstanceID       string
	Name             string
	InstanceType     string
	State            string
	Region           string
	AvailabilityZone string
	PublicIP         string
	PrivateIP        string
	LaunchTime       time.Time
	TTL              string
	IdleTimeout      string
	KeyName          string
	SpotInstance     bool
	Tags             map[string]string
}

// ListInstances returns all spawn-managed instances, optionally filtered by region and state
func (c *Client) ListInstances(ctx context.Context, region string, stateFilter string) ([]InstanceInfo, error) {
	var allInstances []InstanceInfo

	// Determine which regions to search
	regions := []string{}
	if region != "" {
		regions = append(regions, region)
	} else {
		// Query all regions
		var err error
		regions, err = c.getAllRegions(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get regions: %w", err)
		}
	}

	// Search each region for spawn-managed instances
	for _, r := range regions {
		instances, err := c.listInstancesInRegion(ctx, r, stateFilter)
		if err != nil {
			// Log error but continue with other regions
			continue
		}
		allInstances = append(allInstances, instances...)
	}

	return allInstances, nil
}

func (c *Client) listInstancesInRegion(ctx context.Context, region string, stateFilter string) ([]InstanceInfo, error) {
	cfg := c.cfg.Copy()
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	// Build filters
	filters := []types.Filter{
		{
			Name:   aws.String("tag:spawn:managed"),
			Values: []string{"true"},
		},
	}

	// Add state filter if specified
	if stateFilter != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []string{stateFilter},
		})
	} else {
		// Default: show running and stopped instances (not terminated)
		filters = append(filters, types.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []string{"pending", "running", "stopping", "stopped"},
		})
	}

	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	var instances []InstanceInfo

	// Paginate through results
	paginator := ec2.NewDescribeInstancesPaginator(ec2Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances in %s: %w", region, err)
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				info := InstanceInfo{
					InstanceID:       valueOrEmpty(instance.InstanceId),
					InstanceType:     string(instance.InstanceType),
					State:            string(instance.State.Name),
					Region:           region,
					AvailabilityZone: valueOrEmpty(instance.Placement.AvailabilityZone),
					PublicIP:         valueOrEmpty(instance.PublicIpAddress),
					PrivateIP:        valueOrEmpty(instance.PrivateIpAddress),
					KeyName:          valueOrEmpty(instance.KeyName),
					SpotInstance:     instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot,
					Tags:             make(map[string]string),
				}

				if instance.LaunchTime != nil {
					info.LaunchTime = *instance.LaunchTime
				}

				// Extract tags
				for _, tag := range instance.Tags {
					if tag.Key != nil && tag.Value != nil {
						key := *tag.Key
						value := *tag.Value

						switch key {
						case "Name":
							info.Name = value
						case "spawn:ttl":
							info.TTL = value
						case "spawn:idle-timeout":
							info.IdleTimeout = value
						default:
							info.Tags[key] = value
						}
					}
				}

				instances = append(instances, info)
			}
		}
	}

	return instances, nil
}

func (c *Client) getAllRegions(ctx context.Context) ([]string, error) {
	// Use us-east-1 as the base region for the DescribeRegions call
	cfg := c.cfg.Copy()
	cfg.Region = "us-east-1"
	ec2Client := ec2.NewFromConfig(cfg)

	result, err := ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only enabled regions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	var regions []string
	for _, region := range result.Regions {
		if region.RegionName != nil {
			regions = append(regions, *region.RegionName)
		}
	}

	return regions, nil
}

// StopInstance stops an EC2 instance
func (c *Client) StopInstance(ctx context.Context, region, instanceID string, hibernate bool) error {
	cfg := c.cfg.Copy()
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
		Hibernate:   aws.Bool(hibernate),
	}

	_, err := ec2Client.StopInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// StartInstance starts a stopped EC2 instance
func (c *Client) StartInstance(ctx context.Context, region, instanceID string) error {
	cfg := c.cfg.Copy()
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := ec2Client.StartInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

// UpdateInstanceTags updates tags on an EC2 instance
func (c *Client) UpdateInstanceTags(ctx context.Context, region, instanceID string, tags map[string]string) error {
	cfg := c.cfg.Copy()
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	// Convert tags to AWS format
	awsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// Create or update tags
	input := &ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      awsTags,
	}

	_, err := ec2Client.CreateTags(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}

	return nil
}

// SetupSpawndIAMRole creates or retrieves the IAM role and instance profile for spawnd
// Returns the instance profile name
func (c *Client) SetupSpawndIAMRole(ctx context.Context) (string, error) {
	iamClient := iam.NewFromConfig(c.cfg)

	roleName := "spawnd-instance-role"
	instanceProfileName := "spawnd-instance-profile"
	policyName := "spawnd-policy"

	roleCreated := false
	profileCreated := false

	// 1. Check if role exists, create if not
	_, err := iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})

	if err != nil {
		roleCreated = true
		// Role doesn't exist, create it
		trustPolicy := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`

		_, err = iamClient.CreateRole(ctx, &iam.CreateRoleInput{
			RoleName:                 aws.String(roleName),
			AssumeRolePolicyDocument: aws.String(trustPolicy),
			Description:              aws.String("IAM role for spawnd daemon on EC2 instances"),
			Tags: []iamtypes.Tag{
				{Key: aws.String("spawn:managed"), Value: aws.String("true")},
			},
		})
		if err != nil && !contains(err.Error(), "EntityAlreadyExists") {
			return "", fmt.Errorf("failed to create IAM role: %w", err)
		}
	}

	// 2. Attach inline policy to role
	policy := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeTags",
        "ec2:DescribeInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:TerminateInstances",
        "ec2:StopInstances"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/spawn:managed": "true"
        }
      }
    }
  ]
}`

	_, err = iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(policy),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach policy to role: %w", err)
	}

	// 3. Check if instance profile exists, create if not
	_, err = iamClient.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
	})

	if err != nil {
		profileCreated = true
		// Instance profile doesn't exist, create it
		_, err = iamClient.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
			InstanceProfileName: aws.String(instanceProfileName),
			Tags: []iamtypes.Tag{
				{Key: aws.String("spawn:managed"), Value: aws.String("true")},
			},
		})
		if err != nil && !contains(err.Error(), "EntityAlreadyExists") {
			return "", fmt.Errorf("failed to create instance profile: %w", err)
		}

		// Add role to instance profile
		_, err = iamClient.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
			InstanceProfileName: aws.String(instanceProfileName),
			RoleName:            aws.String(roleName),
		})
		if err != nil && !contains(err.Error(), "LimitExceeded") {
			return "", fmt.Errorf("failed to add role to instance profile: %w", err)
		}
	}

	// If we created new resources, wait for IAM to propagate (eventual consistency)
	if roleCreated || profileCreated {
		time.Sleep(10 * time.Second)
	}

	return instanceProfileName, nil
}
