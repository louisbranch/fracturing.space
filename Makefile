PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := api/gen/

PROTO_FILES := \
	$(wildcard $(PROTO_DIR)/common/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/campaign/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/duality/v1/*.proto) \
	$(wildcard $(PROTO_DIR)/session/v1/*.proto)

.PHONY: all proto clean run cover test integration

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

clean:
	rm -rf $(GEN_GO_DIR)

run:
	@bash -euo pipefail -c '\
	  cleanup() { kill -- -$$; } ; trap cleanup EXIT INT TERM; \
	  go run ./cmd/server 2>&1 & \
	  go run ./cmd/mcp 2>&1 & \
	  wait \
	'

cover:
	rm -f coverage.raw coverage.out coverage.html
	@bash -euo pipefail -c 'go test -tags=integration -v -coverpkg=./... -coverprofile=coverage.raw ./... | tee coverage.log'
	awk -v exclude='$(COVER_EXCLUDE_REGEX)' 'NR==1 || $$1 !~ exclude' coverage.raw > coverage.out
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o coverage.html

test:
	go test ./...

integration:
	go test -tags=integration ./...
