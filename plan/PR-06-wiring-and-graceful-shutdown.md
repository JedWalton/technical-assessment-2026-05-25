# PR #6 — Wiring and graceful shutdown

Branch: `feature/uw-6-wiring-and-graceful-shutdown`

## Summary

- Add `internal/config` to load and validate `OUTPUT_DIR`, `BATCH_SIZE`, and optional `HTTP_ADDR`, `SHUTDOWN_TIMEOUT`.
- Wire `FileWriter` → `Service` → `httpapi` in `cmd/orderservice` with `http.Server` timeouts, `Service.Run` in a goroutine, and graceful shutdown (`Shutdown` then final `Flush`).
- Export `batch.SystemClock` for production wiring.
- Add `//go:build integration` test exercising HTTP submit → CSV flush and shutdown drain.

## Scope

- **In scope:** config package, full `run`/`setup`, integration test, README env table update.
- **Out of scope (deferred):** Dockerfile (PR #7).

## Contract / public API

```go
package config

type Config struct {
    OutputDir        string
    BatchSize        int
    HTTPAddr         string
    ShutdownTimeout  time.Duration
    FlushEvery       time.Duration
}

func Load(getenv func(string) string) (Config, error)
```

Environment variables:

| Variable | Required | Default |
|----------|----------|---------|
| `OUTPUT_DIR` | yes | — |
| `BATCH_SIZE` | yes | — |
| `HTTP_ADDR` | no | `:8080` |
| `SHUTDOWN_TIMEOUT` | no | `15s` |

`FlushEvery` defaults to `batch.DurationUntilNextMidnightUTC(now)`.

## Design decisions

- **`Load(getenv)`** — same test seam as `run`; no direct `os.Getenv` in config package.
- **`setup` in main** — returns bound listen address (supports `:0` in tests) and a shutdown function; `run` orchestrates lifecycle.
- **Shutdown order** — `http.Server.Shutdown` first (stop new requests), then `Service.Flush` (drain buffer to CSV).
- **Integration test calls `setup`** — avoids polling logs for a random port; still exercises real HTTP listener and CSV writer.

## Process

TDD Red→Green:

1. Write `config_test.go` for env parsing and validation.
2. Implement `config.go`.
3. Wire `main.go` and add integration test.
4. Run `make ci` and `go test -tags=integration ./cmd/orderservice/...`.

## Test plan

Verified (`make ci` + `make test-integration` on 2026-05-25).

- [x] Missing `OUTPUT_DIR` → error — `TestLoad_missingOutputDir`, `TestSetup_requiresOutputDir`
- [x] Missing / invalid `BATCH_SIZE` → error — `TestLoad_missingBatchSize`, `TestLoad_invalidBatchSize`
- [x] Non-writable `OUTPUT_DIR` → error — `TestLoad_outputDirNotWritable`
- [x] Defaults for `HTTP_ADDR` and `SHUTDOWN_TIMEOUT` — `TestLoad_defaults`
- [x] Custom HTTP and shutdown timeout — `TestLoad_customHTTPAndShutdown`
- [x] Integration: POST orders → CSV file with correct rows — `TestSetup_submitOrdersAndShutdownFlush`
- [x] Integration: shutdown flushes partial buffer — same (3 orders, batch 2 → 2 CSV files, 3 rows total)
- [x] `go test -race ./...` clean (unit tests) — `make ci`
- [x] `go vet` + gofmt clean — `make ci`

## Acceptance criteria

- [x] `make ci` passes.
- [x] `go test -tags=integration -race ./cmd/orderservice/...` passes locally (`make test-integration`).
- [x] `OUTPUT_DIR` and `BATCH_SIZE` documented in README.
