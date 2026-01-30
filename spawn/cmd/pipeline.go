package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	lambdasvc "github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/mycelium/spawn/pkg/pipeline"
	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage multi-stage pipelines",
	Long: `Manage multi-stage pipelines with DAG dependencies.

Pipelines allow you to orchestrate complex workflows where each stage runs on
separate instances with different instance types. Stages can depend on each other,
forming a directed acyclic graph (DAG).

Data can be passed between stages via:
- S3 (batch mode): Stage outputs uploaded to S3, downloaded by next stage
- Network streaming (real-time): Direct TCP/gRPC connections between stages`,
}

var validatePipelineCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a pipeline definition",
	Long: `Validate a pipeline definition file.

Checks:
- JSON syntax and structure
- Required fields present
- Stage dependencies valid (no circular dependencies)
- Instance types, regions, and other configuration valid`,
	Args: cobra.ExactArgs(1),
	RunE: runValidatePipeline,
}

var graphPipelineCmd = &cobra.Command{
	Use:   "graph <file>",
	Short: "Display pipeline DAG as ASCII art",
	Long: `Display the pipeline dependency graph as ASCII art.

Shows:
- Stage names and instance types
- Dependencies between stages
- Fan-out and fan-in patterns
- Data passing modes (S3 or streaming)`,
	Args: cobra.ExactArgs(1),
	RunE: runGraphPipeline,
}

var launchPipelineCmd = &cobra.Command{
	Use:   "launch <file>",
	Short: "Launch a pipeline",
	Long: `Launch a multi-stage pipeline.

The pipeline definition will be uploaded to S3 and a Lambda orchestrator
will be invoked to manage the pipeline execution.`,
	Args: cobra.ExactArgs(1),
	RunE: runLaunchPipeline,
}

var statusPipelineCmd = &cobra.Command{
	Use:   "status <pipeline-id>",
	Short: "Show pipeline status",
	Long: `Show the current status of a running or completed pipeline.

Displays:
- Overall pipeline status
- Per-stage progress
- Instance information
- Cost tracking`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusPipeline,
}

var collectPipelineCmd = &cobra.Command{
	Use:   "collect <pipeline-id>",
	Short: "Download pipeline results",
	Long: `Download all results from a completed pipeline.

Downloads outputs from all stages to a local directory.`,
	Args: cobra.ExactArgs(1),
	RunE: runCollectPipeline,
}

var (
	flagSimpleGraph bool
	flagGraphStats  bool
	flagJSONOutput  bool
	flagDetached    bool
	flagWait        bool
	flagRegion      string
	flagOutputDir   string
	flagStage       string
)

func init() {
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(validatePipelineCmd)
	pipelineCmd.AddCommand(graphPipelineCmd)
	pipelineCmd.AddCommand(launchPipelineCmd)
	pipelineCmd.AddCommand(statusPipelineCmd)
	pipelineCmd.AddCommand(collectPipelineCmd)

	// Graph command flags
	graphPipelineCmd.Flags().BoolVar(&flagSimpleGraph, "simple", false, "Show simplified graph")
	graphPipelineCmd.Flags().BoolVar(&flagGraphStats, "stats", false, "Show graph statistics")
	graphPipelineCmd.Flags().BoolVar(&flagJSONOutput, "json", false, "Output as JSON")

	// Launch command flags
	launchPipelineCmd.Flags().BoolVar(&flagDetached, "detached", false, "Launch and return immediately")
	launchPipelineCmd.Flags().BoolVar(&flagWait, "wait", false, "Wait for pipeline to complete")
	launchPipelineCmd.Flags().StringVar(&flagRegion, "region", "", "AWS region (default: from AWS config)")

	// Collect command flags
	collectPipelineCmd.Flags().StringVar(&flagOutputDir, "output", "./results", "Output directory for downloaded files")
	collectPipelineCmd.Flags().StringVar(&flagStage, "stage", "", "Download results from specific stage only")
}

