ARG GO_VERSION=1.26.0
FROM golang:${GO_VERSION} AS base

ARG DEVCONTAINER_UID=1000
ARG DEVCONTAINER_GID=1000
RUN groupadd --gid "${DEVCONTAINER_GID}" vscode \
  && useradd --uid "${DEVCONTAINER_UID}" --gid "${DEVCONTAINER_GID}" --create-home --shell /bin/bash vscode

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go install github.com/air-verse/air@v1.62.0

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

FROM base AS build-listing

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/listing ./cmd/listing

FROM base AS build-web

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/web ./cmd/web

FROM base AS build-chat

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/chat ./cmd/chat

FROM base AS build-ai

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ai ./cmd/ai

FROM base AS build-notifications

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notifications ./cmd/notifications

FROM base AS build-worker

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

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

FROM gcr.io/distroless/static-debian12:nonroot AS listing

WORKDIR /app

COPY --from=build-listing /out/listing /app/listing

EXPOSE 8091

ENTRYPOINT ["/app/listing"]

FROM gcr.io/distroless/static-debian12:nonroot AS web

WORKDIR /app

COPY --from=build-web /out/web /app/web

EXPOSE 8080

ENTRYPOINT ["/app/web"]

FROM gcr.io/distroless/static-debian12:nonroot AS chat

WORKDIR /app

COPY --from=build-chat /out/chat /app/chat

EXPOSE 8086

ENTRYPOINT ["/app/chat"]

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
