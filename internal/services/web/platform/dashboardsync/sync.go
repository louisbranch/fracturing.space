package dashboardsync

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
)

const (
	defaultProjectionWaitTimeout = 5 * time.Second
	slowProjectionWaitThreshold  = 250 * time.Millisecond
	projectionScopeCampaigns     = "campaign_summary"
	projectionScopeSessions      = "campaign_sessions"
	projectionScopeInvites       = "campaign_invites"
)

// UserHubControlClient exposes cache invalidation calls required by dashboard sync.
type UserHubControlClient interface {
	InvalidateDashboards(context.Context, *userhubv1.InvalidateDashboardsRequest, ...grpc.CallOption) (*userhubv1.InvalidateDashboardsResponse, error)
}

// GameEventClient exposes campaign event queries required by projection sync waits.
type GameEventClient interface {
	ListEvents(context.Context, *gamev1.ListEventsRequest, ...grpc.CallOption) (*gamev1.ListEventsResponse, error)
	SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[gamev1.CampaignUpdate], error)
}

// Service exposes the cross-module dashboard refresh hooks used by the web
// service. Callers consume this contract rather than constructing refresh
// helpers themselves.
type Service interface {
	ProfileSaved(context.Context, string)
	CampaignCreated(context.Context, string, string)
	SessionStarted(context.Context, string, string)
	SessionEnded(context.Context, string, string)
	InviteChanged(context.Context, []string, string)
}

// Syncer coordinates dashboard cache invalidation around web mutations.
//
// It is intentionally fail-open: user actions remain successful even when sync
// verification or invalidation cannot complete.
type Syncer struct {
	userhub     UserHubControlClient
	game        GameEventClient
	logger      *slog.Logger
	waitTimeout time.Duration
}

// Noop provides a safe do-nothing dashboard sync implementation for degraded
// startup modes and tests that do not care about refresh hooks.
type Noop struct{}

// New constructs a shared dashboard sync helper for web mutations.
func New(userhub UserHubControlClient, game GameEventClient, logger *slog.Logger) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{
		userhub:     userhub,
		game:        game,
		logger:      logger,
		waitTimeout: defaultProjectionWaitTimeout,
	}
}

// ProfileSaved invalidates cached dashboards for the acting user.
func (s *Syncer) ProfileSaved(ctx context.Context, userID string) {
	s.invalidate(ctx, []string{userID}, nil, "web.profile_saved")
}

// CampaignCreated waits for campaign summary projection visibility, then invalidates.
func (s *Syncer) CampaignCreated(ctx context.Context, userID, campaignID string) {
	s.syncProjectionAndInvalidate(ctx, userID, campaignID, projectionScopeCampaigns, "web.campaign_created")
}

// SessionStarted waits for session projection visibility, then invalidates.
func (s *Syncer) SessionStarted(ctx context.Context, userID, campaignID string) {
	s.syncProjectionAndInvalidate(ctx, userID, campaignID, projectionScopeSessions, "web.session_started")
}

// SessionEnded waits for session projection visibility, then invalidates.
func (s *Syncer) SessionEnded(ctx context.Context, userID, campaignID string) {
	s.syncProjectionAndInvalidate(ctx, userID, campaignID, projectionScopeSessions, "web.session_ended")
}

// InviteChanged waits for invite projection visibility, then invalidates dashboards.
func (s *Syncer) InviteChanged(ctx context.Context, userIDs []string, campaignID string) {
	syncUserID := ""
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID != "" {
			syncUserID = userID
			break
		}
	}
	s.syncProjectionAndInvalidate(ctx, syncUserID, campaignID, projectionScopeInvites, "web.invite_changed")
	if len(normalizedIDs(userIDs)) > 1 {
		s.invalidate(ctx, userIDs, []string{campaignID}, "web.invite_changed")
	}
}

// syncProjectionAndInvalidate coordinates projection waiting with downstream
// dashboard invalidation so readers do not immediately observe stale campaign data.
func (s *Syncer) syncProjectionAndInvalidate(ctx context.Context, userID, campaignID, scope, reason string) {
	userID = strings.TrimSpace(userID)
	campaignID = strings.TrimSpace(campaignID)
	scope = strings.TrimSpace(scope)
	if campaignID == "" {
		s.invalidate(ctx, []string{userID}, nil, reason)
		return
	}
	waitStarted := time.Now()
	if err := s.waitForProjectionApplied(ctx, userID, campaignID, scope); err != nil {
		if s.logger != nil {
			s.logger.Warn(
				"web dashboard sync degraded",
				"reason", reason,
				"campaign_id", campaignID,
				"user_id", userID,
				"scope", scope,
				"wait", time.Since(waitStarted),
				"error", err,
			)
		}
	} else if s.logger != nil {
		waitDuration := time.Since(waitStarted)
		if waitDuration >= slowProjectionWaitThreshold {
			s.logger.Info(
				"web dashboard sync slow",
				"reason", reason,
				"campaign_id", campaignID,
				"user_id", userID,
				"scope", scope,
				"wait", waitDuration,
			)
		}
	}
	s.invalidate(ctx, []string{userID}, []string{campaignID}, reason)
}

