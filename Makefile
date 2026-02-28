PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := (api/gen/|_templ[.]go|internal/services/admin/templates/|internal/services/game/storage/sqlite/db/|internal/services/auth/storage/sqlite/db/|internal/services/admin/storage/sqlite/db/|internal/tools/eventdocgen/|cmd/|internal/cmd/)
SCENARIO_SMOKE_MANIFEST := internal/test/game/scenarios/smoke.txt
INTEGRATION_SMOKE_FULL_PATTERN := ^(TestMCPStdioEndToEnd|TestMCPHTTPBlackbox)$$
INTEGRATION_SMOKE_PR_PATTERN := ^(TestMCPStdioEndToEnd|TestMCPHTTPBlackboxSmoke)$$

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/auth/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/social/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/listing/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/ai/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/game/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/notifications/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/userhub/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/systems/daggerheart/v1/*.proto)

.PHONY: all proto clean up down cover cover-treemap test test-unit test-changed integration integration-full integration-smoke integration-smoke-full integration-smoke-pr integration-shard integration-shard-check scenario scenario-full scenario-smoke scenario-shard scenario-shard-check scenario-fast templ-generate event-catalog-check topology-generate topology-check i18n-check i18n-status i18n-status-check docs-check docs-path-check docs-link-check docs-index-check docs-lifecycle-check negative-test-assertion-check fmt fmt-check catalog-importer bootstrap bootstrap-prod setup-hooks

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
	@if [ -n "$${INTEGRATION_SHARD_TOTAL:-}" ] || [ -n "$${INTEGRATION_SHARD_INDEX:-}" ]; then \
		$(MAKE) integration-shard; \
	else \
		$(MAKE) integration-full; \
	fi

integration-full:
	$(MAKE) event-catalog-check
	$(MAKE) topology-check
	go test -tags=integration ./...

integration-smoke:
	$(MAKE) integration-smoke-pr

integration-smoke-full:
	$(MAKE) event-catalog-check
	$(MAKE) topology-check
	go test -tags=integration ./internal/test/integration -run '$(INTEGRATION_SMOKE_FULL_PATTERN)'

integration-smoke-pr:
	$(MAKE) event-catalog-check
	$(MAKE) topology-check
	go test -tags=integration ./internal/test/integration -run '$(INTEGRATION_SMOKE_PR_PATTERN)'

integration-shard:
	INTEGRATION_SHARD_TOTAL=$${INTEGRATION_SHARD_TOTAL:?set INTEGRATION_SHARD_TOTAL} INTEGRATION_SHARD_INDEX=$${INTEGRATION_SHARD_INDEX:?set INTEGRATION_SHARD_INDEX} bash ./scripts/integration-shard.sh

integration-shard-check:
	INTEGRATION_VERIFY_SHARDS_TOTAL=$${INTEGRATION_VERIFY_SHARDS_TOTAL:?set INTEGRATION_VERIFY_SHARDS_TOTAL} bash ./scripts/integration-shard.sh --check

scenario:
	$(MAKE) scenario-full

scenario-full:
	go test -tags=scenario ./internal/test/game

scenario-smoke:
	SCENARIO_MANIFEST=$(SCENARIO_SMOKE_MANIFEST) go test -tags=scenario ./internal/test/game

scenario-shard:
	SCENARIO_SHARD_TOTAL=$${SCENARIO_SHARD_TOTAL:?set SCENARIO_SHARD_TOTAL} SCENARIO_SHARD_INDEX=$${SCENARIO_SHARD_INDEX:?set SCENARIO_SHARD_INDEX} go test -tags=scenario ./internal/test/game

scenario-shard-check:
	SCENARIO_VERIFY_SHARDS_TOTAL=$${SCENARIO_VERIFY_SHARDS_TOTAL:?set SCENARIO_VERIFY_SHARDS_TOTAL} go test -tags=scenario ./internal/test/game -run '^TestScenarioShardCoverage$$'

scenario-fast:
	SCENARIO_PARALLELISM=$${SCENARIO_PARALLELISM:-4} go test -parallel=$${SCENARIO_PARALLELISM} -tags=scenario ./internal/test/game

docs-check: docs-path-check docs-link-check docs-index-check docs-lifecycle-check

docs-path-check:
	@bash ./scripts/check-doc-paths.sh

docs-link-check:
	@bash ./scripts/check-doc-links.sh

docs-index-check:
	@bash ./scripts/check-doc-index-coverage.sh

docs-lifecycle-check:
	@bash ./scripts/check-doc-lifecycle.sh

negative-test-assertion-check:
	@bash ./scripts/check-negative-test-assertions.sh

event-catalog-check:
	@bash -euo pipefail -c 'go run ./internal/tools/eventdocgen >/dev/null 2>&1; git diff --exit-code -- docs/events/event-catalog.md docs/events/usage-map.md docs/events/command-catalog.md'

topology-generate:
	go run ./internal/tools/topologygen

topology-check:
	go run ./internal/tools/topologygen -check

i18n-check:
	go run ./internal/tools/i18ncheck

i18n-status:
	go run ./internal/tools/i18nstatus

i18n-status-check:
	@bash -euo pipefail -c 'go run ./internal/tools/i18nstatus >/dev/null 2>&1; git diff --exit-code -- docs/reference/i18n-status.md docs/reference/i18n-status.json'

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
