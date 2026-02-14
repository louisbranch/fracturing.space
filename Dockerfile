ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS build-game

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/game ./cmd/game

FROM base AS build-mcp

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp

FROM base AS build-admin

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/admin ./cmd/admin

FROM base AS build-auth

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth ./cmd/auth

FROM base AS build-web

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/web ./cmd/web

FROM gcr.io/distroless/static-debian12:nonroot AS game

WORKDIR /app

COPY --from=build-game /out/game /app/game

EXPOSE 8080

ENTRYPOINT ["/app/game"]

FROM gcr.io/distroless/static-debian12:nonroot AS mcp

WORKDIR /app

COPY --from=build-mcp /out/mcp /app/mcp

EXPOSE 8081

ENTRYPOINT ["/app/mcp"]

FROM gcr.io/distroless/static-debian12:nonroot AS admin

WORKDIR /app

COPY --from=build-admin /out/admin /app/admin

EXPOSE 8082

ENTRYPOINT ["/app/admin"]

FROM gcr.io/distroless/static-debian12:nonroot AS auth

WORKDIR /app

COPY --from=build-auth /out/auth /app/auth

EXPOSE 8083

ENTRYPOINT ["/app/auth"]

FROM gcr.io/distroless/static-debian12:nonroot AS web

WORKDIR /app

COPY --from=build-web /out/web /app/web

EXPOSE 8086

ENTRYPOINT ["/app/web"]
