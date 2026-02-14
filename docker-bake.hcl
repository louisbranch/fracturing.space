variable "GO_VERSION" {
  default = "1.25.6"
}

variable "GAME_IMAGE" {
  default = "ghcr.io/fracturing-space/game:dev"
}

variable "MCP_IMAGE" {
  default = "ghcr.io/fracturing-space/mcp:dev"
}

variable "ADMIN_IMAGE" {
  default = "ghcr.io/fracturing-space/admin:dev"
}

variable "AUTH_IMAGE" {
  default = "ghcr.io/fracturing-space/auth:dev"
}

variable "WEB_IMAGE" {
  default = "ghcr.io/fracturing-space/web:dev"
}

group "default" {
  targets = ["game", "mcp", "admin", "auth", "web"]
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

target "auth" {
  inherits = ["base"]
  target   = "auth"
  tags     = ["${AUTH_IMAGE}"]
}

target "web" {
  inherits = ["base"]
  target   = "web"
  tags     = ["${WEB_IMAGE}"]
}