// waitForProjectionApplied blocks until the requested campaign projection scope
// has caught up to the latest event sequence visible when the mutation completed.
func (s *Syncer) waitForProjectionApplied(ctx context.Context, userID, campaignID, scope string) error {
	if s == nil || s.game == nil {
		return errors.New("game event client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return errors.New("campaign id is required")
	}
	waitCtx, cancel := context.WithTimeout(ctx, s.projectionWaitTimeout())
	defer cancel()
	waitCtx = grpcauthctx.WithUserID(waitCtx, userID)

	targetSeq, err := latestCampaignSeq(waitCtx, s.game, campaignID)
	if err != nil {
		return err
	}
	if targetSeq == 0 {
		return nil
	}

	stream, err := s.game.SubscribeCampaignUpdates(waitCtx, &gamev1.SubscribeCampaignUpdatesRequest{
		CampaignId:       campaignID,
		AfterSeq:         targetSeq - 1,
		Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
		ProjectionScopes: []string{scope},
	})
	if err != nil {
		return err
	}
	for {
		update, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				return io.EOF
			}
			return recvErr
		}
		if update == nil {
			continue
		}
		if strings.TrimSpace(update.GetCampaignId()) != campaignID {
			continue
		}
		if update.GetProjectionApplied() == nil {
			continue
		}
		if update.GetSeq() >= targetSeq {
			return nil
		}
	}
}

// invalidate forwards the normalized refresh request to userhub when available.
func (s *Syncer) invalidate(ctx context.Context, userIDs []string, campaignIDs []string, reason string) {
	if s == nil || s.userhub == nil {
		if s != nil && s.logger != nil {
			s.logger.Warn(
				"web dashboard invalidation skipped",
				"reason", strings.TrimSpace(reason),
				"error", "userhub control client is not configured",
			)
		}
		return
	}
	req := &userhubv1.InvalidateDashboardsRequest{
		UserIds:     normalizedIDs(userIDs),
		CampaignIds: normalizedIDs(campaignIDs),
		Reason:      strings.TrimSpace(reason),
	}
	if len(req.GetUserIds()) == 0 && len(req.GetCampaignIds()) == 0 {
		return
	}
	callCtx := grpcauthctx.WithServiceID(ctx, "web")
	if _, err := s.userhub.InvalidateDashboards(callCtx, req); err != nil && s.logger != nil {
		s.logger.Warn(
			"web dashboard invalidation failed",
			"reason", req.GetReason(),
			"user_ids", req.GetUserIds(),
			"campaign_ids", req.GetCampaignIds(),
			"error", err,
		)
	}
}

// projectionWaitTimeout supplies a safe default when tests or callers do not override it.
func (s *Syncer) projectionWaitTimeout() time.Duration {
	if s == nil || s.waitTimeout <= 0 {
		return defaultProjectionWaitTimeout
	}
	return s.waitTimeout
}

// latestCampaignSeq reads the newest event sequence so projection waiting can
// subscribe from a stable point-in-time boundary.
func latestCampaignSeq(ctx context.Context, client GameEventClient, campaignID string) (uint64, error) {
	resp, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		return 0, err
	}
	events := resp.GetEvents()
	if len(events) == 0 || events[0] == nil {
		return 0, nil
	}
	return events[0].GetSeq(), nil
}

// normalizedIDs removes blanks before invalidation requests cross service boundaries.
func normalizedIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ProfileSaved keeps the no-op syncer aligned with the shared Service
// contract.
func (Noop) ProfileSaved(context.Context, string) {}

// CampaignCreated keeps the no-op syncer aligned with the shared Service
// contract.
func (Noop) CampaignCreated(context.Context, string, string) {}

// SessionStarted keeps the no-op syncer aligned with the shared Service
// contract.
func (Noop) SessionStarted(context.Context, string, string) {}

// SessionEnded keeps the no-op syncer aligned with the shared Service
// contract.
func (Noop) SessionEnded(context.Context, string, string) {}

// InviteChanged keeps the no-op syncer aligned with the shared Service
// contract.
func (Noop) InviteChanged(context.Context, []string, string) {}
