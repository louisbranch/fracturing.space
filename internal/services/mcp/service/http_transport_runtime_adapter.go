package service

import (
	"fmt"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/mcp/httptransport"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
)

// httpTransportRuntimeFactory exposes only the MCP runtime behavior the HTTP
// bridge needs so the transport package does not depend on service internals.
type httpTransportRuntimeFactory struct {
	runtime *Server
}

// BaseServer returns the shared MCP server used when a session does not need a
// dedicated fixed-authority runtime.
func (f httpTransportRuntimeFactory) BaseServer() *mcp.Server {
	if f.runtime == nil {
		return nil
	}
	return f.runtime.mcpServer
}

// NewSessionRuntime creates a dedicated MCP runtime when bridge headers pin
// one HTTP session to one internal AI authority context.
func (f httpTransportRuntimeFactory) NewSessionRuntime(header http.Header) (httptransport.SessionRuntime, error) {
	if f.runtime == nil || f.runtime.gameMc == nil {
		return nil, nil
	}

	var aiConn *grpc.ClientConn
	if f.runtime.aiMc != nil {
		aiConn = f.runtime.aiMc.Conn()
	}

	sessionCtx := mcpbridge.SessionContextFromHeaders(header)
	if !sessionCtx.Valid() && f.runtime.profile == mcpRegistrationProfileStandard {
		return nil, fmt.Errorf("%w: missing required MCP bridge session headers", httptransport.ErrSessionBootstrapRejected)
	}
	if sessionCtx.Valid() {
		server, err := newInternalAISessionServer(f.runtime.gameMc.Conn(), aiConn, sessionCtx)
		if err != nil {
			return nil, err
		}
		return httpTransportSessionRuntime{server: server}, nil
	}

	server, err := newServerWithAIConnProfile(f.runtime.gameMc.Conn(), aiConn, f.runtime.profile, sessionctx.Context{})
	if err != nil {
		return nil, err
	}
	return httpTransportSessionRuntime{server: server}, nil
}

// httpTransportSessionRuntime narrows the service runtime surface to the
// session contract consumed by the HTTP transport package.
type httpTransportSessionRuntime struct {
	server *Server
}

// Server exposes the MCP server bound to this session's authority.
func (r httpTransportSessionRuntime) Server() *mcp.Server {
	if r.server == nil {
		return nil
	}
	return r.server.mcpServer
}

// Close releases the dedicated gRPC connections opened for one session.
func (r httpTransportSessionRuntime) Close() error {
	if r.server == nil {
		return nil
	}
	return r.server.Close()
}
