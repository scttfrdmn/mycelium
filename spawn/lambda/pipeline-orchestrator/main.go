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
	"github.com/scttfrdmn/mycelium/spawn/pkg/pipeline"
)

const (
	defaultTableName = "spawn-pipeline-orchestration"
	maxExecutionDur  = 13 * time.Minute
	pollInterval     = 10 * time.Second
)

// PipelineEvent is the Lambda input event
type PipelineEvent struct {
	PipelineID string `json:"pipeline_id"`
}

var (
	awsCfg         aws.Config
	dynamodbClient *dynamodb.Client
	s3Client       *s3.Client
	lambdaClient   *lambdasvc.Client
	tableName      string
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

	tableName = getEnv("SPAWN_PIPELINE_ORCHESTRATION_TABLE", defaultTableName)
	log.Printf("Configuration: table=%s", tableName)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event PipelineEvent) error {
	log.Printf("Starting pipeline orchestration for pipeline_id=%s", event.PipelineID)

	// Load state from DynamoDB
	state, err := loadPipelineState(ctx, event.PipelineID)
	if err != nil {
		return fmt.Errorf("failed to load pipeline state: %w", err)
	}

	// Download pipeline definition from S3
	pipelineDef, err := downloadPipelineDefinition(ctx, state.S3ConfigKey)
	if err != nil {
		return fmt.Errorf("failed to download pipeline definition: %w", err)
	}

	// Check if pipeline is already in terminal state
	if state.Status == pipeline.StatusCompleted || state.Status == pipeline.StatusFailed || state.Status == pipeline.StatusCancelled {
		log.Printf("Pipeline already in terminal state: %s. Exiting without reinvocation.", state.Status)
		return nil
	}

	// Setup resources if INITIALIZING
	if state.Status == pipeline.StatusInitializing {
		log.Println("Setting up pipeline resources...")
		if err := setupPipelineResources(ctx, state, pipelineDef); err != nil {
			state.Status = pipeline.StatusFailed
			state.CompletedAt = timePtr(time.Now())
			savePipelineState(ctx, state)
			return fmt.Errorf("failed to setup resources: %w", err)
		}
		state.Status = pipeline.StatusRunning
		if err := savePipelineState(ctx, state); err != nil {
			return fmt.Errorf("failed to update status to RUNNING: %w", err)
		}
	}

	// Run polling loop
	return runPipelinePollingLoop(ctx, state, pipelineDef, event.PipelineID)
}

