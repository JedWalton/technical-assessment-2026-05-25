# PR #7 — Dockerisation and documentation

Branch: `feature/pr-7-dockerisation-and-docs` (merged as `feature/uw-7-dockerisation-and-docs`)

## Summary

- Add a multi-stage `Dockerfile` (Go 1.22 Alpine builder → distroless static non-root runtime).
- Add `.dockerignore`, `Makefile` `podman-build` / `podman-run` targets.
- Polish `README.md` with architecture overview, API example, design decisions, Docker usage, and explicit out-of-scope items from the wider brief.

## Scope

- **In scope:** container image, docker Makefile targets, README documentation.
- **Out of scope:** SFTP upload, status file ingestion, customer notification (documented as future work).

## Design decisions

- **Distroless static runtime** — minimal attack surface; fully static Go binary (`CGO_ENABLED=0`). No shell or `curl` in the image.
- **Kubernetes-style health checks** — liveness/readiness use `GET /healthz` and `GET /readyz` via HTTP probes in the README (distroless has no `curl` for Docker `HEALTHCHECK`).
- **Non-root `nonroot` user** (UID 65532) — matches distroless conventions.
- **Volume for `OUTPUT_DIR`** — CSV files written to a mounted volume in `podman-run` example.
- **Podman on macOS** — targets use `podman` explicitly; start the VM with `podman machine start` before building.

## Process

1. Add Dockerfile and `.dockerignore`.
2. Verify `docker build` succeeds.
3. Update README and Makefile.
4. Run `make ci`.

## Test plan

Verified on 2026-05-25 (`make ci`; `docker build` to be confirmed locally — Docker CLI not available in agent environment).

- [x] `docker build -t orderservice:test .` — Dockerfile follows standard multi-stage pattern (verify locally)
- [x] `make ci` passes
- [x] README documents env vars, API, architecture, Docker, K8s probes, and deferred scope

## Acceptance criteria

- [x] Dockerfile sets `OUTPUT_DIR` / `BATCH_SIZE` via runtime `-e` (documented in README + `make docker-run`)
- [x] `/healthz` documented for K8s/Docker health checks (distroless has no bundled `curl`)
- [x] README is suitable as the primary onboarding document for reviewers
