package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/app"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

// getTimeout returns the configured timeout from environment variable or default.
func getTimeout() time.Duration {
	const defaultTimeout = 60 * time.Second

	if timeoutStr := os.Getenv("PHO_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			return timeout
		}
		// If parsing fails, fall back to default
	}

	return defaultTimeout
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