func loadPipelineState(ctx context.Context, pipelineID string) (*pipeline.PipelineState, error) {
	result, err := dynamodbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pipeline_id": &types.AttributeValueMemberS{Value: pipelineID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dynamodb get failed: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("pipeline %s not found", pipelineID)
	}

	var state pipeline.PipelineState
	if err := attributevalue.UnmarshalMap(result.Item, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

func savePipelineState(ctx context.Context, state *pipeline.PipelineState) error {
	state.UpdatedAt = time.Now()

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

func downloadPipelineDefinition(ctx context.Context, s3Key string) (*pipeline.Pipeline, error) {
	// Parse S3 key (format: s3://bucket/key)
	parts := strings.SplitN(strings.TrimPrefix(s3Key, "s3://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid S3 key format: %s", s3Key)
	}
	bucket, key := parts[0], parts[1]

	// Check cache in /tmp
	cacheFile := fmt.Sprintf("/tmp/%s", strings.ReplaceAll(key, "/", "_"))
	if data, err := os.ReadFile(cacheFile); err == nil {
		log.Printf("Using cached pipeline definition from %s", cacheFile)
		var p pipeline.Pipeline
		if err := json.Unmarshal(data, &p); err == nil {
			return &p, nil
		}
	}

	// Download from S3
	log.Printf("Downloading pipeline definition from s3://%s/%s", bucket, key)
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get failed: %w", err)
	}
	defer result.Body.Close()

	var p pipeline.Pipeline
	if err := json.NewDecoder(result.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %w", err)
	}

	// Cache to /tmp
	if data, err := json.Marshal(p); err == nil {
		os.WriteFile(cacheFile, data, 0644)
	}

	return &p, nil
}

func setupPipelineResources(ctx context.Context, state *pipeline.PipelineState, pipelineDef *pipeline.Pipeline) error {
	// Create security group if streaming stages exist
	if pipelineDef.HasStreamingStages() {
		log.Println("Pipeline has streaming stages, creating security group...")
		sgID, err := createStreamingSecurityGroup(ctx, state.PipelineID)
		if err != nil {
			return fmt.Errorf("create security group: %w", err)
		}
		state.SecurityGroupID = sgID
		log.Printf("Created security group: %s", sgID)
	}

	// Create placement group if EFA stages exist
	if pipelineDef.HasEFAStages() {
		log.Println("Pipeline has EFA stages, creating placement group...")
		pgID, err := createPlacementGroup(ctx, state.PipelineID)
		if err != nil {
			return fmt.Errorf("create placement group: %w", err)
		}
		state.PlacementGroupID = pgID
		log.Printf("Created placement group: %s", pgID)
	}

	return nil
}

func createStreamingSecurityGroup(ctx context.Context, pipelineID string) (string, error) {
	// Get default region EC2 client
	ec2Client := ec2.NewFromConfig(awsCfg)

	sgName := fmt.Sprintf("spawn-pipeline-%s", pipelineID)
	description := fmt.Sprintf("Security group for spawn pipeline %s", pipelineID)

	// Create security group
	createResult, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String(description),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSecurityGroup,
				Tags: []ec2types.Tag{
					{Key: aws.String("spawn:pipeline-id"), Value: aws.String(pipelineID)},
					{Key: aws.String("Name"), Value: aws.String(sgName)},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create security group failed: %w", err)
	}

	sgID := *createResult.GroupId

	// Add ingress rules: allow all TCP 50000-60000 between pipeline instances (self-referential)
	_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(50000),
				ToPort:     aws.Int32(60000),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{GroupId: aws.String(sgID)}, // Self-referential
				},
			},
			// SSH access
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []ec2types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("authorize ingress failed: %w", err)
	}

	return sgID, nil
}

