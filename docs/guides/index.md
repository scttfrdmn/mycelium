---
description: "Task-oriented guides for common workflows."
---

# Guides

Task-oriented guides for common workflows. If you're new, start with **Getting Started**. If you know what you want to do, jump directly to the relevant guide.

## Getting Started

| Guide | What you'll learn |
|-------|------------------|
| [Installation](/guides/installation) | Install truffle, spawn, and optional tools |
| [Your First Instance](/guides/first-instance) | End-to-end: find, launch, connect, terminate |

## Compute

| Guide | What you'll learn |
|-------|------------------|
| [GPU Training Jobs](/guides/gpu-training) | Launch a GPU instance, run a training job, terminate on completion |
| [Jupyter Notebooks](/guides/jupyter) | RStudio or Jupyter with process-aware idle detection |
| [Spot Instances](/guides/spot-instances) | Use Spot for up to 90% cost savings, handle interruptions gracefully |

## Automation & Control

| Guide | What you'll learn |
|-------|------------------|
| [Slack Setup](/guides/slack-setup) | Connect Slack, use `/spore` commands, get lifecycle DMs |
| [Teams Setup](/guides/teams-setup) | Connect Microsoft Teams |
| [AI Assistant (MCP)](/guides/mcp-setup) | Control compute from Claude Desktop or Cursor |
| [Lifecycle Notifications](/guides/notifications) | Configure what events trigger notifications and where they go |

## Advanced

| Guide | What you'll learn |
|-------|------------------|
| [Parameter Sweeps](/guides/parameter-sweeps) | Run the same job across many parameter combinations in parallel |
| [MPI Clusters](/guides/mpi) | Launch multi-node clusters for distributed workloads |
| [Pipelines](/guides/pipelines) | Chain jobs so each stage launches the next |
| [Plugins](/guides/plugins) | Extend instances with pre-built or custom plugins at launch time |
| [Job Arrays](/guides/job-arrays) | Launch and manage groups of related instances |

## Workflow Engines

| Guide | What you'll learn |
|-------|------------------|
| [Workflow Engines](/guides/workflow-engines) | Run Nextflow, WDL, CWL, Snakemake, or Airflow with spawn as the execution backend — each task on an ephemeral instance |
| [Nextflow (nf-spawn)](/guides/nextflow) | Dispatch each Nextflow process to its own ephemeral EC2 instance |
