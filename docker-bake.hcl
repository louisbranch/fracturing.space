variable "GO_VERSION" {
  default = "1.26.0"
}

variable "REGISTRY" {
  default = "ghcr.io"
}

variable "NAMESPACE" {
  default = "fracturing-space"
}

variable "IMAGE_TAG" {
  default = "dev"
}

group "default" {
  targets = [
    "game",
    "admin",
    "auth",
    "social",
    "discovery",
    "ai",
    "notifications",
    "worker",
    "status",
    "invite",
    "userhub",
    "web",
    "play",
    "caddy",
    "openviking-sidecar",
  ]
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
  tags     = ["${REGISTRY}/${NAMESPACE}/game:${IMAGE_TAG}"]
}

target "admin" {
  inherits = ["base"]
  target   = "admin"
  tags     = ["${REGISTRY}/${NAMESPACE}/admin:${IMAGE_TAG}"]
}

target "auth" {
  inherits = ["base"]
  target   = "auth"
  tags     = ["${REGISTRY}/${NAMESPACE}/auth:${IMAGE_TAG}"]
}

target "social" {
  inherits = ["base"]
  target   = "social"
  tags     = ["${REGISTRY}/${NAMESPACE}/social:${IMAGE_TAG}"]
}

target "discovery" {
  inherits = ["base"]
  target   = "discovery"
  tags     = ["${REGISTRY}/${NAMESPACE}/discovery:${IMAGE_TAG}"]
}

target "ai" {
  inherits = ["base"]
  target   = "ai"
  tags     = ["${REGISTRY}/${NAMESPACE}/ai:${IMAGE_TAG}"]
}

target "notifications" {
  inherits = ["base"]
  target   = "notifications"
  tags     = ["${REGISTRY}/${NAMESPACE}/notifications:${IMAGE_TAG}"]
}

target "worker" {
  inherits = ["base"]
  target   = "worker"
  tags     = ["${REGISTRY}/${NAMESPACE}/worker:${IMAGE_TAG}"]
}

target "status" {
  inherits = ["base"]
  target   = "status"
  tags     = ["${REGISTRY}/${NAMESPACE}/status:${IMAGE_TAG}"]
}

target "invite" {
  inherits = ["base"]
  target   = "invite"
  tags     = ["${REGISTRY}/${NAMESPACE}/invite:${IMAGE_TAG}"]
}

target "userhub" {
  inherits = ["base"]
  target   = "userhub"
  tags     = ["${REGISTRY}/${NAMESPACE}/userhub:${IMAGE_TAG}"]
}

target "web" {
  inherits = ["base"]
  target   = "web"
  tags     = ["${REGISTRY}/${NAMESPACE}/web:${IMAGE_TAG}"]
}

target "play" {
  inherits = ["base"]
  target   = "play"
  tags     = ["${REGISTRY}/${NAMESPACE}/play:${IMAGE_TAG}"]
}

target "caddy" {
  context    = "."
  dockerfile = "docker/caddy/Dockerfile"
  tags       = ["${REGISTRY}/${NAMESPACE}/caddy:${IMAGE_TAG}"]
}

target "openviking-sidecar" {
  context    = "."
  dockerfile = "docker/openviking/Dockerfile"
  tags       = ["${REGISTRY}/${NAMESPACE}/openviking-sidecar:${IMAGE_TAG}"]
}
