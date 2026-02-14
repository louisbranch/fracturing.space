PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := (api/gen/|_templ\.go|internal/services/game/storage/sqlite/db/|internal/services/auth/storage/sqlite/db/|internal/services/admin/storage/sqlite/db/|internal/tools/seed/)

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/auth/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/game/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/systems/daggerheart/v1/*.proto)

.PHONY: all proto clean run cover cover-treemap test integration scenario templ-generate event-catalog-check

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
	go run github.com/a-h/templ/cmd/templ@v0.3.977 generate ./...

clean:
	rm -rf $(GEN_GO_DIR)

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
	  if [ -z "$${FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY:-}" ] || [ -z "$${FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY:-}" ]; then \
	    eval "$$(go run ./cmd/join-grant-key)"; \
	  fi; \
	  export FRACTURING_SPACE_JOIN_GRANT_ISSUER="$${FRACTURING_SPACE_JOIN_GRANT_ISSUER:-fracturing.space/auth}"; \
	  export FRACTURING_SPACE_JOIN_GRANT_AUDIENCE="$${FRACTURING_SPACE_JOIN_GRANT_AUDIENCE:-fracturing.space/game}"; \
	  export FRACTURING_SPACE_JOIN_GRANT_TTL="$${FRACTURING_SPACE_JOIN_GRANT_TTL:-5m}"; \
	  FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=dev-secret go run ./cmd/game 2>&1 & pids+=($$!); \
	  go run ./cmd/auth 2>&1 & pids+=($$!); \
	  go run ./cmd/mcp 2>&1 & pids+=($$!); \
	  go run ./cmd/admin 2>&1 & pids+=($$!); \
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

event-catalog-check:
	@bash -euo pipefail -c 'go generate ./internal/services/game/domain/campaign/event >/dev/null 2>&1; git diff --exit-code -- docs/events/event-catalog.md'

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
