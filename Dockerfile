ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS build-grpc

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM base AS build-mcp

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp

FROM base AS build-web

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/web ./cmd/web

FROM gcr.io/distroless/static-debian12:nonroot AS grpc

WORKDIR /app

COPY --from=build-grpc /out/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]

FROM gcr.io/distroless/static-debian12:nonroot AS mcp

WORKDIR /app

COPY --from=build-mcp /out/mcp /app/mcp

EXPOSE 8081

ENTRYPOINT ["/app/mcp"]

FROM gcr.io/distroless/static-debian12:nonroot AS web

WORKDIR /app

COPY --from=build-web /out/web /app/web

EXPOSE 8082

ENTRYPOINT ["/app/web"]
