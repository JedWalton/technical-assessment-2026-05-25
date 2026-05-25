# PR #3 — CSV batch writer

Branch: `feature/uw-3-csv-batch-writer`

## Summary

- Add `internal/batch` package with `FileWriter`, which writes a slice of `order.Order` values to a CSV file in a configured output directory.
- Use `encoding/csv` with header row `customer_number,address,postcode,placed_at` and RFC3339 timestamps.
- Write atomically: create `orders-<timestamp>-<id>.csv.tmp`, flush, then `os.Rename` to `.csv` so downstream readers never see a partial file.
- Empty batches are a no-op (no file created).

## Scope

- **In scope:** `FileWriter`, atomic rename, CSV format, filesystem tests with `t.TempDir()`.
- **Out of scope (deferred):** `Writer` interface consumed by `Service` (declared in PR #4 `service.go`), in-memory buffer, EOD ticker (PR #4), HTTP layer (PR #5).

## Contract / public API

```go
package batch

type FileWriter struct {
    Dir string // output directory (must exist and be writable)
}

func NewFileWriter(dir string) *FileWriter

// Write persists orders to a new CSV file. Returns nil without creating a
// file when orders is empty. Respects ctx cancellation.
func (w *FileWriter) Write(ctx context.Context, orders []order.Order) error
```

## Design decisions

- **Atomic rename pattern** — write to `*.csv.tmp` in the same directory, then `os.Rename` into place. Same-directory rename is atomic on POSIX and the pattern Kubernetes volume consumers expect.
- **Filesystem-safe timestamp** — UTC compact form `20060102T150405Z` plus 8 hex bytes from `crypto/rand` for uniqueness; avoids `:` in filenames.
- **Empty batch no-op** — callers (the batch service in PR #4) may call `Flush` on an empty buffer; the writer must not create zero-byte or header-only files.
- **`PlacedAt` as RFC3339** — standard, sortable, unambiguous in CSV.
- **Context check before I/O** — `ctx.Err()` checked at the start of `Write` so cancelled shutdown does not start a new file.

## Process

TDD Red→Green:

1. Write `writer_test.go` specifying CSV content, atomic behaviour, empty batch, and error paths.
2. Implement `writer.go` until all tests pass.
3. Run `make ci`.

## Test plan

- [ ] Empty slice → nil, no files created in dir
- [ ] Single order → one `.csv` file, correct header and row
- [ ] Multiple orders → all rows present in order
- [ ] No `.csv.tmp` files remain after successful write
- [ ] Write to read-only directory → error, no `.csv` file created
- [ ] `go test -race ./...` clean
- [ ] `go vet` + gofmt clean

## Acceptance criteria

- `make ci` passes.
- `go test -cover ./internal/batch/...` reports high coverage on writer paths.
