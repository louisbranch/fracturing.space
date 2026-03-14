package cmd

import (
	"context"
	"testing"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"google.golang.org/grpc"
)

type fakeReporterConn struct {
	closeCalls int
}

func (c *fakeReporterConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return nil
}

func (c *fakeReporterConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func (c *fakeReporterConn) Close() error {
	c.closeCalls++
	return nil
}

type capabilityRegistration struct {
	name   string
	status platformstatus.CapabilityStatus
}

type fakeReporter struct {
	registered []capabilityRegistration
	startCtx   context.Context
	stopCalls  int
}

func (r *fakeReporter) Register(name string, initial platformstatus.CapabilityStatus) {
	r.registered = append(r.registered, capabilityRegistration{
		name:   name,
		status: initial,
	})
}

func (r *fakeReporter) Start(ctx context.Context) func() {
	r.startCtx = ctx
	return func() {
		r.stopCalls++
	}
}

func TestStartStatusReporterRegistersCapabilitiesAndStops(t *testing.T) {
	restore := stubStatusReporterDeps(t)
	defer restore()

	conn := &fakeReporterConn{}
	reporter := &fakeReporter{}
	dialCalled := false
	clientCalled := false

	dialReporterConn = func(ctx context.Context, addr string, logf func(string, ...any)) reporterConn {
		dialCalled = true
		if ctx == nil {
			t.Fatal("dial context is nil")
		}
		if addr != "status:8087" {
			t.Fatalf("dial addr = %q, want %q", addr, "status:8087")
		}
		if logf == nil {
			t.Fatal("expected log function")
		}
		return conn
	}
	newStatusClient = func(gconn grpc.ClientConnInterface) statusv1.StatusServiceClient {
		clientCalled = true
		if gconn != conn {
			t.Fatal("unexpected status client connection")
		}
		return nil
	}
	newStatusReporter = func(service string, client statusv1.StatusServiceClient) statusReporter {
		if service != "notifications" {
			t.Fatalf("service = %q, want notifications", service)
		}
		if !clientCalled {
			t.Fatal("status client should be created before reporter")
		}
		return reporter
	}

	stop := StartStatusReporter(
		nil,
		"notifications",
		"status:8087",
		Capability(" notifications.inbox ", platformstatus.Operational),
		Capability("", platformstatus.Degraded),
		Capability("notifications.worker", platformstatus.Maintenance),
	)

	if !dialCalled {
		t.Fatal("expected reporter dial")
	}
	if len(reporter.registered) != 2 {
		t.Fatalf("registered capabilities = %d, want 2", len(reporter.registered))
	}
	if reporter.registered[0].name != "notifications.inbox" {
		t.Fatalf("first capability name = %q, want notifications.inbox", reporter.registered[0].name)
	}
	if reporter.registered[1].name != "notifications.worker" {
		t.Fatalf("second capability name = %q, want notifications.worker", reporter.registered[1].name)
	}

	stop()

	if reporter.stopCalls != 1 {
		t.Fatalf("reporter stop calls = %d, want 1", reporter.stopCalls)
	}
	if conn.closeCalls != 1 {
		t.Fatalf("connection close calls = %d, want 1", conn.closeCalls)
	}
}

func TestStartStatusReporterSkipsClientWhenDialFails(t *testing.T) {
	restore := stubStatusReporterDeps(t)
	defer restore()

	reporter := &fakeReporter{}
	clientCalled := false
	dialReporterConn = func(context.Context, string, func(string, ...any)) reporterConn {
		return nil
	}
	newStatusClient = func(grpc.ClientConnInterface) statusv1.StatusServiceClient {
		clientCalled = true
		return nil
	}
	newStatusReporter = func(service string, client statusv1.StatusServiceClient) statusReporter {
		if client != nil {
			t.Fatal("expected nil status client when dial fails")
		}
		return reporter
	}

	stop := StartStatusReporter(context.Background(), "chat", "status:8087")
	stop()

	if clientCalled {
		t.Fatal("status client should not be created without connection")
	}
	if reporter.startCtx == nil {
		t.Fatal("expected reporter start context")
	}
}

func TestStartStatusReporterUsesProvidedContext(t *testing.T) {
	restore := stubStatusReporterDeps(t)
	defer restore()

	type contextKey string
	const key contextKey = "k"
	ctx := context.WithValue(context.Background(), key, "v")

	reporter := &fakeReporter{}
	dialReporterConn = func(dialCtx context.Context, addr string, logf func(string, ...any)) reporterConn {
		if dialCtx != ctx {
			t.Fatal("expected provided context in dial")
		}
		return nil
	}
	newStatusReporter = func(service string, client statusv1.StatusServiceClient) statusReporter {
		return reporter
	}

	stop := StartStatusReporter(ctx, "worker", "status:8087")
	stop()

	if reporter.startCtx != ctx {
		t.Fatal("expected provided context in reporter start")
	}
}

func TestStartStatusReporterSkipsDialWhenStatusAddrBlank(t *testing.T) {
	restore := stubStatusReporterDeps(t)
	defer restore()

	reporter := &fakeReporter{}
	dialCalled := false
	dialReporterConn = func(context.Context, string, func(string, ...any)) reporterConn {
		dialCalled = true
		return nil
	}
	newStatusReporter = func(service string, client statusv1.StatusServiceClient) statusReporter {
		if client != nil {
			t.Fatal("expected nil status client for blank status addr")
		}
		return reporter
	}

	stop := StartStatusReporter(context.Background(), "mcp", "   ")
	stop()

	if dialCalled {
		t.Fatal("expected blank status addr to skip dialing")
	}
}

func stubStatusReporterDeps(t *testing.T) func() {
	t.Helper()
	prevDial := dialReporterConn
	prevNewClient := newStatusClient
	prevNewReporter := newStatusReporter
	return func() {
		dialReporterConn = prevDial
		newStatusClient = prevNewClient
		newStatusReporter = prevNewReporter
	}
}