func runValidatePipeline(cmd *cobra.Command, args []string) error {
	file := args[0]

	// Load pipeline
	p, err := pipeline.LoadPipelineFromFile(file)
	if err != nil {
		return fmt.Errorf("load pipeline: %w", err)
	}

	// Validate
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Fprintf(os.Stdout, "✓ Pipeline is valid\n\n")
	fmt.Fprintf(os.Stdout, "Pipeline: %s\n", p.PipelineName)
	fmt.Fprintf(os.Stdout, "ID: %s\n", p.PipelineID)
	fmt.Fprintf(os.Stdout, "Stages: %d\n", len(p.Stages))
	fmt.Fprintf(os.Stdout, "S3 Bucket: %s\n", p.S3Bucket)

	// Show topological order
	order, err := p.GetTopologicalOrder()
	if err != nil {
		return fmt.Errorf("get topological order: %w", err)
	}
	fmt.Fprintf(os.Stdout, "\nExecution order:\n")
	for i, stageID := range order {
		fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, stageID)
	}

	// Show features
	fmt.Fprintf(os.Stdout, "\nFeatures:\n")
	if p.HasStreamingStages() {
		fmt.Fprintf(os.Stdout, "  • Network streaming enabled\n")
	}
	if p.HasEFAStages() {
		fmt.Fprintf(os.Stdout, "  • EFA (Elastic Fabric Adapter) enabled\n")
	}
	if p.OnFailure == "stop" {
		fmt.Fprintf(os.Stdout, "  • Stops on first failure\n")
	} else {
		fmt.Fprintf(os.Stdout, "  • Continues on failure\n")
	}
	if p.MaxCostUSD != nil {
		fmt.Fprintf(os.Stdout, "  • Budget limit: $%.2f\n", *p.MaxCostUSD)
	}

	return nil
}

func runGraphPipeline(cmd *cobra.Command, args []string) error {
	file := args[0]

	// Load pipeline
	p, err := pipeline.LoadPipelineFromFile(file)
	if err != nil {
		return fmt.Errorf("load pipeline: %w", err)
	}

	// Validate
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// JSON output
	if flagJSONOutput {
		stats := p.GetGraphStats()
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Fprintf(os.Stdout, "%s\n", data)
		return nil
	}

	// Show statistics
	if flagGraphStats {
		stats := p.GetGraphStats()
		fmt.Fprintf(os.Stdout, "Pipeline Statistics\n")
		fmt.Fprintf(os.Stdout, "═══════════════════\n\n")
		fmt.Fprintf(os.Stdout, "Total stages:     %d\n", stats["total_stages"])
		fmt.Fprintf(os.Stdout, "Total instances:  %d\n", stats["total_instances"])
		fmt.Fprintf(os.Stdout, "Max fan-out:      %d\n", stats["max_fan_out"])
		fmt.Fprintf(os.Stdout, "Max fan-in:       %d\n", stats["max_fan_in"])
		fmt.Fprintf(os.Stdout, "Has streaming:    %v\n", stats["has_streaming"])
		fmt.Fprintf(os.Stdout, "Has EFA:          %v\n\n", stats["has_efa"])
		return nil
	}

	// Render graph
	var graph string
	if flagSimpleGraph {
		graph, err = p.RenderSimpleGraph()
	} else {
		graph, err = p.RenderGraph()
	}
	if err != nil {
		return fmt.Errorf("render graph: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", graph)
	return nil
}

func runLaunchPipeline(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	file := args[0]

	// Load and validate pipeline
	p, err := pipeline.LoadPipelineFromFile(file)
	if err != nil {
		return fmt.Errorf("load pipeline: %w", err)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "🚀 Launching pipeline: %s\n", p.PipelineName)
	fmt.Fprintf(os.Stderr, "   Pipeline ID: %s\n", p.PipelineID)
	fmt.Fprintf(os.Stderr, "   Stages: %d\n\n", len(p.Stages))

	// Load AWS config
	region := flagRegion
	if region == "" {
		region = p.Stages[0].Region // Use first stage's region
		if region == "" {
			region = "us-east-1"
		}
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile("mycelium-infra"), // Infrastructure account
	)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	// Get user account ID
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("get caller identity: %w", err)
	}
	userAccountID := *identity.Account

	// Step 1: Upload pipeline definition to S3
	fmt.Fprintf(os.Stderr, "📤 Uploading pipeline definition to S3...\n")
	s3Client := s3.NewFromConfig(cfg)
	bucketName := fmt.Sprintf("spawn-pipelines-%s", region)
	s3Key := fmt.Sprintf("pipelines/%s/config.json", p.PipelineID)

	pipelineJSON, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pipeline: %w", err)
	}

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(pipelineJSON),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Uploaded to s3://%s/%s\n\n", bucketName, s3Key)

	// Step 2: Create DynamoDB record
	fmt.Fprintf(os.Stderr, "💾 Creating pipeline orchestration record...\n")
	dynamoClient := dynamodb.NewFromConfig(cfg)
	tableName := "spawn-pipeline-orchestration"

	// Build initial pipeline state
	pipelineState := map[string]interface{}{
		"pipeline_id":      p.PipelineID,
		"pipeline_name":    p.PipelineName,
		"user_id":          userAccountID,
		"created_at":       time.Now().UTC().Format(time.RFC3339),
		"updated_at":       time.Now().UTC().Format(time.RFC3339),
		"status":           "INITIALIZING",
		"s3_config_key":    fmt.Sprintf("s3://%s/%s", bucketName, s3Key),
		"s3_bucket":        p.S3Bucket,
		"s3_prefix":        p.S3Prefix,
		"result_s3_bucket": p.ResultS3Bucket,
		"result_s3_prefix": p.ResultS3Prefix,
		"on_failure":       p.OnFailure,
		"total_stages":     len(p.Stages),
		"completed_stages": 0,
		"failed_stages":    0,
		"current_cost_usd": 0.0,
	}
	if p.MaxCostUSD != nil {
		pipelineState["max_cost_usd"] = *p.MaxCostUSD
	}

	item, err := attributevalue.MarshalMap(pipelineState)
	if err != nil {
		return fmt.Errorf("marshal DynamoDB item: %w", err)
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("create DynamoDB record: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Created orchestration record\n\n")

	// Step 3: Invoke Lambda orchestrator
	fmt.Fprintf(os.Stderr, "⚡ Invoking pipeline orchestrator Lambda...\n")
	lambdaClient := lambdasvc.NewFromConfig(cfg)
	functionName := "spawn-pipeline-orchestrator"

	payload := map[string]string{
		"pipeline_id": p.PipelineID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal Lambda payload: %w", err)
	}

	_, err = lambdaClient.Invoke(ctx, &lambdasvc.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: lambdatypes.InvocationTypeEvent, // Asynchronous
		Payload:        payloadJSON,
	})
	if err != nil {
		return fmt.Errorf("invoke Lambda: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Lambda orchestrator invoked\n\n")

	fmt.Fprintf(os.Stderr, "✅ Pipeline launched successfully!\n\n")
	fmt.Fprintf(os.Stderr, "Pipeline ID: %s\n\n", p.PipelineID)
	fmt.Fprintf(os.Stderr, "To check status:\n")
	fmt.Fprintf(os.Stderr, "  spawn pipeline status %s\n\n", p.PipelineID)
	fmt.Fprintf(os.Stderr, "To collect results:\n")
	fmt.Fprintf(os.Stderr, "  spawn pipeline collect %s --output ./results/\n", p.PipelineID)

	// Output just the pipeline ID to stdout for scripting
	fmt.Fprintf(os.Stdout, "%s\n", p.PipelineID)

	return nil
}

