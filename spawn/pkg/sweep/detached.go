package sweep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	dynamoTableName = "spawn-sweep-orchestration"
	s3BucketPattern = "spawn-sweeps-%s"
	lambdaFuncName  = "spawn-sweep-orchestrator"
)

// SweepRecord represents the DynamoDB record structure
type SweepRecord struct {
	SweepID       string          `dynamodbav:"sweep_id"`
	SweepName     string          `dynamodbav:"sweep_name"`
	UserID        string          `dynamodbav:"user_id"`
	CreatedAt     string          `dynamodbav:"created_at"`
	UpdatedAt     string          `dynamodbav:"updated_at"`
	CompletedAt   string          `dynamodbav:"completed_at,omitempty"`
	S3ParamsKey   string          `dynamodbav:"s3_params_key"`
	MaxConcurrent int             `dynamodbav:"max_concurrent"`
	LaunchDelay   string          `dynamodbav:"launch_delay"`
	TotalParams   int             `dynamodbav:"total_params"`
	Region        string          `dynamodbav:"region"`
	AWSAccountID  string          `dynamodbav:"aws_account_id"`
	Status        string          `dynamodbav:"status"`
	NextToLaunch  int             `dynamodbav:"next_to_launch"`
	Launched      int             `dynamodbav:"launched"`
	Failed        int             `dynamodbav:"failed"`
	ErrorMessage  string          `dynamodbav:"error_message,omitempty"`
	Instances     []SweepInstance `dynamodbav:"instances"`
}

// SweepInstance tracks individual instance state
type SweepInstance struct {
	Index        int    `dynamodbav:"index"`
	InstanceID   string `dynamodbav:"instance_id"`
	State        string `dynamodbav:"state"`
	LaunchedAt   string `dynamodbav:"launched_at"`
	TerminatedAt string `dynamodbav:"terminated_at,omitempty"`
	ErrorMessage string `dynamodbav:"error_message,omitempty"`
}

// ParamFileFormat matches the CLI parameter file structure
type ParamFileFormat struct {
	Defaults map[string]interface{}   `json:"defaults"`
	Params   []map[string]interface{} `json:"params"`
}

// UploadParamsToS3 uploads parameter file to S3 and returns the S3 key
func UploadParamsToS3(ctx context.Context, cfg aws.Config, paramFormat *ParamFileFormat, sweepID, region string) (string, error) {
	s3Client := s3.NewFromConfig(cfg)

	bucket := fmt.Sprintf(s3BucketPattern, region)
	key := fmt.Sprintf("sweeps/%s/params.json", sweepID)

	// Marshal params to JSON
	data, err := json.Marshal(paramFormat)
	if err != nil {
		return "", fmt.Errorf("failed to marshal params: %w", err)
	}

	// Upload to S3
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	s3Key := fmt.Sprintf("s3://%s/%s", bucket, key)
	return s3Key, nil
}

// CreateSweepRecord creates a new DynamoDB record for the sweep
func CreateSweepRecord(ctx context.Context, cfg aws.Config, record *SweepRecord) error {
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	// Get current user identity for UserID
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}
	record.UserID = *identity.Arn

	// Set timestamps and initial state
	now := time.Now().Format(time.RFC3339)
	record.CreatedAt = now
	record.UpdatedAt = now
	record.Status = "INITIALIZING"
	record.NextToLaunch = 0
	record.Launched = 0
	record.Failed = 0
	record.Instances = []SweepInstance{}

	// Marshal record
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Put item to DynamoDB
	_, err = dynamodbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(dynamoTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create DynamoDB record: %w", err)
	}

	return nil
}

// InvokeSweepOrchestrator invokes the Lambda function for sweep orchestration
func InvokeSweepOrchestrator(ctx context.Context, cfg aws.Config, sweepID string) error {
	lambdaClient := lambda.NewFromConfig(cfg)

	payload, err := json.Marshal(map[string]interface{}{
		"sweep_id":       sweepID,
		"force_download": false,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Invoke Lambda asynchronously
	_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(lambdaFuncName),
		InvocationType: "Event", // Async invocation
		Payload:        payload,
	})
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %w", err)
	}

	return nil
}

// QuerySweepStatus queries DynamoDB for the current sweep status
func QuerySweepStatus(ctx context.Context, cfg aws.Config, sweepID string) (*SweepRecord, error) {
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	result, err := dynamodbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(dynamoTableName),
		Key: map[string]types.AttributeValue{
			"sweep_id": &types.AttributeValueMemberS{Value: sweepID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("sweep %s not found", sweepID)
	}

	var record SweepRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return &record, nil
}

// LoadSweepStateFromDynamoDB loads the sweep state from DynamoDB
func LoadSweepStateFromDynamoDB(ctx context.Context, cfg aws.Config, sweepID string) (*SweepRecord, error) {
	return QuerySweepStatus(ctx, cfg, sweepID)
}

// SaveSweepState saves the sweep state to DynamoDB
func SaveSweepState(ctx context.Context, cfg aws.Config, record *SweepRecord) error {
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	record.UpdatedAt = time.Now().Format(time.RFC3339)

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	_, err = dynamodbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(dynamoTableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}
