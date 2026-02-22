variable "GO_VERSION" {
  default = "1.26.0"
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

variable "CONNECTIONS_IMAGE" {
  default = "ghcr.io/fracturing-space/connections:dev"
}

variable "WEB_IMAGE" {
  default = "ghcr.io/fracturing-space/web:dev"
}

variable "NOTIFICATIONS_IMAGE" {
  default = "ghcr.io/fracturing-space/notifications:dev"
}

variable "WORKER_IMAGE" {
  default = "ghcr.io/fracturing-space/worker:dev"
}

group "default" {
  targets = ["game", "mcp", "admin", "auth", "connections", "web", "notifications", "worker"]
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

target "connections" {
  inherits = ["base"]
  target   = "connections"
  tags     = ["${CONNECTIONS_IMAGE}"]
}

target "web" {
  inherits = ["base"]
  target   = "web"
  tags     = ["${WEB_IMAGE}"]
}

target "notifications" {
  inherits = ["base"]
  target   = "notifications"
  tags     = ["${NOTIFICATIONS_IMAGE}"]
}

target "worker" {
  inherits = ["base"]
  target   = "worker"
  tags     = ["${WORKER_IMAGE}"]
}
