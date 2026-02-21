PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := (api/gen/|_templ\.go|internal/services/admin/templates/|internal/services/game/storage/sqlite/db/|internal/services/auth/storage/sqlite/db/|internal/services/admin/storage/sqlite/db/|internal/tools/eventdocgen/|cmd/|internal/cmd/)

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/auth/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/ai/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/game/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/systems/daggerheart/v1/*.proto)

.PHONY: all proto clean run up down cover cover-treemap test integration scenario scenario-missing-doc-check templ-generate event-catalog-check fmt fmt-check catalog-importer bootstrap bootstrap-prod setup-hooks

all: proto

proto:
	@mkdir -p $(GEN_GO_DIR)
	protoc \
		-I $(PROTO_DIR) \
		--go_out=$(GEN_GO_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_GO_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

templ-generate:
	mkdir -p .tmp/go-build .tmp/go-cache
	go run github.com/a-h/templ/cmd/templ@v0.3.977 generate ./...

fmt:
	@bash -euo pipefail -c '\
	  if [ -n "$${FILE:-}" ]; then \
	    goimports -w "$$FILE"; \
	  elif [ -n "$${FILES:-}" ]; then \
	    goimports -w $$FILES; \
	  else \
	    goimports -w .; \
	  fi \
	'

fmt-check:
	@bash -euo pipefail -c '\
	  unformatted="$$(goimports -l .)"; \
	  if [ -n "$$unformatted" ]; then \
	    echo "Go files need formatting:"; \
	    printf "%s\n" "$$unformatted"; \
	    exit 1; \
	  fi; \
	  echo "Go formatting check passed." \
	'

clean:
	rm -rf $(GEN_GO_DIR)

up: ## Start watcher-based local services (devcontainer-friendly)
	@bash .devcontainer/scripts/post-start.sh

down: ## Stop watcher-based local services
	@bash .devcontainer/scripts/stop-watch-services.sh

run:
	@bash -euo pipefail -c '\
	  interrupted=0; \
	  pids=(); \
	  cleanup() { \
	    trap - EXIT INT TERM; \
	    for pid in "$$@"; do \
	      if kill -0 "$$pid" 2>/dev/null; then \
	        kill "$$pid" 2>/dev/null || true; \
	      fi; \
	    done; \
	    wait || true; \
	  }; \
	  env_file=".env"; \
	  if [ ! -f "$$env_file" ]; then \
	    cp "$${ENV_EXAMPLE:-.env.local.example}" "$$env_file"; \
	  fi; \
	  mkdir -p .tmp/go-build .tmp/go-cache; \
	  set -a; \
	  . "$$env_file"; \
	    set +a; \
	    if [ -z "$${FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY:-}" ] || [ -z "$${FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY:-}" ]; then \
	    eval "$$(go run ./cmd/join-grant-key)"; \
	    export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY; \
	    export FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY; \
	  fi; \
	  export FRACTURING_SPACE_JOIN_GRANT_ISSUER="$${FRACTURING_SPACE_JOIN_GRANT_ISSUER:-fracturing.space/auth}"; \
	  export FRACTURING_SPACE_JOIN_GRANT_AUDIENCE="$${FRACTURING_SPACE_JOIN_GRANT_AUDIENCE:-fracturing.space/game}"; \
	  export FRACTURING_SPACE_JOIN_GRANT_TTL="$${FRACTURING_SPACE_JOIN_GRANT_TTL:-5m}"; \
	  FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=dev-secret go run ./cmd/game 2>&1 & pids+=($$!); \
	  go run ./cmd/auth 2>&1 & pids+=($$!); \
	  go run ./cmd/mcp 2>&1 & pids+=($$!); \
	  go run ./cmd/admin 2>&1 & pids+=($$!); \
	  go run ./cmd/web 2>&1 & pids+=($$!); \
	  trap "cleanup $${pids[*]}" EXIT; \
	  trap "interrupted=1; cleanup $${pids[*]}" INT TERM; \
	  status=0; \
	  wait || status=$$?; \
	  if [ "$$interrupted" -eq 1 ]; then \
	    exit 0; \
	  fi; \
	  exit $$status \
	'

cover:
	rm -f coverage.raw coverage.out coverage.html coverage-treemap.svg
	@bash -euo pipefail -c 'go test -tags=integration -v -coverpkg=./... -coverprofile=coverage.raw ./... | tee coverage.log'
	awk -v exclude='$(COVER_EXCLUDE_REGEX)' 'NR==1 || $$1 !~ exclude' coverage.raw > coverage.out
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o coverage.html

cover-treemap: cover
	go run github.com/nikolaydubina/go-cover-treemap -coverprofile=coverage.out -percent > coverage-treemap.svg

test:
	go test ./...

integration:
	$(MAKE) event-catalog-check
	go test -tags=integration ./...

scenario:
	go test -tags=scenario ./internal/test/game

scenario-missing-doc-check:
	@bash ./scripts/check-scenario-missing-mechanics.sh

event-catalog-check:
	@bash -euo pipefail -c 'go run ./internal/tools/eventdocgen >/dev/null 2>&1; git diff --exit-code -- docs/events/event-catalog.md docs/events/usage-map.md docs/events/command-catalog.md'

seed: ## Seed the local database with demo data (static fixtures)
	go run ./cmd/seed -v

seed-fresh: ## Reset DB and seed with static fixtures
	rm -f data/game-events.db data/game-projections.db && $(MAKE) seed

seed-generate: ## Generate dynamic demo data
	go run ./cmd/seed -generate -preset=demo -v

seed-variety: ## Generate variety of campaigns across all statuses
	go run ./cmd/seed -generate -preset=variety -v

seed-generate-fresh: ## Reset DB and generate demo data
	rm -f data/game-events.db data/game-projections.db && $(MAKE) seed-generate

catalog-importer: ## Import Daggerheart catalog content
	go run ./cmd/catalog-importer -dir internal/tools/importer/content/daggerheart/v1

bootstrap: ## Generate missing keys and start Compose
	./scripts/bootstrap.sh

bootstrap-prod: ## Bootstrap using .env.production.example
	ENV_EXAMPLE=.env.production.example ./scripts/bootstrap.sh

setup-hooks: ## Configure repository-managed git hooks path
	@bash -euo pipefail -c '\
	  if [ ! -f .githooks/pre-commit ]; then \
	    echo ".githooks/pre-commit not found"; \
	    exit 1; \
	  fi; \
	  chmod +x .githooks/pre-commit; \
	  current="$$(git config --local --get core.hooksPath || true)"; \
	  if [ "$$current" = ".githooks" ]; then \
	    echo "core.hooksPath already configured as .githooks"; \
	    exit 0; \
	  fi; \
	  git config --local core.hooksPath .githooks; \
	  echo "Configured core.hooksPath=.githooks" \
	'
