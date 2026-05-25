// Command orderservice is the HTTP entrypoint for the UW broadband order
// batching microservice. It accepts orders, validates them, buffers them in
// memory, and writes them to CSV files in a configurable output directory
// whenever the batch size threshold is reached or the end-of-day ticker
// fires.
//
// This file is intentionally a thin shell around run, which is the unit
// integration tests in later PRs will exercise directly without spawning a
// subprocess.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// serviceName is the static identifier used in structured logs.
const serviceName = "orderservice"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, os.Args, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", serviceName, err)
		os.Exit(1)
	}
}

// run is the testable entrypoint. It blocks until ctx is cancelled (typically
// by SIGINT or SIGTERM) and returns nil on a clean shutdown.
func run(ctx context.Context, _ []string, _ func(string) string, stdout, _ io.Writer) error {
	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("service starting", "service", serviceName)

	<-ctx.Done()

	slog.Info("service shutting down", "service", serviceName)
	return nil
}
