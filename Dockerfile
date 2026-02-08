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
