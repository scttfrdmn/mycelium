package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	lambdasvc "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	tableName       = "spawn-sweep-orchestration"
	maxExecutionDur = 13 * time.Minute
	pollInterval    = 10 * time.Second
	crossAccountRole = "arn:aws:iam::%s:role/SpawnSweepCrossAccountRole"
)

// SweepEvent is the Lambda input event
type SweepEvent struct {
	SweepID       string `json:"sweep_id"`
	ForceDownload bool   `json:"force_download"`
}

// SweepRecord is the DynamoDB record structure
type SweepRecord struct {
	SweepID         string                      `dynamodbav:"sweep_id"`
	SweepName       string                      `dynamodbav:"sweep_name"`
	UserID          string                      `dynamodbav:"user_id"`
	CreatedAt       string                      `dynamodbav:"created_at"`
	UpdatedAt       string                      `dynamodbav:"updated_at"`
	CompletedAt     string                      `dynamodbav:"completed_at,omitempty"`
	S3ParamsKey     string                      `dynamodbav:"s3_params_key"`
	MaxConcurrent   int                         `dynamodbav:"max_concurrent"`
	LaunchDelay     string                      `dynamodbav:"launch_delay"`
	TotalParams     int                         `dynamodbav:"total_params"`
	Region          string                      `dynamodbav:"region"`
	AWSAccountID    string                      `dynamodbav:"aws_account_id"`
	Status          string                      `dynamodbav:"status"`
	CancelRequested bool                        `dynamodbav:"cancel_requested"`
	EstimatedCost   float64                     `dynamodbav:"estimated_cost,omitempty"`
	NextToLaunch    int                         `dynamodbav:"next_to_launch"`
	Launched        int                         `dynamodbav:"launched"`
	Failed          int                         `dynamodbav:"failed"`
	ErrorMessage    string                      `dynamodbav:"error_message,omitempty"`
	Instances       []SweepInstance             `dynamodbav:"instances"`

	// Multi-region support
	MultiRegion     bool                        `dynamodbav:"multi_region"`
	RegionStatus    map[string]*RegionProgress  `dynamodbav:"region_status,omitempty"`
}

// RegionProgress tracks per-region sweep progress
type RegionProgress struct {
	Launched      int   `dynamodbav:"launched"`
	Failed        int   `dynamodbav:"failed"`
	ActiveCount   int   `dynamodbav:"active_count"`
	NextToLaunch  []int `dynamodbav:"next_to_launch"`
}

// SweepInstance tracks individual instance state
type SweepInstance struct {
	Index        int    `dynamodbav:"index"`
	Region       string `dynamodbav:"region"`
	InstanceID   string `dynamodbav:"instance_id"`
	State        string `dynamodbav:"state"`
	LaunchedAt   string `dynamodbav:"launched_at"`
	TerminatedAt string `dynamodbav:"terminated_at,omitempty"`
	ErrorMessage string `dynamodbav:"error_message,omitempty"`
}

// ParamFileFormat matches CLI parameter file structure
type ParamFileFormat struct {
	Defaults map[string]interface{}   `json:"defaults"`
	Params   []map[string]interface{} `json:"params"`
}

// LaunchConfig represents EC2 launch configuration
type LaunchConfig struct {
	InstanceType string
	Region       string
	AMI          string
	KeyName      string
	IAMRole      string
	Tags         map[string]string
	UserData     string
	Spot         bool
	// Additional fields as needed
}

var (
	awsCfg        aws.Config
	dynamodbClient *dynamodb.Client
	s3Client      *s3.Client
	lambdaClient  *lambdasvc.Client
	stsClient     *sts.Client
)

func init() {
	var err error
	awsCfg, err = config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	dynamodbClient = dynamodb.NewFromConfig(awsCfg)
	s3Client = s3.NewFromConfig(awsCfg)
	lambdaClient = lambdasvc.NewFromConfig(awsCfg)
	stsClient = sts.NewFromConfig(awsCfg)
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event SweepEvent) error {
	log.Printf("Starting sweep orchestration for sweep_id=%s", event.SweepID)

	// Load state from DynamoDB
	state, err := loadSweepState(ctx, event.SweepID)
	if err != nil {
		return fmt.Errorf("failed to load sweep state: %w", err)
	}

	// Download parameters from S3
	params, err := downloadParams(ctx, state.S3ParamsKey, event.ForceDownload)
	if err != nil {
		return fmt.Errorf("failed to download parameters: %w", err)
	}

	// Check if sweep is already in terminal state (prevents recursive loop)
	if state.Status == "COMPLETED" || state.Status == "FAILED" || state.Status == "CANCELLED" {
		log.Printf("Sweep already in terminal state: %s. Exiting without reinvocation.", state.Status)
		return nil
	}

	// Setup shared resources if INITIALIZING
	if state.Status == "INITIALIZING" {
		log.Println("Setting up shared resources...")
		// Note: Shared resources (AMI, SSH key, IAM role) are assumed to be pre-configured
		// This is handled by the CLI's existing setup logic
		state.Status = "RUNNING"
		state.UpdatedAt = time.Now().Format(time.RFC3339)
		if err := saveSweepState(ctx, state); err != nil {
			return fmt.Errorf("failed to update status to RUNNING: %w", err)
		}
	}

	// Create cross-account EC2 client
	ec2Client, err := createCrossAccountEC2Client(ctx, state.Region, state.AWSAccountID)
	if err != nil {
		return fmt.Errorf("failed to create cross-account EC2 client: %w", err)
	}

	// Run polling loop
	return runPollingLoop(ctx, state, params, ec2Client, event.SweepID)
}

