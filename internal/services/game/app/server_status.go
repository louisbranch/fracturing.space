package server

import (
	"context"
	"fmt"
	"log"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"google.golang.org/grpc"
)

// dialStatusLenient attempts to connect to the status service.
// On failure it returns nil values — the reporter will accumulate locally.
// The caller must close the returned connection on shutdown.
func dialStatusLenient(addr string) (*grpc.ClientConn, statusv1.StatusServiceClient) {
	if addr == "" {
		return nil, nil
	}
	logf := func(format string, args ...any) {
		log.Printf("status %s", fmt.Sprintf(format, args...))
	}
	conn := platformgrpc.DialLenient(context.Background(), addr, logf)
	if conn == nil {
		log.Printf("status service unavailable; capability reporting disabled")
		return nil, nil
	}
	return conn, statusv1.NewStatusServiceClient(conn)
}

// initStatusReporter creates and configures the game service status reporter.
// It wires capability registrations for the game service's functional areas.
func initStatusReporter(statusClient statusv1.StatusServiceClient, socialAvailable, aiAvailable bool, catalogPopulated bool) *platformstatus.Reporter {
	reporter := platformstatus.NewReporter("game", statusClient)

	reporter.Register("game.campaign.service", platformstatus.Operational)

	if catalogPopulated {
		reporter.Register("game.character.creation", platformstatus.Operational)
		reporter.Register("game.system.daggerheart", platformstatus.Operational)
	} else {
		reporter.Register("game.character.creation", platformstatus.Degraded)
		reporter.Register("game.system.daggerheart", platformstatus.Degraded)
	}

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

// hasCatalogContent checks whether the content store has Daggerheart catalog data.
// An empty catalog means character creation and the game system are degraded.
func hasCatalogContent(store *storagesqlite.Store) bool {
	if store == nil {
		return false
	}
	classes, err := store.ListDaggerheartClasses(context.Background())
	if err != nil {
		log.Printf("catalog content check failed: %v", err)
		return false
	}
	return len(classes) > 0
}

// startStatusReporter launches the background push loop if a reporter is configured.
func (s *Server) startStatusReporter(ctx context.Context) func() {
	if s == nil || s.statusReporter == nil {
		return func() {}
	}
	return s.statusReporter.Start(ctx)
}
