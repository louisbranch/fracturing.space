package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
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

// evaluateCatalogCapabilityState resolves whether catalog-backed capabilities are
// ready and, when degraded, includes a stable operator-facing reason.
func evaluateCatalogCapabilityState(ctx context.Context, store contentstore.DaggerheartCatalogReadinessStore) catalogCapabilityState {
	readiness, err := contentstore.EvaluateDaggerheartCatalogReadiness(ctx, store)
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

	return startCancelableLoop(ctx, func(workerCtx context.Context) {
		runCatalogAvailabilityMonitor(
			workerCtx,
			s.statusReporter,
			s.stores.content,
			catalogAvailabilityMonitorInterval,
			slogInfof,
		)
	})
}

// runCatalogAvailabilityMonitor re-checks catalog readiness until it becomes
// ready, then exits to avoid long-lived polling overhead.
func runCatalogAvailabilityMonitor(
	ctx context.Context,
	reporter *platformstatus.Reporter,
	store contentstore.DaggerheartCatalogReadinessStore,
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