func loadSweepState(ctx context.Context, sweepID string) (*SweepRecord, error) {
	result, err := dynamodbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"sweep_id": &types.AttributeValueMemberS{Value: sweepID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dynamodb get failed: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("sweep %s not found", sweepID)
	}

	var state SweepRecord
	if err := attributevalue.UnmarshalMap(result.Item, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

func saveSweepState(ctx context.Context, state *SweepRecord) error {
	state.UpdatedAt = time.Now().Format(time.RFC3339)

	item, err := attributevalue.MarshalMap(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Retry logic with exponential backoff
	for retries := 0; retries < 3; retries++ {
		_, err = dynamodbClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err == nil {
			return nil
		}
		log.Printf("DynamoDB write failed (attempt %d/3): %v", retries+1, err)
		time.Sleep(time.Duration(retries+1) * time.Second)
	}

	return fmt.Errorf("failed to save state after 3 retries: %w", err)
}

func downloadParams(ctx context.Context, s3Key string, forceDownload bool) (*ParamFileFormat, error) {
	// Parse S3 key (format: s3://bucket/key)
	parts := strings.SplitN(strings.TrimPrefix(s3Key, "s3://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid S3 key format: %s", s3Key)
	}
	bucket, key := parts[0], parts[1]

	// Check cache in /tmp
	cacheFile := fmt.Sprintf("/tmp/%s", strings.ReplaceAll(key, "/", "_"))
	if !forceDownload {
		if data, err := os.ReadFile(cacheFile); err == nil {
			log.Printf("Using cached params from %s", cacheFile)
			var params ParamFileFormat
			if err := json.Unmarshal(data, &params); err == nil {
				return &params, nil
			}
		}
	}

	// Download from S3
	log.Printf("Downloading params from s3://%s/%s", bucket, key)
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get failed: %w", err)
	}
	defer result.Body.Close()

	var params ParamFileFormat
	if err := json.NewDecoder(result.Body).Decode(&params); err != nil {
		return nil, fmt.Errorf("failed to decode params: %w", err)
	}

	// Cache to /tmp
	if data, err := json.Marshal(params); err == nil {
		os.WriteFile(cacheFile, data, 0644)
	}

	return &params, nil
}

func createCrossAccountEC2Client(ctx context.Context, region, accountID string) (*ec2.Client, error) {
	roleARN := fmt.Sprintf(crossAccountRole, accountID)
	log.Printf("Assuming role: %s", roleARN)

	result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String("spawn-sweep-orchestrator"),
		DurationSeconds: aws.Int32(3600),
	})
	if err != nil {
		return nil, fmt.Errorf("assume role failed: %w", err)
	}

	creds := result.Credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     *creds.AccessKeyId,
				SecretAccessKey: *creds.SecretAccessKey,
				SessionToken:    *creds.SessionToken,
				Source:          "AssumeRole",
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return ec2.NewFromConfig(cfg), nil
}

