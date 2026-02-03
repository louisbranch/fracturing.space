variable "GO_VERSION" {
  default = "1.25.6"
}

variable "GRPC_IMAGE" {
  default = "duality-grpc:dev"
}

variable "MCP_IMAGE" {
  default = "duality-mcp:dev"
}

group "default" {
  targets = ["grpc", "mcp"]
}

target "base" {
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    GO_VERSION = "${GO_VERSION}"
  }
}

target "grpc" {
  inherits = ["base"]
  target   = "grpc"
  tags     = ["${GRPC_IMAGE}"]
}

target "mcp" {
  inherits = ["base"]
  target   = "mcp"
  tags     = ["${MCP_IMAGE}"]
}
