package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"orderservice/internal/config"
)

func TestSetup_requiresOutputDir(t *testing.T) {
	t.Parallel()
	_, _, _, err := setup(context.Background(), func(string) string { return "" }, testLogger())
	if !errors.Is(err, config.ErrOutputDirRequired) {
		t.Fatalf("err = %v, want %v", err, config.ErrOutputDirRequired)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
