package cmd

import (
	"context"
	"log"
	"strings"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"google.golang.org/grpc"
)

type reporterConn interface {
	grpc.ClientConnInterface
	Close() error
}

type statusReporter interface {
	Register(name string, initial platformstatus.CapabilityStatus)
	Start(ctx context.Context) func()
}

var (
	dialReporterConn = func(ctx context.Context, addr string, logf func(string, ...any)) reporterConn {
		return platformgrpc.DialLenient(ctx, addr, logf)
	}
	newStatusClient = func(conn grpc.ClientConnInterface) statusv1.StatusServiceClient {
		return statusv1.NewStatusServiceClient(conn)
	}
	newStatusReporter = func(service string, client statusv1.StatusServiceClient) statusReporter {
		return platformstatus.NewReporter(service, client)
	}
)

// CapabilityRegistration describes one status capability registration.
type CapabilityRegistration struct {
	Name   string
	Status platformstatus.CapabilityStatus
}

// Capability creates one capability registration.
func Capability(name string, status platformstatus.CapabilityStatus) CapabilityRegistration {
	return CapabilityRegistration{
		Name:   name,
		Status: status,
	}
}

// StartStatusReporter starts a status reporter and returns its cleanup function.
//
// The returned function must always be called to stop the reporter and close the
// optional status client connection.
func StartStatusReporter(
	ctx context.Context,
	service string,
	statusAddr string,
	capabilities ...CapabilityRegistration,
) func() {
	if ctx == nil {
		ctx = context.Background()
	}

	statusConn := dialReporterConn(ctx, statusAddr, log.Printf)
	var statusClient statusv1.StatusServiceClient
	if statusConn != nil {
		statusClient = newStatusClient(statusConn)
	}

	reporter := newStatusReporter(service, statusClient)
	for _, registration := range capabilities {
		name := strings.TrimSpace(registration.Name)
		if name == "" {
			continue
		}
		reporter.Register(name, registration.Status)
	}
	stopReporter := reporter.Start(ctx)

	return func() {
		stopReporter()
		if statusConn != nil {
			if err := statusConn.Close(); err != nil {
				log.Printf("close status connection: %v", err)
			}
		}
	}
}
