# PR #4 — Batch service

Branch: `feature/pr-4-batch-service` (merged as `feature/uw-4-batch-service`)

## Summary

- Add `Service` with a thread-safe in-memory buffer, `Add`, `Flush`, and `Run`.
- Auto-flush when the buffer reaches `batchSize`; periodic flush via a `Clock` ticker (end-of-day interval supplied by caller).
- Declare `Writer` and `Clock` interfaces in `service.go` (accept interfaces, return concrete types); `FileWriter` satisfies `Writer`.
- Race-detector tests for concurrent `Add`, size-triggered flush, ticker flush, context cancellation, and final `Flush`.

## Scope

- **In scope:** `Service`, `Writer`, `Clock`, `realClock`, fake clock/ticker in tests, `DurationUntilNextMidnightUTC` helper.
- **Out of scope (deferred):** HTTP handlers (PR #5), env wiring and default flush interval from config (PR #6).

## Contract / public API

```go
package batch

type Writer interface {
    Write(ctx context.Context, orders []order.Order) error
}

type Clock interface {
    Now() time.Time
    NewTicker(time.Duration) Ticker
}

type Ticker interface {
    C() <-chan time.Time
    Stop()
}

type Service struct { /* ... */ }

func NewService(writer Writer, clock Clock, batchSize int, flushEvery time.Duration) *Service
func DurationUntilNextMidnightUTC(now time.Time) time.Duration

func (s *Service) Add(ctx context.Context, o order.Order) error
func (s *Service) Flush(ctx context.Context) error
func (s *Service) Run(ctx context.Context)
```

## Design decisions

- **`Writer` defined at call site** — `Service` depends on `Writer`, not `*FileWriter`; tests use a recording fake without the filesystem.
- **Copy buffer on flush** — `Flush` snapshots and clears the buffer under the mutex before calling `Write`, so the writer never holds the lock during I/O.
- **Auto-flush outside the append lock** — `Add` only sets a flag under the mutex, then calls `Flush` without holding the mutex during `Write` (avoids deadlock if `Write` is slow).
- **`Run` blocks until ctx cancelled** — ticker loop selects on `ctx.Done()` and `ticker.C()`; caller runs `Flush` after `Run` returns (PR #6 shutdown).
- **`DurationUntilNextMidnightUTC`** — helper for PR #6 config default; tested independently.

## Process

TDD Red→Green:

1. Write `service_test.go` and `clock_test.go` against the contract.
2. Implement `clock.go` and `service.go`.
3. Run `make ci`.

## Test plan

Verified against `internal/batch/service_test.go` (`make ci` on 2026-05-25).

- [x] Add until batch size → single `Write` with all orders — `TestServiceAdd_flushesWhenBatchSizeReached`
- [x] Ticker tick flushes buffered orders without reaching batch size — `TestServiceRun_tickerFlushesBuffer`
- [x] Concurrent `Add` from many goroutines → no lost orders, race-clean — `TestServiceAdd_concurrentNoLostOrders`
- [x] `Run` returns when context cancelled — `TestServiceRun_stopsWhenContextCancelled`
- [x] `Flush` after partial batch writes remaining orders — `TestServiceFlush_writesPartialBatch`
- [x] Empty `Flush` is no-op (no `Write` call) — `TestServiceFlush_emptyIsNoOp`
- [x] `DurationUntilNextMidnightUTC` returns positive duration before midnight — `TestDurationUntilNextMidnightUTC`
- [x] No goroutine leak after `Run` stops — `TestServiceRun_noGoroutineLeak`
- [x] `go test -race ./...` clean — `make ci`
- [x] `go vet` + gofmt clean — `make ci`

## Acceptance criteria

- [x] `make ci` passes.
- [x] `go test -cover ./internal/batch/...` reports high coverage including service paths (**78.9%**).
