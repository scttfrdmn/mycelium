package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
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
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
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

// RegionalOrchestrator manages EC2 clients for multi-region sweeps
type RegionalOrchestrator struct {
	ec2Clients   map[string]*ec2.Client
	accountID    string
	credentials  *ststypes.Credentials
	mu           sync.Mutex
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

	// Route to appropriate polling loop based on multi-region flag
	if state.MultiRegion && len(state.RegionStatus) > 0 {
		log.Printf("Starting multi-region sweep across %d regions", len(state.RegionStatus))
		orchestrator, err := initializeRegionalOrchestrator(ctx, state)
		if err != nil {
			return fmt.Errorf("failed to initialize regional orchestrator: %w", err)
		}
		return runMultiRegionPollingLoop(ctx, state, params, orchestrator, event.SweepID)
	}

	// Legacy single-region path
	log.Printf("Starting single-region sweep in %s", state.Region)
	ec2Client, err := createCrossAccountEC2Client(ctx, state.Region, state.AWSAccountID)
	if err != nil {
		return fmt.Errorf("failed to create cross-account EC2 client: %w", err)
	}
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// initializeRegionalOrchestrator creates EC2 clients for all regions in the sweep
func initializeRegionalOrchestrator(ctx context.Context, state *SweepRecord) (*RegionalOrchestrator, error) {
	// Assume cross-account role once
	roleARN := fmt.Sprintf(crossAccountRole, state.AWSAccountID)
	log.Printf("Assuming role: %s", roleARN)

	result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String("spawn-sweep-orchestrator"),
		DurationSeconds: aws.Int32(3600),
	})
	if err != nil {
		return nil, fmt.Errorf("assume role failed: %w", err)
	}

	orchestrator := &RegionalOrchestrator{
		ec2Clients:  make(map[string]*ec2.Client),
		accountID:   state.AWSAccountID,
		credentials: result.Credentials,
	}

	// Create EC2 client for each region
	for region := range state.RegionStatus {
		client, err := createEC2ClientWithCredentials(ctx, region, result.Credentials)
		if err != nil {
			log.Printf("WARNING: Failed to create EC2 client for region %s: %v", region, err)
			log.Printf("Region %s may be restricted or unavailable. Params in this region will be marked as failed.", region)
			// Mark all params in this region as failed
			if err := markRegionParamsFailed(ctx, state, region, fmt.Sprintf("Region unavailable: %v", err)); err != nil {
				log.Printf("Failed to mark region %s params as failed: %v", region, err)
			}
			continue
		}

		// Validate region access with a simple API call
		_, err = client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("region-name"),
					Values: []string{region},
				},
			},
		})
		if err != nil {
			log.Printf("WARNING: Region %s is not accessible: %v", region, err)
			log.Printf("This may be due to account restrictions or SCP policies. Params in this region will be marked as failed.", region)
			if err := markRegionParamsFailed(ctx, state, region, fmt.Sprintf("Region access denied: %v", err)); err != nil {
				log.Printf("Failed to mark region %s params as failed: %v", region, err)
			}
			continue
		}

		orchestrator.ec2Clients[region] = client
		log.Printf("Initialized EC2 client for region %s", region)
	}

	if len(orchestrator.ec2Clients) == 0 {
		return nil, fmt.Errorf("no accessible regions found")
	}

	log.Printf("Successfully initialized %d regional EC2 clients", len(orchestrator.ec2Clients))
	return orchestrator, nil
}

