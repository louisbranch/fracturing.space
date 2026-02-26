PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := (api/gen/|_templ[.]go|internal/services/admin/templates/|internal/services/game/storage/sqlite/db/|internal/services/auth/storage/sqlite/db/|internal/services/admin/storage/sqlite/db/|internal/tools/eventdocgen/|cmd/|internal/cmd/)

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/auth/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/social/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/listing/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/ai/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/game/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/notifications/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/systems/daggerheart/v1/*.proto)

.PHONY: all proto clean up down cover cover-treemap test test-unit test-changed integration scenario scenario-missing-doc-check templ-generate event-catalog-check topology-generate topology-check docs-path-check negative-test-assertion-check fmt fmt-check catalog-importer bootstrap bootstrap-prod setup-hooks

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
	goimports -w $$(rg --files -g '*_templ.go')

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

up: ## Start devcontainer and watcher-based local services
	@bash .devcontainer/scripts/start-devcontainer.sh

down: ## Stop watcher-based local services and devcontainer
	@bash .devcontainer/scripts/stop-devcontainer.sh

cover:
	rm -f coverage.raw coverage.out coverage.html coverage-treemap.svg coverage.log
	@bash -euo pipefail -c '\
	  mkdir -p .tmp/coverage; \
	  rm -f .tmp/coverage/*.out; \
	  : > coverage.log; \
	  printf "mode: set\n" > coverage.raw; \
	  i=0; total=$$(go list ./... | wc -l); \
	  for pkg in $$(go list ./...); do \
	    i=$$((i + 1)); \
	    if [ $$((i % 25)) -eq 0 ]; then \
	      printf "[cover %d/%d] %s\n" "$$i" "$$total" "$$pkg"; \
	    fi; \
	    profile=$$(printf "%s" "$$pkg" | tr "/" "_" | tr "." "_").out; \
	    go test -tags=integration -covermode=set -coverprofile=.tmp/coverage/"$$profile" "$$pkg" >> coverage.log 2>&1; \
	    awk "FNR > 1 { print }" .tmp/coverage/"$$profile" >> coverage.raw; \
	  done'
	awk -v exclude='$(COVER_EXCLUDE_REGEX)' 'NR==1 || $$1 !~ exclude' coverage.raw > coverage.out
	go tool cover -func coverage.out > coverage.func
	@awk '/^total:/{print}' coverage.func
	go tool cover -html=coverage.out -o coverage.html

cover-treemap: cover
	go run github.com/nikolaydubina/go-cover-treemap -coverprofile=coverage.out -percent > coverage-treemap.svg

test:
	go test ./...

test-unit:
	go test ./...

test-changed:
	@bash ./scripts/test-changed.sh

integration:
	$(MAKE) event-catalog-check
	$(MAKE) topology-check
	go test -tags=integration ./...

scenario:
	go test -tags=scenario ./internal/test/game

scenario-missing-doc-check:
	@bash ./scripts/check-scenario-missing-mechanics.sh

docs-path-check:
	@bash ./scripts/check-doc-paths.sh

negative-test-assertion-check:
	@bash ./scripts/check-negative-test-assertions.sh

event-catalog-check:
	@bash -euo pipefail -c 'go run ./internal/tools/eventdocgen >/dev/null 2>&1; git diff --exit-code -- docs/events/event-catalog.md docs/events/usage-map.md docs/events/command-catalog.md'

topology-generate:
	go run ./internal/tools/topologygen

topology-check:
	go run ./internal/tools/topologygen -check

seed: ## Seed local database with local-dev manifest
	go run ./cmd/seed -manifest=internal/tools/seed/manifests/local-dev.json -v

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
