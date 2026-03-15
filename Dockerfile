ARG GO_VERSION=1.26.0
FROM node:22-bookworm AS node-toolchain

FROM golang:${GO_VERSION} AS base

ARG DEVCONTAINER_UID=1000
ARG DEVCONTAINER_GID=1000
RUN groupadd --gid "${DEVCONTAINER_GID}" vscode \
  && useradd --uid "${DEVCONTAINER_UID}" --gid "${DEVCONTAINER_GID}" --create-home --shell /bin/bash vscode

COPY --from=node-toolchain /usr/local/bin/ /usr/local/bin/
COPY --from=node-toolchain /usr/local/lib/ /usr/local/lib/
COPY --from=node-toolchain /usr/local/include/ /usr/local/include/
COPY --from=node-toolchain /usr/local/share/ /usr/local/share/

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go install github.com/air-verse/air@v1.62.0

FROM node-toolchain AS build-play-ui

WORKDIR /src/internal/services/play/ui

COPY internal/services/play/ui/package.json ./
COPY internal/services/play/ui/package-lock.json ./
COPY internal/services/play/ui/index.html ./
COPY internal/services/play/ui/tsconfig.json ./
COPY internal/services/play/ui/tsconfig.node.json ./
COPY internal/services/play/ui/vite.config.ts ./
COPY internal/services/play/ui/vitest.setup.ts ./
COPY internal/services/play/ui/src ./src

RUN npm ci
RUN npm run build

FROM base AS build-game

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/game ./cmd/game

FROM base AS build-mcp

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp

FROM base AS build-admin

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/admin ./cmd/admin

FROM base AS build-auth

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth ./cmd/auth

FROM base AS build-social

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/social ./cmd/social

FROM base AS build-discovery

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/discovery ./cmd/discovery

FROM base AS build-web

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/web ./cmd/web

FROM base AS build-play

COPY --from=build-play-ui /src/internal/services/play/ui/dist /src/internal/services/play/ui/dist

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/play ./cmd/play

FROM base AS build-ai

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ai ./cmd/ai

FROM base AS build-notifications

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notifications ./cmd/notifications

FROM base AS build-worker

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM base AS build-status

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/status ./cmd/status

FROM base AS build-userhub

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/userhub ./cmd/userhub

FROM gcr.io/distroless/static-debian12:nonroot AS game

WORKDIR /app

COPY --from=build-game /out/game /app/game

EXPOSE 8082

ENTRYPOINT ["/app/game"]

FROM gcr.io/distroless/static-debian12:nonroot AS mcp

WORKDIR /app

COPY --from=build-mcp /out/mcp /app/mcp

EXPOSE 8085

ENTRYPOINT ["/app/mcp"]

FROM gcr.io/distroless/static-debian12:nonroot AS admin

WORKDIR /app

COPY --from=build-admin /out/admin /app/admin

EXPOSE 8081

ENTRYPOINT ["/app/admin"]

FROM gcr.io/distroless/static-debian12:nonroot AS auth

WORKDIR /app

COPY --from=build-auth /out/auth /app/auth

EXPOSE 8083

ENTRYPOINT ["/app/auth"]

FROM gcr.io/distroless/static-debian12:nonroot AS social

WORKDIR /app

COPY --from=build-social /out/social /app/social

EXPOSE 8090

ENTRYPOINT ["/app/social"]

FROM gcr.io/distroless/static-debian12:nonroot AS discovery

WORKDIR /app

COPY --from=build-discovery /out/discovery /app/discovery

EXPOSE 8091

ENTRYPOINT ["/app/discovery"]

FROM gcr.io/distroless/static-debian12:nonroot AS web

WORKDIR /app

COPY --from=build-web /out/web /app/web

EXPOSE 8080

ENTRYPOINT ["/app/web"]

FROM gcr.io/distroless/static-debian12:nonroot AS play

WORKDIR /app

COPY --from=build-play /out/play /app/play

EXPOSE 8094

ENTRYPOINT ["/app/play"]

FROM gcr.io/distroless/static-debian12:nonroot AS ai

WORKDIR /app

COPY --from=build-ai /out/ai /app/ai

EXPOSE 8087

ENTRYPOINT ["/app/ai"]

FROM gcr.io/distroless/static-debian12:nonroot AS notifications

WORKDIR /app

COPY --from=build-notifications /out/notifications /app/notifications

EXPOSE 8088

ENTRYPOINT ["/app/notifications"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker

WORKDIR /app

COPY --from=build-worker /out/worker /app/worker

EXPOSE 8089

ENTRYPOINT ["/app/worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS status

WORKDIR /app

COPY --from=build-status /out/status /app/status

EXPOSE 8093

ENTRYPOINT ["/app/status"]

FROM gcr.io/distroless/static-debian12:nonroot AS userhub

WORKDIR /app

COPY --from=build-userhub /out/userhub /app/userhub

EXPOSE 8092

ENTRYPOINT ["/app/userhub"]
