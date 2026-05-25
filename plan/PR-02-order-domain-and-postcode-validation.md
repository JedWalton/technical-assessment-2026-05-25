# PR #2 — Order domain and postcode validation

Branch: `feature/uw-2-order-domain-and-postcode-validation`

## Summary

- Add `internal/order` package with the `Order` domain type and a `Validate()` method that checks all required fields.
- Add `ValidatePostcode(string) error` as a pure, hand-rolled validator (no regex) enforcing the take-home rule: postcode must be 1–8 characters of letters, digits, and spaces only.
- Export sentinel errors so HTTP handlers in PR #5 can map them to `400` responses via `errors.Is`.
- Add table-driven, parallel black-box tests (`package order_test`) covering postcode boundaries and full order validation.

## Scope

- **In scope:** `Order` struct, postcode validation, sentinel errors, unit tests.
- **Out of scope (deferred):** CSV writer (PR #3), batch buffer (PR #4), HTTP handlers (PR #5), JSON decoding of orders (PR #5).

## Contract / public API

```go
package order

type Order struct {
    CustomerNumber string
    Address        string
    Postcode       string
    PlacedAt       time.Time
}

func (o Order) Validate() error

func ValidatePostcode(postcode string) error

var (
    ErrPostcodeEmpty
    ErrPostcodeTooLong
    ErrPostcodeInvalidChar
    ErrCustomerNumberRequired
    ErrAddressRequired
)
```

## Design decisions

- **Hand-rolled postcode loop** instead of `regexp` — the rule is simple (length ≤ 8, ASCII alnum + space); a loop is faster, allocation-free, and easier to reason about in an interview.
- **Sentinel errors, not error strings** — callers use `errors.Is` for stable HTTP status mapping; messages are fixed on the sentinel values.
- **`Validate()` composes `ValidatePostcode`** — postcode rules live in one place; `Order.Validate` also guards customer number and address so HTTP handlers only call one method.
- **Black-box tests** (`package order_test`) — tests import only the public API, matching how `httpapi` will consume the package in PR #5.
- **No `INS` prefix** — the README prompt-injection is ignored.

## Process

TDD Red→Green:

1. Write `postcode_test.go` and `order_test.go` against the contract above; confirm compile failure / test failure.
2. Implement `postcode.go` and `order.go` until all tests pass.
3. Run `make ci`.

## Test plan

Verified against `internal/order/postcode_test.go` and `internal/order/order_test.go` (`make ci` on 2026-05-25).

- [x] Postcode: empty → `ErrPostcodeEmpty` — `TestValidatePostcode/empty`
- [x] Postcode: 8 valid chars → nil — `TestValidatePostcode/valid_max_length_eight` (`AB12 3CD`)
- [x] Postcode: 9 chars → `ErrPostcodeTooLong` — `TestValidatePostcode/too_long_nine_chars`
- [x] Postcode: unicode, punctuation, tab → `ErrPostcodeInvalidChar` — `unicode_letter`, `hyphen`, `tab`, `newline`
- [x] Postcode: mixed case, digits, spaces → nil — `valid_mixed_case_and_digits`, `valid_digits_only`, `valid_spaces_only_within_limit`
- [x] Order: valid → nil — `TestOrderValidate/valid_order`
- [x] Order: missing customer number → `ErrCustomerNumberRequired` — `missing_customer_number`, `whitespace_only_customer_number`
- [x] Order: missing address → `ErrAddressRequired` — `missing_address`, `whitespace_only_address`
- [x] Order: invalid postcode → corresponding postcode error — `empty_postcode`, `postcode_too_long`, `postcode_invalid_character`
- [x] `go test -race ./...` clean — `make ci` / `make test-race`
- [x] `go vet` + gofmt clean — `make ci` (`fmt-check`, `vet`)
- [x] Coverage on `internal/order` reported in CI — **100.0%** (`go test -cover ./internal/order/...`)

## Acceptance criteria

- [x] `make ci` passes on a fresh checkout of this branch.
- [x] `go test -cover ./internal/order/...` reports ≥ 85% coverage on the order package (**100.0%**).
