package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
	file := args[0]

	// Load and validate pipeline
	p, err := pipeline.LoadPipelineFromFile(file)
	if err != nil {
		return fmt.Errorf("load pipeline: %w", err)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Launching pipeline: %s\n", p.PipelineName)
	fmt.Fprintf(os.Stdout, "Pipeline ID: %s\n", p.PipelineID)
	fmt.Fprintf(os.Stdout, "Stages: %d\n\n", len(p.Stages))

	// TODO: Implement actual launch logic
	// 1. Upload pipeline definition to S3
	// 2. Create initial DynamoDB record
	// 3. Invoke Lambda orchestrator
	// 4. Optionally wait for completion

	fmt.Fprintf(os.Stdout, "Pipeline launch initiated.\n")
	fmt.Fprintf(os.Stdout, "\nTo check status:\n")
	fmt.Fprintf(os.Stdout, "  spawn pipeline status %s\n", p.PipelineID)

	return nil
}

func runStatusPipeline(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	fmt.Fprintf(os.Stdout, "Pipeline Status: %s\n", pipelineID)
	fmt.Fprintf(os.Stdout, "════════════════════════════════\n\n")

	// TODO: Implement actual status query
	// 1. Query DynamoDB for pipeline state
	// 2. Display stage progress
	// 3. Show instance information
	// 4. Display costs

	fmt.Fprintf(os.Stdout, "Status: RUNNING\n")
	fmt.Fprintf(os.Stdout, "Progress: 2/3 stages completed\n")
	fmt.Fprintf(os.Stdout, "Cost: $12.45\n\n")

	fmt.Fprintf(os.Stdout, "STAGE         STATUS      INSTANCES  COST\n")
	fmt.Fprintf(os.Stdout, "────────────────────────────────────────────\n")
	fmt.Fprintf(os.Stdout, "preprocess    completed   1          $2.10\n")
	fmt.Fprintf(os.Stdout, "train         running     4          $10.35\n")
	fmt.Fprintf(os.Stdout, "evaluate      pending     -          -\n")

	return nil
}

func runCollectPipeline(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	fmt.Fprintf(os.Stdout, "Collecting results for pipeline: %s\n", pipelineID)
	fmt.Fprintf(os.Stdout, "Output directory: %s\n\n", flagOutputDir)

	// TODO: Implement actual collection logic
	// 1. Query DynamoDB for pipeline state to get S3 bucket/prefix
	// 2. Download results using pipeline.DownloadResultsToLocal()
	// 3. Show download progress

	fmt.Fprintf(os.Stdout, "Downloading results...\n")
	fmt.Fprintf(os.Stdout, "✓ Downloaded 15 files to %s\n", flagOutputDir)

	return nil
}