func runStatusPipeline(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipelineID := args[0]

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile("mycelium-infra"),
	)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	// Query DynamoDB for pipeline state
	dynamoClient := dynamodb.NewFromConfig(cfg)
	tableName := "spawn-pipeline-orchestration"

	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pipeline_id": &types.AttributeValueMemberS{Value: pipelineID},
		},
	})
	if err != nil {
		return fmt.Errorf("query DynamoDB: %w", err)
	}

	if result.Item == nil {
		return fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Parse pipeline state
	var state map[string]interface{}
	err = attributevalue.UnmarshalMap(result.Item, &state)
	if err != nil {
		return fmt.Errorf("unmarshal pipeline state: %w", err)
	}

	// Display status
	fmt.Fprintf(os.Stdout, "Pipeline: %s\n", getStringField(state, "pipeline_name"))
	fmt.Fprintf(os.Stdout, "ID: %s\n", pipelineID)
	fmt.Fprintf(os.Stdout, "Status: %s\n", getStringField(state, "status"))
	fmt.Fprintf(os.Stdout, "Created: %s\n", getStringField(state, "created_at"))
	fmt.Fprintf(os.Stdout, "Updated: %s\n", getStringField(state, "updated_at"))
	fmt.Fprintf(os.Stdout, "\n")

	// Progress
	totalStages := getIntField(state, "total_stages")
	completedStages := getIntField(state, "completed_stages")
	failedStages := getIntField(state, "failed_stages")
	fmt.Fprintf(os.Stdout, "Progress: %d/%d stages completed", completedStages, totalStages)
	if failedStages > 0 {
		fmt.Fprintf(os.Stdout, " (%d failed)", failedStages)
	}
	fmt.Fprintf(os.Stdout, "\n")

	// Cost
	currentCost := getFloatField(state, "current_cost_usd")
	fmt.Fprintf(os.Stdout, "Cost: $%.2f", currentCost)
	if maxCost, ok := state["max_cost_usd"].(float64); ok && maxCost > 0 {
		fmt.Fprintf(os.Stdout, " / $%.2f", maxCost)
	}
	fmt.Fprintf(os.Stdout, "\n\n")

	// Stages table
	if stages, ok := state["stages"].([]interface{}); ok && len(stages) > 0 {
		fmt.Fprintf(os.Stdout, "STAGE                STATUS       INSTANCES  COST      DURATION\n")
		fmt.Fprintf(os.Stdout, "─────────────────────────────────────────────────────────────────\n")
		for _, stageVal := range stages {
			if stageMap, ok := stageVal.(map[string]interface{}); ok {
				stageID := getStringField(stageMap, "stage_id")
				status := getStringField(stageMap, "status")
				instanceCount := len(getSliceField(stageMap, "instances"))
				stageCost := getFloatField(stageMap, "stage_cost_usd")

				// Calculate duration
				duration := ""
				if launchedAt := getStringField(stageMap, "launched_at"); launchedAt != "" {
					if completedAt := getStringField(stageMap, "completed_at"); completedAt != "" {
						duration = formatDurationBetween(launchedAt, completedAt)
					} else {
						duration = formatDurationBetween(launchedAt, time.Now().UTC().Format(time.RFC3339))
					}
				}

				instanceStr := "-"
				if instanceCount > 0 {
					instanceStr = fmt.Sprintf("%d", instanceCount)
				}

				costStr := "-"
				if stageCost > 0 {
					costStr = fmt.Sprintf("$%.2f", stageCost)
				}

				fmt.Fprintf(os.Stdout, "%-20s %-12s %-10s %-9s %s\n",
					truncate(stageID, 20), status, instanceStr, costStr, duration)
			}
		}
	}

	return nil
}

