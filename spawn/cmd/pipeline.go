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

var (
	flagSimpleGraph bool
	flagGraphStats  bool
	flagJSONOutput  bool
)

func init() {
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(validatePipelineCmd)
	pipelineCmd.AddCommand(graphPipelineCmd)

	// Graph command flags
	graphPipelineCmd.Flags().BoolVar(&flagSimpleGraph, "simple", false, "Show simplified graph")
	graphPipelineCmd.Flags().BoolVar(&flagGraphStats, "stats", false, "Show graph statistics")
	graphPipelineCmd.Flags().BoolVar(&flagJSONOutput, "json", false, "Output as JSON")
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
