variable "GO_VERSION" {
  default = "1.25.6"
}

variable "GRPC_IMAGE" {
  default = "docker.io/louisbranch/duality-engine-grpc:dev"
}

variable "MCP_IMAGE" {
  default = "docker.io/louisbranch/duality-engine-mcp:dev"
}

variable "WEB_IMAGE" {
  default = "docker.io/louisbranch/duality-engine-web:dev"
}

group "default" {
  targets = ["grpc", "mcp", "web"]
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

target "web" {
  inherits = ["base"]
  target   = "web"
  tags     = ["${WEB_IMAGE}"]
}