// Helper functions for extracting fields
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloatField(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0.0
}

func getSliceField(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

func formatDurationBetween(start, end string) string {
	startTime, err1 := time.Parse(time.RFC3339, start)
	endTime, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return "-"
	}
	duration := endTime.Sub(startTime)
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	return fmt.Sprintf("%.1fh", duration.Hours())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func runCollectPipeline(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	pipelineID := args[0]

	fmt.Fprintf(os.Stderr, "📦 Collecting results for pipeline: %s\n", pipelineID)
	fmt.Fprintf(os.Stderr, "   Output directory: %s\n\n", flagOutputDir)

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile("mycelium-infra"),
	)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	// Query DynamoDB for pipeline state
	dynamoClient := dynamodb.NewFromConfig(cfg)
	tableName := "spawn-pipeline-orchestration"

	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pipeline_id": &types.AttributeValueMemberS{Value: pipelineID},
		},
	})
	if err != nil {
		return fmt.Errorf("query DynamoDB: %w", err)
	}

	if result.Item == nil {
		return fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Parse pipeline state
	var state map[string]interface{}
	err = attributevalue.UnmarshalMap(result.Item, &state)
	if err != nil {
		return fmt.Errorf("unmarshal pipeline state: %w", err)
	}

	// Get S3 result location
	resultBucket := getStringField(state, "result_s3_bucket")
	resultPrefix := getStringField(state, "result_s3_prefix")
	if resultBucket == "" {
		// Fall back to stage output locations
		resultBucket = getStringField(state, "s3_bucket")
		resultPrefix = fmt.Sprintf("%s/stages", getStringField(state, "s3_prefix"))
	}

	if resultBucket == "" {
		return fmt.Errorf("no result bucket configured for pipeline")
	}

	fmt.Fprintf(os.Stderr, "📍 Source: s3://%s/%s\n\n", resultBucket, resultPrefix)

	// Create output directory
	if err := os.MkdirAll(flagOutputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Download from S3
	s3Client := s3.NewFromConfig(cfg)

	// List objects
	fmt.Fprintf(os.Stderr, "🔍 Listing objects...\n")
	listOutput, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(resultBucket),
		Prefix: aws.String(resultPrefix),
	})
	if err != nil {
		return fmt.Errorf("list S3 objects: %w", err)
	}

	if len(listOutput.Contents) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  No results found\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "📥 Downloading %d objects...\n", len(listOutput.Contents))

	// Download each object
	downloadCount := 0
	for _, obj := range listOutput.Contents {
		// Skip directories (keys ending with /)
		if len(*obj.Key) > 0 && (*obj.Key)[len(*obj.Key)-1] == '/' {
			continue
		}

		// Filter by stage if specified
		if flagStage != "" {
			// Check if object key contains the stage ID
			if !contains(*obj.Key, fmt.Sprintf("/stages/%s/", flagStage)) {
				continue
			}
		}

		// Download to local file
		localPath := fmt.Sprintf("%s/%s", flagOutputDir, *obj.Key)
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", localPath, err)
		}

		getOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(resultBucket),
			Key:    obj.Key,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to download %s: %v\n", *obj.Key, err)
			continue
		}

		// Write to file
		file, err := os.Create(localPath)
		if err != nil {
			getOutput.Body.Close()
			return fmt.Errorf("create file %s: %w", localPath, err)
		}

		_, err = io.Copy(file, getOutput.Body)
		file.Close()
		getOutput.Body.Close()
		if err != nil {
			return fmt.Errorf("write file %s: %w", localPath, err)
		}

		downloadCount++
		fmt.Fprintf(os.Stderr, "   ✓ %s\n", *obj.Key)
	}

	fmt.Fprintf(os.Stderr, "\n✅ Downloaded %d files to %s\n", downloadCount, flagOutputDir)

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexString(s, substr) >= 0)
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
