PROTO_DIR := api/proto
GEN_GO_DIR := api/gen/go

PROTO_FILES := \
	$(PROTO_DIR)/duality/v1/dice.proto

.PHONY: all proto clean

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

