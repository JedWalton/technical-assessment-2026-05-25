.PHONY: help build run test test-race test-cover test-integration vet fmt fmt-check ci clean

BINARY      := orderservice
PKG         := ./...
COVER_OUT   := coverage.out
COVER_HTML  := coverage.html

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the orderservice binary.
	go build -o $(BINARY) ./cmd/orderservice

run: build ## Build and run the orderservice binary.
	./$(BINARY)

test: ## Run all tests.
	go test $(PKG)

test-race: ## Run all tests with the race detector.
	go test -race $(PKG)

test-cover: ## Run all tests with coverage profile.
	go test -race -covermode=atomic -coverprofile=$(COVER_OUT) $(PKG)
	@go tool cover -func=$(COVER_OUT) | tail -n 1

test-integration: ## Run integration tests (HTTP + CSV + shutdown).
	go test -tags=integration -race ./cmd/orderservice/...

cover-html: test-cover ## Generate HTML coverage report.
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "Coverage report: $(COVER_HTML)"

vet: ## Run go vet.
	go vet $(PKG)

fmt: ## Format Go source files in place.
	gofmt -w .

fmt-check: ## Fail if any Go file is not gofmt-clean.
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then \
		echo "The following files need gofmt:"; \
		echo "$$out"; \
		exit 1; \
	fi

ci: fmt-check vet test-race ## Run the same checks CI runs.

clean: ## Remove build and coverage artifacts.
	rm -f $(BINARY) $(COVER_OUT) $(COVER_HTML)
