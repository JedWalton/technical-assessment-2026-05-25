// Command orderservice is the HTTP entrypoint for the UW broadband order
// batching microservice.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/batch"
	"github.com/JedWalton/technical-assessment-2026-05-25/internal/config"
	"github.com/JedWalton/technical-assessment-2026-05-25/internal/httpapi"
)

const serviceName = "orderservice"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, os.Args, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", serviceName, err)
		os.Exit(1)
	}
}

// run loads configuration, starts the HTTP server and batch ticker, blocks
// until ctx is cancelled, then shuts down gracefully.
func run(ctx context.Context, _ []string, getenv func(string) string, stdout, _ io.Writer) error {
	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr, shutdown, shutdownTimeout, err := setup(ctx, getenv, logger)
	if err != nil {
		return err
	}

	slog.Info("service starting", "service", serviceName, "addr", addr)

	<-ctx.Done()

	slog.Info("service shutting down", "service", serviceName)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return shutdown(shutdownCtx)
}

// setup wires dependencies and starts HTTP and the batch ticker. It returns the
// bound listen address, a shutdown function, and the configured shutdown timeout.
func setup(ctx context.Context, getenv func(string) string, logger *slog.Logger) (addr string, shutdown func(context.Context) error, shutdownTimeout time.Duration, err error) {
	cfg, err := config.Load(getenv)
	if err != nil {
		return "", nil, 0, err
	}

	writer := batch.NewFileWriter(cfg.OutputDir)
	svc := batch.NewService(writer, batch.SystemClock{}, cfg.BatchSize, cfg.FlushEvery)

	go svc.Run(ctx)

	handler := httpapi.New(svc, logger)
	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return "", nil, 0, fmt.Errorf("listen: %w", err)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("http serve failed", "error", serveErr)
		}
	}()

	shutdownFn := func(shutdownCtx context.Context) error {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		if err := svc.Flush(shutdownCtx); err != nil {
			return fmt.Errorf("final flush: %w", err)
		}
		return nil
	}

	return ln.Addr().String(), shutdownFn, cfg.ShutdownTimeout, nil
}
