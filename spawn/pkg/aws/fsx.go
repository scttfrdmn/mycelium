package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/fsx/types"
)

// FSxInfo contains information about an FSx Lustre filesystem
type FSxInfo struct {
	FileSystemID    string
	DNSName         string
	MountName       string
	StorageCapacity int32
	S3Bucket        string
	S3ImportPath    string
	S3ExportPath    string
}

// FSxConfig contains configuration for creating FSx Lustre filesystem
type FSxConfig struct {
	StackName        string
	Region           string
	StorageCapacity  int32
	S3Bucket         string
	ImportPath       string
	ExportPath       string
	AutoCreateBucket bool
	SubnetID         string // Optional: specify subnet, otherwise uses default VPC
}

// CreateFSxLustreFilesystem creates an FSx for Lustre filesystem with S3 backing
func (c *Client) CreateFSxLustreFilesystem(ctx context.Context, config FSxConfig) (*FSxInfo, error) {
	// 1. Ensure S3 bucket exists (auto-create if specified)
	if config.AutoCreateBucket {
		err := c.CreateS3BucketIfNotExists(ctx, config.S3Bucket, config.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 bucket: %w", err)
		}
	}

	// 2. Get subnet ID (use default VPC if not specified)
	subnetID := config.SubnetID
	if subnetID == "" {
		vpcID, err := c.GetDefaultVPC(ctx, config.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to get default VPC: %w", err)
		}

		subnets, err := c.GetSubnets(ctx, config.Region, vpcID)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnets: %w", err)
		}

		if len(subnets) == 0 {
			return nil, fmt.Errorf("no subnets found in default VPC")
		}

		subnetID = subnets[0]
	}

	// 3. Construct import/export paths
	importPath := config.ImportPath
	if importPath == "" && config.S3Bucket != "" {
		importPath = fmt.Sprintf("s3://%s/", config.S3Bucket)
	}

	exportPath := config.ExportPath
	if exportPath == "" && config.S3Bucket != "" {
		exportPath = fmt.Sprintf("s3://%s/", config.S3Bucket)
	}

	// 4. Create FSx filesystem with DRA
	cfg := c.cfg.Copy()
	cfg.Region = config.Region
	fsxClient := fsx.NewFromConfig(cfg)

	input := &fsx.CreateFileSystemInput{
		FileSystemType:  types.FileSystemTypeLustre,
		StorageCapacity: aws.Int32(config.StorageCapacity),
		SubnetIds:       []string{subnetID}, // FSx Lustre requires single subnet
		LustreConfiguration: &types.CreateFileSystemLustreConfiguration{
			DeploymentType:      types.LustreDeploymentTypeScratch2,
			DataCompressionType: types.DataCompressionTypeLz4,
		},
		Tags: []types.Tag{
			{Key: aws.String("Name"), Value: aws.String(config.StackName)},
			{Key: aws.String("spawn:managed"), Value: aws.String("true")},
			{Key: aws.String("spawn:fsx-s3-backed"), Value: aws.String("true")},
			{Key: aws.String("spawn:fsx-s3-bucket"), Value: aws.String(config.S3Bucket)},
			{Key: aws.String("spawn:fsx-stack-name"), Value: aws.String(config.StackName)},
			{Key: aws.String("spawn:fsx-storage-capacity"), Value: aws.String(fmt.Sprintf("%d", config.StorageCapacity))},
			{Key: aws.String("spawn:fsx-created"), Value: aws.String(time.Now().UTC().Format(time.RFC3339))},
		},
	}

	// Add import/export paths if S3 bucket specified
	if importPath != "" {
		input.LustreConfiguration.ImportPath = aws.String(importPath)
		input.LustreConfiguration.AutoImportPolicy = types.AutoImportPolicyTypeNewChanged
		input.Tags = append(input.Tags, types.Tag{
			Key:   aws.String("spawn:fsx-s3-import-path"),
			Value: aws.String(importPath),
		})
	}

	if exportPath != "" {
		input.LustreConfiguration.ExportPath = aws.String(exportPath)
		input.Tags = append(input.Tags, types.Tag{
			Key:   aws.String("spawn:fsx-s3-export-path"),
			Value: aws.String(exportPath),
		})
	}

	result, err := fsxClient.CreateFileSystem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create FSx filesystem: %w", err)
	}

	filesystemID := *result.FileSystem.FileSystemId

	// 5. Wait for filesystem to be AVAILABLE (~5-8 minutes)
	// Poll until filesystem is available
	maxWaitTime := 15 * time.Minute
	startTime := time.Now()
	for {
		describeResult, err := fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{
			FileSystemIds: []string{filesystemID},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe FSx filesystem: %w", err)
		}

		if len(describeResult.FileSystems) > 0 {
			fs := describeResult.FileSystems[0]
			if fs.Lifecycle == types.FileSystemLifecycleAvailable {
				break
			}
			if fs.Lifecycle == types.FileSystemLifecycleFailed {
				return nil, fmt.Errorf("FSx filesystem creation failed")
			}
		}

		if time.Since(startTime) > maxWaitTime {
			return nil, fmt.Errorf("FSx filesystem creation timeout after %v", maxWaitTime)
		}

		time.Sleep(30 * time.Second)
	}

	// 6. Get filesystem details
	describeResult, err := fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{
		FileSystemIds: []string{filesystemID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe FSx filesystem: %w", err)
	}

	if len(describeResult.FileSystems) == 0 {
		return nil, fmt.Errorf("FSx filesystem not found after creation")
	}

	fs := describeResult.FileSystems[0]

	// 7. Return filesystem info
	return &FSxInfo{
		FileSystemID:    *fs.FileSystemId,
		DNSName:         *fs.DNSName,
		MountName:       *fs.LustreConfiguration.MountName,
		StorageCapacity: *fs.StorageCapacity,
		S3Bucket:        config.S3Bucket,
		S3ImportPath:    importPath,
		S3ExportPath:    exportPath,
	}, nil
}

