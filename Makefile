# OpsGrid developer commands. `make help` lists everything.
#
# Toolchain: Go/Node/pnpm versions come from mise.toml; Go dev tools are pinned
# in tools/go.mod and run via `go tool -modfile=tools/go.mod <tool>`.

# /bin/bash -e works on macOS's GNU Make 3.81 (no .SHELLFLAGS) and in CI.
SHELL := /bin/bash
.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

export GOTOOLCHAIN := go1.27.1
GO        ?= go
GOTOOL    := $(GO) tool -modfile=tools/go.mod
PNPM      ?= pnpm
COMPOSE   ?= docker compose
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
export GIT_COMMIT

# Integration tests reach Docker through the active context's socket.
DOCKER_HOST ?= $(shell docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null)
export DOCKER_HOST

##@ Environment

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n"} \
	/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

.PHONY: doctor
doctor: ## Check toolchain, Docker resources and ports
	@./scripts/doctor.sh

.PHONY: setup
setup: ## Install toolchain (mise), JS deps and browsers
	@command -v mise >/dev/null && mise install || echo "mise not found: install Go/Node/pnpm per mise.toml"
	$(PNPM) install --frozen-lockfile
	$(PNPM) --filter @opsgrid/web exec playwright install chromium
	$(GO) mod download
	$(GOTOOL) -n sqlc >/dev/null 2>&1 || true

##@ Run

# Long-running services; one-shot jobs (migrate, storage-init) are handled
# separately because `up --wait` treats any exited container as a failure.
CORE_SERVICES := postgres redis keycloak storage mailpit api worker relay

.PHONY: dev
dev: ## Build and start the core stack (API serves the SPA on :8080)
	$(COMPOSE) up -d --build --wait $(CORE_SERVICES)
	$(COMPOSE) run --rm storage-init
	@echo
	@echo "  App          http://localhost:8080"
	@echo "  Web (HMR)    make web-dev  → http://localhost:5173"
	@echo "  Keycloak     http://localhost:8180  (admin / see .env.example)"
	@echo "  Mailpit      http://localhost:8025"
	@echo "  Health       http://localhost:9090/readyz"

.PHONY: dev-obs
dev-obs: ## Start the stack with telemetry export + Grafana/Prometheus/Tempo/Loki
	OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 $(COMPOSE) --profile observability up -d --build --wait \
	  $(CORE_SERVICES) otel-collector prometheus tempo loki grafana
	$(COMPOSE) run --rm storage-init
	@echo "  Grafana      http://localhost:3000"
	@echo "  Prometheus   http://localhost:9091"

.PHONY: web-dev
web-dev: ## Vite dev server with HMR on :5173 (proxies /v1 and /auth to :8080)
	$(PNPM) --filter @opsgrid/web dev

.PHONY: stop
stop: ## Stop all containers (keeps data)
	$(COMPOSE) --profile observability --profile chaos stop

.PHONY: down
down: ## Remove containers (keeps volumes)
	$(COMPOSE) --profile observability --profile chaos down

.PHONY: reset
reset: ## Destroy containers AND volumes (all local data)
	$(COMPOSE) --profile observability --profile chaos down -v --remove-orphans

.PHONY: logs
logs: ## Tail application logs
	$(COMPOSE) logs -f --tail=100 api worker relay

.PHONY: migrate
migrate: ## Apply migrations (one-shot container)
	$(COMPOSE) run --rm migrate

.PHONY: psql
psql: ## psql as the runtime role (subject to RLS)
	$(COMPOSE) exec -e PGPASSWORD=$${OPSGRID_APP_PASSWORD:-opsgrid-dev-app} postgres psql -U opsgrid_app -d opsgrid

##@ Quality

.PHONY: lint
lint: lint-go lint-web lint-api lint-sql ## All linters

.PHONY: lint-go
lint-go: ## golangci-lint (architecture boundaries, security, style)
	$(GOTOOL) golangci-lint run ./...

