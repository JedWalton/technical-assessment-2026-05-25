package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/batch"
)

const (
	envOutputDir       = "OUTPUT_DIR"
	envBatchSize       = "BATCH_SIZE"
	envHTTPAddr        = "HTTP_ADDR"
	envShutdownTimeout = "SHUTDOWN_TIMEOUT"

	defaultHTTPAddr        = ":8080"
	defaultShutdownTimeout = 15 * time.Second
)

var (
	ErrOutputDirRequired    = errors.New("OUTPUT_DIR is required")
	ErrOutputDirNotWritable = errors.New("OUTPUT_DIR is not writable")
	ErrBatchSizeRequired    = errors.New("BATCH_SIZE is required")
	ErrBatchSizeInvalid     = errors.New("BATCH_SIZE must be a positive integer")
)

// Config holds runtime settings loaded from the environment.
type Config struct {
	OutputDir       string
	BatchSize       int
	HTTPAddr        string
	ShutdownTimeout time.Duration
	FlushEvery      time.Duration
}

// Load reads configuration from getenv, validates it, and returns Config.
func Load(getenv func(string) string) (Config, error) {
	outDir := strings.TrimSpace(getenv(envOutputDir))
	if outDir == "" {
		return Config{}, ErrOutputDirRequired
	}
	if err := ensureDirWritable(outDir); err != nil {
		return Config{}, err
	}

	batchRaw := strings.TrimSpace(getenv(envBatchSize))
	if batchRaw == "" {
		return Config{}, ErrBatchSizeRequired
	}
	batchSize, err := strconv.Atoi(batchRaw)
	if err != nil || batchSize < 1 {
		return Config{}, ErrBatchSizeInvalid
	}

	httpAddr := strings.TrimSpace(getenv(envHTTPAddr))
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	shutdownTimeout := defaultShutdownTimeout
	if raw := strings.TrimSpace(getenv(envShutdownTimeout)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
		}
		shutdownTimeout = d
	}

	return Config{
		OutputDir:       outDir,
		BatchSize:       batchSize,
		HTTPAddr:        httpAddr,
		ShutdownTimeout: shutdownTimeout,
		FlushEvery:      batch.DurationUntilNextMidnightUTC(time.Now().UTC()),
	}, nil
}

func ensureDirWritable(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return fmt.Errorf("%w: %v", ErrOutputDirNotWritable, mkErr)
			}
			info, err = os.Stat(dir)
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrOutputDirNotWritable, err)
		}
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: not a directory", ErrOutputDirNotWritable)
	}
	f, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOutputDirNotWritable, err)
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return nil
}
