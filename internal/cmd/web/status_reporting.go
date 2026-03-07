package web

import (
	"context"
	"fmt"
	"log"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
)

// startStatusReporter starts integration capability reporting for startup dependencies.
// It always returns a stop function that must be deferred by callers.
func startStatusReporter(ctx context.Context, statusAddr string, statuses map[string]dependencyStatus) (statusv1.StatusServiceClient, func()) {
	statusConn := platformgrpc.DialLenient(ctx, statusAddr, log.Printf)
	var statusClient statusv1.StatusServiceClient
	if statusConn != nil {
		statusClient = statusv1.NewStatusServiceClient(statusConn)
	}

	reporter := platformstatus.NewReporter("web", statusClient)
	registerDependencyCapabilities(reporter, statuses)
	stopReporter := reporter.Start(ctx)

	stop := func() {
		stopReporter()
		if statusConn == nil {
			return
		}
		if err := statusConn.Close(); err != nil {
			log.Printf("close status connection: %v", err)
		}
	}
	return statusClient, stop
}

// registerDependencyCapabilities keeps capability registration order deterministic.
func registerDependencyCapabilities(reporter *platformstatus.Reporter, statuses map[string]dependencyStatus) {
	if reporter == nil {
		return
	}
	for _, dep := range dependencyOrder {
		capName := fmt.Sprintf("web.%s.integration", dep)
		if status, ok := statuses[dep]; ok && status.State == dependencyDialStateConnected {
			reporter.Register(capName, platformstatus.Operational)
			continue
		}
		reporter.Register(capName, platformstatus.Unavailable)
	}
}