.PHONY: lint-web
lint-web: ## Biome + TypeScript
	$(PNPM) exec biome check .
	$(PNPM) -r typecheck

.PHONY: lint-api
lint-api: ## Redocly lint of the OpenAPI contract
	$(PNPM) exec redocly lint api/openapi.yaml

.PHONY: lint-sql
lint-sql: ## squawk: unsafe migration patterns
	$(PNPM) exec squawk --config=.squawk.toml db/migrations/*.sql

.PHONY: fmt
fmt: ## Format Go and web code
	$(GOTOOL) golangci-lint fmt ./...
	$(PNPM) exec biome check --write .

.PHONY: test
test: test-go test-web ## Unit tests (Go + web)

.PHONY: test-go
test-go: ## Go unit tests
	$(GO) test -count=1 ./...

.PHONY: test-race
test-race: ## Go unit tests with the race detector
	$(GO) test -race -count=1 ./...

.PHONY: test-integration
test-integration: ## Go integration tests (real PostgreSQL via testcontainers)
	$(GO) test -tags integration -race -count=1 ./...

.PHONY: test-web
test-web: ## Web unit/component tests (browser) + API client tests
	$(PNPM) -r test

.PHONY: test-stories
test-stories: ## Storybook interaction + accessibility tests
	$(PNPM) --filter @opsgrid/web test:stories

.PHONY: storybook
storybook: ## Component workbench on http://localhost:6006
	$(PNPM) --filter @opsgrid/web storybook

.PHONY: test-e2e
test-e2e: ## Playwright E2E against the running stack (make dev first)
	$(PNPM) --filter @opsgrid/web e2e

.PHONY: test-fuzz
test-fuzz: ## Short fuzz run of every Fuzz* target
	@for pkg in $$($(GO) list ./...); do \
	  for f in $$($(GO) test -list '^Fuzz' $$pkg 2>/dev/null | grep '^Fuzz' || true); do \
	    echo "fuzz $$pkg $$f"; $(GO) test -run=^$$ -fuzz="^$$f$$" -fuzztime=30s $$pkg; \
	  done; \
	done

.PHONY: deps
deps: ## Report available dependency upgrades (read-only)
	@./scripts/deps-report.sh

.PHONY: secrets-scan
secrets-scan: ## gitleaks over the full git history
	docker run --rm -v "$$PWD:/repo" zricethezav/gitleaks:v8.30.1 git /repo --config /repo/.gitleaks.toml --redact

.PHONY: vuln
vuln: ## Known-vulnerability scan (Go)
	$(GOTOOL) govulncheck ./...

.PHONY: check
check: lint test-race test-integration test-stories ## Everything CI runs before E2E

##@ Build & generate

.PHONY: generate
generate: ## Regenerate code from the OpenAPI contract and SQL
	$(GO) generate ./internal/api/...
	$(PNPM) --filter @opsgrid/openapi-gen generate
	@if [ -f sqlc.yaml ]; then $(GOTOOL) sqlc generate; fi

.PHONY: check-generated
check-generated: generate ## Fail if generated code differs from the contract
	@git diff --exit-code -- internal/api/gen packages/api-client/src/gen || \
	  (echo "Generated code is stale: run 'make generate' and commit the result." >&2; exit 1)

.PHONY: build
build: ## Build Go binaries into ./bin (without the embedded SPA)
	$(GO) build -trimpath -o bin/ ./cmd/...

.PHONY: web-embed
web-embed: ## Build the SPA and sync it into the Go embed directory
	$(PNPM) --filter @opsgrid/web build
	rsync -a --delete --exclude .gitkeep apps/web/dist/ internal/platform/webui/dist/

.PHONY: image
image: ## Build the container image
	docker build -t opsgrid:dev --build-arg COMMIT=$(GIT_COMMIT) .

##@ Evidence

.PHONY: proofs
proofs: ## Run engineering proofs and write docs/evidence/proofs-report.md
	@./scripts/proofs.sh