func createPlacementGroup(ctx context.Context, pipelineID string) (string, error) {
	ec2Client := ec2.NewFromConfig(awsCfg)

	pgName := fmt.Sprintf("spawn-pipeline-%s", pipelineID)

	_, err := ec2Client.CreatePlacementGroup(ctx, &ec2.CreatePlacementGroupInput{
		GroupName: aws.String(pgName),
		Strategy:  ec2types.PlacementStrategyCluster,
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypePlacementGroup,
				Tags: []ec2types.Tag{
					{Key: aws.String("spawn:pipeline-id"), Value: aws.String(pipelineID)},
					{Key: aws.String("Name"), Value: aws.String(pgName)},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create placement group failed: %w", err)
	}

	return pgName, nil
}

func runPipelinePollingLoop(ctx context.Context, state *pipeline.PipelineState, pipelineDef *pipeline.Pipeline, pipelineID string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(maxExecutionDur)

	// Get topological order
	order, err := pipelineDef.GetTopologicalOrder()
	if err != nil {
		return fmt.Errorf("get topological order: %w", err)
	}

	for {
		// Reload state to check for cancellation
		currentState, err := loadPipelineState(ctx, state.PipelineID)
		if err != nil {
			log.Printf("Failed to reload pipeline state: %v", err)
		} else if currentState.CancelRequested {
			log.Println("Cancellation requested, stopping orchestration")
			state.Status = pipeline.StatusCancelled
			state.CompletedAt = timePtr(time.Now())
			if err := savePipelineState(ctx, state); err != nil {
				log.Printf("Failed to save cancelled state: %v", err)
			}
			cleanupPipelineResources(ctx, state)
			return nil
		}

		// Check timeout
		if time.Now().After(deadline) {
			if state.Status == pipeline.StatusCompleted || state.Status == pipeline.StatusFailed || state.Status == pipeline.StatusCancelled {
				log.Printf("Pipeline in terminal state %s, not reinvoking", state.Status)
				return nil
			}

			log.Println("Approaching Lambda timeout, re-invoking...")
			if err := savePipelineState(ctx, state); err != nil {
				log.Printf("Failed to save state before re-invocation: %v", err)
			}
			return reinvokeSelf(ctx, pipelineID)
		}

		// Process stages in topological order
		madeProgress := false
		for _, stageID := range order {
			stageState := getStageState(state, stageID)
			if stageState == nil {
				log.Printf("Warning: stage %s not found in state", stageID)
				continue
			}

			stageDef := pipelineDef.GetStage(stageID)
			if stageDef == nil {
				log.Printf("Warning: stage %s not found in definition", stageID)
				continue
			}

			switch stageState.Status {
			case pipeline.StageStatusPending:
				// Check if dependencies are met
				if dependenciesMet(state, stageDef) {
					log.Printf("Stage %s dependencies met, marking as ready", stageID)
					stageState.Status = pipeline.StageStatusReady
					madeProgress = true
				}

			case pipeline.StageStatusReady:
				// Launch stage
				log.Printf("Launching stage %s", stageID)
				if err := launchStage(ctx, state, stageDef, stageState); err != nil {
					log.Printf("Failed to launch stage %s: %v", stageID, err)
					stageState.Status = pipeline.StageStatusFailed
					stageState.ErrorMessage = err.Error()
					stageState.CompletedAt = timePtr(time.Now())
					state.FailedStages++

					// Check on_failure policy
					if state.OnFailure == "stop" {
						log.Printf("Stage %s failed and on_failure=stop, marking remaining stages as skipped", stageID)
						skipRemainingStages(state, order, stageID)
					}
					madeProgress = true
				} else {
					stageState.Status = pipeline.StageStatusLaunching
					stageState.LaunchedAt = timePtr(time.Now())
					madeProgress = true
				}

			case pipeline.StageStatusLaunching:
				// Check if instances are running
				if allInstancesRunning(ctx, stageState) {
					log.Printf("Stage %s instances all running", stageID)
					stageState.Status = pipeline.StageStatusRunning
					madeProgress = true
				}

			case pipeline.StageStatusRunning:
				// Check if stage completed
				completed, err := checkStageCompletion(ctx, state, stageState)
				if err != nil {
					log.Printf("Error checking stage %s completion: %v", stageID, err)
				} else if completed {
					log.Printf("Stage %s completed", stageID)
					stageState.Status = pipeline.StageStatusCompleted
					stageState.CompletedAt = timePtr(time.Now())
					state.CompletedStages++

					// Calculate cost
					calculateStageCost(stageState)
					state.CurrentCostUSD += stageState.StageCostUSD

					// Check budget
					if state.MaxCostUSD != nil && state.CurrentCostUSD > *state.MaxCostUSD {
						log.Printf("Budget exceeded: $%.2f > $%.2f", state.CurrentCostUSD, *state.MaxCostUSD)
						state.Status = pipeline.StatusCancelled
						state.CompletedAt = timePtr(time.Now())
						skipRemainingStages(state, order, stageID)
						break
					}

					madeProgress = true
				}
			}
		}

		// Save state if progress made
		if madeProgress {
			if err := savePipelineState(ctx, state); err != nil {
				log.Printf("Failed to save state: %v", err)
			}
		}

		// Check for completion
		if allStagesComplete(state) {
			log.Println("All stages completed")
			if state.FailedStages > 0 {
				state.Status = pipeline.StatusFailed
			} else {
				state.Status = pipeline.StatusCompleted
			}
			state.CompletedAt = timePtr(time.Now())
			if err := savePipelineState(ctx, state); err != nil {
				return fmt.Errorf("failed to save completion state: %w", err)
			}
			cleanupPipelineResources(ctx, state)
			return nil
		}

		// Wait for next poll
		<-ticker.C
	}
}

func getStageState(state *pipeline.PipelineState, stageID string) *pipeline.StageState {
	for i := range state.Stages {
		if state.Stages[i].StageID == stageID {
			return &state.Stages[i]
		}
	}
	return nil
}

func dependenciesMet(state *pipeline.PipelineState, stageDef *pipeline.Stage) bool {
	for _, depID := range stageDef.DependsOn {
		depState := getStageState(state, depID)
		if depState == nil || depState.Status != pipeline.StageStatusCompleted {
			return false
		}
	}
	return true
}

func launchStage(ctx context.Context, state *pipeline.PipelineState, stageDef *pipeline.Stage, stageState *pipeline.StageState) error {
	// Get EC2 client for stage's region
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(stageDef.Region))
	if err != nil {
		return fmt.Errorf("load config for region %s: %w", stageDef.Region, err)
	}
	ec2Client := ec2.NewFromConfig(cfg)

	// Launch instances for this stage
	for i := 0; i < stageDef.InstanceCount; i++ {
		instanceInfo, err := launchStageInstance(ctx, ec2Client, state, stageDef, stageState, i)
		if err != nil {
			return fmt.Errorf("launch instance %d: %w", i, err)
		}
		stageState.Instances = append(stageState.Instances, *instanceInfo)
		log.Printf("Launched instance %s for stage %s (index %d)", instanceInfo.InstanceID, stageDef.StageID, i)
	}

	return nil
}

func launchStageInstance(ctx context.Context, ec2Client *ec2.Client, state *pipeline.PipelineState, stageDef *pipeline.Stage, stageState *pipeline.StageState, index int) (*pipeline.InstanceInfo, error) {
	// Build RunInstances input
	input := &ec2.RunInstancesInput{
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: ec2types.InstanceType(stageDef.InstanceType),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{Key: aws.String("spawn:pipeline-id"), Value: aws.String(state.PipelineID)},
					{Key: aws.String("spawn:stage-id"), Value: aws.String(stageDef.StageID)},
					{Key: aws.String("spawn:stage-index"), Value: aws.String(fmt.Sprintf("%d", stageState.StageIndex))},
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("%s-%s-%d", state.PipelineName, stageDef.StageID, index))},
				},
			},
		},
	}

	// Add AMI if specified
	if stageDef.AMI != "" {
		input.ImageId = aws.String(stageDef.AMI)
	}

	// Add spot if specified
	if stageDef.Spot {
		input.InstanceMarketOptions = &ec2types.InstanceMarketOptionsRequest{
			MarketType: ec2types.MarketTypeSpot,
		}
	}

	// Add security group if streaming
	if state.SecurityGroupID != "" {
		input.SecurityGroupIds = []string{state.SecurityGroupID}
	}

	// Add placement group if EFA
	if stageDef.EFAEnabled && state.PlacementGroupID != "" {
		input.Placement = &ec2types.Placement{
			GroupName: aws.String(state.PlacementGroupID),
		}
	}

	// Launch
	result, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("run instances failed: %w", err)
	}

	instance := result.Instances[0]
	instanceInfo := &pipeline.InstanceInfo{
		InstanceID: *instance.InstanceId,
		State:      string(instance.State.Name),
		LaunchedAt: time.Now(),
	}

	if instance.PrivateIpAddress != nil {
		instanceInfo.PrivateIP = *instance.PrivateIpAddress
	}
	if instance.PublicIpAddress != nil {
		instanceInfo.PublicIP = *instance.PublicIpAddress
	}

	// Generate DNS name
	instanceInfo.DNSName = fmt.Sprintf("%s-%d.%s.spore.host", stageDef.StageID, index, state.PipelineID)

	return instanceInfo, nil
}

