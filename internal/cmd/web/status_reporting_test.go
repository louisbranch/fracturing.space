package web

import (
	"context"
	"testing"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
)

func TestStartStatusServiceEmptyAddr(t *testing.T) {
	reporter := platformstatus.NewReporter("web", nil)
	mc, client := startStatusService(context.Background(), "", reporter)
	if mc != nil {
		t.Fatal("expected nil ManagedConn for empty address")
	}
	if client != nil {
		t.Fatal("expected nil client for empty address")
	}
}

func TestStartStatusServiceCreatesClient(t *testing.T) {
	stubManagedConn(t)

	reporter := platformstatus.NewReporter("web", nil)
	mc, client := startStatusService(context.Background(), "status:8093", reporter)
	if mc == nil {
		t.Fatal("expected non-nil ManagedConn")
	}
	defer mc.Close()
	if client == nil {
		t.Fatal("expected non-nil StatusServiceClient")
	}
}
