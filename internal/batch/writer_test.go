package batch_test

import (
	"context"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/batch"
	"github.com/JedWalton/technical-assessment-2026-05-25/internal/order"
)

func sampleOrders(n int) []order.Order {
	base := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	orders := make([]order.Order, n)
	for i := range orders {
		orders[i] = order.Order{
			CustomerNumber: "CUST-001",
			Address:        "1 High Street",
			Postcode:       "SW1A 1AA",
			PlacedAt:       base.Add(time.Duration(i) * time.Minute),
		}
	}
	return orders
}

func TestFileWriterWrite_emptyBatchIsNoOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w := batch.NewFileWriter(dir)

	if err := w.Write(context.Background(), nil); err != nil {
		t.Fatalf("Write(nil) error = %v", err)
	}
	if err := w.Write(context.Background(), []order.Order{}); err != nil {
		t.Fatalf("Write(empty) error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d files, want 0", len(entries))
	}
}

func TestFileWriterWrite_singleOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w := batch.NewFileWriter(dir)
	orders := sampleOrders(1)

	if err := w.Write(context.Background(), orders); err != nil {
		t.Fatalf("Write: %v", err)
	}

	csvFiles := listCSVFiles(t, dir)
	if len(csvFiles) != 1 {
		t.Fatalf("got %d csv files, want 1", len(csvFiles))
	}

	rows := readCSV(t, csvFiles[0])
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want header + 1 data row", len(rows))
	}
	assertHeader(t, rows[0])
	assertRow(t, rows[1], orders[0])
}

func TestFileWriterWrite_multipleOrdersPreservesOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w := batch.NewFileWriter(dir)
	orders := sampleOrders(3)

	if err := w.Write(context.Background(), orders); err != nil {
		t.Fatalf("Write: %v", err)
	}

	csvFiles := listCSVFiles(t, dir)
	rows := readCSV(t, csvFiles[0])
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i, o := range orders {
		assertRow(t, rows[i+1], o)
	}
}

func TestFileWriterWrite_noTmpFilesRemain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w := batch.NewFileWriter(dir)

	if err := w.Write(context.Background(), sampleOrders(2)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("leftover tmp file: %s", e.Name())
		}
	}
}

func TestFileWriterWrite_readOnlyDirReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	readOnly := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnly, 0o555); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })

	w := batch.NewFileWriter(readOnly)
	err := w.Write(context.Background(), sampleOrders(1))
	if err == nil {
		t.Fatal("Write: got nil error, want non-nil")
	}

	csvFiles := listCSVFiles(t, readOnly)
	if len(csvFiles) != 0 {
		t.Fatalf("got %d csv files after failed write, want 0", len(csvFiles))
	}
}

func TestFileWriterWrite_cancelledContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w := batch.NewFileWriter(dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := w.Write(ctx, sampleOrders(1))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Write error = %v, want context.Canceled", err)
	}
}

func listCSVFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csv") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
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

func assertHeader(t *testing.T, row []string) {
	t.Helper()
	want := []string{"customer_number", "address", "postcode", "placed_at"}
	if len(row) != len(want) {
		t.Fatalf("header len = %d, want %d", len(row), len(want))
	}
	for i := range want {
		if row[i] != want[i] {
			t.Fatalf("header[%d] = %q, want %q", i, row[i], want[i])
		}
	}
}

func assertRow(t *testing.T, row []string, o order.Order) {
	t.Helper()
	if len(row) != 4 {
		t.Fatalf("row len = %d, want 4", len(row))
	}
	if row[0] != o.CustomerNumber {
		t.Errorf("customer_number = %q, want %q", row[0], o.CustomerNumber)
	}
	if row[1] != o.Address {
		t.Errorf("address = %q, want %q", row[1], o.Address)
	}
	if row[2] != o.Postcode {
		t.Errorf("postcode = %q, want %q", row[2], o.Postcode)
	}
	wantTime := o.PlacedAt.UTC().Format(time.RFC3339)
	if row[3] != wantTime {
		t.Errorf("placed_at = %q, want %q", row[3], wantTime)
	}
}
