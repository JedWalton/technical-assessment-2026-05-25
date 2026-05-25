# PR #1 — Project scaffold

Branch: `feature/uw-1-project-scaffold`

## Summary

- Initialise the Go module (`github.com/JedWalton/technical-assessment-2026-05-25`) targeting Go 1.22+, with **zero third-party dependencies** (the entire service will be built on the standard library).
- Add a `Makefile` exposing `build`, `test`, `test-race`, `test-cover`, `vet`, `fmt-check`, `fmt`, `run`, `clean`, and `help` targets — these are the same commands CI will run, so a contributor can reproduce the pipeline locally.
- Add a GitHub Actions workflow at `.github/workflows/ci.yml` running gofmt check, `go vet ./...`, and `go test -race -cover ./...` on every push and pull request.
- Add a minimal `cmd/orderservice/main.go` entrypoint built around a `run(ctx, args, getenv)` function so subsequent PRs only need to fill the body — the `main` shell, signal handling, and JSON `log/slog` setup are already in place.
- Update README with build, test, and run instructions plus the planned project layout.

## Scope

- **In scope:** module skeleton, build/test/lint toolchain, CI workflow, entrypoint stub, README pointers.
- **Out of scope (deferred):** domain types (PR #2), CSV writer (PR #3), batch service (PR #4), HTTP handlers (PR #5), env-var config and full wiring (PR #6), Dockerfile (PR #7).

## Contract / public API

- Module path: `github.com/JedWalton/technical-assessment-2026-05-25`.
- Binary: `cmd/orderservice` → `orderservice`.
- No exported types yet.

## Design decisions

- **Stdlib-only `go.mod`.** Anchored to Go 1.22 because that is the first release with `http.ServeMux` `METHOD /path` routing (used in PR #5). No third-party deps means no supply-chain surface and a trivial Docker image in PR #7.
- **`run(ctx, args, getenv)` pattern.** Idiomatic for testable Go entrypoints: `main` is a 3-liner that delegates to `run`, which takes its context, command-line args, and env reader as parameters. Subsequent PRs (especially #6) can integration-test `run` directly without spawning a subprocess.
- **`log/slog` JSON handler from the start.** Structured logs are the only logging format used; `slog.SetDefault` is called once at the top of `run` and every package logs via `slog.Info`/`slog.Error` rather than carrying loggers around.
- **`signal.NotifyContext`** wraps the base context — wiring graceful shutdown in PR #6 only requires adding `server.Shutdown` and `service.Flush` calls inside the same `<-ctx.Done()` branch.
- **Makefile mirrors CI.** Each CI step has a corresponding `make` target with the same flags. `make ci` runs the whole pipeline locally.

## Process

This is a scaffold-only PR — there is no domain logic yet, so there are no Red→Green TDD tests. The "tests" for this PR are operational: `make ci` (which delegates to vet, fmt-check, and race-tested go test) must succeed on a fresh checkout.

## Test plan

- [x] `go build ./...` succeeds.
- [x] `go vet ./...` clean.
- [x] `gofmt -l .` returns no files.
- [x] `go test -race -cover ./...` succeeds (no tests yet → exits 0).
- [x] `make ci` green locally.
- [x] CI green on push.

## Acceptance criteria

- A fresh `git clone` followed by `make ci` passes without modification.
- Running `./orderservice` after `make build` logs a JSON `service starting` line, waits for SIGINT/SIGTERM, then logs `service shutting down` and exits 0.
