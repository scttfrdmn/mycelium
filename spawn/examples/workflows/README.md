# Workflow Integration Examples

This directory contains working examples of spawn integration with popular workflow orchestration tools.

## Available Examples

### [Apache Airflow](airflow/)
Platform for programmatically authoring, scheduling, and monitoring workflows.
- Custom operator for spawn sweeps
- DAG with polling and error handling
- Production-ready example

### [Prefect](prefect/)
Modern workflow orchestration with dynamic task generation.
- Task-based sweep launching
- Automatic retry logic
- Clean Python API

### [Nextflow](nextflow/)
Workflow system for computational pipelines, popular in bioinformatics.
- Process-based workflow
- Channel-based data flow
- Containerized execution

### [Snakemake](snakemake/)
Workflow management for reproducible and scalable data analysis.
- Rule-based workflow
- Automatic dependency resolution
- Integration with conda/containers

### [AWS Step Functions](step-functions/)
Serverless workflow orchestration on AWS.
- State machine definition
- Lambda-based integration
- CloudFormation deployment

### [Argo Workflows](argo/)
Kubernetes-native workflow engine for parallel jobs.
- Container-native workflow
- Kubernetes resource management
- DAG-based execution

### [Common Workflow Language (CWL)](cwl/)
Specification for describing command-line tools and workflows.
- Portable tool description
- Docker-based execution
- Wide tool compatibility

### [Workflow Description Language (WDL)](wdl/)
Workflow language for genomic analysis pipelines.
- Task-based workflows
- Runtime configuration
- Cromwell execution engine

## Quick Start

Each example directory contains:
- Working code/configuration
- README with setup instructions
- Sample input files
- Expected output description

## General Pattern

All examples follow this pattern:

1. **Launch**: Start sweep with `--detach --output-id`
2. **Wait**: Poll status with `--check-complete`
3. **Process**: Handle results based on exit code

## Prerequisites

- spawn installed and configured
- AWS credentials configured
- Tool-specific requirements (see individual READMEs)

## Testing Examples

```bash
# Test Airflow example
cd airflow && ./test.sh

# Test Prefect example
cd prefect && python spawn_flow.py

# Test with Docker
cd argo && kubectl apply -f workflows/spawn-sweep.yaml
```

## Best Practices

1. **Always use --detach** for Lambda orchestration
2. **Capture IDs with --output-id** for tracking
3. **Check exit codes** for proper error handling
4. **Set timeouts** to prevent hanging
5. **Log everything** for debugging

## Support

For issues or questions:
- Main documentation: [WORKFLOW_INTEGRATION.md](../../WORKFLOW_INTEGRATION.md)
- GitHub Issues: https://github.com/scttfrdmn/mycelium/issues
