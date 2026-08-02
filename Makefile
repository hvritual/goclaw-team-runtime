.PHONY: help makehelp dev dev-sqlite dev-postgres setup-sqlite setup-postgres start-sqlite start-postgres stop-sqlite server server-sqlite server-postgres daemon cli multica build test migrate-up migrate-down sqlc seed clean setup start stop check worktree-env setup-main start-main stop-main check-main setup-worktree start-worktree stop-worktree check-worktree db-up db-down db-reset selfhost selfhost-build selfhost-stop bootstrap generate generated-clean vet-ddd test-race-ddd lint-ddd

MAIN_ENV_FILE ?= .env
WORKTREE_ENV_FILE ?= .env.worktree
ENV_FILE ?= $(if $(wildcard $(MAIN_ENV_FILE)),$(MAIN_ENV_FILE),$(if $(wildcard $(WORKTREE_ENV_FILE)),$(WORKTREE_ENV_FILE),$(MAIN_ENV_FILE)))

ifneq ($(wildcard $(ENV_FILE)),)
include $(ENV_FILE)
endif

POSTGRES_DB ?= multica
POSTGRES_USER ?= multica
POSTGRES_PASSWORD ?= multica
POSTGRES_PORT ?= 5432
PORT := $(or $(BACKEND_PORT),$(API_PORT),$(SERVER_PORT),$(PORT),8080)
FRONTEND_PORT ?= 3000
FRONTEND_ORIGIN ?= http://localhost:$(FRONTEND_PORT)
MULTICA_APP_URL ?= $(FRONTEND_ORIGIN)
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
NEXT_PUBLIC_API_URL ?= http://localhost:$(PORT)
NEXT_PUBLIC_WS_URL ?= ws://localhost:$(PORT)/ws
GOOGLE_REDIRECT_URI ?= $(FRONTEND_ORIGIN)/auth/callback
MULTICA_SERVER_URL ?= ws://localhost:$(PORT)/ws
LOCAL_UPLOAD_BASE_URL ?= http://localhost:$(PORT)
SQLITE_DATABASE_PATH ?= $(CURDIR)/data/multica-local.db
SQLITE_DEV_CODE ?= 888888
SQLITE_LISTEN_ADDRESS ?= 127.0.0.1

export

MULTICA_ARGS ?= $(ARGS)

COMPOSE := docker compose

define REQUIRE_ENV
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "Missing env file: $(ENV_FILE)"; \
		echo "Create .env from .env.example, or run 'make worktree-env' and use .env.worktree."; \
		exit 1; \
	fi
endef

# Self-hosting requires the Docker Compose CLI plugin (`docker compose`).
# The self-host compose files use compose-spec syntax (top-level `name:`, no
# `version:`) that the legacy v1 `docker-compose` standalone cannot parse, so we
# fail early with an actionable message instead of a cryptic CLI parse error
# (e.g. "unknown shorthand flag: 'f' in -f") when the plugin is missing or v1.
# Keep the message short and OS-agnostic: per-OS install steps belong in docs.
define REQUIRE_COMPOSE
	@if ! compose_version=$$($(COMPOSE) version --short 2>/dev/null); then \
		echo "Docker Compose ('docker compose') was not found."; \
		echo "Self-hosting requires the Compose CLI plugin; legacy 'docker-compose' v1 is not supported."; \
		echo "Install Docker Compose from https://docs.docker.com/compose/install/ and verify with: docker compose version"; \
		exit 1; \
	fi; \
	case "$$compose_version" in \
		1.*|v1.*) \
			echo "'$(COMPOSE)' is legacy Docker Compose v1 ($$compose_version)."; \
			echo "Self-hosting requires the Compose CLI plugin; legacy 'docker-compose' v1 is not supported."; \
			echo "Install Docker Compose from https://docs.docker.com/compose/install/ and verify with: docker compose version"; \
			exit 1; \
			;; \
	esac
endef

# Default target changed from selfhost to help: bare `make` now prints this help
# instead of launching a full Docker Compose build, which is safer for onboarding.
.DEFAULT_GOAL := help

##@ Help

