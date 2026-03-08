package web

import (
	"context"
	"log"
	"strings"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
)

// startStatusService creates a ManagedConn for the status service and
// late-binds the reporter client when the connection becomes healthy.
func startStatusService(
	ctx context.Context,
	statusAddr string,
	reporter *platformstatus.Reporter,
) (*platformgrpc.ManagedConn, statusv1.StatusServiceClient) {
	addr := strings.TrimSpace(statusAddr)
	if addr == "" {
		return nil, nil
	}
	mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "status",
		Addr: addr,
		Mode: platformgrpc.ModeOptional,
		Logf: log.Printf,
	})
	if err != nil {
		log.Printf("web: status managed conn: %v", err)
		return nil, nil
	}

	client := statusv1.NewStatusServiceClient(mc.Conn())

	// Late-bind: once the status service is reachable, attach the client to
	// the reporter so accumulated capabilities flush to the status service.
	go func() {
		if mc.WaitReady(ctx) == nil {
			reporter.SetClient(client)
		}
	}()

	return mc, client
}
