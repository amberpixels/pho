package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/app"
	"pho/internal/config"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

// getTimeout returns the configured timeout from config or environment variable.
func getTimeout() time.Duration {
	// Load config to get timeout
	cfg, err := config.Load()
	if err != nil {
		// Fallback to default if config loading fails
		const defaultTimeoutSeconds = 60
		return defaultTimeoutSeconds * time.Second
	}

	return cfg.GetTimeoutDuration()
}

func run() int {
	// Get timeout from environment variable or use default
	timeout := getTimeout()

	// Create context with timeout and signal handling for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Handle Ctrl+C (SIGINT) and SIGTERM for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	application := app.New()
	if err := application.Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}