func allInstancesRunning(ctx context.Context, stageState *pipeline.StageState) bool {
	// TODO: Query EC2 to check instance states
	// For now, simple heuristic: if launched > 1 minute ago, assume running
	if stageState.LaunchedAt == nil {
		return false
	}
	return time.Since(*stageState.LaunchedAt) > 1*time.Minute
}

func checkStageCompletion(ctx context.Context, state *pipeline.PipelineState, stageState *pipeline.StageState) (bool, error) {
	// Check for S3 completion marker
	markerKey := fmt.Sprintf("%s/stages/%s/COMPLETE", state.S3Prefix, stageState.StageID)

	_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(state.S3Bucket),
		Key:    aws.String(markerKey),
	})
	if err == nil {
		return true, nil // Marker exists
	}

	// Check if all instances terminated (alternative completion signal)
	allTerminated := true
	for _, inst := range stageState.Instances {
		if inst.State != "terminated" && inst.State != "stopped" {
			allTerminated = false
			break
		}
	}

	return allTerminated, nil
}

func calculateStageCost(stageState *pipeline.StageState) {
	if stageState.LaunchedAt == nil || stageState.CompletedAt == nil {
		return
	}

	hours := stageState.CompletedAt.Sub(*stageState.LaunchedAt).Hours()
	instanceHours := hours * float64(len(stageState.Instances))
	stageState.InstanceHours = instanceHours

	// Simplified cost calculation (use $0.10/hour as default)
	stageState.StageCostUSD = instanceHours * 0.10
}

