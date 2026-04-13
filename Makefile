.PHONY: dev dev-down build build-agents build-agent-linux build-agent-macos build-agent-windows build-agent-windows-arm64 sign-windows test test-integration lint lint-tools lint-frontend lint-all \
        sqlc proto proto-tools api-client clean migrate migrate-hub migrate-status seed seed-hub setup-hooks tidy fmt \
        ci ci-full ci-quick ci-codegen-check

GOLANGCI_LINT_VERSION := v2.10.1

# ── Development ──────────────────────────────────────────────

# Auto-generate .env with per-user port offsets if missing
.env:
	@echo "No .env found — generating for user $$USER..."
	@./scripts/dev-env.sh

dev: .env
	set -a && . ./.env && set +a && docker compose -f docker-compose.dev.yml up -d
	@if command -v air > /dev/null 2>&1; then \
		set -a && . ./.env && set +a && \
		trap 'kill 0' SIGINT SIGTERM EXIT; \
		air -c .air.toml & \
		air -c .air.agent.toml & \
		air -c .air.hub.toml & \
		wait; \
	else \
		echo "air is not installed. Install it with: go install github.com/air-verse/air@latest"; \
	fi

dev-down: .env
	set -a && . ./.env && set +a && docker compose -f docker-compose.dev.yml down

dev-env:
	@./scripts/dev-env.sh

dev-ports:
	@./scripts/dev-env.sh --print

# ── Build ────────────────────────────────────────────────────

build:
	go build -o bin/server ./cmd/server
	go build -o bin/agent ./cmd/agent
	go build -o bin/hub ./cmd/hub

build-agent-linux:
	@mkdir -p dist/agents
	chmod +x "$(CURDIR)/cmd/agent/dist/install.desktop"
	GOOS=linux GOARCH=amd64 go build -o bin/patchiq-agent-linux-amd64 ./cmd/agent
	tar czf dist/agents/patchiq-agent-linux-amd64.tar.gz -C bin patchiq-agent-linux-amd64 --transform='s/patchiq-agent-linux-amd64/patchiq-agent/' -C "$(CURDIR)/cmd/agent/dist" install.desktop README.txt
	GOOS=linux GOARCH=arm64 go build -o bin/patchiq-agent-linux-arm64 ./cmd/agent
	tar czf dist/agents/patchiq-agent-linux-arm64.tar.gz -C bin patchiq-agent-linux-arm64 --transform='s/patchiq-agent-linux-arm64/patchiq-agent/' -C "$(CURDIR)/cmd/agent/dist" install.desktop README.txt

build-agent-macos:
	GOOS=darwin GOARCH=arm64 go build -o bin/patchiq-agent-darwin-arm64 ./cmd/agent
	GOOS=darwin GOARCH=amd64 go build -o bin/patchiq-agent-darwin-amd64 ./cmd/agent

# Build the Windows agent with the public server address baked in via -ldflags.
# Writes directly into the repo cache dir so the PM UI Agent Downloads page
# picks it up automatically (the server handler scans REPO_CACHE_DIR/windows/
# for patchiq-agent-windows-amd64.exe on every request).
#
# Override SERVER_ADDR on the command line for ad-hoc release builds:
#   make build-agent-windows SERVER_ADDR=patchiq.example.com:3013
SERVER_ADDR ?=
build-agent-windows:
	@if [ -z "$(SERVER_ADDR)" ]; then \
		echo "ERROR: SERVER_ADDR is required. Example:"; \
		echo "  make build-agent-windows SERVER_ADDR=patchiq.example.com:3013"; \
		exit 1; \
	fi
	@mkdir -p $(REPO_CACHE_DIR)/windows bin
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=$(SERVER_ADDR)" \
		-o $(REPO_CACHE_DIR)/windows/patchiq-agent-windows-amd64.exe ./cmd/agent
	@cp $(REPO_CACHE_DIR)/windows/patchiq-agent-windows-amd64.exe bin/patchiq-agent.exe
	@echo "Built patchiq-agent-windows-amd64.exe with server address $(SERVER_ADDR)"
	@echo "  Repo cache: $(REPO_CACHE_DIR)/windows/patchiq-agent-windows-amd64.exe"
	@echo "  Local copy: bin/patchiq-agent.exe"

