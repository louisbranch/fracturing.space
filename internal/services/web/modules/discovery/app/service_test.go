package app

import (
	"context"
	"errors"
	"testing"
)

type gatewayStub struct {
	entries []StarterEntry
	err     error
}

func (g gatewayStub) ListStarterEntries(context.Context) ([]StarterEntry, error) {
	return g.entries, g.err
}

func TestNewServiceWithoutGatewayUsesExplicitDegradedContract(t *testing.T) {
	t.Parallel()

	page := NewService(nil).LoadPage(context.Background())
	if page.Status != PageStatusUnavailable {
		t.Fatalf("Status = %q, want %q", page.Status, PageStatusUnavailable)
	}
}

func TestLoadPageReturnsExplicitDegradedStateOnGatewayError(t *testing.T) {
	t.Parallel()

	page := NewService(gatewayStub{err: errors.New("boom")}).LoadPage(context.Background())
	if page.Status != PageStatusUnavailable {
		t.Fatalf("Status = %q, want %q", page.Status, PageStatusUnavailable)
	}
}

func TestLoadPageTreatsZeroEntriesAsUnavailable(t *testing.T) {
	t.Parallel()

	page := NewService(gatewayStub{}).LoadPage(context.Background())
	if page.Status != PageStatusUnavailable {
		t.Fatalf("Status = %q, want %q", page.Status, PageStatusUnavailable)
	}
}

func TestLoadPageReturnsEntriesWithoutDegradation(t *testing.T) {
	t.Parallel()

	page := NewService(gatewayStub{
		entries: []StarterEntry{{EntryID: "starter:one", Title: "Starter"}},
	}).LoadPage(context.Background())
	if page.Status != PageStatusReady {
		t.Fatalf("Status = %q, want %q", page.Status, PageStatusReady)
	}
	if len(page.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(page.Entries))
	}
}

func TestIsGatewayHealthy(t *testing.T) {
	t.Parallel()

	if IsGatewayHealthy(nil) {
		t.Fatal("nil gateway should be unhealthy")
	}
	if IsGatewayHealthy(NewUnavailableGateway()) {
		t.Fatal("unavailable gateway should be unhealthy")
	}
	if !IsGatewayHealthy(gatewayStub{}) {
		t.Fatal("configured gateway should be healthy")
	}
}
