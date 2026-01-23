PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go
COVER_EXCLUDE_REGEX := api/gen/

PROTO_FILES := \
	$(PROTO_DIR)/campaign/v1/campaign.proto \
	$(PROTO_DIR)/duality/v1/duality.proto

.PHONY: all proto clean run cover

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
	  echo "waiting for port 8080..."; \
	  until nc -z 127.0.0.1 8080; do sleep 0.2; done; \
	  go run ./cmd/mcp 2>&1 & \
	  wait \
	'

cover:
	go test -v -coverpkg=./... -coverprofile=coverage.raw ./...
	awk -v exclude='$(COVER_EXCLUDE_REGEX)' 'NR==1 || $$1 !~ exclude' coverage.raw > coverage.out
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o coverage.html