func runPollingLoop(ctx context.Context, state *SweepRecord, params *ParamFileFormat, ec2Client *ec2.Client, sweepID string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(maxExecutionDur)
	launchDelay := parseDuration(state.LaunchDelay)

	for {
		// Reload state from DynamoDB to check for cancellation
		currentState, err := loadSweepState(ctx, state.SweepID)
		if err != nil {
			log.Printf("Failed to reload sweep state: %v", err)
		} else if currentState.CancelRequested {
			log.Println("Cancellation requested, stopping orchestration")
			state.Status = "CANCELLED"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			if err := saveSweepState(ctx, state); err != nil {
				log.Printf("Failed to save cancelled state: %v", err)
			}
			return nil
		}

		// Check timeout
		if time.Now().After(deadline) {
			// Double-check status before reinvoking (safety guard)
			if state.Status == "COMPLETED" || state.Status == "FAILED" || state.Status == "CANCELLED" {
				log.Printf("Sweep in terminal state %s, not reinvoking", state.Status)
				return nil
			}

			log.Println("Approaching Lambda timeout, re-invoking...")
			if err := saveSweepState(ctx, state); err != nil {
				log.Printf("Failed to save state before re-invocation: %v", err)
			}
			return reinvokeSelf(ctx, sweepID)
		}

		// Query active instances
		activeCount, err := countActiveInstances(ctx, ec2Client, state)
		if err != nil {
			log.Printf("Failed to query instances: %v", err)
		} else {
			log.Printf("Active instances: %d/%d", activeCount, state.MaxConcurrent)
		}

		// Launch next batch if slots available
		available := state.MaxConcurrent - activeCount
		if available > 0 && state.NextToLaunch < state.TotalParams {
			toLaunch := min(available, state.TotalParams-state.NextToLaunch)
			log.Printf("Launching %d instances (slots available: %d)", toLaunch, available)

			for i := 0; i < toLaunch; i++ {
				paramIndex := state.NextToLaunch
				paramSet := params.Params[paramIndex]

				// Merge defaults with param set
				config := mergeParams(params.Defaults, paramSet)

				// Launch instance
				if err := launchInstance(ctx, ec2Client, state, config, paramIndex); err != nil {
					log.Printf("Failed to launch instance %d: %v", paramIndex, err)
					state.Failed++
					state.Instances = append(state.Instances, SweepInstance{
						Index:        paramIndex,
						State:        "failed",
						ErrorMessage: err.Error(),
						LaunchedAt:   time.Now().Format(time.RFC3339),
					})
				} else {
					state.Launched++
				}

				state.NextToLaunch++

				// Save state after each launch
				if err := saveSweepState(ctx, state); err != nil {
					log.Printf("Failed to save state: %v", err)
				}

				// Delay between launches
				if launchDelay > 0 && i < toLaunch-1 {
					time.Sleep(launchDelay)
				}
			}
		}

		// Check for completion
		if state.NextToLaunch >= state.TotalParams && activeCount == 0 {
			log.Println("All instances launched and completed")
			state.Status = "COMPLETED"
			state.CompletedAt = time.Now().Format(time.RFC3339)
			if err := saveSweepState(ctx, state); err != nil {
				return fmt.Errorf("failed to save completion state: %w", err)
			}
			return nil
		}

		// Wait for next poll
		<-ticker.C
	}
}

func countActiveInstances(ctx context.Context, ec2Client *ec2.Client, state *SweepRecord) (int, error) {
	// Build instance IDs list
	var instanceIDs []string
	for _, inst := range state.Instances {
		if inst.InstanceID != "" && (inst.State == "pending" || inst.State == "running") {
			instanceIDs = append(instanceIDs, inst.InstanceID)
		}
	}

	if len(instanceIDs) == 0 {
		return 0, nil
	}

	// Query instance states
	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		return 0, fmt.Errorf("describe instances failed: %w", err)
	}

	activeCount := 0
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			state := instance.State.Name
			if state == ec2types.InstanceStateNamePending || state == ec2types.InstanceStateNameRunning {
				activeCount++
			}
		}
	}

	return activeCount, nil
}

func launchInstance(ctx context.Context, ec2Client *ec2.Client, state *SweepRecord, config map[string]interface{}, paramIndex int) error {
	// Extract launch configuration
	instanceType := getStringParam(config, "instance_type", "t3.micro")
	ami := getStringParam(config, "ami", "")
	keyName := getStringParam(config, "key_name", "")
	iamRole := getStringParam(config, "iam_role", "spawnd-role")
	spot := getBoolParam(config, "spot", false)

	// Build RunInstances input
	input := &ec2.RunInstancesInput{
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: ec2types.InstanceType(instanceType),
		IamInstanceProfile: &ec2types.IamInstanceProfileSpecification{
			Name: aws.String(iamRole),
		},
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{Key: aws.String("spawn:sweep-id"), Value: aws.String(state.SweepID)},
					{Key: aws.String("spawn:sweep-index"), Value: aws.String(fmt.Sprintf("%d", paramIndex))},
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%d", state.SweepName, paramIndex))},
				},
			},
		},
	}

	if ami != "" {
		input.ImageId = aws.String(ami)
	}

	if keyName != "" {
		input.KeyName = aws.String(keyName)
	}

	if spot {
		input.InstanceMarketOptions = &ec2types.InstanceMarketOptionsRequest{
			MarketType: ec2types.MarketTypeSpot,
		}
	}

	// Launch
	result, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("run instances failed: %w", err)
	}

	instanceID := *result.Instances[0].InstanceId
	log.Printf("Launched instance %s for param %d", instanceID, paramIndex)

	// Record instance
	state.Instances = append(state.Instances, SweepInstance{
		Index:      paramIndex,
		InstanceID: instanceID,
		State:      "pending",
		LaunchedAt: time.Now().Format(time.RFC3339),
	})

	return nil
}

func reinvokeSelf(ctx context.Context, sweepID string) error {
	functionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	if functionName == "" {
		return fmt.Errorf("AWS_LAMBDA_FUNCTION_NAME not set")
	}

	payload, _ := json.Marshal(SweepEvent{
		SweepID:       sweepID,
		ForceDownload: false,
	})

	_, err := lambdaClient.Invoke(ctx, &lambdasvc.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: "Event", // Async invocation
		Payload:        payload,
	})

	return err
}

// Helper functions

func mergeParams(defaults, params map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range params {
		result[k] = v
	}
	return result
}

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
