//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetup_submitOrdersAndShutdownFlush(t *testing.T) {
	dir := t.TempDir()
	getenv := func(k string) string {
		switch k {
		case "OUTPUT_DIR":
			return dir
		case "BATCH_SIZE":
			return "2"
		case "HTTP_ADDR":
			return "127.0.0.1:0"
		case "SHUTDOWN_TIMEOUT":
			return "5s"
		}
		return ""
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	addr, shutdown, _, err := setup(ctx, getenv, logger)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	baseURL := "http://" + addr
	client := &http.Client{Timeout: 5 * time.Second}

	postOrder := func(customer string) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{
			"customer_number": customer,
			"address":         "1 High Street",
			"postcode":        "AB12 3CD",
			"placed_at":       "2026-05-25T10:00:00Z",
		})
		resp, err := client.Post(baseURL+"/orders", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
		}
	}

	postOrder("C1")
	postOrder("C2") // batch size 2 → first CSV flush

	waitForCSVCount(t, dir, 1)

	postOrder("C3") // partial buffer

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	files := listCSVFiles(t, dir)
	if len(files) != 2 {
		t.Fatalf("got %d csv files, want 2 (batch flush + shutdown flush)", len(files))
	}

	totalRows := 0
	for _, f := range files {
		rows := readCSV(t, f)
		if len(rows) < 2 {
			t.Fatalf("file %s: expected header + data rows", f)
		}
		totalRows += len(rows) - 1
	}
	if totalRows != 3 {
		t.Fatalf("total data rows = %d, want 3", totalRows)
	}
}

func waitForCSVCount(t *testing.T, dir string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(listCSVFiles(t, dir)) >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d csv files in %s", want, dir)
}

func listCSVFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csv") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return rows
}
