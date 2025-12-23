package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scttfrdmn/mycelium/spawn/pkg/agent"
)

const Version = "0.1.0"

func main() {
	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("spored version %s\n", Version)
			os.Exit(0)
		case "complete":
			handleComplete(os.Args[2:])
			os.Exit(0)
		case "help", "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	// Setup logging
	logFile, err := os.OpenFile("/var/log/spored.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Could not open log file: %v", err)
		log.SetOutput(os.Stderr)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Printf("spored v%s starting...", Version)

	// Create agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent, err := agent.NewAgent(ctx)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring
	go agent.Monitor(ctx)

	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown - run cleanup tasks
	cancel()

	// Run cleanup with a timeout context
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	agent.Cleanup(cleanupCtx)

	log.Printf("spored stopped")
}

func handleComplete(args []string) {
	// Default completion file
	completionFile := "/tmp/SPAWN_COMPLETE"
	var status, message string

	// Simple flag parsing
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file", "-f":
			if i+1 < len(args) {
				completionFile = args[i+1]
				i++
			}
		case "--status", "-s":
			if i+1 < len(args) {
				status = args[i+1]
				i++
			}
		case "--message", "-m":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: spored complete [options]")
			fmt.Println()
			fmt.Println("Signal completion to trigger on-complete action (terminate/stop/hibernate)")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  -f, --file PATH      Completion file path (default: /tmp/SPAWN_COMPLETE)")
			fmt.Println("  -s, --status STATUS  Optional status (e.g., 'success', 'failed')")
			fmt.Println("  -m, --message MSG    Optional message")
			fmt.Println("  -h, --help           Show this help")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  spored complete")
			fmt.Println("  spored complete --status success")
			fmt.Println("  spored complete --status success --message 'Job completed successfully'")
			os.Exit(0)
		}
	}

	// Build metadata if provided
	var content []byte
	if status != "" || message != "" {
		metadata := make(map[string]string)
		if status != "" {
			metadata["status"] = status
		}
		if message != "" {
			metadata["message"] = message
		}
		metadata["timestamp"] = time.Now().Format(time.RFC3339)

		var err error
		content, err = json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding metadata: %v\n", err)
			os.Exit(1)
		}
	}

	// Write completion file
	if err := os.WriteFile(completionFile, content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing completion file: %v\n", err)
		os.Exit(1)
	}

	// Success message
	fmt.Printf("✓ Completion signal sent to %s\n", completionFile)
	if status != "" {
		fmt.Printf("  Status: %s\n", status)
	}
	if message != "" {
		fmt.Printf("  Message: %s\n", message)
	}
}

func printHelp() {
	fmt.Printf("spored v%s - Spawn EC2 instance agent\n\n", Version)
	fmt.Println("Usage:")
	fmt.Println("  spored                Run as daemon (monitors instance lifecycle)")
	fmt.Println("  spored complete       Signal completion to trigger on-complete action")
	fmt.Println("  spored version        Show version")
	fmt.Println("  spored help           Show this help")
	fmt.Println()
	fmt.Println("Daemon Mode:")
	fmt.Println("  Runs as a systemd service and monitors:")
	fmt.Println("  - Spot interruption warnings")
	fmt.Println("  - Completion signals (file-based)")
	fmt.Println("  - TTL (time-to-live) expiration")
	fmt.Println("  - Idle timeout detection")
	fmt.Println()
	fmt.Println("  Configuration is loaded from EC2 instance tags (set by spawn launch).")
	fmt.Println()
	fmt.Println("Complete Subcommand:")
	fmt.Println("  Signal that your workload has finished. If --on-complete was set during")
	fmt.Println("  launch, the instance will terminate/stop/hibernate after a grace period.")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    spored complete")
	fmt.Println("    spored complete --status success")
	fmt.Println("    spored complete --status success --message 'Job completed'")
	fmt.Println()
	fmt.Println("For more information: https://github.com/scttfrdmn/mycelium")
}