help: ## Show available make targets and common local workflows
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nQuick start:\n  \033[36mmake dev\033[0m          Start the SQLite-backed application\n  \033[36mmake dev-postgres\033[0m Start the retained PostgreSQL development path\n  \033[36mmake check\033[0m        Run the full local verification pipeline\n\nSQLite data is stored under \033[36mdata/\033[0m and does not require Docker.\n\n"} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9_.-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

makehelp: help ## Alias for `make help`

# ---------- Self-hosting (Docker Compose) ----------
##@ Self-hosting

selfhost: ## Create .env if needed, then pull and start the official self-hosted images
	$(REQUIRE_COMPOSE)
	@if [ ! -f .env ]; then \
		echo "==> Creating .env from .env.example..."; \
		cp .env.example .env; \
		JWT=$$(openssl rand -hex 32); \
		PGPASS=$$(openssl rand -hex 24); \
		VCSKEY=$$(openssl rand -base64 32); \
		if [ "$$(uname)" = "Darwin" ]; then \
			sed -i '' "s/^JWT_SECRET=.*/JWT_SECRET=$$JWT/" .env; \
			sed -i '' "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$PGPASS/" .env; \
			sed -i '' -E "s#^(DATABASE_URL=postgres://[^:]+:)[^@]*(@.*)#\1$$PGPASS\2#" .env; \
			sed -i '' "s#^MULTICA_VCS_SECRET_KEY=.*#MULTICA_VCS_SECRET_KEY=$$VCSKEY#" .env; \
		else \
			sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$$JWT/" .env; \
			sed -i "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$PGPASS/" .env; \
			sed -i -E "s#^(DATABASE_URL=postgres://[^:]+:)[^@]*(@.*)#\1$$PGPASS\2#" .env; \
			sed -i "s#^MULTICA_VCS_SECRET_KEY=.*#MULTICA_VCS_SECRET_KEY=$$VCSKEY#" .env; \
		fi; \
		echo "==> Generated random JWT_SECRET, POSTGRES_PASSWORD, and MULTICA_VCS_SECRET_KEY"; \
	fi
	@echo "==> Pulling official Multica images..."
	@if ! $(COMPOSE) -f docker-compose.selfhost.yml pull; then \
		echo ""; \
		echo "Official images for tag '$${MULTICA_IMAGE_TAG:-latest}' are not published yet."; \
		echo "If this is before the first GHCR release, build from the current checkout:"; \
		echo "  make selfhost-build"; \
		exit 1; \
	fi
	@echo "==> Starting Multica via Docker Compose..."
	$(COMPOSE) -f docker-compose.selfhost.yml up -d
	@echo "==> Waiting for backend to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -sf http://localhost:$${PORT:-8080}/health > /dev/null 2>&1; then \
			break; \
		fi; \
		sleep 2; \
	done
	@if curl -sf http://localhost:$${PORT:-8080}/health > /dev/null 2>&1; then \
		echo ""; \
		echo "✓ Multica is running!"; \
		echo "  Frontend: http://localhost:$${FRONTEND_PORT:-3000}"; \
		echo "  Backend:  http://localhost:$${PORT:-8080}"; \
		echo ""; \
		echo "Images: $${MULTICA_BACKEND_IMAGE:-ghcr.io/multica-ai/multica-backend}:$${MULTICA_IMAGE_TAG:-latest}"; \
		echo "        $${MULTICA_WEB_IMAGE:-ghcr.io/multica-ai/multica-web}:$${MULTICA_IMAGE_TAG:-latest}"; \
		echo ""; \
		echo "Log in: configure RESEND_API_KEY in .env for email codes,"; \
		echo "        or read the generated code from backend logs when Resend is unset."; \
		echo ""; \
		echo "Next — install the CLI and connect your machine:"; \
		echo "  brew install multica-ai/tap/multica"; \
		echo "  multica setup self-host"; \
	else \
		echo ""; \
		echo "Services are still starting. Check logs:"; \
		echo "  $(COMPOSE) -f docker-compose.selfhost.yml logs"; \
	fi

selfhost-build: ## Build backend/web from the current checkout and start the self-hosted stack
	$(REQUIRE_COMPOSE)
	@if [ ! -f .env ]; then \
		echo "==> Creating .env from .env.example..."; \
		cp .env.example .env; \
		JWT=$$(openssl rand -hex 32); \
		PGPASS=$$(openssl rand -hex 24); \
		VCSKEY=$$(openssl rand -base64 32); \
		if [ "$$(uname)" = "Darwin" ]; then \
			sed -i '' "s/^JWT_SECRET=.*/JWT_SECRET=$$JWT/" .env; \
			sed -i '' "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$PGPASS/" .env; \
			sed -i '' -E "s#^(DATABASE_URL=postgres://[^:]+:)[^@]*(@.*)#\1$$PGPASS\2#" .env; \
			sed -i '' "s#^MULTICA_VCS_SECRET_KEY=.*#MULTICA_VCS_SECRET_KEY=$$VCSKEY#" .env; \
		else \
			sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$$JWT/" .env; \
			sed -i "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$PGPASS/" .env; \
			sed -i -E "s#^(DATABASE_URL=postgres://[^:]+:)[^@]*(@.*)#\1$$PGPASS\2#" .env; \
			sed -i "s#^MULTICA_VCS_SECRET_KEY=.*#MULTICA_VCS_SECRET_KEY=$$VCSKEY#" .env; \
		fi; \
		echo "==> Generated random JWT_SECRET, POSTGRES_PASSWORD, and MULTICA_VCS_SECRET_KEY"; \
	fi
	@echo "==> Building Multica from the current checkout..."
	$(COMPOSE) -f docker-compose.selfhost.yml -f docker-compose.selfhost.build.yml up -d --build
	@echo "==> Waiting for backend to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -sf http://localhost:$${PORT:-8080}/health > /dev/null 2>&1; then \
			break; \
		fi; \
		sleep 2; \
	done
	@if curl -sf http://localhost:$${PORT:-8080}/health > /dev/null 2>&1; then \
		echo ""; \
		echo "✓ Multica is running!"; \
		echo "  Frontend: http://localhost:$${FRONTEND_PORT:-3000}"; \
		echo "  Backend:  http://localhost:$${PORT:-8080}"; \
		echo ""; \
		echo "Log in: configure RESEND_API_KEY in .env for email codes,"; \
		echo "        or read the generated code from backend logs when Resend is unset."; \
		echo ""; \
		echo "Built images locally via docker-compose.selfhost.build.yml."; \
		echo "Local tags: multica-backend:dev and multica-web:dev."; \
		echo ""; \
		echo "Next — install the CLI and connect your machine:"; \
		echo "  brew install multica-ai/tap/multica"; \
		echo "  multica setup self-host"; \
	else \
		echo ""; \
		echo "Services are still starting. Check logs:"; \
		echo "  $(COMPOSE) -f docker-compose.selfhost.yml logs"; \
	fi

selfhost-stop: ## Stop the self-hosted Docker Compose stack
	$(REQUIRE_COMPOSE)
	@echo "==> Stopping Multica services..."
	$(COMPOSE) -f docker-compose.selfhost.yml down
	@echo "✓ All services stopped."

# ---------- One-click commands ----------
##@ One-click

setup-postgres: ## Prepare the retained PostgreSQL development path
	$(REQUIRE_ENV)
	@echo "==> Using env file: $(ENV_FILE)"
	@echo "==> Installing dependencies..."
	pnpm install
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	@echo "==> Running migrations..."
	cd server && go run ./cmd/migrate up
	@echo ""
	@echo "✓ Setup complete! Run 'make start' to launch the app."

start-postgres: ## Start the retained PostgreSQL backend and frontend
	$(REQUIRE_ENV)
	@echo "Using env file: $(ENV_FILE)"
	@echo "Backend: http://localhost:$(PORT)"
	@echo "Frontend: http://localhost:$(FRONTEND_PORT)"
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	@echo "Running migrations..."
	cd server && go run ./cmd/migrate up
	@echo "Starting backend and frontend..."
	@trap 'kill 0' EXIT; \
		(cd server && go run ./cmd/server) & \
		pnpm dev:web & \
		wait

stop: ## Stop backend and frontend processes for the current checkout
	$(REQUIRE_ENV)
	@echo "Stopping services..."
	@-lsof -ti:$(PORT) | xargs kill -9 2>/dev/null
	@-lsof -ti:$(FRONTEND_PORT) | xargs kill -9 2>/dev/null
	@case "$(DATABASE_URL)" in \
		""|*@localhost:*|*@localhost/*|*@127.0.0.1:*|*@127.0.0.1/*|*@\[::1\]:*|*@\[::1\]/*) \
			echo "✓ App processes stopped. Shared PostgreSQL is still running on localhost:$(POSTGRES_PORT)." ;; \
		*) \
			echo "✓ App processes stopped. Remote PostgreSQL was not affected." ;; \
	esac

check: ## Run typecheck, TS tests, Go tests, and Playwright E2E for the current checkout
	$(REQUIRE_ENV)
	@ENV_FILE="$(ENV_FILE)" bash scripts/check.sh

db-up: ## Start the shared PostgreSQL container used by main and worktrees
	@$(COMPOSE) up -d postgres

db-down: ## Stop the shared PostgreSQL container without removing its Docker volume
	@$(COMPOSE) down

# Drop + recreate the current env's database, then run all migrations.
# Use for a clean slate in local dev. Only affects the DB named in
# ENV_FILE (POSTGRES_DB); the shared postgres container and other
# worktree DBs are untouched. Refuses to run against a remote host.
db-reset: ## Drop and recreate the current env's database, then re-run all migrations
	$(REQUIRE_ENV)
	@case "$(DATABASE_URL)" in \
		""|*@localhost:*|*@localhost/*|*@127.0.0.1:*|*@127.0.0.1/*|*@\[::1\]:*|*@\[::1\]/*) ;; \
		*) echo "Refusing to reset: DATABASE_URL points at a remote host."; exit 1 ;; \
	esac
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	@echo "==> Dropping and recreating database '$(POSTGRES_DB)'..."
	@$(COMPOSE) exec -T postgres psql -U $(POSTGRES_USER) -d postgres -v ON_ERROR_STOP=1 \
		-c "DROP DATABASE IF EXISTS \"$(POSTGRES_DB)\" WITH (FORCE);" \
		-c "CREATE DATABASE \"$(POSTGRES_DB)\";"
	@echo "==> Running migrations..."
	cd server && go run ./cmd/migrate up
	@echo ""
	@echo "✓ Database '$(POSTGRES_DB)' reset. Run 'make start' to launch the app."

worktree-env: ## Generate .env.worktree with a unique DB name and app ports for this worktree
	@bash scripts/init-worktree-env.sh .env.worktree

setup-main: ## Prepare the main checkout using .env
	@$(MAKE) setup ENV_FILE=$(MAIN_ENV_FILE)

start-main: ## Start the main checkout using .env
	@$(MAKE) start ENV_FILE=$(MAIN_ENV_FILE)

stop-main: ## Stop the main checkout processes defined by .env
	@$(MAKE) stop ENV_FILE=$(MAIN_ENV_FILE)

check-main: ## Run the full verification pipeline for the main checkout
	@ENV_FILE=$(MAIN_ENV_FILE) bash scripts/check.sh

setup-worktree: ## Ensure .env.worktree exists, then prepare this worktree
	@if [ ! -f "$(WORKTREE_ENV_FILE)" ]; then \
		echo "==> Generating $(WORKTREE_ENV_FILE) with unique ports..."; \
		bash scripts/init-worktree-env.sh $(WORKTREE_ENV_FILE); \
	else \
		echo "==> Using existing $(WORKTREE_ENV_FILE)"; \
	fi
	@$(MAKE) setup ENV_FILE=$(WORKTREE_ENV_FILE)

start-worktree: ## Start this worktree using .env.worktree
	@$(MAKE) start ENV_FILE=$(WORKTREE_ENV_FILE)

stop-worktree: ## Stop this worktree's backend and frontend processes
	@$(MAKE) stop ENV_FILE=$(WORKTREE_ENV_FILE)

check-worktree: ## Run the full verification pipeline for this worktree
	@ENV_FILE=$(WORKTREE_ENV_FILE) bash scripts/check.sh

# ---------- Individual commands ----------
##@ Individual commands

dev: dev-sqlite ## Start the SQLite-backed application (default)

dev-postgres: ## Bootstrap the retained PostgreSQL development path
	@bash scripts/dev.sh

dev-sqlite: setup-sqlite start-sqlite ## Install dependencies and start the local-only SQLite backend and web app

setup-sqlite: ## Prepare dependencies for SQLite local mode (no Docker or PostgreSQL)
	@echo "==> Installing frontend dependencies..."
	pnpm install --frozen-lockfile
	@echo "==> Downloading Go dependencies..."
	cd server && go mod download
	@echo "✓ SQLite local mode is ready."

start-sqlite: ## Start the SQLite local backend and web app
	@echo "SQLite database: $(SQLITE_DATABASE_PATH)"
	@echo "Backend: http://$(SQLITE_LISTEN_ADDRESS):$(PORT)"
	@echo "Frontend: http://localhost:$(FRONTEND_PORT)"
	@echo "Local login code: $(SQLITE_DEV_CODE)"
	@echo "Agent execution and cloud-only endpoints are disabled in this mode."
	@trap 'kill 0' EXIT; \
		(cd server && \
			SQLITE_DATABASE_PATH="$(SQLITE_DATABASE_PATH)" \
			MULTICA_DEV_VERIFICATION_CODE="$(SQLITE_DEV_CODE)" \
			SQLITE_LISTEN_ADDRESS="$(SQLITE_LISTEN_ADDRESS)" \
			FRONTEND_ORIGIN="$(FRONTEND_ORIGIN)" \
			PORT="$(PORT)" \
			go run ./cmd/sqlite-server) & \
		REMOTE_API_URL="http://$(SQLITE_LISTEN_ADDRESS):$(PORT)" \
		NEXT_PUBLIC_API_URL="" \
		NEXT_PUBLIC_WS_URL="" \
		BACKEND_PORT="$(PORT)" \
		pnpm dev:web & \
		wait

stop-sqlite: ## Stop the SQLite local backend and web app
	@echo "Stopping SQLite local services..."
	@-lsof -ti:$(PORT) | xargs kill -9 2>/dev/null
	@-lsof -ti:$(FRONTEND_PORT) | xargs kill -9 2>/dev/null
	@echo "✓ SQLite data remains at $(SQLITE_DATABASE_PATH)."

server-postgres: ## Run only the retained PostgreSQL Go server
	$(REQUIRE_ENV)
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	cd server && go run ./cmd/server

server-sqlite: ## Run only the local SQLite API server
	cd server && \
		SQLITE_DATABASE_PATH="$(SQLITE_DATABASE_PATH)" \
		MULTICA_DEV_VERIFICATION_CODE="$(SQLITE_DEV_CODE)" \
		SQLITE_LISTEN_ADDRESS="$(SQLITE_LISTEN_ADDRESS)" \
		FRONTEND_ORIGIN="$(FRONTEND_ORIGIN)" \
		PORT="$(PORT)" \
		go run ./cmd/sqlite-server

setup: setup-sqlite ## Prepare the default SQLite-backed application

start: start-sqlite ## Start the default SQLite-backed application

server: server-sqlite ## Run only the default SQLite API server

daemon: ## Restart the local agent daemon using the CLI's stored auth/session
	@$(MAKE) multica MULTICA_ARGS="daemon restart --profile local"

cli: ## Run the multica CLI with ARGS or MULTICA_ARGS from source
	@$(MAKE) multica MULTICA_ARGS="$(MULTICA_ARGS)"

multica: ## Run the multica CLI entrypoint directly from the Go source tree
	cd server && go run -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" ./cmd/multica $(MULTICA_ARGS)

VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

build: ## Build the server, CLI, and migrate binaries into server/bin
	cd server && go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/server ./cmd/server
	cd server && go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/multica ./cmd/multica
	cd server && go build -o bin/migrate ./cmd/migrate

test: ## Run Go tests after ensuring the target DB exists and migrations are applied
	$(REQUIRE_ENV)
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	cd server && go run ./cmd/migrate up
	bash scripts/test-go.sh --race

DDD_BASE_REV ?= HEAD^
DDD_TOOLS_BIN ?= $(CURDIR)/server/bin/ddd-tools
DDD_SCAFFOLD_MODULE ?= github.com/fworld/go-ddd-scaffold
DDD_SCAFFOLD_VERSION ?= v0.0.0-20260802042746-1c5b2054726a
DDDGEN ?= dddgen
PROTOC_GEN_ACCESS ?= protoc-gen-access

bootstrap: ## Install the pinned Proto and externally distributed dddgen tools
	mkdir -p "$(DDD_TOOLS_BIN)"
	GOBIN="$(DDD_TOOLS_BIN)" go install github.com/bufbuild/buf/cmd/buf@v1.61.0
	GOBIN="$(DDD_TOOLS_BIN)" go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	GOBIN="$(DDD_TOOLS_BIN)" go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	GOBIN="$(DDD_TOOLS_BIN)" go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3@v3.0.0-20260626125723-668db92c2c00
	GOBIN="$(DDD_TOOLS_BIN)" go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.1
	@dddgen_path=$$(command -v "$(DDDGEN)") || { echo "dddgen is required; install the externally distributed go-ddd-scaffold tool first"; exit 1; }; cp "$$dddgen_path" "$(DDD_TOOLS_BIN)/dddgen"
	@access_path=$$(command -v "$(PROTOC_GEN_ACCESS)") || { echo "protoc-gen-access is required; install the externally distributed go-ddd-scaffold tool first"; exit 1; }; cp "$$access_path" "$(DDD_TOOLS_BIN)/protoc-gen-access"
	@$(MAKE) verify-ddd-tools

verify-ddd-tools: ## Verify staged dddgen binaries match the pinned module version
	bash scripts/verify-ddd-tools.sh "$(DDD_TOOLS_BIN)" "$(DDD_SCAFFOLD_MODULE)" "$(DDD_SCAFFOLD_VERSION)"

generate: verify-ddd-tools ## Reconcile dddgen services and regenerate Proto, OpenAPI, and access artifacts
	cd server && PATH="$(DDD_TOOLS_BIN):$$PATH" dddgen -root . proto-services
	cd server && PATH="$(DDD_TOOLS_BIN):$$PATH" buf dep update
	cd server && PATH="$(DDD_TOOLS_BIN):$$PATH" buf lint
	cd server && PATH="$(DDD_TOOLS_BIN):$$PATH" buf generate
	cd server && go run ./cmd/postprocess-generated

generated-clean: generate ## Fail when committed dddgen or Proto artifacts are stale
	@changes=$$(git status --porcelain --untracked-files=all -- server/gen server/internal/modules server/tests/contract); \
	if [ -n "$$changes" ]; then echo "Generated artifacts are stale:"; echo "$$changes"; exit 1; fi

vet-ddd: ## Vet the native dddgen modules and architecture/bootstrap packages
	cd server && go vet ./internal/modules/... ./internal/platform/... ./internal/bootstrap ./internal/architecturetest ./tests/contract/...

test-race-ddd: ## Race-test native dddgen modules and their local/gRPC contracts
	cd server && go test -race ./internal/modules/... ./internal/platform/... ./internal/bootstrap ./internal/architecturetest ./tests/contract/...

lint-ddd: ## Run pinned DDD architecture lint plus changed-code coverage
	cd server && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./modules/space/...
	cd server && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --disable=revive ./internal/modules/... ./internal/platform/... ./internal/bootstrap ./internal/architecturetest ./tests/contract/...
	cd server && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --disable=unused ./cmd/server/space.go
	cd server && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --new-from-rev=$(DDD_BASE_REV) ./...

# Database
##@ Database

migrate-up: ## Create the target DB if needed, then apply database migrations
	$(REQUIRE_ENV)
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	cd server && go run ./cmd/migrate up

migrate-down: ## Create the target DB if needed, then roll back database migrations
	$(REQUIRE_ENV)
	@bash scripts/ensure-postgres.sh "$(ENV_FILE)"
	cd server && go run ./cmd/migrate down

sqlc: ## Regenerate sqlc code
	cd server && sqlc generate

# Cleanup
##@ Cleanup

clean: ## Remove build caches, generated binaries, and temp files
	rm -rf server/bin server/tmp
	rm -rf apps/*/.next apps/*/.source apps/*/.expo
	rm -rf apps/*/out apps/*/dist apps/*/dist-electron packages/*/dist
	rm -rf .turbo apps/*/.turbo packages/*/.turbo
	rm -rf apps/*/*.tsbuildinfo packages/*/*.tsbuildinfo
	@echo "✓ Clean complete."
