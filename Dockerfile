ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp

FROM gcr.io/distroless/static-debian12:nonroot AS grpc

WORKDIR /app

COPY --from=build /out/server /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]

FROM gcr.io/distroless/static-debian12:nonroot AS mcp

WORKDIR /app

COPY --from=build /out/mcp /app/mcp

EXPOSE 8081

ENTRYPOINT ["/app/mcp"]
