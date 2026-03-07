package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
)

const (
	capabilityGameCampaignService   = "game.campaign.service"
	capabilityGameCharacterCreation = "game.character.creation"
	capabilityGameSystemDaggerheart = "game.system.daggerheart"

	catalogAvailabilityMonitorInterval = 2 * time.Second
)

// catalogCapabilityState tracks runtime status for catalog-backed capabilities.
type catalogCapabilityState struct {
	Ready  bool
	Detail string
}

// dialStatusLenient attempts to connect to the status service.
// On failure it returns nil values — the reporter will accumulate locally.
// The caller must close the returned connection on shutdown.
func dialStatusLenient(ctx context.Context, addr string) (*grpc.ClientConn, statusv1.StatusServiceClient) {
	if addr == "" {
		return nil, nil
	}
	logf := func(format string, args ...any) {
		log.Printf("status %s", fmt.Sprintf(format, args...))
	}
	conn := platformgrpc.DialLenient(startupContext(ctx), addr, logf)
	if conn == nil {
		log.Printf("status service unavailable; capability reporting disabled")
		return nil, nil
	}
	return conn, statusv1.NewStatusServiceClient(conn)
}

// initStatusReporter creates and configures the game service status reporter.
// It wires capability registrations for the game service's functional areas.
func initStatusReporter(
	statusClient statusv1.StatusServiceClient,
	socialAvailable, aiAvailable bool,
	catalogState catalogCapabilityState,
) *platformstatus.Reporter {
	reporter := platformstatus.NewReporter("game", statusClient)

	reporter.Register(capabilityGameCampaignService, platformstatus.Operational)
	reporter.Register(capabilityGameCharacterCreation, platformstatus.Degraded)
	reporter.Register(capabilityGameSystemDaggerheart, platformstatus.Degraded)
	applyCatalogCapabilityState(reporter, catalogState)

	if socialAvailable {
		reporter.Register("game.social.integration", platformstatus.Operational)
	} else {
		reporter.Register("game.social.integration", platformstatus.Degraded)
	}

	if aiAvailable {
		reporter.Register("game.ai.integration", platformstatus.Operational)
	} else {
		reporter.Register("game.ai.integration", platformstatus.Degraded)
	}

	return reporter
}

// evaluateCatalogCapabilityState resolves whether catalog-backed capabilities are
// ready and, when degraded, includes a stable operator-facing reason.
func evaluateCatalogCapabilityState(ctx context.Context, store storage.DaggerheartCatalogReadinessStore) catalogCapabilityState {
	readiness, err := storage.EvaluateDaggerheartCatalogReadiness(ctx, store)
	if err != nil {
		return catalogCapabilityState{
			Ready:  false,
			Detail: fmt.Sprintf("catalog readiness check failed: %v", err),
		}
	}
	if readiness.Ready {
		return catalogCapabilityState{Ready: true}
	}
	return catalogCapabilityState{
		Ready:  false,
		Detail: fmt.Sprintf("missing daggerheart catalog sections: %s", strings.Join(readiness.MissingSectionNames(), ", ")),
	}
}

// applyCatalogCapabilityState writes catalog-backed capability status transitions.
func applyCatalogCapabilityState(reporter *platformstatus.Reporter, state catalogCapabilityState) {
	if reporter == nil {
		return
	}
	if state.Ready {
		reporter.SetOperational(capabilityGameCharacterCreation)
		reporter.SetOperational(capabilityGameSystemDaggerheart)
		return
	}
	detail := strings.TrimSpace(state.Detail)
	if detail == "" {
		detail = "catalog content unavailable"
	}
	reporter.SetDegraded(capabilityGameCharacterCreation, detail)
	reporter.SetDegraded(capabilityGameSystemDaggerheart, detail)
}

// startStatusReporter launches the background push loop if a reporter is configured.
func (s *Server) startStatusReporter(ctx context.Context) func() {
	if s == nil || s.statusReporter == nil {
		return func() {}
	}
	return s.statusReporter.Start(ctx)
}

// startCatalogAvailabilityMonitor keeps catalog-backed capability status
// current until catalog content becomes ready.
func (s *Server) startCatalogAvailabilityMonitor(ctx context.Context) func() {
	if s == nil ||
		s.statusReporter == nil ||
		s.stores == nil ||
		s.stores.content == nil ||
		s.catalogReadyAtStartup {
		return func() {}
	}

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runCatalogAvailabilityMonitor(
			workerCtx,
			s.statusReporter,
			s.stores.content,
			catalogAvailabilityMonitorInterval,
			log.Printf,
		)
	}()
	return func() {
		cancel()
		<-done
	}
}

// runCatalogAvailabilityMonitor re-checks catalog readiness until it becomes
// ready, then exits to avoid long-lived polling overhead.
func runCatalogAvailabilityMonitor(
	ctx context.Context,
	reporter *platformstatus.Reporter,
	store storage.DaggerheartCatalogReadinessStore,
	interval time.Duration,
	logf func(string, ...any),
) {
	if reporter == nil || store == nil || interval <= 0 {
		return
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		state := evaluateCatalogCapabilityState(ctx, store)
		applyCatalogCapabilityState(reporter, state)
		if state.Ready {
			logf("catalog availability monitor: catalog is ready; upgraded catalog-backed capabilities")
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
