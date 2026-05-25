package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/config"
)

func TestLoad_missingOutputDir(t *testing.T) {
	t.Parallel()
	_, err := config.Load(func(string) string { return "" })
	if !errors.Is(err, config.ErrOutputDirRequired) {
		t.Fatalf("err = %v, want %v", err, config.ErrOutputDirRequired)
	}
}

func TestLoad_missingBatchSize(t *testing.T) {
	t.Parallel()
	getenv := func(k string) string {
		if k == "OUTPUT_DIR" {
			return t.TempDir()
		}
		return ""
	}
	_, err := config.Load(getenv)
	if !errors.Is(err, config.ErrBatchSizeRequired) {
		t.Fatalf("err = %v, want %v", err, config.ErrBatchSizeRequired)
	}
}

func TestLoad_invalidBatchSize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	getenv := func(k string) string {
		switch k {
		case "OUTPUT_DIR":
			return dir
		case "BATCH_SIZE":
			return "0"
		}
		return ""
	}
	_, err := config.Load(getenv)
	if !errors.Is(err, config.ErrBatchSizeInvalid) {
		t.Fatalf("err = %v, want %v", err, config.ErrBatchSizeInvalid)
	}
}

func TestLoad_outputDirNotWritable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	readonly := filepath.Join(dir, "ro")
	if err := os.Mkdir(readonly, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readonly, 0o755) })

	getenv := func(k string) string {
		switch k {
		case "OUTPUT_DIR":
			return readonly
		case "BATCH_SIZE":
			return "10"
		}
		return ""
	}
	_, err := config.Load(getenv)
	if !errors.Is(err, config.ErrOutputDirNotWritable) {
		t.Fatalf("err = %v, want %v", err, config.ErrOutputDirNotWritable)
	}
}

func TestLoad_defaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	getenv := func(k string) string {
		switch k {
		case "OUTPUT_DIR":
			return dir
		case "BATCH_SIZE":
			return "100"
		}
		return ""
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
	if cfg.FlushEvery <= 0 {
		t.Fatalf("FlushEvery = %v, want positive", cfg.FlushEvery)
	}
}

func TestLoad_customHTTPAndShutdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	getenv := func(k string) string {
		switch k {
		case "OUTPUT_DIR":
			return dir
		case "BATCH_SIZE":
			return "50"
		case "HTTP_ADDR":
			return "127.0.0.1:9090"
		case "SHUTDOWN_TIMEOUT":
			return "5s"
		}
		return ""
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
}