// GetFSxFilesystem retrieves info for existing FSx filesystem
func (c *Client) GetFSxFilesystem(ctx context.Context, filesystemID, region string) (*FSxInfo, error) {
	cfg := c.cfg.Copy()
	cfg.Region = region
	fsxClient := fsx.NewFromConfig(cfg)

	result, err := fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{
		FileSystemIds: []string{filesystemID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe FSx filesystem: %w", err)
	}

	if len(result.FileSystems) == 0 {
		return nil, fmt.Errorf("FSx filesystem not found: %s", filesystemID)
	}

	fs := result.FileSystems[0]

	// Extract S3 info from tags
	s3Bucket := ""
	s3ImportPath := ""
	s3ExportPath := ""
	for _, tag := range fs.Tags {
		switch *tag.Key {
		case "spawn:fsx-s3-bucket":
			s3Bucket = *tag.Value
		case "spawn:fsx-s3-import-path":
			s3ImportPath = *tag.Value
		case "spawn:fsx-s3-export-path":
			s3ExportPath = *tag.Value
		}
	}

	return &FSxInfo{
		FileSystemID:    *fs.FileSystemId,
		DNSName:         *fs.DNSName,
		MountName:       *fs.LustreConfiguration.MountName,
		StorageCapacity: *fs.StorageCapacity,
		S3Bucket:        s3Bucket,
		S3ImportPath:    s3ImportPath,
		S3ExportPath:    s3ExportPath,
	}, nil
}

// RecallFSxFilesystem finds and recreates FSx filesystem by stack name
func (c *Client) RecallFSxFilesystem(ctx context.Context, stackName, region string) (*FSxInfo, error) {
	cfg := c.cfg.Copy()
	cfg.Region = region
	fsxClient := fsx.NewFromConfig(cfg)

	// 1. Search for filesystems with this stack name tag
	result, err := fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list FSx filesystems: %w", err)
	}

	// 2. Find filesystem with matching stack name (may be deleted, so check all)
	var foundConfig *FSxConfig
	for _, fs := range result.FileSystems {
		for _, tag := range fs.Tags {
			if *tag.Key == "spawn:fsx-stack-name" && *tag.Value == stackName {
				// Extract configuration from tags
				config := &FSxConfig{
					StackName:       stackName,
					Region:          region,
					StorageCapacity: *fs.StorageCapacity,
				}

				for _, t := range fs.Tags {
					switch *t.Key {
					case "spawn:fsx-s3-bucket":
						config.S3Bucket = *t.Value
					case "spawn:fsx-s3-import-path":
						config.ImportPath = *t.Value
					case "spawn:fsx-s3-export-path":
						config.ExportPath = *t.Value
					}
				}

				// If filesystem is already available, return it
				if fs.Lifecycle == types.FileSystemLifecycleAvailable {
					return &FSxInfo{
						FileSystemID:    *fs.FileSystemId,
						DNSName:         *fs.DNSName,
						MountName:       *fs.LustreConfiguration.MountName,
						StorageCapacity: *fs.StorageCapacity,
						S3Bucket:        config.S3Bucket,
						S3ImportPath:    config.ImportPath,
						S3ExportPath:    config.ExportPath,
					}, nil
				}

				foundConfig = config
				break
			}
		}
		if foundConfig != nil {
			break
		}
	}

	if foundConfig == nil {
		return nil, fmt.Errorf("no FSx filesystem found with stack name: %s", stackName)
	}

	// 3. Create new filesystem with same configuration
	return c.CreateFSxLustreFilesystem(ctx, *foundConfig)
}

// GetSubnets returns subnet IDs for a VPC
func (c *Client) GetSubnets(ctx context.Context, region, vpcID string) ([]string, error) {
	cfg := c.cfg.Copy()
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	result, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	subnetIDs := make([]string, 0, len(result.Subnets))
	for _, subnet := range result.Subnets {
		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	}

	return subnetIDs, nil
}
