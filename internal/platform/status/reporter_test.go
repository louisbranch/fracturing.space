package status

import (
	"context"
	"sync"
	"testing"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"google.golang.org/grpc"
)

// fakeClient records ReportStatus calls for testing.
type fakeClient struct {
	mu      sync.Mutex
	reports []*statusv1.ReportStatusRequest
}

func (f *fakeClient) ReportStatus(_ context.Context, req *statusv1.ReportStatusRequest, _ ...grpc.CallOption) (*statusv1.ReportStatusResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reports = append(f.reports, req)
	return &statusv1.ReportStatusResponse{}, nil
}

func (f *fakeClient) GetSystemStatus(_ context.Context, _ *statusv1.GetSystemStatusRequest, _ ...grpc.CallOption) (*statusv1.GetSystemStatusResponse, error) {
	return &statusv1.GetSystemStatusResponse{}, nil
}

func (f *fakeClient) SetOverride(_ context.Context, _ *statusv1.SetOverrideRequest, _ ...grpc.CallOption) (*statusv1.SetOverrideResponse, error) {
	return &statusv1.SetOverrideResponse{}, nil
}

func (f *fakeClient) ClearOverride(_ context.Context, _ *statusv1.ClearOverrideRequest, _ ...grpc.CallOption) (*statusv1.ClearOverrideResponse, error) {
	return &statusv1.ClearOverrideResponse{}, nil
}

func (f *fakeClient) reportCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reports)
}

func (f *fakeClient) lastReport() *statusv1.ReportStatusRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.reports) == 0 {
		return nil
	}
	return f.reports[len(f.reports)-1]
}

func TestReporter_nil_client_safe(t *testing.T) {
	r := NewReporter("game", nil, WithPushInterval(time.Millisecond))
	r.Register("game.service", Operational)

	ctx, cancel := context.WithCancel(context.Background())
	stop := r.Start(ctx)

	// Let a few push cycles pass.
	time.Sleep(10 * time.Millisecond)

	cancel()
	stop()
	// Should not panic.
}

func TestReporter_push_on_set(t *testing.T) {
	client := &fakeClient{}
	r := NewReporter("game", client, WithPushInterval(time.Hour))
	r.Register("game.service", Operational)

	ctx, cancel := context.WithCancel(context.Background())
	stop := r.Start(ctx)

	// Set triggers immediate push.
	r.SetDegraded("game.service", "social down")
	time.Sleep(50 * time.Millisecond)

	if client.reportCount() < 1 {
		t.Fatal("expected at least 1 report after Set")
	}

	last := client.lastReport()
	if last.Report.Service != "game" {
		t.Fatalf("service = %q, want game", last.Report.Service)
	}

	cancel()
	stop()
}

func TestReporter_snapshot(t *testing.T) {
	r := NewReporter("test", nil)
	r.Register("cap.a", Operational)
	r.Register("cap.b", Degraded)

	snap := r.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snap))
	}
}

func TestReporter_concurrent_set(t *testing.T) {
	r := NewReporter("test", nil)
	r.Register("cap.a", Operational)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Set("cap.a", Degraded, "concurrent")
		}()
	}
	wg.Wait()

	snap := r.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot len = %d, want 1", len(snap))
	}
	if snap[0].Status != Degraded {
		t.Fatalf("status = %v, want DEGRADED", snap[0].Status)
	}
}

func TestReporter_convenience_methods(t *testing.T) {
	r := NewReporter("test", nil)

	r.SetOperational("cap.a")
	r.SetDegraded("cap.b", "slow")
	r.SetUnavailable("cap.c", "down")

	snap := r.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("snapshot len = %d, want 3", len(snap))
	}

	statusMap := make(map[string]CapabilityStatus)
	for _, c := range snap {
		statusMap[c.Name] = c.Status
	}

	if statusMap["cap.a"] != Operational {
		t.Fatalf("cap.a = %v, want OPERATIONAL", statusMap["cap.a"])
	}
	if statusMap["cap.b"] != Degraded {
		t.Fatalf("cap.b = %v, want DEGRADED", statusMap["cap.b"])
	}
	if statusMap["cap.c"] != Unavailable {
		t.Fatalf("cap.c = %v, want UNAVAILABLE", statusMap["cap.c"])
	}
}

func TestReporter_periodic_push(t *testing.T) {
	client := &fakeClient{}
	r := NewReporter("game", client, WithPushInterval(10*time.Millisecond))
	r.Register("game.service", Operational)

	ctx, cancel := context.WithCancel(context.Background())
	stop := r.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	if client.reportCount() < 2 {
		t.Fatalf("expected at least 2 periodic pushes, got %d", client.reportCount())
	}

	cancel()
	stop()
}
