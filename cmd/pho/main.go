package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pho/internal/app"
	"time"
)

func main() {
	// TODO: make timeout configurable via CLI flag or environment variable
	const defaultTimeout = 5 * time.Minute

	// Create context with timeout and signal handling for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Handle Ctrl+C (SIGINT) and SIGTERM for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	app := app.New()
	if err := app.Run(ctx, os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
