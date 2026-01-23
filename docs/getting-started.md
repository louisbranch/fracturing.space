# Getting Started

## Prerequisites

- Go 1.21+

## Run locally

Start the gRPC server and MCP bridge together:

```sh
make run
```

This runs the gRPC server on `localhost:8080` and the MCP server on stdio.
The MCP server will wait for the gRPC server to be healthy before accepting requests.

## Run services individually

Start the gRPC server:

```sh
go run ./cmd/server
```

Start the MCP server (requires the gRPC server running):

```sh
go run ./cmd/mcp
```