// createEC2ClientWithCredentials creates an EC2 client for a specific region using provided credentials
func createEC2ClientWithCredentials(ctx context.Context, region string, creds *ststypes.Credentials) (*ec2.Client, error) {
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

// markRegionParamsFailed marks all params in a region as failed
func markRegionParamsFailed(ctx context.Context, state *SweepRecord, region, errorMsg string) error {
	regionStatus := state.RegionStatus[region]
	if regionStatus == nil {
		return nil
	}

	// Mark all pending params in this region as failed
	for _, paramIndex := range regionStatus.NextToLaunch {
		state.Instances = append(state.Instances, SweepInstance{
			Index:        paramIndex,
			Region:       region,
			State:        "failed",
			ErrorMessage: errorMsg,
			LaunchedAt:   time.Now().Format(time.RFC3339),
		})
		state.Failed++
		regionStatus.Failed++
	}

	// Clear the queue since all are now failed
	regionStatus.NextToLaunch = []int{}
	return saveSweepState(ctx, state)
}

// runMultiRegionPollingLoop orchestrates launches across multiple regions
func runMultiRegionPollingLoop(ctx context.Context, state *SweepRecord, params *ParamFileFormat, orchestrator *RegionalOrchestrator, sweepID string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(maxExecutionDur)
	launchDelay := parseDuration(state.LaunchDelay)

	for {
		// Reload state from DynamoDB to check for cancellation
		currentState, err := loadSweepState(ctx, state.SweepID)
		if err != nil {
			log.Printf("Failed to reload sweep state: %v", err)
		} else {
			state = currentState // Update local state
			if currentState.CancelRequested {
				log.Println("Cancellation requested, stopping orchestration")
				state.Status = "CANCELLED"
				state.CompletedAt = time.Now().Format(time.RFC3339)
				if err := saveSweepState(ctx, state); err != nil {
					log.Printf("Failed to save cancelled state: %v", err)
				}
				return nil
			}
		}

		// Check timeout
		if time.Now().After(deadline) {
			// Double-check status before reinvoking
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

		// Query active instances per region concurrently
		activeByRegion := queryActiveInstancesByRegion(ctx, orchestrator, state)

		// Calculate global capacity
		totalActive := 0
		for _, count := range activeByRegion {
			totalActive += count
		}
		globalAvailable := state.MaxConcurrent - totalActive

		log.Printf("Global: %d active, %d available (max: %d)", totalActive, globalAvailable, state.MaxConcurrent)

		// Launch instances if capacity available
		if globalAvailable > 0 {
			if err := launchAcrossRegions(ctx, orchestrator, state, params, activeByRegion, globalAvailable, launchDelay); err != nil {
				log.Printf("Error during launch: %v", err)
			}
		}

		// Check for completion
		allDone := true
		for _, regionStatus := range state.RegionStatus {
			if len(regionStatus.NextToLaunch) > 0 {
				allDone = false
				break
			}
		}

		if allDone && totalActive == 0 {
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

// queryActiveInstancesByRegion queries active instance counts per region concurrently
func queryActiveInstancesByRegion(ctx context.Context, orchestrator *RegionalOrchestrator, state *SweepRecord) map[string]int {
	activeByRegion := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for region, client := range orchestrator.ec2Clients {
		wg.Add(1)
		go func(r string, c *ec2.Client) {
			defer wg.Done()
			count, err := countActiveInstancesInRegion(ctx, c, state, r)
			if err != nil {
				log.Printf("Failed to query instances in %s: %v", r, err)
				return
			}
			mu.Lock()
			activeByRegion[r] = count
			// Update region status
			if state.RegionStatus[r] != nil {
				state.RegionStatus[r].ActiveCount = count
			}
			mu.Unlock()
		}(region, client)
	}
	wg.Wait()

	return activeByRegion
}

// countActiveInstancesInRegion counts active instances in a specific region
func countActiveInstancesInRegion(ctx context.Context, ec2Client *ec2.Client, state *SweepRecord, region string) (int, error) {
	// Build instance IDs list for this region
	var instanceIDs []string
	for _, inst := range state.Instances {
		if inst.Region == region && inst.InstanceID != "" && (inst.State == "pending" || inst.State == "running") {
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

// launchAcrossRegions launches instances across regions with fair distribution
func launchAcrossRegions(ctx context.Context, orchestrator *RegionalOrchestrator, state *SweepRecord, params *ParamFileFormat, activeByRegion map[string]int, globalAvailable int, launchDelay time.Duration) error {
	// Count regions with pending work
	regionsWithWork := 0
	for region, regionStatus := range state.RegionStatus {
		if _, hasClient := orchestrator.ec2Clients[region]; hasClient && len(regionStatus.NextToLaunch) > 0 {
			regionsWithWork++
		}
	}

	if regionsWithWork == 0 {
		return nil
	}

	// Calculate fair share per region
	fairShare := max(1, globalAvailable/regionsWithWork)
	log.Printf("Fair share: %d instances per region (%d regions with work)", fairShare, regionsWithWork)

	// Sort regions for deterministic ordering
	regions := make([]string, 0, len(orchestrator.ec2Clients))
	for region := range orchestrator.ec2Clients {
		if len(state.RegionStatus[region].NextToLaunch) > 0 {
			regions = append(regions, region)
		}
	}
	sort.Strings(regions)

	// Launch instances concurrently per region
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, region := range regions {
		regionStatus := state.RegionStatus[region]
		client := orchestrator.ec2Clients[region]

		if len(regionStatus.NextToLaunch) == 0 {
			continue
		}

		wg.Add(1)
		go func(r string, rs *RegionProgress, c *ec2.Client) {
			defer wg.Done()

			toLaunch := min(fairShare, len(rs.NextToLaunch))
			log.Printf("Region %s: launching %d instances", r, toLaunch)

			for i := 0; i < toLaunch; i++ {
				if len(rs.NextToLaunch) == 0 {
					break
				}

				paramIndex := rs.NextToLaunch[0]
				paramSet := params.Params[paramIndex]

				// Merge defaults with param set
				config := mergeParams(params.Defaults, paramSet)

				// Launch instance in this region
				if err := launchInstanceInRegion(ctx, c, state, config, paramIndex, r); err != nil {
					log.Printf("Failed to launch instance %d in %s: %v", paramIndex, r, err)
					mu.Lock()
					state.Failed++
					rs.Failed++
					state.Instances = append(state.Instances, SweepInstance{
						Index:        paramIndex,
						Region:       r,
						State:        "failed",
						ErrorMessage: err.Error(),
						LaunchedAt:   time.Now().Format(time.RFC3339),
					})
					mu.Unlock()
				} else {
					mu.Lock()
					state.Launched++
					rs.Launched++
					mu.Unlock()
				}

				// Remove from queue
				mu.Lock()
				rs.NextToLaunch = rs.NextToLaunch[1:]
				mu.Unlock()

				// Delay between launches
				if launchDelay > 0 && i < toLaunch-1 {
					time.Sleep(launchDelay)
				}
			}
		}(region, regionStatus, client)
	}

	wg.Wait()

	// Save state after batch
	return saveSweepState(ctx, state)
}

// launchInstanceInRegion launches an instance in a specific region
func launchInstanceInRegion(ctx context.Context, ec2Client *ec2.Client, state *SweepRecord, config map[string]interface{}, paramIndex int, region string) error {
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
	log.Printf("Launched instance %s in %s for param %d", instanceID, region, paramIndex)

	// Record instance
	state.Instances = append(state.Instances, SweepInstance{
		Index:      paramIndex,
		Region:     region,
		InstanceID: instanceID,
		State:      "pending",
		LaunchedAt: time.Now().Format(time.RFC3339),
	})

	return nil
}
