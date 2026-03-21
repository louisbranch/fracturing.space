PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := (api/gen/|_templ[.]go|internal/services/admin/templates/|internal/services/game/storage/sqlite/db/|internal/services/auth/storage/sqlite/db/|internal/test/|internal/tools/|cmd/|internal/cmd/)
COVERAGE_FLOORS_FILE ?= docs/reference/coverage-floors.json
CRITICAL_DOMAIN_COVERPKG := ./internal/services/game/domain/action,./internal/services/game/domain/aggregate,./internal/services/game/domain/authz,./internal/services/game/domain/systems,./internal/services/game/domain/systems/daggerheart,./internal/services/game/domain/systems/daggerheart/domain,./internal/services/game/domain/systems/daggerheart/profile,./internal/services/game/domain/systems/daggerheart/internal/mechanics,./internal/services/game/domain/systems/daggerheart/internal/reducer,./internal/services/game/domain/systems/daggerheart/internal/snapstate,./internal/services/game/domain/systems/daggerheart/internal/decider,./internal/services/game/domain/systems/daggerheart/internal/payload,./internal/services/game/domain/systems/daggerheart/internal/validator,./internal/services/game/domain/systems/daggerheart/internal/folder,./internal/services/game/domain/systems/daggerheart/internal/adapter,./internal/services/game/domain/systems/manifest,./internal/services/game/domain/campaign,./internal/services/game/domain/character,./internal/services/game/domain/checkpoint,./internal/services/game/domain/command,./internal/services/game/domain/engine,./internal/services/game/domain/event,./internal/services/game/domain/fork,./internal/services/game/domain/invite,./internal/services/game/domain/journal,./internal/services/game/domain/module,./internal/services/game/domain/participant,./internal/services/game/domain/readiness,./internal/services/game/domain/replay,./internal/services/game/domain/session,./internal/services/game/domain/session/gate,./internal/services/shared/joingrant
CRITICAL_DOMAIN_TEST_PKGS := ./internal/services/game/domain/... ./internal/services/shared/joingrant
SCENARIO_SMOKE_MANIFEST := internal/test/game/scenarios/manifests/smoke.txt
SCENARIO_DEFAULT_PARALLELISM ?= 4
GO_TEST_CACHE_DIR ?= $(CURDIR)/.tmp/go-cache
GO_TEST_TMP_DIR ?= $(CURDIR)/.tmp/go-build
TEST_TMP_ROOT ?= $(CURDIR)/.tmp/test-tmp
TEST_PROGRESS_DIR ?= $(CURDIR)/.tmp/test-status
TEST_PROGRESS_HEARTBEAT ?= 10s
INTEGRATION_COVERAGE_SHARDS ?= 4
INTEGRATION_COVERAGE_PARALLELISM ?= 1
INTEGRATION_SMOKE_FULL_PATTERN := ^(TestMCPStdioEndToEnd|TestMCPHTTPBlackbox)$$
INTEGRATION_SMOKE_PR_PATTERN := ^(TestMCPStdioEndToEnd|TestMCPHTTPBlackboxSmoke)$$

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/auth/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/social/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/discovery/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/ai/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/game/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/notifications/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/userhub/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/systems/daggerheart/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/status/v1/*.proto)

.PHONY: all proto clean up down cover cover-core cover-critical-domain cover-critical-domain-core check-coverage cover-package-floors coverage-floors-ratchet cover-treemap test test-changed smoke smoke-integration smoke-scenario check check-core check-focused check-runtime ci-integration-shard ci-integration-shard-check ci-scenario-shard ci-scenario-shard-check templ-generate event-catalog-check topology-generate topology-check i18n-check i18n-status i18n-status-check docs-check docs-path-check docs-link-check docs-index-check docs-nav-quality-check docs-lifecycle-check docs-web-route-check docs-architecture-budget-check web-architecture-check game-architecture-check admin-architecture-check play-architecture-check web-package-comment-check web-declaration-comment-check web-comment-quality-check web-doc-baseline-update negative-test-assertion-check tool-cli-contract-check tools-check fmt fmt-check catalog-importer bootstrap bootstrap-prod setup-hooks

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
	    files="$$(rg --files -g "*.go" -g "!.tmp/**")"; \
	    if [ -n "$$files" ]; then \
	      goimports -w $$files; \
	    fi; \
	  fi \
	'

fmt-check:
	@bash -euo pipefail -c '\
	  files="$$(rg --files -g "*.go" -g "!.tmp/**")"; \
	  unformatted="$$(goimports -l $$files)"; \
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
	@COVERAGE_LOCK_LABEL='make cover' \
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	bash ./scripts/with-coverage-lock.sh \
		bash ./scripts/with-test-temp.sh cover -- \
		$(MAKE) cover-core

cover-core:
	COVER_EXCLUDE_REGEX='$(COVER_EXCLUDE_REGEX)' \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" \
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	GO_TEST_TMP_DIR="$(GO_TEST_TMP_DIR)" \
	INTEGRATION_COVERAGE_SHARDS="$(INTEGRATION_COVERAGE_SHARDS)" \
	INTEGRATION_COVERAGE_PARALLELISM="$(INTEGRATION_COVERAGE_PARALLELISM)" \
	TEST_PROGRESS_HEARTBEAT="$(TEST_PROGRESS_HEARTBEAT)" \
	bash ./scripts/cover-full.sh

cover-critical-domain:
	@COVERAGE_LOCK_LABEL='make cover-critical-domain' \
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	bash ./scripts/with-coverage-lock.sh \
		bash ./scripts/with-test-temp.sh cover-critical-domain -- \
		$(MAKE) cover-critical-domain-core

cover-critical-domain-core:
	mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	rm -f coverage-critical-domain.out coverage-critical-domain.func
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go test -count=1 -tags=integration -covermode=set -coverpkg=$(CRITICAL_DOMAIN_COVERPKG) -coverprofile=coverage-critical-domain.out $(CRITICAL_DOMAIN_TEST_PKGS)
	go tool cover -func=coverage-critical-domain.out > coverage-critical-domain.func
	@awk '/^total:/{print}' coverage-critical-domain.func

check-coverage:
	@COVERAGE_LOCK_LABEL='make check-coverage' \
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	COVER_EXCLUDE_REGEX='$(COVER_EXCLUDE_REGEX)' \
	bash ./scripts/with-coverage-lock.sh \
		bash ./scripts/with-test-temp.sh check-coverage -- \
		bash ./scripts/check-coverage.sh

cover-package-floors:
	@test -f coverage.out || (echo "coverage.out not found; run 'make cover' first" && exit 1)
	go run ./internal/tools/coveragefloors check -profile=coverage.out -floors=$(COVERAGE_FLOORS_FILE)

coverage-floors-ratchet:
	@test -f coverage.out || (echo "coverage.out not found; run 'make cover' first" && exit 1)
	go run ./internal/tools/coveragefloors ratchet -profile=coverage.out -seed=$(COVERAGE_FLOORS_FILE) -existing=coverage-package-floors.json -out=coverage-package-floors.json

cover-treemap: cover
	go run github.com/nikolaydubina/go-cover-treemap -coverprofile=coverage.out -percent > coverage-treemap.svg

test:
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" TEST_PROGRESS_HEARTBEAT="$(TEST_PROGRESS_HEARTBEAT)" \
	bash ./scripts/with-test-temp.sh test -- \
		bash ./scripts/go-test-progress.sh \
			--label test \
			--status-dir "$(TEST_PROGRESS_DIR)/test" \
			-- \
			go test -json ./...

test-changed:
	@GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	bash ./scripts/with-test-temp.sh test-changed -- \
		bash ./scripts/test-changed.sh

smoke:
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" MAKE="$(MAKE)" \
	bash ./scripts/with-test-temp.sh smoke -- \
		bash ./scripts/smoke.sh

smoke-integration:
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" TEST_PROGRESS_HEARTBEAT="$(TEST_PROGRESS_HEARTBEAT)" \
	bash ./scripts/with-test-temp.sh smoke-integration -- \
		bash ./scripts/go-test-progress.sh \
			--label smoke-integration \
			--status-dir "$(TEST_PROGRESS_DIR)/smoke/integration" \
			-- \
			env INTEGRATION_SHARED_FIXTURE="$${INTEGRATION_SHARED_FIXTURE:-true}" \
			go test -json -tags=integration ./internal/test/integration -run '$(INTEGRATION_SMOKE_PR_PATTERN)'

smoke-scenario:
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" TEST_PROGRESS_HEARTBEAT="$(TEST_PROGRESS_HEARTBEAT)" \
	bash ./scripts/with-test-temp.sh smoke-scenario -- \
		bash ./scripts/go-test-progress.sh \
			--label smoke-scenario \
			--status-dir "$(TEST_PROGRESS_DIR)/smoke/scenario" \
			-- \
			env SCENARIO_MANIFEST="$(SCENARIO_SMOKE_MANIFEST)" \
			go test -json -parallel="$${SCENARIO_PARALLELISM:-$(SCENARIO_DEFAULT_PARALLELISM)}" -tags=scenario ./internal/test/game

check:
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" MAKE="$(MAKE)" \
	bash ./scripts/with-test-temp.sh check -- \
		bash ./scripts/check.sh

check-core:
	$(MAKE) docs-check
	$(MAKE) fmt-check
	$(MAKE) i18n-check
	$(MAKE) i18n-status-check
	$(MAKE) negative-test-assertion-check

check-focused:
	@bash ./scripts/check-focused-gates.sh

check-runtime:
	@echo "[check-runtime] stage 1/3: event-catalog-check"
	$(MAKE) event-catalog-check
	@echo "[check-runtime] stage 2/3: topology-check"
	$(MAKE) topology-check
	@echo "[check-runtime] stage 3/3: scenario"
	GO_TEST_CACHE_DIR="$(GO_TEST_CACHE_DIR)" \
	TEST_TMP_ROOT="$(TEST_TMP_ROOT)" \
	TEST_PROGRESS_DIR="$(TEST_PROGRESS_DIR)" TEST_PROGRESS_HEARTBEAT="$(TEST_PROGRESS_HEARTBEAT)" \
	bash ./scripts/with-test-temp.sh check-runtime -- \
		bash ./scripts/go-test-progress.sh \
			--label check-runtime-scenario \
			--status-dir "$(TEST_PROGRESS_DIR)/check-runtime/scenario" \
			-- \
			go test -json -parallel="$${SCENARIO_PARALLELISM:-$(SCENARIO_DEFAULT_PARALLELISM)}" -tags=scenario ./internal/test/game

ci-integration-shard:
	INTEGRATION_SHARED_FIXTURE=$${INTEGRATION_SHARED_FIXTURE:-true} INTEGRATION_SHARD_TOTAL=$${INTEGRATION_SHARD_TOTAL:?set INTEGRATION_SHARD_TOTAL} INTEGRATION_SHARD_INDEX=$${INTEGRATION_SHARD_INDEX:?set INTEGRATION_SHARD_INDEX} bash ./scripts/integration-shard.sh

ci-integration-shard-check:
	INTEGRATION_VERIFY_SHARDS_TOTAL=$${INTEGRATION_VERIFY_SHARDS_TOTAL:?set INTEGRATION_VERIFY_SHARDS_TOTAL} bash ./scripts/integration-shard.sh --check

ci-scenario-shard:
	@bash -euo pipefail -c ' \
		scenario_parallelism="$${SCENARIO_PARALLELISM:-$(SCENARIO_DEFAULT_PARALLELISM)}"; \
		SCENARIO_SHARD_TOTAL="$${SCENARIO_SHARD_TOTAL:?set SCENARIO_SHARD_TOTAL}" \
		SCENARIO_SHARD_INDEX="$${SCENARIO_SHARD_INDEX:?set SCENARIO_SHARD_INDEX}" \
		go test -parallel="$$scenario_parallelism" -tags=scenario ./internal/test/game \
	'

ci-scenario-shard-check:
	SCENARIO_VERIFY_SHARDS_TOTAL=$${SCENARIO_VERIFY_SHARDS_TOTAL:?set SCENARIO_VERIFY_SHARDS_TOTAL} go test -tags=scenario ./internal/test/game -run '^TestScenarioShardCoverage$$'

docs-check: docs-path-check docs-link-check docs-index-check docs-nav-quality-check docs-lifecycle-check docs-web-route-check docs-architecture-budget-check

docs-path-check:
	@bash ./scripts/check-doc-paths.sh

docs-link-check:
	@bash ./scripts/check-doc-links.sh

docs-index-check:
	@bash ./scripts/check-doc-index-coverage.sh

docs-nav-quality-check:
	@bash ./scripts/check-doc-nav-quality.sh

docs-lifecycle-check:
	@bash ./scripts/check-doc-lifecycle.sh

docs-web-route-check:
	@bash ./scripts/check-web-route-doc-consistency.sh

docs-architecture-budget-check:
	@bash ./scripts/check-architecture-page-budget.sh

web-architecture-check: web-package-comment-check web-declaration-comment-check web-comment-quality-check
	go test ./internal/services/web/modules ./internal/services/web/routepath ./internal/services/web/templates ./internal/services/web

game-architecture-check:
	go test ./internal/services/game/domain/internaltest/contracts
	go test ./internal/services/game/api/grpc/game -run '^TestDirectAppendEventUsageIsRestrictedToMaintenancePaths$$|^TestDirectDomainExecuteUsageIsForbidden$$'
	go test ./internal/services/game/api/grpc/systems/daggerheart -run '^TestDaggerheartHandlersUseSharedDomainWriteHelper$$|^TestDaggerheartWritePathArchitecture$$|^TestDaggerheartArchScanIncludesNonLegacyFiles$$'

admin-architecture-check:
	go test ./internal/services/admin/modules ./internal/services/admin/routepath ./internal/services/admin

play-architecture-check:
	go test ./internal/services/play/app -run '^TestAppPackageDoesNotConstructRuntimeInfrastructure$$'

web-package-comment-check:
	@bash ./scripts/check-web-package-comments.sh

web-declaration-comment-check:
	@bash ./scripts/check-web-declaration-comments.sh

web-comment-quality-check:
	@bash ./scripts/check-web-comment-quality.sh

web-doc-baseline-update:
	@bash ./scripts/update-web-declaration-comment-baseline.sh

negative-test-assertion-check:
	@bash ./scripts/check-negative-test-assertions.sh

tool-cli-contract-check:
	@bash ./scripts/check-tool-cli-contracts.sh

tools-check: tool-cli-contract-check
	go test ./internal/tools/...

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
