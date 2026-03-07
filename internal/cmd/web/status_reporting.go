package web

import (
	"context"
	"log"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
)

// startStatusReporter starts integration capability reporting for startup dependencies.
// It always returns a stop function that must be deferred by callers.
func startStatusReporter(
	ctx context.Context,
	statusAddr string,
	dialTimeout time.Duration,
	requirements []dependencyRequirement,
	statuses map[string]dependencyStatus,
) (statusv1.StatusServiceClient, func()) {
	statusConn := platformgrpc.DialLenientWithTimeout(ctx, statusAddr, dialTimeout, log.Printf)
	var statusClient statusv1.StatusServiceClient
	if statusConn != nil {
		statusClient = statusv1.NewStatusServiceClient(statusConn)
	}

	reporter := platformstatus.NewReporter("web", statusClient)
	registerDependencyCapabilities(reporter, requirements, statuses)
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
func registerDependencyCapabilities(
	reporter *platformstatus.Reporter,
	requirements []dependencyRequirement,
	statuses map[string]dependencyStatus,
) {
	if reporter == nil {
		return
	}
	for _, dep := range requirements {
		if status, ok := statuses[dep.name]; ok && status.State == dependencyDialStateConnected {
			reporter.Register(dep.capability, platformstatus.Operational)
			continue
		}
		reporter.Register(dep.capability, platformstatus.Unavailable)
	}
}