# Build Windows agent for ARM64 (Surface Pro X, Parallels, etc.)
# NOTE: To embed the admin manifest + version info, install go-winres
#   (go install github.com/tc-hib/go-winres@latest) and run:
#     go-winres make --in cmd/agent/winres/winres.json --out cmd/agent/rsrc
#   before building. The generated .syso file will be picked up automatically.
build-agent-windows-arm64:
	@if [ -z "$(SERVER_ADDR)" ]; then \
		echo "ERROR: SERVER_ADDR is required. Example:"; \
		echo "  make build-agent-windows-arm64 SERVER_ADDR=patchiq.example.com:3013"; \
		exit 1; \
	fi
	@mkdir -p $(REPO_CACHE_DIR)/windows bin
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build \
		-ldflags "-X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=$(SERVER_ADDR)" \
		-o $(REPO_CACHE_DIR)/windows/patchiq-agent-windows-arm64.exe ./cmd/agent
	@echo "Built: $(REPO_CACHE_DIR)/windows/patchiq-agent-windows-arm64.exe"

# Code signing for Windows binaries (requires signtool.exe and a valid certificate).
# Usage: make sign-windows BINARY=path/to/patchiq-agent.exe
BINARY ?=
sign-windows:
	@if [ -z "$(BINARY)" ]; then \
		echo "ERROR: BINARY is required. Example:"; \
		echo "  make sign-windows BINARY=bin/patchiq-agent.exe"; \
		exit 1; \
	fi
	@echo "Signing $(BINARY)..."
	signtool sign /tr http://timestamp.digicert.com /td sha256 /fd sha256 /a "$(BINARY)"

# Cross-compile every agent binary into dist/agents/, which is the
# directory the Patch Manager serves via /api/v1/agent-binaries. Darwin
# builds use CGO for Mach kernel metrics — cross-compiling from Linux
# works without the macOS SDK as long as CGO_ENABLED=0 is acceptable
# (metrics fall back to /proc-style probes on darwin in that mode).
#
# Windows binaries need the server address baked in at build time via
# -ldflags; override SERVER_ADDR on the command line for release builds:
#   make build-agents SERVER_ADDR=patchiq.example.com:3013
# The default targets the local dev gRPC port so `make build-agents`
# works on a fresh checkout.
SERVER_ADDR_DEFAULT := localhost:50451
SERVER_ADDR ?= $(SERVER_ADDR_DEFAULT)
WIN_LDFLAGS := -X github.com/skenzeriq/patchiq/cmd/agent/cli.DefaultServerAddress=$(SERVER_ADDR)
build-agents:
	@echo "Cross-compiling agent binaries into dist/agents/..."
	@mkdir -p dist/agents bin
	chmod +x "$(CURDIR)/cmd/agent/dist/install.desktop"
	# Linux amd64 (tarball with server.txt injected at download time)
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o bin/patchiq-agent-linux-amd64 ./cmd/agent
	tar czf dist/agents/patchiq-agent-linux-amd64.tar.gz -C bin patchiq-agent-linux-amd64 --transform='s/patchiq-agent-linux-amd64/patchiq-agent/' -C "$(CURDIR)/cmd/agent/dist" install.desktop README.txt
	# Linux arm64
	CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o bin/patchiq-agent-linux-arm64 ./cmd/agent
	tar czf dist/agents/patchiq-agent-linux-arm64.tar.gz -C bin patchiq-agent-linux-arm64 --transform='s/patchiq-agent-linux-arm64/patchiq-agent/' -C "$(CURDIR)/cmd/agent/dist" install.desktop README.txt
	# Windows amd64/arm64 — server address baked in via -ldflags
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(WIN_LDFLAGS)" -o dist/agents/patchiq-agent-windows-amd64.exe ./cmd/agent
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "$(WIN_LDFLAGS)" -o dist/agents/patchiq-agent-windows-arm64.exe ./cmd/agent
	# macOS (best-effort; darwin/amd64 and darwin/arm64 require cgo for full
	# metrics, so we skip silently if cross-compile fails without SDK).
	-CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/patchiq-agent-darwin-amd64 ./cmd/agent 2>/dev/null && \
		tar czf dist/agents/patchiq-agent-darwin-amd64.tar.gz -C bin patchiq-agent-darwin-amd64 || \
		echo "  (macOS amd64 build skipped — build natively on macOS for full cgo metrics)"
	-CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/patchiq-agent-darwin-arm64 ./cmd/agent 2>/dev/null && \
		tar czf dist/agents/patchiq-agent-darwin-arm64.tar.gz -C bin patchiq-agent-darwin-arm64 || \
		echo "  (macOS arm64 build skipped — build natively on macOS for full cgo metrics)"
	@echo "Agent binaries staged in dist/agents/ (SERVER_ADDR=$(SERVER_ADDR)):"
	@ls -lh dist/agents/

