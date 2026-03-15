package app

import (
	"context"
	"log/slog"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// StarterEntry is the app-layer discovery card model.
type StarterEntry struct {
	EntryID     string
	Title       string
	Description string
	Tags        []string
	Difficulty  string
	Duration    string
	GmMode      string
	System      string
	Level       int32
	Players     string
}

// PageStatus describes the public discovery page availability contract.
type PageStatus string

const (
	// PageStatusReady means discovery starters were loaded successfully.
	PageStatusReady PageStatus = "ready"
	// PageStatusUnavailable means discovery cannot currently provide a usable curated catalog.
	PageStatusUnavailable PageStatus = "unavailable"
)

// Gateway loads discovery entries from the backing discovery service.
type Gateway interface {
	ListStarterEntries(context.Context) ([]StarterEntry, error)
}

// Page is the explicit discovery-page contract returned to transport.
type Page struct {
	Status  PageStatus
	Entries []StarterEntry
}

// Service orchestrates discovery-page loading.
type Service interface {
	LoadPage(context.Context) Page
}

// service defines an internal contract used at this web package boundary.
type service struct {
	gateway Gateway
	logger  *slog.Logger
}

// unavailableGateway preserves fail-closed gateway behavior while letting the
// service return an explicit degraded page contract.
type unavailableGateway struct{}

// NewUnavailableGateway returns a gateway that always reports unavailable.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// IsGatewayHealthy reports whether a configured discovery gateway is available.
func IsGatewayHealthy(gateway Gateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// NewService constructs a discovery service with explicit degraded-mode policy.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = NewUnavailableGateway()
	}
	return service{
		gateway: gateway,
		logger:  slog.Default(),
	}
}

// LoadPage returns a discovery page contract. Dependency failures are surfaced
// as a degraded page instead of being hidden in handlers.
func (s service) LoadPage(ctx context.Context) Page {
	entries, err := s.gateway.ListStarterEntries(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("discovery starters unavailable", "error", err)
		}
		return Page{Status: PageStatusUnavailable}
	}
	if len(entries) == 0 {
		if s.logger != nil {
			s.logger.Warn("discovery starters unavailable", "reason", "zero_entries")
		}
		return Page{Status: PageStatusUnavailable}
	}
	return Page{
		Status:  PageStatusReady,
		Entries: entries,
	}
}

// ListStarterEntries always returns an unavailable error.
func (unavailableGateway) ListStarterEntries(context.Context) ([]StarterEntry, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "discovery service client is not configured")
}
