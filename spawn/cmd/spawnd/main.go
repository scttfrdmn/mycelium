package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/spawn/pkg/agent"
)

const Version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("spawnd version %s\n", Version)
		os.Exit(0)
	}

	// Setup logging
	logFile, err := os.OpenFile("/var/log/spawnd.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Could not open log file: %v", err)
		log.SetOutput(os.Stderr)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	log.Printf("spawnd v%s starting...", Version)

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

	// Graceful shutdown
	cancel()
	time.Sleep(2 * time.Second)
	log.Printf("spawnd stopped")
}