# ── Test ─────────────────────────────────────────────────────

test:
	go test -race ./...

test-integration:
	go test -race -tags=integration -timeout 15m ./test/integration/...

# ── Lint ─────────────────────────────────────────────────────

lint-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint:
	@command -v golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not found. Run: make lint-tools"; exit 1; }
	@golangci-lint version 2>&1 | grep -q "version 2\." || { echo "golangci-lint v2 required. Run: make lint-tools"; exit 1; }
	golangci-lint run ./...

lint-frontend:
	pnpm -r lint
	pnpm prettier --check "**/*.{ts,tsx,js,jsx,json}" --ignore-path .prettierignore
	pnpm eslint "**/*.{ts,tsx}"

lint-all: lint lint-frontend
	buf lint

# ── Database ─────────────────────────────────────────────────

migrate: .env
	set -a && . ./.env && set +a && go run ./cmd/tools/migrate --db=server up

migrate-hub: .env
	set -a && . ./.env && set +a && go run ./cmd/tools/migrate --db=hub up

migrate-status: .env
	set -a && . ./.env && set +a && go run ./cmd/tools/migrate --db=server status

seed: seed-demo

seed-demo: migrate seed-hub
	set -a && . ./.env && set +a && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_SERVER:-patchiq_dev} --set ON_ERROR_STOP=1 -f scripts/seed.sql && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_SERVER:-patchiq_dev} --set ON_ERROR_STOP=1 -f scripts/seed-dev.sql && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_HUB:-patchiq_hub_dev} --set ON_ERROR_STOP=1 -f scripts/seed-hub.sql

seed-clean: migrate seed-hub
	set -a && . ./.env && set +a && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_SERVER:-patchiq_dev} --set ON_ERROR_STOP=1 -f scripts/seed-clean.sql && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_HUB:-patchiq_hub_dev} --set ON_ERROR_STOP=1 -f scripts/seed-hub.sql

seed-hub: migrate-hub
	@set -a && . ./.env 2>/dev/null && set +a && \
	psql -U $$PGUSER -h $$PGHOST -p $$PGPORT -d $${DB_NAME_HUB:-patchiq_hub_dev} -f scripts/seed-hub.sql; true

seed-agent:
	PATCHIQ_AGENT_SEED=true go run ./cmd/agent --seed-only 2>/dev/null || PATCHIQ_AGENT_SEED=true ./bin/agent --seed-only 2>/dev/null || \
	PATCHIQ_AGENT_SEED=true go run ./cmd/agent &sleep 2 && kill %% 2>/dev/null; true

# ── Codegen ──────────────────────────────────────────────────

sqlc:
	sqlc generate

proto-tools:
	@command -v buf > /dev/null 2>&1 || go install github.com/bufbuild/buf/cmd/buf@v1.66.0

