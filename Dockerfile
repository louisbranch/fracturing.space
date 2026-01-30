ARG GO_VERSION=1.25.6
FROM golang:${GO_VERSION} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mcp ./cmd/mcp
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/entrypoint ./cmd/entrypoint

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /out/mcp /app/mcp
COPY --from=build /out/entrypoint /app/entrypoint

EXPOSE 8081

ENTRYPOINT ["/app/entrypoint"]