func skipRemainingStages(state *pipeline.PipelineState, order []string, failedStageID string) {
	found := false
	for _, stageID := range order {
		if stageID == failedStageID {
			found = true
			continue
		}
		if found {
			stageState := getStageState(state, stageID)
			if stageState != nil && stageState.Status == pipeline.StageStatusPending {
				stageState.Status = pipeline.StageStatusSkipped
			}
		}
	}
}

func allStagesComplete(state *pipeline.PipelineState) bool {
	for _, stage := range state.Stages {
		if stage.Status != pipeline.StageStatusCompleted &&
			stage.Status != pipeline.StageStatusFailed &&
			stage.Status != pipeline.StageStatusSkipped {
			return false
		}
	}
	return true
}

func cleanupPipelineResources(ctx context.Context, state *pipeline.PipelineState) {
	// Cleanup security group
	if state.SecurityGroupID != "" {
		go func() {
			time.Sleep(30 * time.Second) // Wait for instances to terminate
			ec2Client := ec2.NewFromConfig(awsCfg)
			_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
				GroupId: aws.String(state.SecurityGroupID),
			})
			if err != nil {
				log.Printf("Warning: Failed to delete security group %s: %v", state.SecurityGroupID, err)
			} else {
				log.Printf("Deleted security group: %s", state.SecurityGroupID)
			}
		}()
	}

	// Cleanup placement group
	if state.PlacementGroupID != "" {
		go func() {
			time.Sleep(30 * time.Second)
			ec2Client := ec2.NewFromConfig(awsCfg)
			_, err := ec2Client.DeletePlacementGroup(ctx, &ec2.DeletePlacementGroupInput{
				GroupName: aws.String(state.PlacementGroupID),
			})
			if err != nil {
				log.Printf("Warning: Failed to delete placement group %s: %v", state.PlacementGroupID, err)
			} else {
				log.Printf("Deleted placement group: %s", state.PlacementGroupID)
			}
		}()
	}
}

func reinvokeSelf(ctx context.Context, pipelineID string) error {
	functionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	if functionName == "" {
		return fmt.Errorf("AWS_LAMBDA_FUNCTION_NAME not set")
	}

	payload, _ := json.Marshal(PipelineEvent{
		PipelineID: pipelineID,
	})

	_, err := lambdaClient.Invoke(ctx, &lambdasvc.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: "Event", // Async invocation
		Payload:        payload,
	})

	return err
}

func timePtr(t time.Time) *time.Time {
	return &t
}