proto:
	@command -v buf > /dev/null 2>&1 || { echo "buf not found. Run: make proto-tools"; exit 1; }
	buf lint
	buf generate

api-client:
	@echo "Validating OpenAPI specs..."
	npx @openapitools/openapi-generator-cli validate -i api/server.yaml
	npx @openapitools/openapi-generator-cli validate -i api/hub.yaml
	@echo "api-client generation not yet configured (deferred to M1)"

# ── Utilities ────────────────────────────────────────────────

setup-hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks path set to .githooks/"

tidy:
	go mod tidy

fmt:
	go fmt ./...

clean:
	rm -rf bin/ tmp/
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && docker compose -f docker-compose.dev.yml down -v

# ── Local CI (mirrors GitHub Actions) ────────────────────────

ci: ## Fast local CI: codegen check + lint + test + build
	@echo "=== Local CI: Codegen Drift Check ==="
	$(MAKE) ci-codegen-check
	@echo "=== Local CI: Lint ==="
	$(MAKE) -j3 ci-go-lint ci-frontend-lint ci-proto-lint
	@echo "=== Local CI: Test ==="
	$(MAKE) -j2 ci-go-test ci-frontend-test
	@echo "=== Local CI: Build ==="
	$(MAKE) -j2 ci-go-build ci-frontend-build
	@echo "=== Local CI: All checks passed ==="

ci-full: ci ## Full local CI: includes integration tests + Docker image builds
	@echo "=== Integration Tests ==="
	$(MAKE) ci-go-test-integration
	@echo "=== Docker Builds ==="
	$(MAKE) -j2 ci-docker-server ci-docker-hub
	@echo "=== Full CI: All checks passed ==="

ci-quick: ## Quick CI: lint + test only changed code since main
	@echo "=== Quick CI: Lint (changed only) ==="
	golangci-lint run --new-from-rev=main ./...
	@echo "=== Quick CI: Test (changed packages) ==="
	@CHANGED=$$(git diff --name-only main -- '*.go' | sed 's|/[^/]*$$||' | sort -u | grep -v '^test/integration' | sed 's|^|./|'); \
	if [ -n "$$CHANGED" ]; then \
		go test -race $$CHANGED; \
	else \
		echo "No changed Go packages to test"; \
	fi
	@echo "=== Quick CI: Passed ==="

ci-codegen-check:
	sqlc generate
	@git diff --exit-code internal/*/store/sqlcgen/ || { echo "ERROR: sqlc generated code is out of date. Run 'make sqlc' and commit the result."; exit 1; }
	buf generate
	@git diff --exit-code gen/ || { echo "ERROR: protobuf generated code is out of date. Run 'make proto' and commit the result."; exit 1; }

ci-go-lint:
	@command -v golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not found. Run: make lint-tools"; exit 1; }
	@golangci-lint version 2>&1 | grep -q "version 2\." || { echo "golangci-lint v2 required. Run: make lint-tools"; exit 1; }
	golangci-lint run ./...

ci-frontend-lint:
	pnpm -r lint
	pnpm prettier --check "**/*.{ts,tsx,js,jsx,json}" --ignore-path .prettierignore
	pnpm eslint "**/*.{ts,tsx}"

ci-proto-lint:
	buf lint

ci-go-test:
	go test -race -count=1 ./...

ci-go-test-integration:
	go test -race -count=1 -tags=integration ./...

ci-frontend-test:
	pnpm -r test

ci-go-build:
	go build -o /dev/null ./cmd/server
	go build -o /dev/null ./cmd/hub
	go build -o /dev/null ./cmd/agent
	go build -o /dev/null ./cmd/tools/migrate
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/agent
	GOOS=windows GOARCH=arm64 go build -o /dev/null ./cmd/agent

ci-frontend-build:
	pnpm -r build

ci-docker-server:
	docker build -f deploy/docker/Dockerfile.server .

ci-docker-hub:
	docker build -f deploy/docker/Dockerfile.hub .
