variable "GO_VERSION" {
  default = "1.25.6"
}

variable "GAME_IMAGE" {
  default = "docker.io/louisbranch/fracturing.space-game:dev"
}

variable "MCP_IMAGE" {
  default = "docker.io/louisbranch/fracturing.space-mcp:dev"
}

variable "ADMIN_IMAGE" {
  default = "docker.io/louisbranch/fracturing.space-admin:dev"
}

group "default" {
  targets = ["game", "mcp", "admin"]
}

target "base" {
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    GO_VERSION = "${GO_VERSION}"
  }
}

target "game" {
  inherits = ["base"]
  target   = "game"
  tags     = ["${GAME_IMAGE}"]
}

target "mcp" {
  inherits = ["base"]
  target   = "mcp"
  tags     = ["${MCP_IMAGE}"]
}

target "admin" {
  inherits = ["base"]
  target   = "admin"
  tags     = ["${ADMIN_IMAGE}"]
}
