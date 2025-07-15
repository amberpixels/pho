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

const defaultTimeout = 60 * time.Second // TODO: flag/env

func main() {
	os.Exit(run())
}

func run() int {
	// TODO: make timeout configurable via CLI flag or environment variable
	const defaultTimeout = 60 * time.Second

	// Create context with timeout and signal handling for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
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
