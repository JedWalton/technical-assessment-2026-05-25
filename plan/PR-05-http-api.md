# PR #5 — HTTP API

Branch: `feature/uw-5-http-api`

## Summary

- Add `internal/httpapi` with `POST /orders`, `GET /healthz`, and `GET /readyz`.
- JSON request decoding with `DisallowUnknownFields` and a body size cap.
- Map domain validation errors to `400` with a structured `{ "error", "code" }` envelope; `202 Accepted` on success.
- Middleware for structured request logging (`log/slog`) and panic recovery.

## Scope

- **In scope:** handlers, middleware, `OrderAdder` interface, httptest coverage.
- **Out of scope (deferred):** wiring `batch.Service` in `main` (PR #6), env config (PR #6).

## Contract / public API

```go
package httpapi

type OrderAdder interface {
    Add(ctx context.Context, o order.Order) error
}

func New(adder OrderAdder, logger *slog.Logger) http.Handler
```

Routes (Go 1.22 `ServeMux`):

- `POST /orders` — submit order
- `GET /healthz` — liveness
- `GET /readyz` — readiness

## Design decisions

- **`OrderAdder` interface** — handler depends on a one-method interface satisfied by `batch.Service`; tests use a fake without the batch package's filesystem or clock.
- **202 Accepted** — order is accepted into the buffer asynchronously; the client does not wait for CSV flush.
- **Stable error codes** — string codes (`POSTCODE_TOO_LONG`, etc.) derived from sentinel errors via `errors.Is`; HTTP layer does not parse error strings.
- **`placed_at` optional in JSON** — defaults to `time.Now().UTC()` when omitted so mobile clients with clock skew are not rejected.
- **Request ID** — generated per request, attached to logs and returned as `X-Request-ID`.
- **Panic recovery** — panics log stack at Error level and return `500` without crashing the process.

## Process

TDD Red→Green:

1. Write `handler_test.go` for all routes and error mappings.
2. Implement `handler.go` and `middleware.go`.
3. Run `make ci`.

## Test plan

Verified against `internal/httpapi/handler_test.go` (`make ci` on 2026-05-25).

- [x] Valid order → 202, `OrderAdder.Add` called — `TestPOSTOrders_validOrderAccepted`
- [x] Each validation error → 400 with correct `code` — `TestPOSTOrders_validationErrors` (5 cases)
- [x] Malformed JSON → 400 — `TestPOSTOrders_malformedJSON`
- [x] Unknown JSON fields → 400 — `TestPOSTOrders_unknownFieldsRejected`
- [x] Oversized body → 413 — `TestPOSTOrders_oversizedBody`
- [x] Wrong HTTP method on `/orders` → 405 — `TestPOSTOrders_wrongMethod`
- [x] Unsupported Content-Type → 415 — `TestPOSTOrders_unsupportedContentType`
- [x] `OrderAdder` internal error → 500 — `TestPOSTOrders_adderErrorReturns500`
- [x] Panic in handler → 500, process survives — `TestPOSTOrders_panicRecoveredAs500`
- [x] `GET /healthz` → 200 — `TestGETHealthz`
- [x] `GET /readyz` → 200 — `TestGETReadyz`
- [x] `go test -race ./...` clean — `make ci`
- [x] `go vet` + gofmt clean — `make ci`

## Acceptance criteria

- [x] `make ci` passes.
- [x] Black-box tests in `package httpapi_test` (**94.5%** coverage on `internal/httpapi`).
