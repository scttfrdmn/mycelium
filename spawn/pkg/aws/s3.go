package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// CreateS3BucketIfNotExists creates an S3 bucket if it doesn't already exist
func (c *Client) CreateS3BucketIfNotExists(ctx context.Context, bucketName, region string) error {
	cfg := c.cfg.Copy()
	cfg.Region = region
	s3Client := s3.NewFromConfig(cfg)

	// Check if bucket exists
	_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err == nil {
		// Bucket exists
		return nil
	}

	// Check if error is "not found" - if so, create bucket
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchBucket" {
			// Bucket doesn't exist, create it
			createInput := &s3.CreateBucketInput{
				Bucket: aws.String(bucketName),
			}

			// For regions other than us-east-1, need to specify location constraint
			if region != "us-east-1" {
				createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
					LocationConstraint: types.BucketLocationConstraint(region),
				}
			}

			_, err = s3Client.CreateBucket(ctx, createInput)
			if err != nil {
				return fmt.Errorf("failed to create S3 bucket: %w", err)
			}

			// Add tags to identify spawn-managed bucket
			_, err = s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
				Bucket: aws.String(bucketName),
				Tagging: &types.Tagging{
					TagSet: []types.Tag{
						{
							Key:   aws.String("spawn:managed"),
							Value: aws.String("true"),
						},
						{
							Key:   aws.String("spawn:fsx-backing-bucket"),
							Value: aws.String("true"),
						},
					},
				},
			})
			if err != nil {
				// Non-fatal error - bucket was created successfully
				return nil
			}

			return nil
		}
	}

	// Some other error occurred
	return fmt.Errorf("failed to check if bucket exists: %w", err)
}
